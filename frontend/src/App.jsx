import { useEffect, useState } from 'react'
import { Routes, Route, NavLink, useNavigate, useLocation, Navigate } from 'react-router-dom'
import { useAuth } from './store.jsx'
import { Spinner, Btn, Modal, Field, Input, useToast, Logo } from './ui.jsx'
import { api, roleLabel } from './api.js'

import Login from './pages/Login.jsx'
import Dashboard from './pages/Dashboard.jsx'
import Accounts from './pages/Accounts.jsx'
import Partners from './pages/Partners.jsx'
import Products from './pages/Products.jsx'
import Invoices from './pages/Invoices.jsx'
import Money from './pages/Money.jsx'
import Pos from './pages/Pos.jsx'
import FixedAssets from './pages/FixedAssets.jsx'
import Payroll from './pages/Payroll.jsx'
import EInvoice from './pages/EInvoice.jsx'
import Journal from './pages/Journal.jsx'
import Reports from './pages/Reports.jsx'
import Ledger from './pages/Ledger.jsx'
import Settings from './pages/Settings.jsx'
import Companies from './pages/Companies.jsx'
import Users from './pages/Users.jsx'
import Tenants from './pages/Tenants.jsx'

const REPORTS = [
  { id: 'trial', t: 'Dövriyyə balansı', path: '/reports/trial' },
  { id: 'balance', t: 'Balans hesabatı', path: '/reports/balance' },
  { id: 'pl', t: 'Mənfəət və zərər', path: '/reports/pl' },
  { id: 'vat', t: 'ƏDV bəyannaməsi', path: '/reports/vat' },
  { id: 'partnerbal', t: 'Debitor / Kreditor', path: '/reports/partners' },
  { id: 'stock', t: 'Anbar qalıqları', path: '/reports/stock', mod: 'inventory' },
]

function buildNav(auth) {
  const { isSuper, enabledModules } = auth
  const has = (m) => !m || !enabledModules || enabledModules.includes(m)
  if (isSuper) {
    // Superadmin manages only tenants + subscriptions — never company data.
    return [
      { g: 'İdarəetmə' },
      { t: 'Tenantlar (abunələr)', ic: '🗂', path: '/tenants' },
    ]
  }
  // A cashier sees only the POS terminal.
  if (auth.company?.role === 'cashier') {
    return [
      { g: 'Kassa' },
      { t: 'Kassa (POS)', ic: '🛒', path: '/pos' },
    ]
  }
  const items = [
    { g: 'Əsas' },
    { t: 'İdarə paneli', ic: '▚', path: '/' },
    has('journal') && { t: 'Mühasibat jurnalı', ic: '≣', path: '/journal' },
    { g: 'Ticarət' },
    has('pos') && { t: 'Kassa (POS)', ic: '🛒', path: '/pos' },
    has('sales') && { t: 'Satış fakturaları', ic: '↗', path: '/sales' },
    has('purchases') && { t: 'Alış fakturaları', ic: '↙', path: '/purchases' },
    has('money') && { t: 'Kassa / Bank', ic: '₼', path: '/money' },
    { g: 'Kataloq' },
    has('partners') && { t: 'Tərəfdaşlar', ic: '☺', path: '/partners' },
    has('products') && { t: 'Məhsul / Xidmət', ic: '▤', path: '/products' },
    { t: 'Hesablar planı', ic: '❏', path: '/accounts' },
    has('fixed_assets') && { t: 'Əsas vəsaitlər', ic: '🏗', path: '/fixed-assets' },
    has('payroll') && { t: 'Əmək haqqı', ic: '👔', path: '/payroll' },
    has('einvoice') && { t: 'e-Qaimə', ic: '🧾', path: '/einvoice' },
    { g: 'Hesabatlar' },
    ...REPORTS.filter((r) => has(r.mod)).map((r) => ({ t: r.t, ic: '∑', path: r.path })),
    { g: 'İdarəetmə' },
    auth.canManage() && { t: 'Şirkətlər', ic: '🏢', path: '/companies' },
    auth.canManage() && { t: 'İstifadəçilər', ic: '👥', path: '/users' },
    { t: 'Parametrlər', ic: '⚙', path: '/settings' },
  ]
  return items.filter(Boolean)
}

function CompanySwitcher({ onClose }) {
  const auth = useAuth()
  const [list, setList] = useState(auth.companies)
  useEffect(() => { api.get('/companies').then(setList).catch(() => {}) }, [])
  return (
    <Modal title="Şirkət seçin" onClose={onClose}>
      <div className="space-y-2">
        {list.length === 0 && <p className="text-muted">Sizə hələ şirkət təyin edilməyib. Administratora müraciət edin.</p>}
        {list.map((c) => (
          <button key={c.id}
            className="flex w-full items-center justify-between rounded-xl border border-line bg-surface2 px-4 py-3 hover:border-brand"
            onClick={() => { auth.chooseCompany(c); onClose() }}>
            <span><b>{c.name}</b> <span className="ml-1 text-xs text-muted">{roleLabel(c.role)}</span></span>
            <span>›</span>
          </button>
        ))}
      </div>
    </Modal>
  )
}

function Sidebar({ onSwitch }) {
  const auth = useAuth()
  const nav = buildNav(auth)
  return (
    <aside className="flex w-60 flex-col overflow-y-auto border-r border-line bg-surface">
      <div className="flex items-center gap-2.5 px-4 pb-3 pt-4">
        <Logo className="h-8 w-8" />
        <b className="text-[15px]">OAWO Mühasibat</b>
      </div>
      {!auth.isSuper && (
        <button onClick={onSwitch}
          className="mx-3 mb-2.5 flex items-center justify-between gap-2 rounded-xl border border-line bg-surface2 px-3 py-2.5 hover:border-brand">
          <span className="flex flex-col overflow-hidden text-left">
            <span className="truncate font-bold">{auth.company ? auth.company.name : 'Şirkət seç'}</span>
            <span className="text-[11px] text-muted">{auth.company ? roleLabel(auth.company.role) : '—'}</span>
          </span>
          <span className="text-muted">⇅</span>
        </button>
      )}
      <nav className="px-2.5 pb-3">
        {nav.map((n, i) =>
          n.g ? (
            <div key={i} className="px-2.5 pb-1 pt-3 text-[11px] uppercase tracking-wider text-muted">{n.g}</div>
          ) : (
            <NavLink key={i} to={n.path} end={n.path === '/'}
              className={({ isActive }) =>
                `flex items-center gap-2.5 rounded-lg px-3 py-2 font-medium ${
                  isActive ? 'bg-brand/10 text-brand' : 'text-muted hover:bg-surface2 hover:text-text'}`}>
              <span className="w-4 text-center">{n.ic}</span>{n.t}
            </NavLink>
          ),
        )}
      </nav>
    </aside>
  )
}

// Top-right bar: user, theme toggle, logout.
function TopBar({ theme, onToggleTheme }) {
  const auth = useAuth()
  return (
    <header className="sticky top-0 z-20 flex items-center justify-end gap-2 border-b border-line bg-surface/80 px-6 py-2.5 backdrop-blur">
      <span className="mr-1 hidden truncate text-sm text-muted sm:inline">{auth.user?.name || auth.user?.email}</span>
      <button onClick={onToggleTheme} title="Tema" className="rounded-lg border border-line bg-surface2 px-2.5 py-1.5 text-sm hover:border-brand">
        {theme === 'dark' ? '☀' : '🌙'}
      </button>
      <button onClick={auth.logout}
        className="flex items-center gap-1.5 rounded-lg border border-line bg-surface2 px-3 py-1.5 text-sm font-semibold text-text hover:border-danger hover:text-danger">
        ⇥ Çıxış
      </button>
    </header>
  )
}


// Guard for tenant-scoped pages: require an active company.
// Superadmin has no company context — they only manage tenants.
function RequireCompany({ children }) {
  const auth = useAuth()
  if (auth.isSuper) return <Navigate to="/tenants" replace />
  if (!auth.company) {
    return <div className="p-10 text-center text-muted">Şirkət seçilməyib. Yuxarıdakı düymə ilə şirkət seçin.</div>
  }
  return children
}

export default function App() {
  const auth = useAuth()
  const [switching, setSwitching] = useState(false)
  const [theme, setTheme] = useState(() => localStorage.getItem('oawo_theme') || 'light')
  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
    localStorage.setItem('oawo_theme', theme)
  }, [theme])
  const toggleTheme = () => setTheme((t) => (t === 'dark' ? 'light' : 'dark'))

  if (!auth.ready) return <div className="grid h-full place-items-center"><Spinner /></div>
  if (!auth.user) return <Login />

  return (
    <div className="flex h-full">
      <Sidebar onSwitch={() => setSwitching(true)} />
      <main className="flex-1 overflow-y-auto">
        <TopBar theme={theme} onToggleTheme={toggleTheme} />
        <div className="px-6 py-5">
          <Routes>
            <Route path="/" element={auth.isSuper ? <Navigate to="/tenants" replace /> : auth.company?.role === 'cashier' ? <Navigate to="/pos" replace /> : <RequireCompany><Dashboard /></RequireCompany>} />
            <Route path="/tenants" element={auth.isSuper ? <Tenants /> : <Navigate to="/" replace />} />
            <Route path="/companies" element={<Companies />} />
            <Route path="/users" element={<RequireCompany><Users /></RequireCompany>} />
            <Route path="/accounts" element={<RequireCompany><Accounts /></RequireCompany>} />
            <Route path="/journal" element={<RequireCompany><Journal /></RequireCompany>} />
            <Route path="/partners" element={<RequireCompany><Partners /></RequireCompany>} />
            <Route path="/products" element={<RequireCompany><Products /></RequireCompany>} />
            <Route path="/sales" element={<RequireCompany><Invoices key="sales" type="sales_invoice" /></RequireCompany>} />
            <Route path="/purchases" element={<RequireCompany><Invoices key="purchases" type="purchase_invoice" /></RequireCompany>} />
            <Route path="/money" element={<RequireCompany><Money /></RequireCompany>} />
            <Route path="/pos" element={<RequireCompany><Pos /></RequireCompany>} />
            <Route path="/fixed-assets" element={<RequireCompany><FixedAssets /></RequireCompany>} />
            <Route path="/payroll" element={<RequireCompany><Payroll /></RequireCompany>} />
            <Route path="/einvoice" element={<RequireCompany><EInvoice /></RequireCompany>} />
            <Route path="/reports/:kind" element={<RequireCompany><Reports /></RequireCompany>} />
            <Route path="/ledger/:id" element={<RequireCompany><Ledger /></RequireCompany>} />
            <Route path="/settings" element={<RequireCompany><Settings /></RequireCompany>} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </div>
      </main>
      {switching && <CompanySwitcher onClose={() => setSwitching(false)} />}
    </div>
  )
}
