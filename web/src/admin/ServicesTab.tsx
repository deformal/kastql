import { useEffect, useState } from 'react'
import { metaCall } from './api'
import {
  PageHeader, Card, Table, Empty, ErrorMsg, Spinner,
  Btn, Modal, Field, Input, Select, FormActions, useAsync,
} from './ui'

interface Service {
  name: string; url: string; type: string; headers: Record<string, string>; enabled: boolean
}

export default function ServicesTab() {
  const [services, setServices] = useState<Service[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)

  async function load() {
    setLoading(true)
    setError(null)
    try {
      const data = await metaCall('export_metadata')
      setServices(data.services ?? [])
    } catch (e: any) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  async function handleRemove(name: string) {
    if (!confirm(`Remove service "${name}"?`)) return
    try {
      await metaCall('remove_remote_schema', { args: { name } })
      load()
    } catch (e: any) {
      alert(e.message)
    }
  }

  async function handleReload(name: string) {
    try {
      await metaCall('reload_remote_schema', { args: { name } })
      load()
    } catch (e: any) {
      alert(e.message)
    }
  }

  return (
    <div>
      <PageHeader
        title="Services"
        action={<Btn onClick={() => setShowAdd(true)}>+ Add Service</Btn>}
      />
      <Card>
        {loading && <Spinner />}
        {error && <ErrorMsg msg={error} />}
        {!loading && !error && !services.length && <Empty msg="No services registered yet." />}
        {!loading && !!services.length && (
          <Table cols={['Name', 'URL', 'Type', 'Status', '']}>
            {services.map(s => (
              <tr key={s.name}>
                <td><code>{s.name}</code></td>
                <td className="adm-url">{s.url}</td>
                <td><span className={`adm-badge adm-badge-${s.type}`}>{s.type}</span></td>
                <td>
                  <span className={`adm-badge ${s.enabled ? 'adm-badge-ok' : 'adm-badge-off'}`}>
                    {s.enabled ? 'enabled' : 'disabled'}
                  </span>
                </td>
                <td className="adm-actions">
                  <Btn variant="ghost" size="sm" onClick={() => handleReload(s.name)}>Reload</Btn>
                  <Btn variant="danger" size="sm" onClick={() => handleRemove(s.name)}>Remove</Btn>
                </td>
              </tr>
            ))}
          </Table>
        )}
      </Card>

      {showAdd && <AddServiceModal onClose={() => setShowAdd(false)} onDone={() => { setShowAdd(false); load() }} />}
    </div>
  )
}

function AddServiceModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [form, setForm] = useState({ name: '', url: '', type: 'stitching', headers: '' })
  const { loading, error, run } = useAsync()

  function set(k: string, v: string) { setForm(f => ({ ...f, [k]: v })) }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    await run(async () => {
      let headers = {}
      if (form.headers.trim()) {
        try { headers = JSON.parse(form.headers) } catch { throw new Error('Headers must be valid JSON') }
      }
      await metaCall('add_remote_schema', { args: { name: form.name, url: form.url, type: form.type, headers } })
      onDone()
    })
  }

  return (
    <Modal title="Add Service" onClose={onClose}>
      <form onSubmit={submit}>
        <Field label="Name"><Input value={form.name} onChange={e => set('name', e.target.value)} placeholder="my-service" required /></Field>
        <Field label="URL"><Input value={form.url} onChange={e => set('url', e.target.value)} placeholder="http://service:4000/graphql" required /></Field>
        <Field label="Type">
          <Select value={form.type} onChange={e => set('type', e.target.value)}>
            <option value="stitching">Stitching</option>
            <option value="federation">Federation v2</option>
          </Select>
        </Field>
        <Field label="Headers (JSON, optional)">
          <Input value={form.headers} onChange={e => set('headers', e.target.value)} placeholder='{"Authorization": "Bearer token"}' />
        </Field>
        {error && <ErrorMsg msg={error} />}
        <FormActions>
          <Btn variant="ghost" onClick={onClose}>Cancel</Btn>
          <Btn disabled={loading}>{loading ? 'Adding…' : 'Add Service'}</Btn>
        </FormActions>
      </form>
    </Modal>
  )
}
