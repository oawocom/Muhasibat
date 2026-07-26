import { useState } from 'react'
import { useList } from '../hooks.js'
import { api, money } from '../api.js'
import { Spinner, PageHeader, Table, Btn, Field, Input, useToast } from '../ui.jsx'
import { CrudModal } from './crud.jsx'
import { useAuth } from '../store.jsx'

export default function Settings() {
  const auth = useAuth()
  const toast = useToast()
  const currencies = useList('/currencies', [auth.company?.id])
  const taxes = useList('/tax-rates', [auth.company?.id])
  const warehouses = useList('/warehouses', [auth.company?.id])
  const [edit, setEdit] = useState(null)
  if (!currencies.data || !taxes.data || !warehouses.data) return <Spinner />

  const open = (cfg) => setEdit(cfg)
  const curFields = [{ k: 'code', label: 'Kod' }, { k: 'name', label: 'Ad' }, { k: 'symbol', label: 'Simvol' }, { k: 'rate', label: 'Məzənnə', type: 'number', step: '0.000001', def: 1 }]
  const taxFields = [{ k: 'name', label: 'Ad' }, { k: 'rate', label: 'Dərəcə %', type: 'number', step: '0.01' }]
  const whFields = [{ k: 'code', label: 'Kod' }, { k: 'name', label: 'Ad' }, { k: 'address', label: 'Ünvan', type: 'textarea' }]

  return (
    <>
      <PageHeader title="Parametrlər" />
      {auth.isTenantAdmin && <Subscription />}

      <Table title="Valyutalar" columns={[
        { h: 'Kod', mono: true, k: 'code' }, { h: 'Ad', k: 'name' },
        { h: 'Məzənnə', right: true, render: (r) => (r.is_base ? 'Baza' : money(r.rate)) },
        { h: '', right: true, render: (r) => (!r.is_base && auth.canWrite()) ? <Btn sm variant="ghost" onClick={() => open({ path: '/currencies', fields: [{ k: 'rate', label: 'Məzənnə', type: 'number', step: '0.000001' }], item: r, title: 'Valyuta məzənnəsi' })}>Düzəliş</Btn> : null },
      ]} rows={currencies.data} actions={auth.canWrite() && <Btn sm variant="ghost" onClick={() => open({ path: '/currencies', fields: curFields, item: { enabled: true }, title: 'Yeni valyuta' })}>+ Valyuta</Btn>} />

      <Table title="ƏDV dərəcələri" columns={[
        { h: 'Ad', k: 'name' }, { h: 'Dərəcə %', right: true, render: (r) => money(r.rate) },
        { h: '', right: true, render: (r) => auth.canWrite() ? <Btn sm variant="ghost" onClick={() => open({ path: '/tax-rates', fields: taxFields, item: r, title: 'ƏDV dərəcəsi' })}>Düzəliş</Btn> : null },
      ]} rows={taxes.data} actions={auth.canWrite() && <Btn sm variant="ghost" onClick={() => open({ path: '/tax-rates', fields: taxFields, item: { enabled: true }, title: 'Yeni dərəcə' })}>+ Dərəcə</Btn>} />

      <Table title="Anbarlar" columns={[
        { h: 'Kod', mono: true, k: 'code' }, { h: 'Ad', k: 'name' },
        { h: '', right: true, render: (r) => auth.canWrite() ? <Btn sm variant="ghost" onClick={() => open({ path: '/warehouses', fields: whFields, item: r, title: 'Anbar' })}>Düzəliş</Btn> : null },
      ]} rows={warehouses.data} actions={auth.canWrite() && <Btn sm variant="ghost" onClick={() => open({ path: '/warehouses', fields: whFields, item: { enabled: true }, title: 'Yeni anbar' })}>+ Anbar</Btn>} />

      <PasswordCard />

      {edit && <CrudModal {...edit} onClose={() => setEdit(null)} onSaved={() => { setEdit(null); currencies.reload(); taxes.reload(); warehouses.reload(); toast.ok('Yadda saxlanıldı') }} />}
    </>
  )
}

// Subscription: tenant admin self-selects modules; price follows the formula.
function Subscription() {
  const auth = useAuth()
  const toast = useToast()
  const tenant = useList('/my-tenant', [])
  const catalog = useList('/module-catalog', [])
  const [sel, setSel] = useState(null)
  const [busy, setBusy] = useState(false)
  if (!tenant.data || !catalog.data) return <div className="card mb-5"><Spinner /></div>

  const current = sel || new Set(tenant.data.modules)
  const toggle = (key) => {
    const n = new Set(current); n.has(key) ? n.delete(key) : n.add(key); setSel(n)
  }
  const price = catalog.data.filter((m) => !m.core && current.has(m.key)).reduce((s, m) => s + (m.price || 0), 0)
  const changed = sel !== null

  async function save() {
    setBusy(true)
    try {
      const modules = catalog.data.filter((m) => m.core || current.has(m.key)).map((m) => m.key)
      await api.put('/my-tenant/modules', { modules })
      toast.ok('Modullar yeniləndi')
      await auth.refreshCompanies() // refresh nav with new modules
      tenant.reload(); setSel(null)
    } catch (e) { toast.err(e.message) } finally { setBusy(false) }
  }

  return (
    <div className="card mb-5">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div>
          <div className="text-sm font-semibold">Abunə və modullar — {tenant.data.name}</div>
          <div className="text-xs text-muted">İstədiyiniz modulları seçin. Ödəniş avtomatik hesablanır.</div>
        </div>
        <div className="text-right">
          <div className="text-xs text-muted">Aylıq ödəniş</div>
          <div className="mono text-2xl font-extrabold text-brand">{money(price)} ₼</div>
        </div>
      </div>
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {catalog.data.map((m) => {
          const on = m.core || current.has(m.key)
          return (
            <label key={m.key}
              className={`flex items-center justify-between gap-2 rounded-xl border px-3 py-2.5 text-sm ${on ? 'border-brand bg-brand/5' : 'border-line'} ${m.core ? 'opacity-70' : 'cursor-pointer'}`}>
              <span className="flex items-center gap-2">
                <input type="checkbox" checked={on} disabled={m.core} onChange={() => toggle(m.key)} />
                {m.label}
              </span>
              <span className="text-xs text-muted">{m.core ? 'əsas' : (m.price ? m.price + '₼' : 'pulsuz')}</span>
            </label>
          )
        })}
      </div>
      {changed && <div className="mt-3"><Btn variant="primary" disabled={busy} onClick={save}>{busy ? 'Yadda saxlanılır...' : 'Modulları yadda saxla'}</Btn></div>}
    </div>
  )
}

function PasswordCard() {
  const toast = useToast()
  const [f, setF] = useState({ old_password: '', new_password: '' })
  async function save() {
    try { await api.post('/auth/change-password', f); toast.ok('Şifrə dəyişdirildi'); setF({ old_password: '', new_password: '' }) } catch (e) { toast.err(e.message) }
  }
  return (
    <div className="panel p-5">
      <h3 className="mb-3 text-[15px] font-semibold">Şifrə dəyişdir</h3>
      <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2">
        <Field label="Köhnə şifrə"><Input type="password" value={f.old_password} onChange={(e) => setF({ ...f, old_password: e.target.value })} /></Field>
        <Field label="Yeni şifrə"><Input type="password" value={f.new_password} onChange={(e) => setF({ ...f, new_password: e.target.value })} /></Field>
      </div>
      <Btn variant="primary" onClick={save}>Dəyiş</Btn>
    </div>
  )
}
