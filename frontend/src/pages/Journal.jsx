import { useState } from 'react'
import { useList } from '../hooks.js'
import { api, money, fmtDate, today } from '../api.js'
import { Spinner, PageHeader, Table, Btn, Modal, Field, Input, Select, Badge, useToast } from '../ui.jsx'
import { useAuth } from '../store.jsx'

export default function Journal() {
  const auth = useAuth()
  const { data, reload } = useList('/journal', [auth.company?.id])
  const { data: accounts } = useList('/accounts', [auth.company?.id])
  const [edit, setEdit] = useState(null)
  if (!data || !accounts) return <Spinner />

  const columns = [
    { h: 'Nömrə', mono: true, k: 'number' },
    { h: 'Tarix', render: (r) => fmtDate(r.date) },
    { h: 'Təsvir', render: (r) => <span>{r.description}{r.reference && <span className="ml-1 text-muted">/ {r.reference}</span>}</span> },
    { h: 'Debet', right: true, render: (r) => money(r.total_debit) },
    { h: 'Kredit', right: true, render: (r) => money(r.total_credit) },
    { h: 'Status', render: (r) => <Badge status={r.status} /> },
  ]
  return (
    <>
      <PageHeader title="Mühasibat jurnalı">
        {auth.canWrite() && <Btn variant="primary" onClick={() => setEdit({ _new: true })}>+ Yeni yazılış</Btn>}
      </PageHeader>
      <Table title={`Yazılışlar (${data.length})`} columns={columns} rows={data} onRow={(r) => setEdit(r)} />
      {edit && <JournalModal entry={edit._new ? null : edit} accounts={accounts} canWrite={auth.canWrite()}
        onClose={() => setEdit(null)} onSaved={() => { setEdit(null); reload() }} />}
    </>
  )
}

function JournalModal({ entry, accounts, canWrite, onClose, onSaved }) {
  const toast = useToast()
  const posted = entry && entry.status === 'posted'
  const editable = !posted && canWrite
  const [head, setHead] = useState({ date: entry ? fmtDate(entry.date) : today(), description: entry?.description || '', reference: entry?.reference || '' })
  const [lines, setLines] = useState(
    entry?.lines?.length ? entry.lines.map((l) => ({ ...l })) : [{ account_id: '', description: '', debit: 0, credit: 0 }, { account_id: '', description: '', debit: 0, credit: 0 }],
  )
  const postable = accounts.filter((a) => !a.is_group)
  const accName = (id) => { const a = accounts.find((x) => x.id === id); return a ? a.code + ' — ' + a.name : id }

  function setLine(i, patch) { setLines((ls) => ls.map((l, j) => (j === i ? { ...l, ...patch } : l))) }
  const d = lines.reduce((s, l) => s + (Number(l.debit) || 0), 0)
  const cr = lines.reduce((s, l) => s + (Number(l.credit) || 0), 0)
  const balanced = Math.abs(d - cr) < 0.005

  async function save(post) {
    const body = {
      date: head.date, description: head.description, reference: head.reference,
      lines: lines.filter((l) => l.account_id).map((l, i) => ({
        account_id: Number(l.account_id), description: l.description || '',
        debit: Number(l.debit) || 0, credit: Number(l.credit) || 0, sort_order: i,
      })),
    }
    try {
      if (entry) { await api.put('/journal/' + entry.id, body); if (post) await api.post('/journal/' + entry.id + '/post') }
      else { await api.post('/journal' + (post ? '?post=1' : ''), body) }
      toast.ok('Yadda saxlanıldı'); onSaved()
    } catch (e) { toast.err(e.message) }
  }
  async function unpost() { try { await api.post('/journal/' + entry.id + '/unpost'); toast.ok('Geri qaytarıldı'); onSaved() } catch (e) { toast.err(e.message) } }

  const footer = (
    <>
      {posted ? <>
        {canWrite && <Btn onClick={unpost}>Geri qaytar</Btn>}
        <span className="chip"><Badge status="posted" /></span>
      </> : editable ? <>
        <Btn onClick={() => save(false)}>Layihə saxla</Btn>
        <Btn variant="primary" disabled={!balanced} onClick={() => save(true)}>Təsdiqlə</Btn>
      </> : null}
    </>
  )
  return (
    <Modal wide title={'Mühasibat yazılışı' + (entry ? ' ' + entry.number : '')} onClose={onClose} footer={footer}>
      <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-3">
        <Field label="Tarix"><Input type="date" disabled={!editable} value={head.date} onChange={(e) => setHead({ ...head, date: e.target.value })} /></Field>
        <Field label="Təsvir"><Input disabled={!editable} value={head.description} onChange={(e) => setHead({ ...head, description: e.target.value })} /></Field>
        <Field label="İstinad"><Input disabled={!editable} value={head.reference} onChange={(e) => setHead({ ...head, reference: e.target.value })} /></Field>
      </div>
      <div className="panel mt-2">
        <div className="flex items-center justify-between border-b border-line px-4 py-2.5">
          <h3 className="text-sm font-semibold">Yazılış sətirləri</h3>
          {editable && <Btn sm variant="ghost" onClick={() => setLines([...lines, { account_id: '', description: '', debit: 0, credit: 0 }])}>+ Sətir</Btn>}
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead><tr><th className="th">Hesab</th><th className="th">Təsvir</th><th className="th w-28 text-right">Debet</th><th className="th w-28 text-right">Kredit</th><th className="th w-8"></th></tr></thead>
            <tbody>
              {lines.map((l, i) => (
                <tr key={i}>
                  <td className="td">
                    {editable ? (
                      <select className="input" value={l.account_id || ''} onChange={(e) => setLine(i, { account_id: e.target.value })}>
                        <option value="">— hesab —</option>
                        {postable.map((a) => <option key={a.id} value={a.id}>{a.code} — {a.name}</option>)}
                      </select>
                    ) : accName(l.account_id)}
                  </td>
                  <td className="td"><Input disabled={!editable} value={l.description || ''} onChange={(e) => setLine(i, { description: e.target.value })} /></td>
                  <td className="td"><Input disabled={!editable} type="number" step="0.01" className="mono text-right" value={l.debit || 0} onChange={(e) => setLine(i, { debit: e.target.value, credit: Number(e.target.value) ? 0 : l.credit })} /></td>
                  <td className="td"><Input disabled={!editable} type="number" step="0.01" className="mono text-right" value={l.credit || 0} onChange={(e) => setLine(i, { credit: e.target.value, debit: Number(e.target.value) ? 0 : l.debit })} /></td>
                  <td className="td text-right">{editable && <button className="text-danger" onClick={() => setLines(lines.filter((_, j) => j !== i))}>✕</button>}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
      <div className="mt-3 flex justify-end gap-8 pr-1">
        <div><span className="text-muted">Debet:</span> <b className="mono">{money(d)}</b></div>
        <div><span className="text-muted">Kredit:</span> <b className="mono">{money(cr)}</b></div>
        <div><span className="text-muted">Fərq:</span> <b className={`mono ${balanced ? 'text-ok' : 'text-danger'}`}>{money(d - cr)}</b></div>
      </div>
    </Modal>
  )
}
