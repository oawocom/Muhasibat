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
      {auth.tenant && (
        <div className="card mb-5">
          <div className="text-slate-400 text-sm">Abunə paketi (modullar administrator tərəfindən idarə olunur)</div>
          <div className="mt-1 font-semibold">{auth.tenant.name} · {auth.tenant.plan}</div>
          <div className="mt-1 text-xs text-slate-400">Aktiv modullar: {(auth.tenant.modules || []).length}</div>
        </div>
      )}

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
