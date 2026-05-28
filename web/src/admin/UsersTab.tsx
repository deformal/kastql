import { useEffect, useState } from 'react'
import { getUsers, createUser, deleteUser } from './api'
import {
  PageHeader, Card, Table, Empty, ErrorMsg, Spinner,
  Btn, Modal, Field, Input, FormActions, useAsync,
} from './ui'

interface User { id: number; username: string; created_at: string }

export default function UsersTab() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)

  async function load() {
    setLoading(true); setError(null)
    try { setUsers(await getUsers()) }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  async function handleDelete(u: User) {
    if (!confirm(`Delete user "${u.username}"? They will lose playground access immediately.`)) return
    try { await deleteUser(u.id); load() }
    catch (e: any) { alert(e.message) }
  }

  return (
    <div>
      <PageHeader title="Playground Users" action={<Btn onClick={() => setShowAdd(true)}>+ Create User</Btn>} />
      <Card>
        {loading && <Spinner />}
        {error && <ErrorMsg msg={error} />}
        {!loading && !error && !users.length && <Empty msg="No users created yet. Add users to grant playground access." />}
        {!loading && !!users.length && (
          <Table cols={['ID', 'Username', 'Created', '']}>
            {users.map(u => (
              <tr key={u.id}>
                <td style={{ opacity: .5, fontSize: '12px' }}>{u.id}</td>
                <td><strong>{u.username}</strong></td>
                <td style={{ opacity: .6, fontSize: '12px' }}>{new Date(u.created_at).toLocaleString()}</td>
                <td className="adm-actions">
                  <Btn variant="danger" size="sm" onClick={() => handleDelete(u)}>Delete</Btn>
                </td>
              </tr>
            ))}
          </Table>
        )}
      </Card>
      {showAdd && <AddUserModal onClose={() => setShowAdd(false)} onDone={() => { setShowAdd(false); load() }} />}
    </div>
  )
}

function AddUserModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const { loading, error, run, setError } = useAsync()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (password !== confirm) { setError('Passwords do not match'); return }
    await run(async () => {
      await createUser(username, password)
      onDone()
    })
  }

  return (
    <Modal title="Create User" onClose={onClose}>
      <form onSubmit={submit}>
        <Field label="Username">
          <Input value={username} onChange={e => setUsername(e.target.value)} placeholder="alice" required autoFocus />
        </Field>
        <Field label="Password">
          <Input type="password" value={password} onChange={e => setPassword(e.target.value)} required />
        </Field>
        <Field label="Confirm Password">
          <Input type="password" value={confirm} onChange={e => setConfirm(e.target.value)} required />
        </Field>
        {error && <ErrorMsg msg={error} />}
        <FormActions>
          <Btn variant="ghost" onClick={onClose}>Cancel</Btn>
          <Btn disabled={loading}>{loading ? 'Creating…' : 'Create User'}</Btn>
        </FormActions>
      </form>
    </Modal>
  )
}
