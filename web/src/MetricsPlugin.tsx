import { useCallback, useEffect, useState } from 'react'
import type { GraphiQLPlugin } from '@graphiql/react'
import './MetricsPlugin.css'

// ── Types ─────────────────────────────────────────────────────────────────────

interface OpStat {
  name: string
  type: string
  count: number
  error_count: number
  avg_ms: number
}

interface RecentError {
  timestamp: string
  operation_type: string
  operation_name: string
  duration_ms: number
  message: string
}

interface MetricsSummary {
  total_queries: number
  success_count: number
  error_count: number
  error_rate: number
  latency_p50_ms: number
  latency_p95_ms: number
  latency_p99_ms: number
  operations: OpStat[]
  recent_errors: RecentError[]
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function fmtNum(n?: number) {
  if (n === undefined || n === null) return '—'
  if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M'
  if (n >= 1e3) return (n / 1e3).toFixed(1) + 'k'
  return String(n)
}

function fmtPct(r?: number) {
  if (r === undefined || r === null) return '—'
  return (r * 100).toFixed(1) + '%'
}

function fmtMs(ms?: number) {
  if (ms === undefined || ms === null) return '—'
  return ms + ' ms'
}

function fmtTs(ts: string) {
  try { return new Date(ts).toLocaleTimeString() } catch { return ts }
}

function errorRateClass(rate?: number) {
  if (!rate) return 'good'
  if (rate < 0.05) return 'good'
  if (rate < 0.15) return 'warn'
  return 'bad'
}

function opBadgeClass(type: string) {
  switch (type?.toLowerCase()) {
    case 'query':        return 'badge badge-query'
    case 'mutation':     return 'badge badge-mutation'
    case 'subscription': return 'badge badge-subscription'
    default:             return 'badge badge-unknown'
  }
}

// ── Panel component ───────────────────────────────────────────────────────────

function MetricsPanel() {
  const [data, setData]       = useState<MetricsSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError]     = useState<string | null>(null)
  const [refreshedAt, setAt]  = useState<Date | null>(null)

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    fetch('/v1/metrics?limit=15')
      .then(r => {
        if (!r.ok) throw new Error('HTTP ' + r.status)
        return r.json() as Promise<MetricsSummary>
      })
      .then(d => { setData(d); setAt(new Date()); setLoading(false) })
      .catch(e => { setError(String(e.message)); setLoading(false) })
  }, [])

  useEffect(() => { load() }, [load])

  if (loading && !data) {
    return <div className="km-panel"><div className="km-loading">Loading metrics…</div></div>
  }

  if (error && !data) {
    return <div className="km-panel"><div className="km-error-state">⚠ {error}</div></div>
  }

  return (
    <div className="km-panel">
      {/* Header */}
      <div className="km-header">
        <h2>⚡ kastql Metrics</h2>
        <div className="km-header-right">
          {refreshedAt && <time>Updated {refreshedAt.toLocaleTimeString()}</time>}
          <button className="km-refresh" onClick={load} disabled={loading}>
            {loading ? '…' : 'Refresh'}
          </button>
        </div>
      </div>

      <div className="km-body">

        {/* Summary */}
        <section className="km-section">
          <h3 className="km-section-title">Summary</h3>
          <div className="km-stats">
            <Stat label="Total Queries"  value={fmtNum(data?.total_queries)} />
            <Stat label="Success"        value={fmtNum(data?.success_count)} cls="good" />
            <Stat label="Errors"         value={fmtNum(data?.error_count)}   cls={data?.error_count ? 'bad' : 'good'} />
            <Stat label="Error Rate"     value={fmtPct(data?.error_rate)}    cls={errorRateClass(data?.error_rate)} />
          </div>
        </section>

        {/* Latency */}
        <section className="km-section">
          <h3 className="km-section-title">Latency</h3>
          <div className="km-stats">
            <Stat label="p50" value={fmtMs(data?.latency_p50_ms)} />
            <Stat label="p95" value={fmtMs(data?.latency_p95_ms)} />
            <Stat label="p99" value={fmtMs(data?.latency_p99_ms)} />
          </div>
        </section>

        {/* Operations */}
        <section className="km-section">
          <h3 className="km-section-title">Operations</h3>
          {!data?.operations?.length ? (
            <p className="km-empty">No operations recorded yet.</p>
          ) : (
            <table className="km-table">
              <thead>
                <tr>
                  <th>Operation</th>
                  <th>Type</th>
                  <th className="num">Count</th>
                  <th className="num">Errors</th>
                  <th className="num">Avg ms</th>
                </tr>
              </thead>
              <tbody>
                {data.operations.map((op, i) => (
                  <tr key={i}>
                    <td>{op.name || <em>anonymous</em>}</td>
                    <td><span className={opBadgeClass(op.type)}>{op.type || 'unknown'}</span></td>
                    <td className="num">{op.count}</td>
                    <td className="num">
                      {op.error_count > 0
                        ? <span style={{ color: 'var(--color-error, #dc2626)' }}>{op.error_count}</span>
                        : '0'}
                    </td>
                    <td className="num">{op.avg_ms?.toFixed(1) ?? '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>

        {/* Recent errors */}
        <section className="km-section">
          <h3 className="km-section-title">Recent Errors</h3>
          {!data?.recent_errors?.length ? (
            <p className="km-empty">No errors — looking good! 🎉</p>
          ) : (
            data.recent_errors.map((e, i) => (
              <div key={i} className="km-error-entry">
                <div className="km-error-meta">
                  {fmtTs(e.timestamp)}
                  {' · '}
                  {[e.operation_type, e.operation_name].filter(Boolean).join(' / ') || 'unknown'}
                  {' · '}
                  {e.duration_ms} ms
                </div>
                <div className="km-error-msg">{e.message || '(no message)'}</div>
              </div>
            ))
          )}
        </section>

      </div>
    </div>
  )
}

function Stat({ label, value, cls }: { label: string; value: string; cls?: string }) {
  return (
    <div className="km-stat">
      <span className="km-stat-label">{label}</span>
      <span className={`km-stat-value${cls ? ' ' + cls : ''}`}>{value}</span>
    </div>
  )
}

// ── Icon ──────────────────────────────────────────────────────────────────────

function MetricsIcon() {
  return (
    <svg viewBox="0 0 24 24" width="1em" height="1em" fill="none"
      stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
      <line x1="18" y1="20" x2="18" y2="10" />
      <line x1="12" y1="20" x2="12" y2="4" />
      <line x1="6"  y1="20" x2="6"  y2="14" />
    </svg>
  )
}

// ── Plugin descriptor ─────────────────────────────────────────────────────────

export const MetricsPlugin: GraphiQLPlugin = {
  title: 'Metrics',
  icon: MetricsIcon,
  content: MetricsPanel,
}
