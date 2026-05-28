import { useEffect, useState } from 'react'
import {
  listJWTSecrets, addJWTSecret, deactivateJWTSecret, type JWTSecret,
  listRouterKeys, createRouterKey, deactivateRouterKey, type RouterKey,
  getSettings, updateSetting,
  listCORSOrigins, addCORSOrigin, deleteCORSOrigin, type CORSOrigin,
  listIPRules, addIPRule, deleteIPRule, type IPRule,
  listPersistedQueries, addPersistedQuery, deletePersistedQuery, type PersistedQuery,
  listAuditLog, type AuditEntry,
  listBlockedRequests, type BlockedRequest,
} from './api'
import {
  PageHeader, Card, Table, Empty, ErrorMsg, Spinner,
  Btn, Modal, Field, Input, Select, Textarea, FormActions, useAsync,
} from './ui'

// ── Toggle helper ─────────────────────────────────────────────────────────────

function SettingToggle({
  settingKey, label, description, onSaved,
}: {
  settingKey: string
  label: string
  description: string
  onSaved?: () => void
}) {
  const [enabled, setEnabled] = useState<boolean | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    getSettings()
      .then(s => setEnabled(s[settingKey] === '1' || s[settingKey] === 'true'))
      .catch(() => setEnabled(false))
  }, [settingKey])

  async function toggle() {
    if (enabled === null) return
    const next = !enabled
    setSaving(true)
    try {
      await updateSetting(settingKey, next ? '1' : '0')
      setEnabled(next)
      onSaved?.()
    } catch (e: any) {
      alert(e.message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
      <div style={{ flex: 1, marginRight: 16 }}>
        <div style={{ fontWeight: 600, color: '#f1f5f9', marginBottom: 4 }}>{label}</div>
        <div style={{ fontSize: '.8rem', color: '#64748b' }}>{description}</div>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexShrink: 0 }}>
        {enabled !== null && (
          <span style={{ fontSize: '.8rem', color: enabled ? '#86efac' : '#f87171' }}>
            {enabled ? 'Enabled' : 'Disabled'}
          </span>
        )}
        <button
          onClick={toggle}
          disabled={saving || enabled === null}
          style={{
            position: 'relative', width: 44, height: 24, borderRadius: 12,
            border: 'none', cursor: saving || enabled === null ? 'not-allowed' : 'pointer',
            background: enabled ? '#e10098' : '#334155', transition: 'background .2s',
            padding: 0, flexShrink: 0,
          }}
        >
          <span style={{
            position: 'absolute', top: 3, left: enabled ? 23 : 3, width: 18, height: 18,
            borderRadius: '50%', background: '#fff', transition: 'left .2s',
          }} />
        </button>
      </div>
    </div>
  )
}

// ── Numeric setting row ───────────────────────────────────────────────────────

function NumericSetting({
  settingKey, label, description, suffix = '',
}: {
  settingKey: string
  label: string
  description: string
  suffix?: string
}) {
  const [value, setValue] = useState('')
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    getSettings().then(s => setValue(s[settingKey] ?? '0')).catch(() => setValue('0'))
  }, [settingKey])

  async function save() {
    setSaving(true)
    try {
      await updateSetting(settingKey, value)
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (e: any) {
      alert(e.message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', padding: '10px 0', borderBottom: '1px solid #1e293b' }}>
      <div style={{ flex: 1 }}>
        <div style={{ fontSize: '.85rem', fontWeight: 500, color: '#e2e8f0' }}>{label}</div>
        <div style={{ fontSize: '.75rem', color: '#64748b' }}>{description}</div>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}>
        <Input
          type="number"
          min="0"
          value={value}
          onChange={e => setValue(e.target.value)}
          style={{ width: 80, textAlign: 'right' }}
        />
        {suffix && <span style={{ fontSize: '.8rem', color: '#64748b' }}>{suffix}</span>}
        <Btn size="sm" onClick={save} disabled={saving}>
          {saved ? 'Saved' : saving ? '…' : 'Save'}
        </Btn>
      </div>
    </div>
  )
}

// ── JWT Secrets ───────────────────────────────────────────────────────────────

function JWTSecretsSection() {
  const [secrets, setSecrets] = useState<JWTSecret[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)

  async function load() {
    setLoading(true); setError(null)
    try { setSecrets(await listJWTSecrets()) }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  async function handleDeactivate(id: number) {
    if (!confirm('Deactivate this secret? JWTs signed with it will stop being accepted.')) return
    try { await deactivateJWTSecret(id); load() }
    catch (e: any) { alert(e.message) }
  }

  return (
    <Card>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1rem' }}>
        <div>
          <div style={{ fontWeight: 600, color: '#f1f5f9', marginBottom: 4 }}>JWT Secrets</div>
          <div style={{ fontSize: '.8rem', color: '#64748b' }}>
            Secrets used to validate incoming JWTs. Multiple active secrets allow zero-downtime rotation.
          </div>
        </div>
        <Btn onClick={() => setShowAdd(true)}>+ Add Secret</Btn>
      </div>
      {loading && <Spinner />}
      {error && <ErrorMsg msg={error} />}
      {!loading && !error && !secrets.length && (
        <Empty msg="No JWT secrets configured. Requests without a valid JWT will use the default role." />
      )}
      {!loading && !!secrets.length && (
        <Table cols={['Name', 'Algorithm', 'Status', 'Created', '']}>
          {secrets.map(s => (
            <tr key={s.id}>
              <td><strong>{s.name}</strong></td>
              <td style={{ fontFamily: 'monospace', fontSize: '.85rem', opacity: .7 }}>{s.algorithm}</td>
              <td><span className={`adm-badge ${s.active ? 'adm-badge-ok' : 'adm-badge-off'}`}>{s.active ? 'Active' : 'Inactive'}</span></td>
              <td style={{ opacity: .5, fontSize: '12px' }}>{s.created_at.split('T')[0]}</td>
              <td className="adm-actions">
                {s.active && <Btn variant="danger" size="sm" onClick={() => handleDeactivate(s.id)}>Deactivate</Btn>}
              </td>
            </tr>
          ))}
        </Table>
      )}
      {showAdd && (
        <AddJWTSecretModal onClose={() => setShowAdd(false)} onDone={() => { setShowAdd(false); load() }} />
      )}
    </Card>
  )
}

function AddJWTSecretModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [name, setName] = useState('')
  const [secret, setSecret] = useState('')
  const [algorithm, setAlgorithm] = useState('HS256')
  const { loading, error, run, setError } = useAsync()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim() || !secret) { setError('Name and secret are required'); return }
    await run(async () => { await addJWTSecret(name.trim(), secret, algorithm); onDone() })
  }

  return (
    <Modal title="Add JWT Secret" onClose={onClose}>
      <form onSubmit={submit}>
        <Field label="Name">
          <Input value={name} onChange={e => setName(e.target.value)} placeholder="production-v1" required autoFocus />
        </Field>
        <Field label="Secret">
          <Input type="password" value={secret} onChange={e => setSecret(e.target.value)}
            placeholder="HMAC secret your app signs JWTs with" required style={{ fontFamily: 'monospace' }} />
        </Field>
        <Field label="Algorithm">
          <Select value={algorithm} onChange={e => setAlgorithm(e.target.value)}>
            <option value="HS256">HS256</option>
            <option value="HS384">HS384</option>
            <option value="HS512">HS512</option>
          </Select>
        </Field>
        {error && <ErrorMsg msg={error} />}
        <FormActions>
          <Btn variant="ghost" onClick={onClose}>Cancel</Btn>
          <Btn disabled={loading}>{loading ? 'Adding…' : 'Add Secret'}</Btn>
        </FormActions>
      </form>
    </Modal>
  )
}

// ── Router Keys ───────────────────────────────────────────────────────────────

function RouterKeysSection() {
  const [keys, setKeys] = useState<RouterKey[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)
  const [revealedKey, setRevealedKey] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  async function load() {
    setLoading(true); setError(null)
    try { setKeys(await listRouterKeys()) }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  async function handleDeactivate(id: number) {
    if (!confirm('Deactivate this key? Clients using it will lose access immediately.')) return
    try { await deactivateRouterKey(id); load() }
    catch (e: any) { alert(e.message) }
  }

  function handleCopy() {
    if (!revealedKey) return
    navigator.clipboard.writeText(revealedKey).then(() => {
      setCopied(true); setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <Card>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1rem' }}>
        <div>
          <div style={{ fontWeight: 600, color: '#f1f5f9', marginBottom: 4 }}>Router Keys</div>
          <div style={{ fontSize: '.8rem', color: '#64748b' }}>
            Send as <code style={{ color: '#94a3b8' }}>X-Router-Key</code> header with every API request.
            If no active keys exist the endpoint is open. To rotate: generate a new key, update clients, then deactivate the old one.
          </div>
        </div>
        <Btn onClick={() => setShowAdd(true)}>+ Generate Key</Btn>
      </div>
      {loading && <Spinner />}
      {error && <ErrorMsg msg={error} />}
      {!loading && !error && !keys.length && (
        <Empty msg="No router keys configured. API endpoints are accessible without a key." />
      )}
      {!loading && !!keys.length && (
        <Table cols={['Name', 'Status', 'Created', '']}>
          {keys.map(k => (
            <tr key={k.id}>
              <td><strong>{k.name}</strong></td>
              <td><span className={`adm-badge ${k.active ? 'adm-badge-ok' : 'adm-badge-off'}`}>{k.active ? 'Active' : 'Inactive'}</span></td>
              <td style={{ opacity: .5, fontSize: '12px' }}>{k.created_at.split('T')[0]}</td>
              <td className="adm-actions">
                {k.active && <Btn variant="danger" size="sm" onClick={() => handleDeactivate(k.id)}>Deactivate</Btn>}
              </td>
            </tr>
          ))}
        </Table>
      )}
      {showAdd && (
        <GenerateKeyModal
          onClose={() => setShowAdd(false)}
          onDone={(rawKey) => { setShowAdd(false); setRevealedKey(rawKey); load() }}
        />
      )}
      {revealedKey && (
        <Modal title="Copy Your Key" onClose={() => setRevealedKey(null)}>
          <div style={{ background: '#0f172a', border: '1px solid #e10098', borderRadius: 8, padding: 16, marginBottom: 16 }}>
            <div style={{ fontSize: '.8rem', color: '#f59e0b', fontWeight: 600, marginBottom: 8 }}>
              Copy this key now — it will never be shown again.
            </div>
            <code style={{ display: 'block', wordBreak: 'break-all', fontSize: '.85rem', color: '#e2e8f0' }}>
              {revealedKey}
            </code>
          </div>
          <div style={{ fontSize: '.8rem', color: '#64748b', marginBottom: '1rem' }}>
            Pass this as the <code style={{ color: '#94a3b8' }}>X-Router-Key</code> header in all API requests.
          </div>
          <FormActions>
            <Btn onClick={handleCopy}>{copied ? 'Copied!' : 'Copy to Clipboard'}</Btn>
            <Btn variant="ghost" onClick={() => setRevealedKey(null)}>Done</Btn>
          </FormActions>
        </Modal>
      )}
    </Card>
  )
}

function GenerateKeyModal({ onClose, onDone }: { onClose: () => void; onDone: (key: string) => void }) {
  const [name, setName] = useState('')
  const { loading, error, run, setError } = useAsync()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) { setError('Name is required'); return }
    await run(async () => {
      const result = await createRouterKey(name.trim())
      onDone(result.key)
    })
  }

  return (
    <Modal title="Generate Router Key" onClose={onClose}>
      <form onSubmit={submit}>
        <Field label="Name">
          <Input value={name} onChange={e => setName(e.target.value)} placeholder="ios-app-prod" required autoFocus />
        </Field>
        {error && <ErrorMsg msg={error} />}
        <FormActions>
          <Btn variant="ghost" onClick={onClose}>Cancel</Btn>
          <Btn disabled={loading}>{loading ? 'Generating…' : 'Generate'}</Btn>
        </FormActions>
      </form>
    </Modal>
  )
}

// ── Introspection ─────────────────────────────────────────────────────────────

function IntrospectionToggle() {
  return (
    <Card>
      <SettingToggle
        settingKey="introspection_enabled"
        label="GraphQL Introspection"
        description="When disabled, kastql rejects __schema and __type queries with a standard GraphQL error. Recommended: disable in production once clients are configured."
      />
    </Card>
  )
}

// ── CORS ──────────────────────────────────────────────────────────────────────

function CORSSection() {
  const [origins, setOrigins] = useState<CORSOrigin[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)

  async function load() {
    setLoading(true); setError(null)
    try { setOrigins(await listCORSOrigins()) }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  async function handleDelete(id: number) {
    if (!confirm('Remove this origin?')) return
    try { await deleteCORSOrigin(id); load() }
    catch (e: any) { alert(e.message) }
  }

  return (
    <Card>
      <div style={{ marginBottom: '1rem' }}>
        <SettingToggle
          settingKey="cors_enabled"
          label="CORS"
          description="Enable Cross-Origin Resource Sharing headers. Required for browser-based clients on different domains."
        />
        <div style={{ borderTop: '1px solid #1e293b', marginTop: '1rem', paddingTop: '1rem' }}>
          <SettingToggle
            settingKey="cors_allow_all"
            label="Allow all origins (*)"
            description="When enabled, any origin is accepted. Not recommended for production."
          />
        </div>
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem', marginTop: '1rem' }}>
        <div style={{ fontSize: '.85rem', color: '#94a3b8', fontWeight: 500 }}>Allowed Origins</div>
        <Btn size="sm" onClick={() => setShowAdd(true)}>+ Add Origin</Btn>
      </div>
      {loading && <Spinner />}
      {error && <ErrorMsg msg={error} />}
      {!loading && !error && !origins.length && (
        <Empty msg="No specific origins configured. Enable 'Allow all origins' or add specific origins." />
      )}
      {!loading && !!origins.length && (
        <Table cols={['Origin', 'Added', '']}>
          {origins.map(o => (
            <tr key={o.id}>
              <td style={{ fontFamily: 'monospace', fontSize: '.85rem' }}>{o.origin}</td>
              <td style={{ opacity: .5, fontSize: '12px' }}>{o.created_at.split('T')[0]}</td>
              <td className="adm-actions">
                <Btn variant="danger" size="sm" onClick={() => handleDelete(o.id)}>Remove</Btn>
              </td>
            </tr>
          ))}
        </Table>
      )}
      {showAdd && (
        <AddCORSOriginModal onClose={() => setShowAdd(false)} onDone={() => { setShowAdd(false); load() }} />
      )}
    </Card>
  )
}

function AddCORSOriginModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [origin, setOrigin] = useState('')
  const { loading, error, run, setError } = useAsync()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!origin.trim()) { setError('Origin is required'); return }
    await run(async () => { await addCORSOrigin(origin.trim()); onDone() })
  }

  return (
    <Modal title="Add Allowed Origin" onClose={onClose}>
      <form onSubmit={submit}>
        <Field label="Origin">
          <Input value={origin} onChange={e => setOrigin(e.target.value)}
            placeholder="https://app.example.com" required autoFocus />
        </Field>
        {error && <ErrorMsg msg={error} />}
        <FormActions>
          <Btn variant="ghost" onClick={onClose}>Cancel</Btn>
          <Btn disabled={loading}>{loading ? 'Adding…' : 'Add'}</Btn>
        </FormActions>
      </form>
    </Modal>
  )
}

// ── IP Filter ─────────────────────────────────────────────────────────────────

function IPFilterSection() {
  const [rules, setRules] = useState<IPRule[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)

  async function load() {
    setLoading(true); setError(null)
    try { setRules(await listIPRules()) }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  async function handleDelete(id: number) {
    if (!confirm('Remove this rule?')) return
    try { await deleteIPRule(id); load() }
    catch (e: any) { alert(e.message) }
  }

  return (
    <Card>
      <div style={{ marginBottom: '1rem' }}>
        <SettingToggle
          settingKey="ip_filter_enabled"
          label="IP Filtering"
          description="Allow or deny requests based on client IP address or CIDR range."
        />
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem', marginTop: '1rem' }}>
        <div style={{ fontSize: '.85rem', color: '#94a3b8', fontWeight: 500 }}>IP Rules</div>
        <Btn size="sm" onClick={() => setShowAdd(true)}>+ Add Rule</Btn>
      </div>
      {loading && <Spinner />}
      {error && <ErrorMsg msg={error} />}
      {!loading && !error && !rules.length && (
        <Empty msg="No IP rules. Default action: allow." />
      )}
      {!loading && !!rules.length && (
        <Table cols={['CIDR', 'Mode', 'Note', '']}>
          {rules.map(r => (
            <tr key={r.id}>
              <td style={{ fontFamily: 'monospace', fontSize: '.85rem' }}>{r.cidr}</td>
              <td>
                <span className={`adm-badge ${r.mode === 'allow' ? 'adm-badge-ok' : 'adm-badge-off'}`}>
                  {r.mode}
                </span>
              </td>
              <td style={{ opacity: .6, fontSize: '.8rem' }}>{r.note}</td>
              <td className="adm-actions">
                <Btn variant="danger" size="sm" onClick={() => handleDelete(r.id)}>Remove</Btn>
              </td>
            </tr>
          ))}
        </Table>
      )}
      {showAdd && (
        <AddIPRuleModal onClose={() => setShowAdd(false)} onDone={() => { setShowAdd(false); load() }} />
      )}
    </Card>
  )
}

function AddIPRuleModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [cidr, setCidr] = useState('')
  const [mode, setMode] = useState<'allow' | 'deny'>('deny')
  const [note, setNote] = useState('')
  const { loading, error, run, setError } = useAsync()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!cidr.trim()) { setError('CIDR is required'); return }
    await run(async () => { await addIPRule(cidr.trim(), mode, note.trim()); onDone() })
  }

  return (
    <Modal title="Add IP Rule" onClose={onClose}>
      <form onSubmit={submit}>
        <Field label="CIDR">
          <Input value={cidr} onChange={e => setCidr(e.target.value)}
            placeholder="192.168.1.0/24 or 10.0.0.1/32" required autoFocus />
        </Field>
        <Field label="Mode">
          <Select value={mode} onChange={e => setMode(e.target.value as 'allow' | 'deny')}>
            <option value="deny">Deny</option>
            <option value="allow">Allow</option>
          </Select>
        </Field>
        <Field label="Note (optional)">
          <Input value={note} onChange={e => setNote(e.target.value)} placeholder="e.g. office network" />
        </Field>
        {error && <ErrorMsg msg={error} />}
        <FormActions>
          <Btn variant="ghost" onClick={onClose}>Cancel</Btn>
          <Btn disabled={loading}>{loading ? 'Adding…' : 'Add Rule'}</Btn>
        </FormActions>
      </form>
    </Modal>
  )
}

// ── Rate Limiting ─────────────────────────────────────────────────────────────

function RateLimitSection() {
  return (
    <Card>
      <div style={{ marginBottom: '1.25rem' }}>
        <SettingToggle
          settingKey="rate_limit_enabled"
          label="Rate Limiting"
          description="Token-bucket rate limiting enforced per-IP, globally, and per mutation. Set to 0 to disable individual limits."
        />
      </div>
      <NumericSetting settingKey="rate_limit_global_rpm" label="Global RPM" description="Max requests per minute across all clients. 0 = unlimited." suffix="rpm" />
      <NumericSetting settingKey="rate_limit_ip_rpm" label="Per-IP RPM" description="Max requests per minute per IP address. 0 = unlimited." suffix="rpm" />
      <NumericSetting settingKey="rate_limit_mutation_rpm" label="Mutation RPM" description="Max mutation operations per minute globally. 0 = unlimited." suffix="rpm" />
    </Card>
  )
}

// ── Query Guards ──────────────────────────────────────────────────────────────

function QueryGuardsSection() {
  return (
    <Card>
      <div style={{ marginBottom: '1.25rem' }}>
        <div style={{ fontWeight: 600, color: '#f1f5f9', marginBottom: 4 }}>Query Guards</div>
        <div style={{ fontSize: '.8rem', color: '#64748b' }}>
          Structural limits on incoming GraphQL queries. Set 0 to disable each individual guard.
        </div>
      </div>
      <NumericSetting settingKey="query_depth_limit" label="Max Depth" description="Maximum nesting depth. Prevents deeply nested queries from exploding." />
      <NumericSetting settingKey="query_complexity_limit" label="Max Complexity" description="Maximum field count (simple complexity scoring). 0 = unlimited." />
      <NumericSetting settingKey="query_alias_limit" label="Max Aliases" description="Maximum number of field aliases. Prevents alias-based amplification." />
      <NumericSetting settingKey="query_directive_limit" label="Max Directives" description="Maximum number of directives per query. 0 = unlimited." />
      <NumericSetting settingKey="query_timeout_ms" label="Query Timeout" description="Hard timeout on query execution. 0 = no timeout." suffix="ms" />
    </Card>
  )
}

// ── Size Limits ───────────────────────────────────────────────────────────────

function SizeLimitsSection() {
  return (
    <Card>
      <div style={{ marginBottom: '1.25rem' }}>
        <div style={{ fontWeight: 600, color: '#f1f5f9', marginBottom: 4 }}>Size Limits</div>
        <div style={{ fontSize: '.8rem', color: '#64748b' }}>
          Cap request body and response body sizes. 0 = unlimited.
        </div>
      </div>
      <NumericSetting settingKey="max_request_body_kb" label="Max Request Body" description="Maximum request body size before the request is rejected." suffix="KB" />
      <NumericSetting settingKey="max_response_body_kb" label="Max Response Body" description="Maximum response body size before it is truncated." suffix="KB" />
    </Card>
  )
}

// ── Persisted Queries ─────────────────────────────────────────────────────────

function PersistedQueriesSection() {
  const [queries, setQueries] = useState<PersistedQuery[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)

  async function load() {
    setLoading(true); setError(null)
    try { setQueries(await listPersistedQueries()) }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  async function handleDelete(id: string) {
    if (!confirm('Remove this persisted query?')) return
    try { await deletePersistedQuery(id); load() }
    catch (e: any) { alert(e.message) }
  }

  return (
    <Card>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1rem' }}>
        <div>
          <SettingToggle
            settingKey="persisted_only"
            label="Persisted Queries Only"
            description="When enabled, only pre-registered queries are accepted. Ad-hoc query strings are rejected."
          />
        </div>
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem', marginTop: '1rem' }}>
        <div style={{ fontSize: '.85rem', color: '#94a3b8', fontWeight: 500 }}>Registered Queries</div>
        <Btn size="sm" onClick={() => setShowAdd(true)}>+ Add Query</Btn>
      </div>
      {loading && <Spinner />}
      {error && <ErrorMsg msg={error} />}
      {!loading && !error && !queries.length && (
        <Empty msg="No persisted queries registered." />
      )}
      {!loading && !!queries.length && (
        <Table cols={['Name', 'ID', 'Added', '']}>
          {queries.map(q => (
            <tr key={q.id}>
              <td><strong>{q.name}</strong></td>
              <td style={{ fontFamily: 'monospace', fontSize: '.75rem', opacity: .6, maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis' }}>{q.id}</td>
              <td style={{ opacity: .5, fontSize: '12px' }}>{q.created_at.split('T')[0]}</td>
              <td className="adm-actions">
                <Btn variant="danger" size="sm" onClick={() => handleDelete(q.id)}>Remove</Btn>
              </td>
            </tr>
          ))}
        </Table>
      )}
      {showAdd && (
        <AddPersistedQueryModal onClose={() => setShowAdd(false)} onDone={() => { setShowAdd(false); load() }} />
      )}
    </Card>
  )
}

function AddPersistedQueryModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [id, setId] = useState('')
  const [name, setName] = useState('')
  const [query, setQuery] = useState('')
  const { loading, error, run, setError } = useAsync()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!id.trim() || !query.trim()) { setError('ID and query are required'); return }
    await run(async () => { await addPersistedQuery(id.trim(), name.trim() || id.trim(), query.trim()); onDone() })
  }

  return (
    <Modal title="Add Persisted Query" onClose={onClose}>
      <form onSubmit={submit}>
        <Field label="ID (SHA-256 hash or custom string)">
          <Input value={id} onChange={e => setId(e.target.value)} placeholder="abc123def..." required autoFocus />
        </Field>
        <Field label="Name (optional)">
          <Input value={name} onChange={e => setName(e.target.value)} placeholder="GetUserProfile" />
        </Field>
        <Field label="Query">
          <Textarea value={query} onChange={e => setQuery(e.target.value)}
            placeholder="query GetUserProfile { ... }" rows={5} required />
        </Field>
        {error && <ErrorMsg msg={error} />}
        <FormActions>
          <Btn variant="ghost" onClick={onClose}>Cancel</Btn>
          <Btn disabled={loading}>{loading ? 'Adding…' : 'Add'}</Btn>
        </FormActions>
      </form>
    </Modal>
  )
}

// ── Other toggles ─────────────────────────────────────────────────────────────

function OtherSection() {
  return (
    <Card>
      <div style={{ marginBottom: '1rem' }}>
        <div style={{ fontWeight: 600, color: '#f1f5f9', marginBottom: '1rem' }}>Other</div>
        <SettingToggle
          settingKey="batch_queries_enabled"
          label="Batch Queries"
          description="Allow sending multiple operations as a JSON array in a single request."
        />
        <div style={{ borderTop: '1px solid #1e293b', marginTop: '1rem', paddingTop: '1rem' }}>
          <SettingToggle
            settingKey="audit_log_enabled"
            label="Audit Log"
            description="Record admin mutations (settings changes, key rotations) to the audit log."
          />
        </div>
      </div>
      <NumericSetting settingKey="ws_max_connections" label="Max WebSocket Connections" description="Maximum concurrent WebSocket subscription connections. 0 = unlimited." />
    </Card>
  )
}

// ── Audit Log ─────────────────────────────────────────────────────────────────

function AuditLogSection() {
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  async function load() {
    setLoading(true); setError(null)
    try { setEntries(await listAuditLog(100)) }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  return (
    <Card>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1rem' }}>
        <div>
          <div style={{ fontWeight: 600, color: '#f1f5f9', marginBottom: 4 }}>Audit Log</div>
          <div style={{ fontSize: '.8rem', color: '#64748b' }}>Admin actions: settings changes, key creation, and security mutations.</div>
        </div>
        <Btn variant="ghost" size="sm" onClick={load}>Refresh</Btn>
      </div>
      {loading && <Spinner />}
      {error && <ErrorMsg msg={error} />}
      {!loading && !error && !entries.length && <Empty msg="No audit log entries." />}
      {!loading && !!entries.length && (
        <Table cols={['Time', 'Admin', 'Action', 'Detail', 'IP']}>
          {entries.map(e => (
            <tr key={e.id}>
              <td style={{ opacity: .5, fontSize: '12px', whiteSpace: 'nowrap' }}>{e.created_at.replace('T', ' ').split('.')[0]}</td>
              <td style={{ fontFamily: 'monospace', fontSize: '.8rem' }}>{e.admin}</td>
              <td style={{ fontFamily: 'monospace', fontSize: '.8rem', color: '#e10098' }}>{e.action}</td>
              <td style={{ fontSize: '.8rem', opacity: .7 }}>{e.detail}</td>
              <td style={{ fontFamily: 'monospace', fontSize: '.75rem', opacity: .5 }}>{e.ip}</td>
            </tr>
          ))}
        </Table>
      )}
    </Card>
  )
}

// ── Blocked Requests ──────────────────────────────────────────────────────────

function BlockedRequestsSection() {
  const [entries, setEntries] = useState<BlockedRequest[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  async function load() {
    setLoading(true); setError(null)
    try { setEntries(await listBlockedRequests(100)) }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  return (
    <Card>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1rem' }}>
        <div>
          <div style={{ fontWeight: 600, color: '#f1f5f9', marginBottom: 4 }}>Blocked Requests</div>
          <div style={{ fontSize: '.8rem', color: '#64748b' }}>Recent requests blocked by rate limiting, IP filter, or query guards. Capped at 10,000 rows.</div>
        </div>
        <Btn variant="ghost" size="sm" onClick={load}>Refresh</Btn>
      </div>
      {loading && <Spinner />}
      {error && <ErrorMsg msg={error} />}
      {!loading && !error && !entries.length && <Empty msg="No blocked requests recorded." />}
      {!loading && !!entries.length && (
        <Table cols={['Time', 'Reason', 'IP', 'Path']}>
          {entries.map(e => (
            <tr key={e.id}>
              <td style={{ opacity: .5, fontSize: '12px', whiteSpace: 'nowrap' }}>{e.created_at.replace('T', ' ').split('.')[0]}</td>
              <td style={{ fontFamily: 'monospace', fontSize: '.8rem', color: '#f87171' }}>{e.reason}</td>
              <td style={{ fontFamily: 'monospace', fontSize: '.8rem', opacity: .7 }}>{e.ip}</td>
              <td style={{ fontFamily: 'monospace', fontSize: '.75rem', opacity: .5 }}>{e.path}</td>
            </tr>
          ))}
        </Table>
      )}
    </Card>
  )
}

// ── Tab root ──────────────────────────────────────────────────────────────────

export default function SecurityTab() {
  return (
    <div>
      <PageHeader title="Security" />
      <div style={{ display: 'flex', flexDirection: 'column', gap: '24px', marginTop: '1.5rem' }}>
        <IntrospectionToggle />
        <RouterKeysSection />
        <JWTSecretsSection />
        <CORSSection />
        <IPFilterSection />
        <RateLimitSection />
        <QueryGuardsSection />
        <SizeLimitsSection />
        <PersistedQueriesSection />
        <OtherSection />
        <AuditLogSection />
        <BlockedRequestsSection />
      </div>
    </div>
  )
}
