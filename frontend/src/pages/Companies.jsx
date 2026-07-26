import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, roleLabel } from '../api.js'
import { Spinner, PageHeader, Table, Btn, Modal, Field, Input, Select, useToast } from '../ui.jsx'
import { useAuth } from '../store.jsx'

export default function Companies() {
  const auth = useAuth()
  const toast = useToast()
  const nav = useNavigate()
  const [list, setList] = useState(null)
  const [create, setCreate] = useState(false)

  function reload() { api.get('/companies').then(setList).catch((e) => toast.err(e.message)) }
  useEffect(() => { reload() }, []) // eslint-disable-line
  if (!list) return <Spinner />

  const columns = [
    { h: 'Şirkət', render: (r) => <span><b>{r.name}</b>{auth.company?.id === r.id && <span className="chip ml-2">aktiv</span>}</span> },
    { h: 'VÖEN', mono: true, k: 'tax_id' },
    { h: 'Rolunuz', render: (r) => roleLabel(r.role) },
    { h: '', right: true, render: (r) => (
      <div className="flex justify-end gap-2">
        <Btn sm variant="ghost" onClick={() => { auth.chooseCompany(r); toast.ok('Aktiv: ' + r.name); nav('/') }}>Keç</Btn>
      </div>
    ) },
  ]
  return (
    <>
      <PageHeader title="Şirkətlər">
        {auth.canCreateCompany() && <Btn variant="primary" onClick={() => setCreate(true)}>+ Yeni şirkət</Btn>}
      </PageHeader>
      <Table title={`Şirkətlər (${list.length})`} columns={columns} rows={list} />
      {create && <CreateCompany onClose={() => setCreate(false)} onSaved={() => { setCreate(false); reload(); auth.refreshCompanies() }} />}
    </>
  )
}

function CreateCompany({ onClose, onSaved }) {
  const auth = useAuth()
  const toast = useToast()
  const [f, setF] = useState({ name: '', tax_id: '', tenant_id: '' })
  const [tenants, setTenants] = useState([])
  const [busy, setBusy] = useState(false)
  useEffect(() => { if (auth.isSuper) api.get('/tenants').then(setTenants).catch(() => {}) }, [auth.isSuper])

  async function save() {
    if (!f.name) { toast.err('Ad tələb olunur'); return }
    if (auth.isSuper && !f.tenant_id) { toast.err('Tenant seçin'); return }
    setBusy(true)
    try {
      await api.post('/companies', { name: f.name, tax_id: f.tax_id, tenant_id: Number(f.tenant_id) || undefined })
      toast.ok('Şirkət yaradıldı'); onSaved()
    } catch (e) { toast.err(e.message) } finally { setBusy(false) }
  }
  return (
    <Modal title="Yeni şirkət" onClose={onClose} footer={<Btn variant="primary" disabled={busy} onClick={save}>{busy ? 'Yaradılır...' : 'Yarat'}</Btn>}>
      {auth.isSuper && (
        <Field label="Tenant">
          <Select value={f.tenant_id} onChange={(e) => setF({ ...f, tenant_id: e.target.value })}
            options={[{ value: '', label: '— seçin —' }, ...tenants.map((t) => ({ value: t.id, label: t.name }))]} />
        </Field>
      )}
      <Field label="Şirkət adı"><Input value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} /></Field>
      <Field label="VÖEN"><Input value={f.tax_id} onChange={(e) => setF({ ...f, tax_id: e.target.value })} /></Field>
    </Modal>
  )
}
