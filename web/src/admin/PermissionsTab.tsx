import { useEffect, useState } from 'react'
import { metaCall } from './api'
import {
  PageHeader, Card, Table, Empty, ErrorMsg, Spinner,
  Btn, Modal, Field, Input, Select, FormActions, useAsync,
} from './ui'

interface Permission {
  role: string; service: string; type_name: string; field_name: string; allow: boolean
}

export default function PermissionsTab() {
  const [rows, setRows] = useState<Permission[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)

  async function load() {
    setLoading(true); setError(null)
    try {
      const data = await metaCall('export_metadata')
      setRows(data.permissions ?? [])
    } catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  async function handleDrop(p: Permission) {
    if (!confirm(`Drop permission for role "${p.role}" on ${p.service}/${p.type_name}/${p.field_name || '*'}?`)) return
    try {
      await metaCall('drop_permission', { args: { role: p.role, service: p.service, type_name: p.type_name, field_name: p.field_name } })
      load()
    } catch (e: any) { alert(e.message) }
  }

  return (
    <div>
      <PageHeader title="Permissions" action={<Btn onClick={() => setShowAdd(true)}>+ Add Permission</Btn>} />
      <Card>
        {loading && <Spinner />}
        {error && <ErrorMsg msg={error} />}
        {!loading && !error && !rows.length && <Empty msg="No permissions defined. All roles have full access." />}
        {!loading && !!rows.length && (
          <Table cols={['Role', 'Service', 'Type', 'Field', 'Access', '']}>
            {rows.map((p, i) => (
              <tr key={i}>
                <td><code>{p.role}</code></td>
                <td>{p.service}</td>
                <td>{p.type_name}</td>
                <td>{p.field_name || <em style={{ opacity: .5 }}>all fields</em>}</td>
                <td><span className={`adm-badge ${p.allow ? 'adm-badge-ok' : 'adm-badge-deny'}`}>{p.allow ? 'allow' : 'deny'}</span></td>
                <td className="adm-actions">
                  <Btn variant="danger" size="sm" onClick={() => handleDrop(p)}>Drop</Btn>
                </td>
              </tr>
            ))}
          </Table>
        )}
      </Card>
      {showAdd && <AddPermModal onClose={() => setShowAdd(false)} onDone={() => { setShowAdd(false); load() }} />}
    </div>
  )
}

function AddPermModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [form, setForm] = useState({ role: '', service: '', type_name: '', field_name: '', allow: 'true' })
  const { loading, error, run } = useAsync()

  function set(k: string, v: string) { setForm(f => ({ ...f, [k]: v })) }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    await run(async () => {
      await metaCall('create_permission', {
        args: { ...form, field_name: form.field_name || '', allow: form.allow === 'true' },
      })
      onDone()
    })
  }

  return (
    <Modal title="Add Permission" onClose={onClose}>
      <form onSubmit={submit}>
        <Field label="Role"><Input value={form.role} onChange={e => set('role', e.target.value)} placeholder="admin" required /></Field>
        <Field label="Service"><Input value={form.service} onChange={e => set('service', e.target.value)} placeholder="users-service" required /></Field>
        <Field label="Type"><Input value={form.type_name} onChange={e => set('type_name', e.target.value)} placeholder="User" required /></Field>
        <Field label="Field (leave blank for all fields)"><Input value={form.field_name} onChange={e => set('field_name', e.target.value)} placeholder="email" /></Field>
        <Field label="Access">
          <Select value={form.allow} onChange={e => set('allow', e.target.value)}>
            <option value="true">Allow</option>
            <option value="false">Deny</option>
          </Select>
        </Field>
        {error && <ErrorMsg msg={error} />}
        <FormActions>
          <Btn variant="ghost" onClick={onClose}>Cancel</Btn>
          <Btn disabled={loading}>{loading ? 'Saving…' : 'Create Permission'}</Btn>
        </FormActions>
      </form>
    </Modal>
  )
}
