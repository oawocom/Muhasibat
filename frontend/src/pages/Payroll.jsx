import { useState } from 'react'
import { useList } from '../hooks.js'
import { api, money, fmtDate, today } from '../api.js'
import { Spinner, PageHeader, Table, Btn, Modal, Field, Input, Select, Badge, useToast, useConfirm } from '../ui.jsx'
import { CrudModal } from './crud.jsx'
import { useAuth } from '../store.jsx'

const TABS = [{ k: 'employees', t: 'İşçilər' }, { k: 'runs', t: 'Hesablamalar' }, { k: 'config', t: 'Parametrlər' }]

export default function Payroll() {
  const [tab, setTab] = useState('employees')
  return (
    <>
      <PageHeader title="Əmək haqqı" />
      <div className="mb-4 flex gap-2">
        {TABS.map((t) => (
          <button key={t.k} onClick={() => setTab(t.k)}
            className={`btn btn-sm ${tab === t.k ? 'btn-primary' : 'btn-ghost'}`}>{t.t}</button>
        ))}
      </div>
      {tab === 'employees' && <Employees />}
      {tab === 'runs' && <Runs />}
      {tab === 'config' && <Config />}
    </>
  )
}

function Employees() {
  const auth = useAuth()
  const toast = useToast()
  const { data, reload } = useList('/employees', [auth.company?.id])
  const [edit, setEdit] = useState(null)
  if (!data) return <Spinner />
  const fields = [
    { k: 'full_name', label: 'Ad, soyad' }, { k: 'code', label: 'Tabel №' }, { k: 'position', label: 'Vəzifə' },
    { k: 'tax_id', label: 'FİN' }, { k: 'bank_account', label: 'Bank hesabı / kart' },
    { k: 'hire_date', label: 'İşə qəbul tarixi', type: 'date' },
    { k: 'salary', label: 'Aylıq əmək haqqı (gross, ₼)', type: 'number', step: '0.01' },
    { k: 'status', label: 'Status', type: 'select', def: 'active', options: [{ value: 'active', label: 'Aktiv' }, { value: 'inactive', label: 'Deaktiv' }] },
    { k: 'notes', label: 'Qeyd', type: 'textarea' },
  ]
  const columns = [
    { h: 'Ad', render: (r) => <b>{r.full_name}</b> },
    { h: 'Vəzifə', k: 'position' },
    { h: 'FİN', mono: true, k: 'tax_id' },
    { h: 'Əmək haqqı', right: true, render: (r) => money(r.salary) + ' ₼' },
    { h: 'Status', render: (r) => <span className="chip">{r.status === 'active' ? 'Aktiv' : 'Deaktiv'}</span> },
  ]
  return (
    <>
      {auth.canWrite() && <div className="mb-3"><Btn variant="primary" onClick={() => setEdit({})}>+ Yeni işçi</Btn></div>}
      <Table title={`İşçilər (${data.length})`} columns={columns} rows={data} onRow={auth.canWrite() ? setEdit : undefined} />
      {edit && <CrudModal title={edit.id ? 'İşçi' : 'Yeni işçi'} fields={fields} item={edit} path="/employees"
        onClose={() => setEdit(null)} onSaved={() => { setEdit(null); reload(); toast.ok('Yadda saxlanıldı') }} />}
    </>
  )
}

function Runs() {
  const auth = useAuth()
  const toast = useToast()
  const { data, reload } = useList('/payroll/runs', [auth.company?.id])
  const [detail, setDetail] = useState(null)
  const [creating, setCreating] = useState(false)
  const [confirm, confirmNode] = useConfirm()
  if (!data) return <Spinner />

  async function openRun(id) {
    try { setDetail(await api.get('/payroll/runs/' + id)) } catch (e) { toast.err(e.message) }
  }
  const columns = [
    { h: 'Period', render: (r) => fmtDate(r.period).slice(0, 7) },
    { h: 'Gross', right: true, render: (r) => money(r.gross_total) },
    { h: 'Tutulmalar', right: true, render: (r) => money(r.deductions_total) },
    { h: 'İşəgötürən', right: true, render: (r) => money(r.employer_total) },
    { h: 'Net (ödəniləcək)', right: true, render: (r) => money(r.net_total) },
    { h: 'Status', render: (r) => <Badge status={r.status} /> },
  ]
  return (
    <>
      {auth.canWrite() && <div className="mb-3"><Btn variant="primary" onClick={() => setCreating(true)}>+ Yeni hesablama</Btn></div>}
      <Table title={`Hesablamalar (${data.length})`} columns={columns} rows={data} onRow={(r) => openRun(r.id)} />
      {creating && <CreateRun onClose={() => setCreating(false)} onCreated={(run) => { setCreating(false); reload(); setDetail(run) }} />}
      {detail && <RunDetail run={detail} canWrite={auth.canWrite()} onClose={() => setDetail(null)}
        onChanged={() => { openRun(detail.id); reload() }}
        onDelete={detail.status === 'draft' ? () => confirm('Hesablama silinsin?', async () => { try { await api.del('/payroll/runs/' + detail.id); setDetail(null); reload() } catch (e) { toast.err(e.message) } }) : null} />}
      {confirmNode}
    </>
  )
}

function CreateRun({ onClose, onCreated }) {
  const toast = useToast()
  const [period, setPeriod] = useState(today().slice(0, 7))
  const [busy, setBusy] = useState(false)
  async function run() {
    setBusy(true)
    try { onCreated(await api.post('/payroll/runs?period=' + period + '-01')) } catch (e) { toast.err(e.message) } finally { setBusy(false) }
  }
  return (
    <Modal title="Yeni əmək haqqı hesablaması" onClose={onClose}
      footer={<Btn variant="primary" disabled={busy} onClick={run}>{busy ? 'Hesablanır...' : 'Hesabla'}</Btn>}>
      <p className="mb-3 text-sm text-muted">Seçilmiş ay üçün bütün aktiv işçilərin əmək haqqı, vergi və sosial tutulmaları avtomatik hesablanacaq (layihə kimi). Sonra təsdiqləyib jurnala yaza bilərsiniz.</p>
      <Field label="Ay"><Input type="month" value={period} onChange={(e) => setPeriod(e.target.value)} /></Field>
    </Modal>
  )
}

function RunDetail({ run, canWrite, onClose, onChanged, onDelete }) {
  const toast = useToast()
  const posted = run.status === 'posted'
  async function post() {
    try { await api.post('/payroll/runs/' + run.id + '/post'); toast.ok('Kitablaşdırıldı'); onChanged() } catch (e) { toast.err(e.message) }
  }
  const cols = [
    { h: 'İşçi', k: 'employee_name' },
    { h: 'Gross', right: true, render: (r) => money(r.gross) },
    { h: 'Gəlir vergisi', right: true, render: (r) => money(r.income_tax) },
    { h: 'DSMF (3%)', right: true, render: (r) => money(r.dsmf_emp) },
    { h: 'İşsizlik', right: true, render: (r) => money(r.unemp_emp) },
    { h: 'Tibbi', right: true, render: (r) => money(r.medical_emp) },
    { h: 'Net', right: true, render: (r) => <b>{money(r.net)}</b> },
  ]
  return (
    <Modal wide title={'Hesablama — ' + fmtDate(run.period).slice(0, 7)} onClose={onClose}
      footer={<>
        {onDelete && <Btn variant="danger" onClick={onDelete}>Sil</Btn>}
        {!posted && canWrite && <Btn variant="primary" onClick={post}>Təsdiqlə və jurnala yaz</Btn>}
        {posted && <span className="chip"><Badge status="posted" /></span>}
      </>}>
      <div className="mb-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Stat label="Gross cəmi" v={run.gross_total} />
        <Stat label="Tutulmalar" v={run.deductions_total} />
        <Stat label="İşəgötürən payı" v={run.employer_total} />
        <Stat label="Net ödəniş" v={run.net_total} tone="ok" />
      </div>
      <Table columns={cols} rows={run.lines || []} />
    </Modal>
  )
}
function Stat({ label, v, tone }) {
  return <div className="card"><div className="text-xs text-muted">{label}</div><div className={`mono text-lg font-bold ${tone === 'ok' ? 'text-ok' : ''}`}>{money(v)} ₼</div></div>
}

function Config() {
  const auth = useAuth()
  const toast = useToast()
  const cfg = useList('/payroll/config', [auth.company?.id])
  const accts = useList('/accounts', [auth.company?.id])
  const [f, setF] = useState(null)
  if (!cfg.data || !accts.data) return <Spinner />
  const state = f || cfg.data
  const set = (k) => (e) => setF({ ...state, [k]: e.target.value })
  const opts = [{ value: 0, label: '— seçin —' }, ...accts.data.filter((a) => !a.is_group).map((a) => ({ value: a.id, label: a.code + ' — ' + a.name }))]
  const num = (k, label) => <Field label={label}><Input type="number" step="0.01" value={state[k] ?? 0} onChange={set(k)} /></Field>

  async function save() {
    const body = {}
    Object.keys(state).forEach((k) => { body[k] = Number(state[k]) || 0 })
    try { await api.put('/payroll/config', body); toast.ok('Parametrlər yadda saxlanıldı'); cfg.reload(); setF(null) } catch (e) { toast.err(e.message) }
  }
  return (
    <div className="max-w-3xl">
      <div className="panel mb-4 p-5">
        <h3 className="mb-3 text-[15px] font-semibold">Dərəcələr (%)</h3>
        <div className="grid grid-cols-2 gap-3.5 sm:grid-cols-3">
          <Field label="Gəlir vergisi güzəşti (₼)"><Input type="number" step="0.01" value={state.income_tax_exempt ?? 0} onChange={set('income_tax_exempt')} /></Field>
          {num('income_tax_rate', 'Gəlir vergisi %')}
          <div />
          {num('dsmf_emp', 'DSMF işçi %')}
          {num('dsmf_empr', 'DSMF işəgötürən %')}
          <div />
          {num('unemp_emp', 'İşsizlik işçi %')}
          {num('unemp_empr', 'İşsizlik işəgötürən %')}
          <div />
          {num('medical_emp', 'Tibbi sığorta işçi %')}
          {num('medical_empr', 'Tibbi sığorta işəgötürən %')}
        </div>
        <p className="mt-2 text-xs text-muted">Standart AZ dərəcələri göstərilib — qanun dəyişəndə buradan yeniləyin.</p>
      </div>
      <div className="panel mb-4 p-5">
        <h3 className="mb-3 text-[15px] font-semibold">Mühasibat hesabları</h3>
        <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-3">
          <Field label="Əmək haqqı xərci"><Select value={state.salary_expense_account ?? 0} onChange={set('salary_expense_account')} options={opts} /></Field>
          <Field label="Ödəniləcək əmək haqqı (net)"><Select value={state.wages_payable_account ?? 0} onChange={set('wages_payable_account')} options={opts} /></Field>
          <Field label="Vergi/sosial öhdəliklər"><Select value={state.statutory_payable_account ?? 0} onChange={set('statutory_payable_account')} options={opts} /></Field>
        </div>
      </div>
      <Btn variant="primary" onClick={save}>Yadda saxla</Btn>
    </div>
  )
}
