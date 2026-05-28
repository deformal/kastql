import { useEffect, useState } from 'react'
import { metaCall } from './api'
import {
  PageHeader, Card, Table, Empty, ErrorMsg, Spinner,
  Btn, Modal, Field, Input, Select, Textarea, FormActions, useAsync,
} from './ui'

interface RESTEndpoint {
  name: string; method: string; path: string; graphql_query: string; variables: Record<string, unknown>
}

const METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']

export default function RestEndpointsTab() {
  const [rows, setRows] = useState<RESTEndpoint[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)

  async function load() {
    setLoading(true); setError(null)
    try {
      const data = await metaCall('export_metadata')
      setRows(data.rest_endpoints ?? [])
    } catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  async function handleDrop(name: string) {
    if (!confirm(`Drop REST endpoint "${name}"?`)) return
    try { await metaCall('drop_rest_endpoint', { args: { name } }); load() }
    catch (e: any) { alert(e.message) }
  }

  return (
    <div>
      <PageHeader title="REST Endpoints" action={<Btn onClick={() => setShowAdd(true)}>+ Add Endpoint</Btn>} />
      <Card>
        {loading && <Spinner />}
        {error && <ErrorMsg msg={error} />}
        {!loading && !error && !rows.length && <Empty msg="No REST endpoints defined yet." />}
        {!loading && !!rows.length && (
          <Table cols={['Name', 'Method', 'Path', 'Query', '']}>
            {rows.map(r => (
              <tr key={r.name}>
                <td><code>{r.name}</code></td>
                <td><span className={`adm-badge adm-badge-method-${r.method.toLowerCase()}`}>{r.method}</span></td>
                <td><code>/api{r.path}</code></td>
                <td className="adm-query-preview">{r.graphql_query.slice(0, 60)}{r.graphql_query.length > 60 ? '…' : ''}</td>
                <td className="adm-actions">
                  <Btn variant="danger" size="sm" onClick={() => handleDrop(r.name)}>Drop</Btn>
                </td>
              </tr>
            ))}
          </Table>
        )}
      </Card>
      {showAdd && <AddEndpointModal onClose={() => setShowAdd(false)} onDone={() => { setShowAdd(false); load() }} />}
    </div>
  )
}

function AddEndpointModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [form, setForm] = useState({ name: '', method: 'GET', path: '', graphql_query: '', variables: '{}' })
  const { loading, error, run } = useAsync()

  function set(k: string, v: string) { setForm(f => ({ ...f, [k]: v })) }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    await run(async () => {
      let variables = {}
      try { variables = JSON.parse(form.variables) } catch { throw new Error('Variables must be valid JSON') }
      await metaCall('create_rest_endpoint', { args: { ...form, variables } })
      onDone()
    })
  }

  return (
    <Modal title="Add REST Endpoint" onClose={onClose}>
      <form onSubmit={submit}>
        <Field label="Name"><Input value={form.name} onChange={e => set('name', e.target.value)} placeholder="get-user" required /></Field>
        <Field label="Method">
          <Select value={form.method} onChange={e => set('method', e.target.value)}>
            {METHODS.map(m => <option key={m}>{m}</option>)}
          </Select>
        </Field>
        <Field label="Path (will be served at /api{path})">
          <Input value={form.path} onChange={e => set('path', e.target.value)} placeholder="/users/:id" required />
        </Field>
        <Field label="GraphQL Query">
          <Textarea value={form.graphql_query} onChange={e => set('graphql_query', e.target.value)} rows={5}
            placeholder={'query GetUser($id: ID!) {\n  user(id: $id) { id name }\n}'} required />
        </Field>
        <Field label="Default Variables (JSON)">
          <Textarea value={form.variables} onChange={e => set('variables', e.target.value)} rows={2} />
        </Field>
        {error && <ErrorMsg msg={error} />}
        <FormActions>
          <Btn variant="ghost" onClick={onClose}>Cancel</Btn>
          <Btn disabled={loading}>{loading ? 'Saving…' : 'Create Endpoint'}</Btn>
        </FormActions>
      </form>
    </Modal>
  )
}
