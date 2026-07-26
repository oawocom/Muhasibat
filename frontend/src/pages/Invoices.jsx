import { useState } from 'react'
import { useList } from '../hooks.js'
import { api, money, fmtDate, today } from '../api.js'
import { Spinner, PageHeader, Table, Btn, Modal, Field, Input, Select, Badge, useToast } from '../ui.jsx'
import { useAuth } from '../store.jsx'
import { printInvoice } from './print.js'

export default function Invoices({ type }) {
  const auth = useAuth()
  const toast = useToast()
  const isSales = type === 'sales_invoice'
  const partnerType = isSales ? 'customer' : 'supplier'
  const { data, reload } = useList('/documents?type=' + type, [type, auth.company?.id])
  const { data: partners } = useList('/partners', [auth.company?.id])
  const { data: products } = useList('/products', [auth.company?.id])
  const { data: warehouses } = useList('/warehouses', [auth.company?.id])
  const [edit, setEdit] = useState(null)
  if (!data || !partners || !products || !warehouses) return <Spinner />

  const pname = (id) => partners.find((p) => p.id === id)?.name || '—'
  const columns = [
    { h: 'Nömrə', mono: true, k: 'number' },
    { h: 'Tarix', render: (r) => fmtDate(r.date) },
    { h: 'Tərəfdaş', render: (r) => pname(r.partner_id) },
    { h: 'Cəmi', right: true, render: (r) => money(r.total) + ' ₼' },
    { h: 'ƏDV', right: true, render: (r) => money(r.tax_total) },
    { h: 'Status', render: (r) => <Badge status={r.status} /> },
  ]
  return (
    <>
      <PageHeader title={isSales ? 'Satış fakturaları' : 'Alış fakturaları'}>
        {auth.canWrite() && <Btn variant="primary" onClick={() => setEdit({ _new: true })}>+ Yeni faktura</Btn>}
      </PageHeader>
      <Table title={`Fakturalar (${data.length})`} columns={columns} rows={data} onRow={setEdit} />
      {edit && (
        <InvoiceModal type={type} isSales={isSales} partnerType={partnerType}
          doc={edit._new ? null : edit} partners={partners} products={products} warehouses={warehouses}
          company={auth.company} canWrite={auth.canWrite()}
          onClose={() => setEdit(null)} onSaved={() => { setEdit(null); reload() }} />
      )}
    </>
  )
}

function InvoiceModal({ type, isSales, partnerType, doc, partners, products, warehouses, company, canWrite, onClose, onSaved }) {
  const toast = useToast()
  const editable = !doc || doc.status === 'draft'
  const [head, setHead] = useState({
    partner_id: doc?.partner_id || '',
    date: doc ? fmtDate(doc.date) : today(),
    warehouse_id: doc?.warehouse_id || (warehouses[0]?.id ?? ''),
  })
  const [lines, setLines] = useState(
    doc?.lines?.length ? doc.lines.map((l) => ({ ...l })) : [{ product_id: '', description: '', quantity: 1, unit_price: 0, tax_rate: 18 }],
  )
  const partnerOpts = [{ value: '', label: '— seçin —' }, ...partners.filter((p) => p.type === partnerType || p.type === 'both').map((p) => ({ value: p.id, label: p.name }))]

  function setLine(i, patch) { setLines((ls) => ls.map((l, j) => (j === i ? { ...l, ...patch } : l))) }
  function onProduct(i, pid) {
    const p = products.find((x) => x.id === Number(pid))
    if (p) setLine(i, { product_id: pid, description: p.name, unit_price: isSales ? p.sale_price : p.cost_price })
    else setLine(i, { product_id: '' })
  }
  const sub = lines.reduce((s, l) => s + (Number(l.quantity) || 0) * (Number(l.unit_price) || 0), 0)
  const tax = lines.reduce((s, l) => s + ((Number(l.quantity) || 0) * (Number(l.unit_price) || 0) * (Number(l.tax_rate) || 0)) / 100, 0)

  function payload() {
    return {
      type, partner_id: Number(head.partner_id) || null, warehouse_id: Number(head.warehouse_id) || null,
      date: head.date, fx_rate: 1,
      lines: lines.map((l, idx) => ({
        product_id: Number(l.product_id) || null, description: l.description || '',
        quantity: Number(l.quantity) || 0, unit_price: Number(l.unit_price) || 0, tax_rate: Number(l.tax_rate) || 0, sort_order: idx,
      })),
    }
  }
  async function persist(post) {
    const body = payload()
    if (!body.partner_id) { toast.err('Tərəfdaş seçin'); return }
    try {
      if (doc) { await api.put('/documents/' + doc.id, body); if (post) await api.post('/documents/' + doc.id + '/post') }
      else { await api.post('/documents' + (post ? '?post=1' : ''), body) }
      toast.ok(post ? 'Faktura təsdiqləndi' : 'Yadda saxlanıldı'); onSaved()
    } catch (e) { toast.err(e.message) }
  }

  function doPrint() {
    printInvoice({
      type, company, partner: partners.find((p) => p.id === Number(head.partner_id)),
      number: doc?.number || '(layihə)', date: head.date, lines, sub, tax, total: sub + tax,
    })
  }

  const footer = (
    <>
      {doc && <Btn variant="ghost" onClick={doPrint}>🖨 Çap / PDF</Btn>}
      {editable && canWrite && <>
        <Btn onClick={() => persist(false)}>Layihə saxla</Btn>
        <Btn variant="primary" onClick={() => persist(true)}>Təsdiqlə</Btn>
      </>}
      {!editable && <span className="chip"><Badge status={doc.status} /></span>}
    </>
  )

  return (
    <Modal wide title={(isSales ? 'Satış fakturası' : 'Alış fakturası') + (doc ? ' ' + doc.number : '')} onClose={onClose} footer={footer}>
      <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-3">
        <Field label={isSales ? 'Müştəri' : 'Təchizatçı'}>
          <Select value={head.partner_id} onChange={(e) => setHead({ ...head, partner_id: e.target.value })} options={partnerOpts} />
        </Field>
        <Field label="Tarix"><Input type="date" value={head.date} onChange={(e) => setHead({ ...head, date: e.target.value })} /></Field>
        <Field label="Anbar">
          <Select value={head.warehouse_id} onChange={(e) => setHead({ ...head, warehouse_id: e.target.value })}
            options={[{ value: '', label: '—' }, ...warehouses.map((w) => ({ value: w.id, label: w.name }))]} />
        </Field>
      </div>

      <div className="panel mt-2">
        <div className="flex items-center justify-between border-b border-line px-4 py-2.5">
          <h3 className="text-sm font-semibold">Sətirlər</h3>
          {editable && <Btn sm variant="ghost" onClick={() => setLines([...lines, { product_id: '', description: '', quantity: 1, unit_price: 0, tax_rate: 18 }])}>+ Sətir</Btn>}
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead><tr>
              <th className="th">Məhsul / təsvir</th><th className="th w-20">Say</th><th className="th w-28">Qiymət</th><th className="th w-20">ƏDV %</th><th className="th w-28 text-right">Məbləğ</th><th className="th w-8"></th>
            </tr></thead>
            <tbody>
              {lines.map((l, i) => (
                <tr key={i}>
                  <td className="td">
                    <select className="input mb-1" disabled={!editable} value={l.product_id || ''} onChange={(e) => onProduct(i, e.target.value)}>
                      <option value="">— sərbəst —</option>
                      {products.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
                    </select>
                    <Input disabled={!editable} placeholder="Təsvir" value={l.description || ''} onChange={(e) => setLine(i, { description: e.target.value })} />
                  </td>
                  <td className="td"><Input disabled={!editable} type="number" step="0.001" className="mono text-right" value={l.quantity} onChange={(e) => setLine(i, { quantity: e.target.value })} /></td>
                  <td className="td"><Input disabled={!editable} type="number" step="0.01" className="mono text-right" value={l.unit_price} onChange={(e) => setLine(i, { unit_price: e.target.value })} /></td>
                  <td className="td"><Input disabled={!editable} type="number" step="0.01" className="mono text-right" value={l.tax_rate} onChange={(e) => setLine(i, { tax_rate: e.target.value })} /></td>
                  <td className="td mono text-right">{money((Number(l.quantity) || 0) * (Number(l.unit_price) || 0))}</td>
                  <td className="td text-right">{editable && <button className="text-danger" onClick={() => setLines(lines.filter((_, j) => j !== i))}>✕</button>}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="mt-3 flex justify-end gap-8 pr-1">
        <div><span className="text-muted">Ara cəm:</span> <b className="mono">{money(sub)} ₼</b></div>
        <div><span className="text-muted">ƏDV:</span> <b className="mono">{money(tax)} ₼</b></div>
        <div><span className="text-muted">Yekun:</span> <b className="mono text-base">{money(sub + tax)} ₼</b></div>
      </div>
    </Modal>
  )
}
