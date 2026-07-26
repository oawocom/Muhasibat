import { useState } from 'react'
import { useList } from '../hooks.js'
import { api, money, fmtDate } from '../api.js'
import { Spinner, PageHeader, Table, Btn, Modal, Field, Input, Badge, useToast } from '../ui.jsx'
import { useAuth } from '../store.jsx'

const statusMap = { draft: ['Layihə', 'draft'], sent: ['Göndərilib', 'posted'], registered: ['Qeydiyyatlı', 'posted'], cancelled: ['Ləğv', 'void'] }

export default function EInvoice() {
  const auth = useAuth()
  const toast = useToast()
  const eligible = useList('/einvoices/eligible', [auth.company?.id])
  const list = useList('/einvoices', [auth.company?.id])
  const cfg = useList('/einvoice/config', [auth.company?.id])
  const [detail, setDetail] = useState(null)
  if (!eligible.data || !list.data || !cfg.data) return <Spinner />

  async function create(docId) {
    try { await api.post('/einvoices', { document_id: docId }); toast.ok('e-Qaimə yaradıldı'); eligible.reload(); list.reload() }
    catch (e) { toast.err(e.message) }
  }
  async function open(id) { try { setDetail(await api.get('/einvoices/' + id)) } catch (e) { toast.err(e.message) } }

  const eligCols = [
    { h: 'Faktura', mono: true, k: 'number' },
    { h: 'Tarix', render: (r) => fmtDate(r.date) },
    { h: 'Cəmi', right: true, render: (r) => money(r.total) + ' ₼' },
    { h: '', right: true, render: (r) => auth.canWrite() && <Btn sm variant="primary" onClick={() => create(r.id)}>e-Qaimə yarat</Btn> },
  ]
  const eiCols = [
    { h: 'Faktura', mono: true, render: (r) => r.invoice_number },
    { h: 'Alıcı', render: (r) => r.buyer },
    { h: 'Seriya/№', render: (r) => (r.einvoice.series || r.einvoice.number) ? `${r.einvoice.series} ${r.einvoice.number}` : '—' },
    { h: 'Məbləğ', right: true, render: (r) => money(r.einvoice.total) + ' ₼' },
    { h: 'Status', render: (r) => <Badge status={(statusMap[r.einvoice.status] || ['', 'draft'])[1]} /> },
  ]
  return (
    <>
      <PageHeader title="e-Qaimə" />
      <ConfigCard cfg={cfg} companyName={auth.company?.name} canWrite={auth.canWrite()} />
      <Table title={`Hazır satış fakturaları (${eligible.data.length})`} columns={eligCols} rows={eligible.data} empty="e-qaimə üçün təsdiqlənmiş satış fakturası yoxdur" />
      <Table title={`e-Qaimələr (${list.data.length})`} columns={eiCols} rows={list.data} onRow={(r) => open(r.einvoice.id)} />
      {detail && <DetailModal e={detail} cfg={cfg.data} canWrite={auth.canWrite()} onClose={() => setDetail(null)} onChanged={() => { open(detail.id); list.reload() }} />}
    </>
  )
}

function ConfigCard({ cfg, companyName, canWrite }) {
  const toast = useToast()
  const [open, setOpen] = useState(false)
  const [f, setF] = useState(null)
  const state = f || { ...cfg.data, company_name: cfg.data.company_name || companyName || '' }
  const set = (k) => (e) => setF({ ...state, [k]: e.target.value })
  async function save() { try { await api.put('/einvoice/config', state); toast.ok('Yadda saxlanıldı'); cfg.reload(); setF(null); setOpen(false) } catch (e) { toast.err(e.message) } }
  return (
    <div className="card mb-4">
      <div className="flex items-center justify-between">
        <div><b>Satıcı VÖEN:</b> <span className="mono">{cfg.data.seller_voen || '— təyin edilməyib'}</span>{cfg.data.endpoint && <span className="chip ml-2">inteqrasiya aktiv</span>}</div>
        {canWrite && <Btn sm variant="ghost" onClick={() => setOpen(!open)}>{open ? 'Bağla' : 'Parametrlər'}</Btn>}
      </div>
      {open && (
        <div className="mt-3 grid grid-cols-1 gap-3.5 sm:grid-cols-2">
          <Field label="Satıcı VÖEN"><Input value={state.seller_voen || ''} onChange={set('seller_voen')} /></Field>
          <Field label="Şirkət adı"><Input value={state.company_name || ''} onChange={set('company_name')} /></Field>
          <Field label="e-Qaimə API ünvanı (endpoint) — gələcək inteqrasiya"><Input value={state.endpoint || ''} onChange={set('endpoint')} placeholder="https://..." /></Field>
          <Field label="API token"><Input value={state.token || ''} onChange={set('token')} /></Field>
          <div className="sm:col-span-2"><Btn variant="primary" onClick={save}>Yadda saxla</Btn>
            <p className="mt-2 text-xs text-muted">Qeyd: dövlət e-qaimə sisteminə avtomatik göndəriş üçün e-taxes.gov.az rəsmi API ünvanı və token lazımdır. Onlar olmadan e-qaimə hazırlanır, rəsmi seriya/nömrəni əl ilə qeyd edirsiniz.</p>
          </div>
        </div>
      )}
    </div>
  )
}

function DetailModal({ e, cfg, canWrite, onClose, onChanged }) {
  const toast = useToast()
  const ei = e
  const [reg, setReg] = useState({ series: ei.series || '', number: ei.number || '', status: 'registered', note: ei.note || '' })
  async function saveReg() {
    try { await api.put('/einvoices/' + ei.id, reg); toast.ok('Qeydiyyat yeniləndi'); onChanged() } catch (er) { toast.err(er.message) }
  }
  async function send() {
    try { const r = await api.post('/einvoices/' + ei.id + '/send'); toast.ok('Göndərildi'); onChanged() } catch (er) { toast.err(er.message) }
  }
  function printIt() {
    const w = window.open('', '_blank'); if (!w) return
    w.document.write('<pre style="font-family:monospace;padding:24px;white-space:pre-wrap">' +
      JSON.stringify(ei.payload, null, 2).replace(/</g, '&lt;') + '</pre><script>window.print()</script>')
    w.document.close()
  }
  return (
    <Modal wide title="e-Qaimə" onClose={onClose}
      footer={<>
        <Btn variant="ghost" onClick={printIt}>🖨 Payload</Btn>
        {cfg.endpoint && canWrite && <Btn onClick={send}>Portala göndər</Btn>}
        {canWrite && <Btn variant="primary" onClick={saveReg}>Qeydiyyatı təsdiqlə</Btn>}
      </>}>
      <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-3">
        <Field label="Rəsmi seriya"><Input value={reg.series} onChange={(e2) => setReg({ ...reg, series: e2.target.value })} /></Field>
        <Field label="Rəsmi nömrə"><Input value={reg.number} onChange={(e2) => setReg({ ...reg, number: e2.target.value })} /></Field>
        <Field label="Status">
          <select className="input" value={reg.status} onChange={(e2) => setReg({ ...reg, status: e2.target.value })}>
            <option value="draft">Layihə</option><option value="registered">Qeydiyyatlı</option><option value="cancelled">Ləğv</option>
          </select>
        </Field>
      </div>
      <div className="panel">
        <div className="border-b border-line px-4 py-2.5 text-sm font-semibold">e-Qaimə məzmunu (payload)</div>
        <pre className="max-h-80 overflow-auto p-4 text-xs text-muted">{JSON.stringify(ei.payload, null, 2)}</pre>
      </div>
    </Modal>
  )
}
