import { useState } from 'react'
import { useList } from '../hooks.js'
import { api, money } from '../api.js'
import { Spinner, PageHeader, Table, Btn, Modal, Field, Input, Select, useToast } from '../ui.jsx'

export default function Tenants() {
  const toast = useToast()
  const { data, reload } = useList('/tenants', [])
  const { data: catalog } = useList('/module-catalog', [])
  const [edit, setEdit] = useState(null) // tenant or {_new:true}
  if (!data || !catalog) return <Spinner />

  const columns = [
    { h: 'Tenant / müqavilə', render: (r) => <span><b>{r.name}</b>{!r.enabled && <span className="chip ml-2 text-danger">deaktiv</span>}</span> },
    { h: 'Admin', render: (r) => r.admin_email || '—' },
    { h: 'Paket', render: (r) => r.plan },
    { h: 'Modullar', right: false, render: (r) => (r.modules || []).length + ' / ' + catalog.length },
    { h: 'Şirkət', right: true, render: (r) => r.company_count },
    { h: 'Abunə', right: true, render: (r) => money(r.subscription_amount) + ' ₼/' + (r.billing_cycle === 'yearly' ? 'il' : 'ay') },
    { h: '', right: true, render: (r) => <Btn sm variant="ghost" onClick={() => setEdit(r)}>Düzəliş</Btn> },
  ]
  return (
    <>
      <PageHeader title="Tenantlar (abunələr)">
        <Btn variant="primary" onClick={() => setEdit({ _new: true })}>+ Yeni tenant</Btn>
      </PageHeader>
      <div className="card mb-5">
        <div className="text-slate-400 text-sm">Cəmi aktiv abunə gəliri (aylıq ekvivalent)</div>
        <div className="mono text-2xl font-extrabold text-ok">
          {money(data.filter((t) => t.enabled).reduce((s, t) => s + (t.billing_cycle === 'yearly' ? t.subscription_amount / 12 : t.subscription_amount), 0))} ₼
        </div>
      </div>
      <Table title={`Tenantlar (${data.length})`} columns={columns} rows={data} />
      {edit && <TenantModal tenant={edit._new ? null : edit} catalog={catalog}
        onClose={() => setEdit(null)} onSaved={() => { setEdit(null); reload() }} />}
    </>
  )
}

function TenantModal({ tenant, catalog, onClose, onSaved }) {
  const toast = useToast()
  const isNew = !tenant
  const [f, setF] = useState({
    name: tenant?.name || '', contact_email: tenant?.contact_email || '', contact_phone: tenant?.contact_phone || '',
    plan: tenant?.plan || 'standard', billing_cycle: tenant?.billing_cycle || 'monthly',
    subscription_amount: tenant?.subscription_amount ?? 0,
    admin_email: '', admin_name: '', admin_password: '', company_name: '',
    enabled: tenant ? tenant.enabled : true,
  })
  const [mods, setMods] = useState(() => new Set(tenant ? tenant.modules : catalog.map((m) => m.key)))
  const up = (k) => (e) => setF({ ...f, [k]: e.target.type === 'checkbox' ? e.target.checked : e.target.value })
  const toggle = (key) => setMods((s) => { const n = new Set(s); n.has(key) ? n.delete(key) : n.add(key); return n })

  const suggested = catalog.filter((m) => mods.has(m.key)).reduce((s, m) => s + (m.price || 0), 0)

  async function save() {
    const modules = catalog.filter((m) => m.core || mods.has(m.key)).map((m) => m.key)
    try {
      if (isNew) {
        if (!f.name || !f.admin_email || f.admin_password.length < 6) { toast.err('Tenant adı, admin email və şifrə (min 6) tələb olunur'); return }
        await api.post('/tenants', {
          name: f.name, contact_email: f.contact_email, contact_phone: f.contact_phone, plan: f.plan,
          modules, subscription_amount: Number(f.subscription_amount) || 0, billing_cycle: f.billing_cycle,
          admin_email: f.admin_email, admin_name: f.admin_name, admin_password: f.admin_password, company_name: f.company_name,
        })
        toast.ok('Tenant və admin yaradıldı')
      } else {
        await api.put('/tenants/' + tenant.id, {
          name: f.name, contact_email: f.contact_email, contact_phone: f.contact_phone, plan: f.plan,
          modules, subscription_amount: Number(f.subscription_amount) || 0, billing_cycle: f.billing_cycle, enabled: f.enabled,
        })
        toast.ok('Yeniləndi')
      }
      onSaved()
    } catch (e) { toast.err(e.message) }
  }

  return (
    <Modal wide title={isNew ? 'Yeni tenant (müqavilə)' : 'Tenant: ' + tenant.name} onClose={onClose}
      footer={<Btn variant="primary" onClick={save}>{isNew ? 'Yarat' : 'Yadda saxla'}</Btn>}>
      <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2">
        <Field label="Tenant / şirkət adı"><Input value={f.name} onChange={up('name')} /></Field>
        <Field label="Əlaqə email"><Input value={f.contact_email} onChange={up('contact_email')} /></Field>
        <Field label="Əlaqə telefon"><Input value={f.contact_phone} onChange={up('contact_phone')} /></Field>
        <Field label="Paket"><Select value={f.plan} onChange={up('plan')} options={[{ value: 'basic', label: 'Basic' }, { value: 'standard', label: 'Standard' }, { value: 'pro', label: 'Pro' }]} /></Field>
      </div>

      <div className="panel my-3">
        <div className="border-b border-line px-4 py-2.5 text-sm font-semibold">Abunə olunan modullar (qiyməti müəyyən edir)</div>
        <div className="grid grid-cols-1 gap-2 p-4 sm:grid-cols-3">
          {catalog.map((m) => (
            <label key={m.key} className={`flex items-center gap-2 text-sm ${m.core ? 'opacity-60' : ''}`}>
              <input type="checkbox" checked={m.core || mods.has(m.key)} disabled={m.core} onChange={() => toggle(m.key)} />
              <span>{m.label}{m.core ? <span className="chip ml-1">əsas</span> : m.price ? <span className="text-slate-400"> · {m.price}₼</span> : null}</span>
            </label>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-3">
        <Field label={`Abunə məbləği (təklif: ${money(suggested)}₼)`}><Input type="number" step="0.01" value={f.subscription_amount} onChange={up('subscription_amount')} /></Field>
        <Field label="Dövr"><Select value={f.billing_cycle} onChange={up('billing_cycle')} options={[{ value: 'monthly', label: 'Aylıq' }, { value: 'yearly', label: 'İllik' }]} /></Field>
        {!isNew && <Field label="Status"><Select value={String(f.enabled)} onChange={(e) => setF({ ...f, enabled: e.target.value === 'true' })} options={[{ value: 'true', label: 'Aktiv' }, { value: 'false', label: 'Deaktiv' }]} /></Field>}
      </div>

      {isNew && (
        <div className="panel mt-2">
          <div className="border-b border-line px-4 py-2.5 text-sm font-semibold">Tenant admini (ona bir hesab verilir — qalanını özü idarə edir)</div>
          <div className="grid grid-cols-1 gap-3.5 p-4 sm:grid-cols-2">
            <Field label="Admin email"><Input value={f.admin_email} onChange={up('admin_email')} /></Field>
            <Field label="Admin ad"><Input value={f.admin_name} onChange={up('admin_name')} /></Field>
            <Field label="Admin şifrə (min 6)"><Input type="password" value={f.admin_password} onChange={up('admin_password')} /></Field>
            <Field label="İlk şirkət adı (boş = tenant adı)"><Input value={f.company_name} onChange={up('company_name')} /></Field>
          </div>
        </div>
      )}
    </Modal>
  )
}
