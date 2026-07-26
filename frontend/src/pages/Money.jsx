import { useState } from 'react'
import { useList } from '../hooks.js'
import { api, money, fmtDate, today } from '../api.js'
import { Spinner, PageHeader, Table, Btn, Modal, Field, Input, Select, Badge, useToast } from '../ui.jsx'
import { useAuth } from '../store.jsx'

export default function Money() {
  const auth = useAuth()
  const { data, reload } = useList('/documents', [auth.company?.id])
  const { data: partners } = useList('/partners', [auth.company?.id])
  const { data: accounts } = useList('/accounts', [auth.company?.id])
  const [edit, setEdit] = useState(null)
  if (!data || !partners || !accounts) return <Spinner />

  const list = data.filter((d) => d.type === 'payment' || d.type === 'receipt')
  const pname = (id) => partners.find((p) => p.id === id)?.name || '—'
  const columns = [
    { h: 'Nömrə', mono: true, k: 'number' },
    { h: 'Tarix', render: (r) => fmtDate(r.date) },
    { h: 'Növ', render: (r) => (r.type === 'receipt' ? <span className="text-ok">Mədaxil</span> : <span className="text-danger">Məxaric</span>) },
    { h: 'Tərəfdaş', render: (r) => pname(r.partner_id) },
    { h: 'Məbləğ', right: true, render: (r) => money(r.total) + ' ₼' },
    { h: 'Status', render: (r) => <Badge status={r.status} /> },
  ]
  return (
    <>
      <PageHeader title="Kassa / Bank">
        {auth.canWrite() && <>
          <Btn variant="primary" onClick={() => setEdit({ type: 'receipt' })}>+ Mədaxil</Btn>
          <Btn onClick={() => setEdit({ type: 'payment' })}>+ Məxaric</Btn>
        </>}
      </PageHeader>
      <Table title={`Əməliyyatlar (${list.length})`} columns={columns} rows={list} />
      {edit && <MoneyModal type={edit.type} partners={partners} accounts={accounts}
        onClose={() => setEdit(null)} onSaved={() => { setEdit(null); reload() }} />}
    </>
  )
}

function MoneyModal({ type, partners, accounts, onClose, onSaved }) {
  const toast = useToast()
  const isReceipt = type === 'receipt'
  const partnerType = isReceipt ? 'customer' : 'supplier'
  const cash = accounts.filter((a) => a.system_key === 'cash' || a.system_key === 'bank')
  const [f, setF] = useState({ partner_id: '', cash_account_id: cash[0]?.id ?? '', date: today(), amount: '', notes: '' })
  const up = (k) => (e) => setF({ ...f, [k]: e.target.value })

  async function save() {
    if (!f.partner_id || !f.cash_account_id || !f.amount) { toast.err('Bütün sahələri doldurun'); return }
    const body = {
      type, partner_id: Number(f.partner_id), cash_account_id: Number(f.cash_account_id),
      date: f.date, notes: f.notes, fx_rate: 1,
      lines: [{ description: isReceipt ? 'Müştəridən mədaxil' : 'Təchizatçıya ödəniş', quantity: 1, unit_price: Number(f.amount), tax_rate: 0 }],
    }
    try { await api.post('/documents?post=1', body); toast.ok('Təsdiqləndi'); onSaved() } catch (e) { toast.err(e.message) }
  }
  return (
    <Modal title={isReceipt ? 'Mədaxil order' : 'Məxaric order'} onClose={onClose}
      footer={<Btn variant="primary" onClick={save}>Təsdiqlə</Btn>}>
      <Field label={isReceipt ? 'Müştəri' : 'Təchizatçı'}>
        <Select value={f.partner_id} onChange={up('partner_id')}
          options={[{ value: '', label: '— seçin —' }, ...partners.filter((p) => p.type === partnerType || p.type === 'both').map((p) => ({ value: p.id, label: p.name }))]} />
      </Field>
      <Field label="Kassa / Bank">
        <Select value={f.cash_account_id} onChange={up('cash_account_id')}
          options={cash.map((a) => ({ value: a.id, label: a.code + ' — ' + a.name }))} />
      </Field>
      <Field label="Tarix"><Input type="date" value={f.date} onChange={up('date')} /></Field>
      <Field label="Məbləğ (₼)"><Input type="number" step="0.01" value={f.amount} onChange={up('amount')} /></Field>
      <Field label="Təyinat"><Input value={f.notes} onChange={up('notes')} /></Field>
    </Modal>
  )
}
