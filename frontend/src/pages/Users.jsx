import { useState } from 'react'
import { useList } from '../hooks.js'
import { api, roleLabel } from '../api.js'
import { Spinner, PageHeader, Table, Btn, Modal, Field, Input, Select, useConfirm, useToast } from '../ui.jsx'
import { useAuth } from '../store.jsx'

export default function Users() {
  const auth = useAuth()
  const toast = useToast()
  const cid = auth.company?.id
  const { data, reload } = useList('/companies/' + cid + '/users', [cid])
  const { data: roles } = useList('/roles', [])
  const [add, setAdd] = useState(false)
  const [editRole, setEditRole] = useState(null)
  const [confirm, confirmNode] = useConfirm()
  if (!data || !roles) return <Spinner />

  const roleOpts = roles.map((r) => ({ value: r.key, label: r.label }))
  const columns = [
    { h: 'Ad', render: (r) => <b>{r.name || '—'}</b> },
    { h: 'Email', k: 'email' },
    { h: 'Rol', render: (r) => roleLabel(r.role) },
    { h: '', right: true, render: (r) => (
      <div className="flex justify-end gap-2">
        <Btn sm variant="ghost" onClick={() => setEditRole(r)}>Rol</Btn>
        <Btn sm variant="danger" onClick={() => confirm('İstifadəçi şirkətdən çıxarılsın?', async () => { try { await api.del('/companies/' + cid + '/users/' + r.id); toast.ok('Silindi'); reload() } catch (e) { toast.err(e.message) } })}>Sil</Btn>
      </div>
    ) },
  ]
  return (
    <>
      <PageHeader title="İstifadəçilər">
        <Btn variant="primary" onClick={() => setAdd(true)}>+ İstifadəçi əlavə et</Btn>
      </PageHeader>
      <div className="card mb-4"><b>{auth.company?.name}</b> <span className="text-slate-400">— bu şirkətin istifadəçiləri və rolları</span></div>
      <Table title={`İstifadəçilər (${data.length})`} columns={columns} rows={data} />
      {add && <AddUser cid={cid} roleOpts={roleOpts} onClose={() => setAdd(false)} onSaved={() => { setAdd(false); reload() }} />}
      {editRole && <EditRole cid={cid} user={editRole} roleOpts={roleOpts} onClose={() => setEditRole(null)} onSaved={() => { setEditRole(null); reload() }} />}
      {confirmNode}
    </>
  )
}

function AddUser({ cid, roleOpts, onClose, onSaved }) {
  const toast = useToast()
  const [f, setF] = useState({ email: '', name: '', role: 'accountant', password: '' })
  const up = (k) => (e) => setF({ ...f, [k]: e.target.value })
  async function save() {
    if (!f.email || !f.role) { toast.err('Email və rol tələb olunur'); return }
    try { await api.post('/companies/' + cid + '/users', f); toast.ok('Əlavə olundu'); onSaved() } catch (e) { toast.err(e.message) }
  }
  return (
    <Modal title="İstifadəçi əlavə et" onClose={onClose} footer={<Btn variant="primary" onClick={save}>Əlavə et</Btn>}>
      <Field label="Email"><Input value={f.email} onChange={up('email')} /></Field>
      <Field label="Ad"><Input value={f.name} onChange={up('name')} /></Field>
      <Field label="Rol"><Select value={f.role} onChange={up('role')} options={roleOpts} /></Field>
      <Field label="Şifrə (yeni istifadəçi üçün, min 6)"><Input type="password" value={f.password} onChange={up('password')} /></Field>
    </Modal>
  )
}

function EditRole({ cid, user, roleOpts, onClose, onSaved }) {
  const toast = useToast()
  const [role, setRole] = useState(user.role)
  async function save() { try { await api.put('/companies/' + cid + '/users/' + user.id, { role }); toast.ok('Yeniləndi'); onSaved() } catch (e) { toast.err(e.message) } }
  return (
    <Modal title={'Rol — ' + (user.name || user.email)} onClose={onClose} footer={<Btn variant="primary" onClick={save}>Yadda saxla</Btn>}>
      <Field label="Rol"><Select value={role} onChange={(e) => setRole(e.target.value)} options={roleOpts} /></Field>
    </Modal>
  )
}
