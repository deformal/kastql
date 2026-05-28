import { useCallback, useEffect, useState } from 'react'
import { getMetrics } from './api'
import { PageHeader, Card, ErrorMsg, Spinner } from './ui'

interface OpStat { name: string; type: string; count: number; error_count: number; avg_ms: number }
interface RecentError { timestamp: string; operation_type: string; operation_name: string; duration_ms: number; message: string }
interface MetricsSummary {
  total_queries: number; success_count: number; error_count: number; error_rate: number
  latency_p50_ms: number; latency_p95_ms: number; latency_p99_ms: number
  operations: OpStat[]; recent_errors: RecentError[]
}

function fmt(n?: number) {
  if (n == null) return '—'
  if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M'
  if (n >= 1e3) return (n / 1e3).toFixed(1) + 'k'
  return String(n)
}
function fmtMs(n?: number) { return n == null ? '—' : n + ' ms' }
function fmtPct(r?: number) { return r == null ? '—' : (r * 100).toFixed(1) + '%' }
function fmtTs(ts: string) { try { return new Date(ts).toLocaleTimeString() } catch { return ts } }

export default function MetricsTab() {
  const [data, setData] = useState<MetricsSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [refreshedAt, setRefreshedAt] = useState<Date | null>(null)

  const load = useCallback(async () => {
    setLoading(true); setError(null)
    try { setData(await getMetrics(20)); setRefreshedAt(new Date()) }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { load() }, [load])

  return (
    <div>
      <PageHeader
        title="Metrics"
        action={
          <button className="adm-btn adm-btn-ghost adm-btn-md" onClick={load} disabled={loading}>
            {loading ? '…' : 'Refresh'}
          </button>
        }
      />
      {refreshedAt && <p style={{ opacity: .5, fontSize: 12, marginBottom: 16 }}>Updated {refreshedAt.toLocaleTimeString()}</p>}
      {loading && !data && <Spinner />}
      {error && <ErrorMsg msg={error} />}
      {data && (
        <>
          <div className="adm-metrics-grid">
            <StatCard label="Total Queries" value={fmt(data.total_queries)} />
            <StatCard label="Success" value={fmt(data.success_count)} cls="ok" />
            <StatCard label="Errors" value={fmt(data.error_count)} cls={data.error_count ? 'bad' : 'ok'} />
            <StatCard label="Error Rate" value={fmtPct(data.error_rate)} cls={!data.error_rate ? 'ok' : data.error_rate < 0.05 ? 'ok' : data.error_rate < 0.15 ? 'warn' : 'bad'} />
            <StatCard label="p50" value={fmtMs(data.latency_p50_ms)} />
            <StatCard label="p95" value={fmtMs(data.latency_p95_ms)} />
            <StatCard label="p99" value={fmtMs(data.latency_p99_ms)} />
          </div>

          <Card>
            <h3 className="adm-section-title">Operations</h3>
            {!data.operations?.length ? <p className="adm-empty">No operations recorded yet.</p> : (
              <table className="adm-table">
                <thead><tr><th>Operation</th><th>Type</th><th className="num">Count</th><th className="num">Errors</th><th className="num">Avg ms</th></tr></thead>
                <tbody>
                  {data.operations.map((op, i) => (
                    <tr key={i}>
                      <td>{op.name || <em style={{ opacity: .5 }}>anonymous</em>}</td>
                      <td><span className={`adm-badge adm-badge-op-${op.type?.toLowerCase()}`}>{op.type || 'unknown'}</span></td>
                      <td className="num">{op.count}</td>
                      <td className="num" style={{ color: op.error_count ? 'var(--c-bad)' : undefined }}>{op.error_count}</td>
                      <td className="num">{op.avg_ms?.toFixed(1) ?? '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </Card>

          <Card>
            <h3 className="adm-section-title">Recent Errors</h3>
            {!data.recent_errors?.length ? <p className="adm-empty">No errors — looking good.</p> : (
              data.recent_errors.map((e, i) => (
                <div key={i} className="adm-error-entry">
                  <div className="adm-error-meta">{fmtTs(e.timestamp)} · {[e.operation_type, e.operation_name].filter(Boolean).join(' / ') || 'unknown'} · {e.duration_ms} ms</div>
                  <div className="adm-error-msg">{e.message || '(no message)'}</div>
                </div>
              ))
            )}
          </Card>
        </>
      )}
    </div>
  )
}

function StatCard({ label, value, cls }: { label: string; value: string; cls?: string }) {
  return (
    <div className="adm-stat-card">
      <span className="adm-stat-label">{label}</span>
      <span className={`adm-stat-value${cls ? ' ' + cls : ''}`}>{value}</span>
    </div>
  )
}
