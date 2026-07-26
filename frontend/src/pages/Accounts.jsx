import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useList } from '../hooks.js'
import { api } from '../api.js'
import { Spinner, PageHeader, Table, Btn, Modal, Field, Input, Select, Textarea, useToast } from '../ui.jsx'
import { useAuth } from '../store.jsx'

const TYPES = [
  { value: 'asset', label: 'Aktiv' }, { value: 'liability', label: 'Öhdəlik' },
  { value: 'equity', label: 'Kapital' }, { value: 'income', label: 'Gəlir' }, { value: 'expense', label: 'Xərc' },
]
const typeName = Object.fromEntries(TYPES.map((t) => [t.value, t.label]))

export default function Accounts() {
  const auth = useAuth()
  const toast = useToast()
  const nav = useNavigate()
  const { data, reload } = useList('/accounts', [auth.company?.id])
  const [edit, setEdit] = useState(null) // account or {} for new

  if (!data) return <Spinner />
  const columns = [
    { h: 'Kod', mono: true, render: (r) => <span className="mono">{r.code}</span> },
    { h: 'Ad', render: (r) => <span>{r.is_group ? <b>{r.name}</b> : r.name}{r.system_key && <span className="chip ml-2">sistem</span>}</span> },
    { h: 'Növ', render: (r) => typeName[r.type] || r.type },
    { h: '', right: true, render: (r) => (
      <div className="flex justify-end gap-2">
        <Btn sm variant="ghost" onClick={() => nav('/ledger/' + r.id)}>Baş kitab</Btn>
        {!r.is_group && auth.canWrite() && <Btn sm variant="ghost" onClick={() => setEdit(r)}>Düzəliş</Btn>}
      </div>
    ) },
  ]
  return (
    <>
      <PageHeader title="Hesablar planı">
        {auth.canWrite() && <Btn variant="primary" onClick={() => setEdit({ type: 'asset' })}>+ Yeni hesab</Btn>}
      </PageHeader>
      <Table title={`Hesablar (${data.length})`} columns={columns} rows={data} />
      {edit && <AccountModal item={edit} onClose={() => setEdit(null)} onSaved={() => { setEdit(null); reload(); toast.ok('Yadda saxlanıldı') }} />}
    </>
  )
}

function AccountModal({ item, onClose, onSaved }) {
  const toast = useToast()
  const isNew = !item.id
  const [f, setF] = useState({ code: '', name: '', type: 'asset', is_group: false, description: '', ...item })
  const up = (k) => (e) => setF({ ...f, [k]: e.target.type === 'checkbox' ? e.target.checked : e.target.value })
  async function save() {
    try {
      if (isNew) await api.post('/accounts', f)
      else await api.put('/accounts/' + item.id, f)
      onSaved()
    } catch (e) { toast.err(e.message) }
  }
  return (
    <Modal title={isNew ? 'Yeni hesab' : 'Hesab düzəlişi'} onClose={onClose}
      footer={<Btn variant="primary" onClick={save}>Yadda saxla</Btn>}>
      <Field label="Hesab kodu"><Input value={f.code} onChange={up('code')} /></Field>
      <Field label="Adı"><Input value={f.name} onChange={up('name')} /></Field>
      <Field label="Növ"><Select value={f.type} onChange={up('type')} options={TYPES} /></Field>
      <label className="mb-3.5 flex items-center gap-2 text-sm"><input type="checkbox" checked={!!f.is_group} onChange={up('is_group')} /> Qrup hesabı (yazılış qəbul etmir)</label>
      <Field label="Qeyd"><Textarea value={f.description || ''} onChange={up('description')} /></Field>
    </Modal>
  )
}
