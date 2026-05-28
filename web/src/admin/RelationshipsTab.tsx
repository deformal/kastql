import { useEffect, useState } from 'react'
import { metaCall } from './api'
import {
  PageHeader, Card, Table, Empty, ErrorMsg, Spinner,
  Btn, Modal, Field, Input, Textarea, FormActions, useAsync,
} from './ui'

interface Relationship {
  name: string; source_service: string; source_type: string; source_field: string
  target_service: string; target_type: string; join_config: Record<string, unknown>
}

export default function RelationshipsTab() {
  const [rows, setRows] = useState<Relationship[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)

  async function load() {
    setLoading(true); setError(null)
    try {
      const data = await metaCall('export_metadata')
      setRows(data.relationships ?? [])
    } catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  async function handleRemove(name: string) {
    if (!confirm(`Remove relationship "${name}"?`)) return
    try { await metaCall('remove_relationship', { args: { name } }); load() }
    catch (e: any) { alert(e.message) }
  }

  return (
    <div>
      <PageHeader title="Relationships" action={<Btn onClick={() => setShowAdd(true)}>+ Add Relationship</Btn>} />
      <Card>
        {loading && <Spinner />}
        {error && <ErrorMsg msg={error} />}
        {!loading && !error && !rows.length && <Empty msg="No relationships defined yet." />}
        {!loading && !!rows.length && (
          <Table cols={['Name', 'Source', 'Target', '']}>
            {rows.map(r => (
              <tr key={r.name}>
                <td><code>{r.name}</code></td>
                <td>{r.source_service}.<strong>{r.source_type}</strong>.{r.source_field}</td>
                <td>{r.target_service}.<strong>{r.target_type}</strong></td>
                <td className="adm-actions">
                  <Btn variant="danger" size="sm" onClick={() => handleRemove(r.name)}>Remove</Btn>
                </td>
              </tr>
            ))}
          </Table>
        )}
      </Card>
      {showAdd && <AddRelModal onClose={() => setShowAdd(false)} onDone={() => { setShowAdd(false); load() }} />}
    </div>
  )
}

function AddRelModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [form, setForm] = useState({
    name: '', source_service: '', source_type: '', source_field: '',
    target_service: '', target_type: '', join_config: '{}',
  })
  const { loading, error, run } = useAsync()

  function set(k: string, v: string) { setForm(f => ({ ...f, [k]: v })) }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    await run(async () => {
      let join_config = {}
      try { join_config = JSON.parse(form.join_config) } catch { throw new Error('Join config must be valid JSON') }
      await metaCall('add_relationship', { args: { ...form, join_config } })
      onDone()
    })
  }

  return (
    <Modal title="Add Relationship" onClose={onClose}>
      <form onSubmit={submit}>
        <Field label="Name"><Input value={form.name} onChange={e => set('name', e.target.value)} placeholder="user_orders" required /></Field>
        <Field label="Source Service"><Input value={form.source_service} onChange={e => set('source_service', e.target.value)} placeholder="users-service" required /></Field>
        <Field label="Source Type"><Input value={form.source_type} onChange={e => set('source_type', e.target.value)} placeholder="User" required /></Field>
        <Field label="Source Field"><Input value={form.source_field} onChange={e => set('source_field', e.target.value)} placeholder="orders" required /></Field>
        <Field label="Target Service"><Input value={form.target_service} onChange={e => set('target_service', e.target.value)} placeholder="orders-service" required /></Field>
        <Field label="Target Type"><Input value={form.target_type} onChange={e => set('target_type', e.target.value)} placeholder="Order" required /></Field>
        <Field label="Join Config (JSON)">
          <Textarea value={form.join_config} onChange={e => set('join_config', e.target.value)} rows={3} />
        </Field>
        {error && <ErrorMsg msg={error} />}
        <FormActions>
          <Btn variant="ghost" onClick={onClose}>Cancel</Btn>
          <Btn disabled={loading}>{loading ? 'Adding…' : 'Add Relationship'}</Btn>
        </FormActions>
      </form>
    </Modal>
  )
}
