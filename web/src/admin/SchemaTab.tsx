import { useEffect, useState } from 'react'
import { getMergedSchema, flushCache } from './api'
import { PageHeader, Card, ErrorMsg, Spinner, Btn } from './ui'

export default function SchemaTab() {
  const [sdl, setSdl] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [flushing, setFlushing] = useState(false)

  async function load() {
    setLoading(true)
    setError(null)
    try {
      setSdl(await getMergedSchema())
    } catch (e: any) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  async function handleFlushCache() {
    setFlushing(true)
    try {
      const res = await flushCache()
      alert(`Cache flushed — ${res.flushed} entries removed.`)
    } catch (e: any) {
      alert(e.message)
    } finally {
      setFlushing(false)
    }
  }

  const lines = sdl.split('\n')
  const filtered = search.trim()
    ? lines.filter(l => l.toLowerCase().includes(search.toLowerCase()))
    : lines

  return (
    <div>
      <PageHeader
        title="Schema"
        action={
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <input
              className="adm-search"
              placeholder="Search schema…"
              value={search}
              onChange={e => setSearch(e.target.value)}
            />
            <Btn variant="ghost" onClick={load}>Refresh</Btn>
            <Btn variant="ghost" onClick={handleFlushCache} disabled={flushing}>
              {flushing ? 'Flushing…' : 'Flush Cache'}
            </Btn>
          </div>
        }
      />
      <Card>
        {loading && <Spinner />}
        {error && <ErrorMsg msg={error} />}
        {!loading && !error && !sdl && (
          <p style={{ color: 'var(--muted)', padding: '16px 0' }}>
            No schema loaded — register at least one service.
          </p>
        )}
        {!loading && !!sdl && (
          <pre className="adm-sdl">
            <code>{filtered.join('\n')}</code>
          </pre>
        )}
      </Card>
    </div>
  )
}
