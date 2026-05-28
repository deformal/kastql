import { useState } from 'react'
import { Routes, Route, NavLink, useNavigate } from 'react-router-dom'
import ServicesTab from './ServicesTab'
import RelationshipsTab from './RelationshipsTab'
import PermissionsTab from './PermissionsTab'
import RestEndpointsTab from './RestEndpointsTab'
import UsersTab from './UsersTab'
import MetricsTab from './MetricsTab'
import SecurityTab from './SecurityTab'
import SchemaTab from './SchemaTab'
import './admin.css'

const NAV = [
  { to: '/admin/services',      label: 'Services' },
  // { to: '/admin/relationships', label: 'Relationships' },  // hidden until core is stable
  // { to: '/admin/permissions',   label: 'Permissions' },    // hidden until core is stable
  // { to: '/admin/rest',          label: 'REST Endpoints' }, // hidden until core is stable
  { to: '/admin/schema',        label: 'Schema' },
  { to: '/admin/users',         label: 'Users' },
  { to: '/admin/security',      label: 'Security' },
  { to: '/admin/metrics',       label: 'Metrics' },
]

export default function AdminApp() {
  const [logoutPending, setLogoutPending] = useState(false)

  async function handleLogout() {
    setLogoutPending(true)
    await fetch('/admin/logout', { method: 'POST' })
    window.location.href = '/admin/login'
  }

  return (
    <div className="adm-shell">
      <aside className="adm-sidebar">
        <div className="adm-logo">⚡ kastql</div>
        <div className="adm-logo-sub">Admin Panel</div>
        <nav className="adm-nav">
          {NAV.map(n => (
            <NavLink key={n.to} to={n.to} className={({ isActive }) => 'adm-nav-item' + (isActive ? ' active' : '')}>
              {n.label}
            </NavLink>
          ))}
        </nav>
        <div className="adm-sidebar-footer">
          <a className="adm-playground-link" href="/" target="_blank" rel="noreferrer">
            Open Playground ↗
          </a>
          <button className="adm-logout" onClick={handleLogout} disabled={logoutPending}>
            {logoutPending ? '…' : 'Sign out'}
          </button>
        </div>
      </aside>

      <main className="adm-main">
        <Routes>
          <Route index element={<Redirect to="/admin/services" />} />
          <Route path="services"      element={<ServicesTab />} />
          <Route path="relationships" element={<RelationshipsTab />} />
          <Route path="permissions"   element={<PermissionsTab />} />
          <Route path="rest"          element={<RestEndpointsTab />} />
          <Route path="schema"        element={<SchemaTab />} />
          <Route path="users"         element={<UsersTab />} />
          <Route path="security"      element={<SecurityTab />} />
          <Route path="metrics"       element={<MetricsTab />} />
        </Routes>
      </main>
    </div>
  )
}

function Redirect({ to }: { to: string }) {
  const nav = useNavigate()
  useState(() => { nav(to, { replace: true }) })
  return null
}
