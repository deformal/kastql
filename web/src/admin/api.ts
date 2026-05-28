// Thin wrappers around the kastql admin APIs.

export async function metaCall(type: string, args: Record<string, unknown> = {}) {
  const res = await fetch('/v1/metadata', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ type, ...args }),
  })
  const json = await res.json()
  if (!res.ok) throw new Error(json?.error ?? json?.message ?? 'request failed')
  return json
}

export async function getUsers() {
  const res = await fetch('/v1/admin/users')
  if (!res.ok) throw new Error('failed to fetch users')
  return res.json() as Promise<{ id: number; username: string; created_at: string }[]>
}

export async function createUser(username: string, password: string) {
  const res = await fetch('/v1/admin/users', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  })
  const json = await res.json()
  if (!res.ok) throw new Error(json?.error ?? 'failed to create user')
  return json
}

export async function deleteUser(id: number) {
  const res = await fetch(`/v1/admin/users/${id}`, { method: 'DELETE' })
  if (!res.ok) {
    const json = await res.json().catch(() => ({}))
    throw new Error(json?.error ?? 'failed to delete user')
  }
}

export async function getMetrics(limit = 20) {
  const res = await fetch(`/v1/metrics?limit=${limit}`)
  if (!res.ok) throw new Error('failed to fetch metrics')
  return res.json()
}

// ── Settings ─────────────────────────────────────────────────────────────────

export async function getSettings(): Promise<Record<string, string>> {
  const res = await fetch('/v1/admin/settings')
  if (!res.ok) throw new Error('failed to fetch settings')
  return res.json()
}

export async function updateSetting(key: string, value: string) {
  const res = await fetch(`/v1/admin/settings/${key}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ value }),
  })
  const json = await res.json()
  if (!res.ok) throw new Error(json?.error ?? 'failed to update setting')
  return json
}

// ── JWT Secrets ───────────────────────────────────────────────────────────────

export interface JWTSecret {
  id: number
  name: string
  algorithm: string
  active: boolean
  created_at: string
}

export async function listJWTSecrets(): Promise<JWTSecret[]> {
  const res = await fetch('/v1/admin/jwt-secrets')
  if (!res.ok) throw new Error('failed to fetch jwt secrets')
  return res.json()
}

export async function addJWTSecret(name: string, secret: string, algorithm = 'HS256'): Promise<JWTSecret> {
  const res = await fetch('/v1/admin/jwt-secrets', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, secret, algorithm }),
  })
  const json = await res.json()
  if (!res.ok) throw new Error(json?.error ?? 'failed to add jwt secret')
  return json
}

export async function deactivateJWTSecret(id: number) {
  const res = await fetch(`/v1/admin/jwt-secrets/${id}`, { method: 'DELETE' })
  if (!res.ok) {
    const json = await res.json().catch(() => ({}))
    throw new Error(json?.error ?? 'failed to deactivate secret')
  }
}

// ── Router Keys ───────────────────────────────────────────────────────────────

export interface RouterKey {
  id: number
  name: string
  active: boolean
  created_at: string
}

export interface NewRouterKey extends RouterKey {
  key: string // raw key — shown once only
}

export async function listRouterKeys(): Promise<RouterKey[]> {
  const res = await fetch('/v1/admin/router-keys')
  if (!res.ok) throw new Error('failed to fetch router keys')
  return res.json()
}

export async function createRouterKey(name: string): Promise<NewRouterKey> {
  const res = await fetch('/v1/admin/router-keys', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  const json = await res.json()
  if (!res.ok) throw new Error(json?.error ?? 'failed to create router key')
  return json
}

export async function deactivateRouterKey(id: number) {
  const res = await fetch(`/v1/admin/router-keys/${id}`, { method: 'DELETE' })
  if (!res.ok) {
    const json = await res.json().catch(() => ({}))
    throw new Error(json?.error ?? 'failed to deactivate key')
  }
}

// ── CORS Origins ──────────────────────────────────────────────────────────────

export interface CORSOrigin {
  id: number
  origin: string
  created_at: string
}

export async function listCORSOrigins(): Promise<CORSOrigin[]> {
  const res = await fetch('/v1/admin/cors-origins')
  if (!res.ok) throw new Error('failed to fetch cors origins')
  return res.json()
}

export async function addCORSOrigin(origin: string): Promise<CORSOrigin> {
  const res = await fetch('/v1/admin/cors-origins', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ origin }),
  })
  const json = await res.json()
  if (!res.ok) throw new Error(json?.error ?? 'failed to add cors origin')
  return json
}

export async function deleteCORSOrigin(id: number) {
  const res = await fetch(`/v1/admin/cors-origins/${id}`, { method: 'DELETE' })
  if (!res.ok) {
    const json = await res.json().catch(() => ({}))
    throw new Error(json?.error ?? 'failed to delete cors origin')
  }
}

// ── IP Rules ──────────────────────────────────────────────────────────────────

export interface IPRule {
  id: number
  cidr: string
  mode: 'allow' | 'deny'
  note: string
  created_at: string
}

export async function listIPRules(): Promise<IPRule[]> {
  const res = await fetch('/v1/admin/ip-rules')
  if (!res.ok) throw new Error('failed to fetch ip rules')
  return res.json()
}

export async function addIPRule(cidr: string, mode: 'allow' | 'deny', note: string): Promise<IPRule> {
  const res = await fetch('/v1/admin/ip-rules', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ cidr, mode, note }),
  })
  const json = await res.json()
  if (!res.ok) throw new Error(json?.error ?? 'failed to add ip rule')
  return json
}

export async function deleteIPRule(id: number) {
  const res = await fetch(`/v1/admin/ip-rules/${id}`, { method: 'DELETE' })
  if (!res.ok) {
    const json = await res.json().catch(() => ({}))
    throw new Error(json?.error ?? 'failed to delete ip rule')
  }
}

// ── Persisted Queries ─────────────────────────────────────────────────────────

export interface PersistedQuery {
  id: string
  name: string
  query: string
  created_at: string
}

export async function listPersistedQueries(): Promise<PersistedQuery[]> {
  const res = await fetch('/v1/admin/persisted-queries')
  if (!res.ok) throw new Error('failed to fetch persisted queries')
  return res.json()
}

export async function addPersistedQuery(id: string, name: string, query: string): Promise<PersistedQuery> {
  const res = await fetch('/v1/admin/persisted-queries', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id, name, query }),
  })
  const json = await res.json()
  if (!res.ok) throw new Error(json?.error ?? 'failed to add persisted query')
  return json
}

export async function deletePersistedQuery(id: string) {
  const res = await fetch(`/v1/admin/persisted-queries/${encodeURIComponent(id)}`, { method: 'DELETE' })
  if (!res.ok) {
    const json = await res.json().catch(() => ({}))
    throw new Error(json?.error ?? 'failed to delete persisted query')
  }
}

// ── Health ────────────────────────────────────────────────────────────────────

export type CircuitState = 'closed' | 'open' | 'half_open'

export interface ServiceHealth {
  name: string
  url: string
  circuit: CircuitState
  healthy: boolean
  consec_failures: number
  latency_ms: number
  checked_at: string
  healthy_since?: string
  last_error?: string
}

export async function listHealth(): Promise<ServiceHealth[]> {
  const res = await fetch('/v1/admin/health')
  if (!res.ok) throw new Error('failed to fetch health')
  return res.json()
}

// ── Schema ────────────────────────────────────────────────────────────────────

export async function getMergedSchema(): Promise<string> {
  const res = await fetch('/v1/admin/schema')
  if (!res.ok) throw new Error('failed to fetch schema')
  const json = await res.json()
  return json.sdl ?? ''
}

// ── Cache ─────────────────────────────────────────────────────────────────────

export async function flushCache(): Promise<{ flushed: number }> {
  const res = await fetch('/v1/admin/cache/flush', { method: 'POST' })
  const json = await res.json()
  if (!res.ok) throw new Error(json?.error ?? 'failed to flush cache')
  return json
}

// ── Audit Log ─────────────────────────────────────────────────────────────────

export interface AuditEntry {
  id: number
  admin: string
  action: string
  detail: string
  ip: string
  created_at: string
}

export async function listAuditLog(limit = 100): Promise<AuditEntry[]> {
  const res = await fetch(`/v1/admin/audit-log?limit=${limit}`)
  if (!res.ok) throw new Error('failed to fetch audit log')
  return res.json()
}

// ── Blocked Requests ──────────────────────────────────────────────────────────

export interface BlockedRequest {
  id: number
  reason: string
  ip: string
  path: string
  created_at: string
}

export async function listBlockedRequests(limit = 100): Promise<BlockedRequest[]> {
  const res = await fetch(`/v1/admin/blocked-requests?limit=${limit}`)
  if (!res.ok) throw new Error('failed to fetch blocked requests')
  return res.json()
}
