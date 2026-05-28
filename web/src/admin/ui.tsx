// Shared UI primitives for the admin panel.
import { useState, type ReactNode } from 'react'

export function PageHeader({ title, action }: { title: string; action?: ReactNode }) {
  return (
    <div className="adm-page-header">
      <h1 className="adm-page-title">{title}</h1>
      {action && <div>{action}</div>}
    </div>
  )
}

export function Card({ children }: { children: ReactNode }) {
  return <div className="adm-card">{children}</div>
}

export function Table({ cols, children }: { cols: string[]; children: ReactNode }) {
  return (
    <div className="adm-table-wrap">
      <table className="adm-table">
        <thead>
          <tr>{cols.map(c => <th key={c}>{c}</th>)}</tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
    </div>
  )
}

export function Empty({ msg }: { msg: string }) {
  return <p className="adm-empty">{msg}</p>
}

export function ErrorMsg({ msg }: { msg: string }) {
  return <p className="adm-error">{msg}</p>
}

export function Spinner() {
  return <p className="adm-loading">Loading…</p>
}

export function Btn({
  children, onClick, variant = 'primary', disabled, size = 'md',
}: {
  children: ReactNode
  onClick?: () => void
  variant?: 'primary' | 'danger' | 'ghost'
  disabled?: boolean
  size?: 'sm' | 'md'
}) {
  return (
    <button
      className={`adm-btn adm-btn-${variant} adm-btn-${size}`}
      onClick={onClick}
      disabled={disabled}
    >
      {children}
    </button>
  )
}

export function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return (
    <div className="adm-modal-overlay" onClick={onClose}>
      <div className="adm-modal" onClick={e => e.stopPropagation()}>
        <div className="adm-modal-header">
          <span className="adm-modal-title">{title}</span>
          <button className="adm-modal-close" onClick={onClose}>✕</button>
        </div>
        <div className="adm-modal-body">{children}</div>
      </div>
    </div>
  )
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="adm-field">
      <label className="adm-field-label">{label}</label>
      {children}
    </div>
  )
}

export function Input(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return <input className="adm-input" {...props} />
}

export function Select(props: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return <select className="adm-input" {...props} />
}

export function Textarea(props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className="adm-input adm-textarea" {...props} />
}

export function FormActions({ children }: { children: ReactNode }) {
  return <div className="adm-form-actions">{children}</div>
}

// Simple hook for async operations with loading + error state
export function useAsync() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function run(fn: () => Promise<void>) {
    setLoading(true)
    setError(null)
    try {
      await fn()
    } catch (e: any) {
      setError(e.message ?? 'Something went wrong')
    } finally {
      setLoading(false)
    }
  }

  return { loading, error, run, setError }
}
