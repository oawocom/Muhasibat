import { useState } from 'react'
import { useList } from '../hooks.js'
import { api, money, fmtDate, today } from '../api.js'
import { Spinner, PageHeader, Table, Btn, Modal, Field, Input, Select, Badge, useToast, useConfirm } from '../ui.jsx'
import { useAuth } from '../store.jsx'

const statusMap = { active: 'Aktiv', fully_depreciated: 'Tam amortizasiya', disposed: 'Silinmiş' }

export default function FixedAssets() {
  const auth = useAuth()
  const toast = useToast()
  const { data, reload } = useList('/fixed-assets', [auth.company?.id])
  const { data: accounts } = useList('/accounts', [auth.company?.id])
  const [edit, setEdit] = useState(null)
  const [history, setHistory] = useState(null)
  const [runOpen, setRunOpen] = useState(false)
  const [confirm, confirmNode] = useConfirm()
  if (!data || !accounts) return <Spinner />

  const columns = [
    { h: 'Kod', mono: true, k: 'code' },
    { h: 'Ad', render: (r) => <b>{r.name}</b> },
    { h: 'İlkin dəyər', right: true, render: (r) => money(r.cost) },
    { h: 'Amortizasiya', right: true, render: (r) => money(r.accumulated_depreciation) },
    { h: 'Qalıq dəyər', right: true, render: (r) => money(r.book_value) },
    { h: 'Müddət', right: true, render: (r) => `${r.months_depreciated}/${r.useful_life_months} ay` },
    { h: 'Status', render: (r) => <span className={`chip ${r.status === 'active' ? '' : 'text-ok'}`}>{statusMap[r.status] || r.status}</span> },
    { h: '', right: true, render: (r) => (
      <div className="flex justify-end gap-2">
        <Btn sm variant="ghost" onClick={() => setHistory(r)}>Cədvəl</Btn>
        {auth.canWrite() && <Btn sm variant="ghost" onClick={() => setEdit(r)}>Düzəliş</Btn>}
      </div>
    ) },
  ]
  return (
    <>
      <PageHeader title="Əsas vəsaitlər">
        {auth.canWrite() && <>
          <Btn onClick={() => setRunOpen(true)}>⚙ Amortizasiya hesabla</Btn>
          <Btn variant="primary" onClick={() => setEdit({})}>+ Yeni vəsait</Btn>
        </>}
      </PageHeader>
      <Table title={`Əsas vəsaitlər (${data.length})`} columns={columns} rows={data} />
      {edit && <AssetModal item={edit} accounts={accounts} onClose={() => setEdit(null)}
        onSaved={() => { setEdit(null); reload(); toast.ok('Yadda saxlanıldı') }}
        onDelete={edit.id ? () => confirm('Vəsait silinsin?', async () => { try { await api.del('/fixed-assets/' + edit.id); setEdit(null); reload(); toast.ok('Silindi') } catch (e) { toast.err(e.message) } }) : null} />}
      {history && <HistoryModal asset={history} onClose={() => setHistory(null)} />}
      {runOpen && <RunModal onClose={() => setRunOpen(false)} onDone={() => { setRunOpen(false); reload() }} />}
      {confirmNode}
    </>
  )
}

function bySys(accounts, key) { return accounts.find((a) => a.system_key === key) }

function AssetModal({ item, accounts, onClose, onSaved, onDelete }) {
  const toast = useToast()
  const isNew = !item.id
  const postable = accounts.filter((a) => !a.is_group)
  const opts = [{ value: '', label: '— seçin —' }, ...postable.map((a) => ({ value: a.id, label: a.code + ' — ' + a.name }))]
  const [f, setF] = useState({
    code: '', name: '', category: '', acquisition_date: today(), cost: 0, salvage_value: 0, useful_life_months: 60,
    asset_account_id: bySys(accounts, 'ppe')?.id || '',
    accum_account_id: bySys(accounts, 'accum_dep')?.id || '',
    expense_account_id: bySys(accounts, 'dep_expense')?.id || '',
    ...item,
    acquisition_date: item.acquisition_date ? fmtDate(item.acquisition_date) : today(),
  })
  const up = (k) => (e) => setF({ ...f, [k]: e.target.value })
  const locked = !isNew && item.months_depreciated > 0

  async function save() {
    const body = {
      code: f.code, name: f.name, category: f.category, acquisition_date: f.acquisition_date,
      cost: Number(f.cost) || 0, salvage_value: Number(f.salvage_value) || 0, useful_life_months: Number(f.useful_life_months) || 0,
      asset_account_id: Number(f.asset_account_id) || 0, accum_account_id: Number(f.accum_account_id) || 0, expense_account_id: Number(f.expense_account_id) || 0,
      notes: f.notes || '',
    }
    if (!body.name || !body.accum_account_id || !body.expense_account_id) { toast.err('Ad, yığılmış amortizasiya və xərc hesabı tələb olunur'); return }
    try { if (isNew) await api.post('/fixed-assets', body); else await api.put('/fixed-assets/' + item.id, body); onSaved() } catch (e) { toast.err(e.message) }
  }
  return (
    <Modal wide title={isNew ? 'Yeni əsas vəsait' : 'Əsas vəsait'} onClose={onClose}
      footer={<>{onDelete && <Btn variant="danger" onClick={onDelete}>Sil</Btn>}<Btn variant="primary" onClick={save}>Yadda saxla</Btn></>}>
      <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2">
        <Field label="Kod"><Input value={f.code} onChange={up('code')} /></Field>
        <Field label="Ad"><Input value={f.name} onChange={up('name')} /></Field>
        <Field label="Kateqoriya"><Input value={f.category} onChange={up('category')} /></Field>
        <Field label="Alınma tarixi"><Input type="date" disabled={locked} value={f.acquisition_date} onChange={up('acquisition_date')} /></Field>
        <Field label="İlkin dəyər (₼)"><Input type="number" step="0.01" disabled={locked} value={f.cost} onChange={up('cost')} /></Field>
        <Field label="Qalıq (ləğv) dəyər"><Input type="number" step="0.01" disabled={locked} value={f.salvage_value} onChange={up('salvage_value')} /></Field>
        <Field label="İstifadə müddəti (ay)"><Input type="number" disabled={locked} value={f.useful_life_months} onChange={up('useful_life_months')} /></Field>
      </div>
      {locked && <div className="mb-3 text-xs text-warn">Amortizasiya başlayıb — dəyər/müddət dəyişdirilə bilməz.</div>}
      <div className="panel my-2">
        <div className="border-b border-line px-4 py-2.5 text-sm font-semibold">Amortizasiya hesabları</div>
        <div className="grid grid-cols-1 gap-3.5 p-4 sm:grid-cols-3">
          <Field label="Aktiv hesabı"><Select value={f.asset_account_id} onChange={up('asset_account_id')} options={opts} /></Field>
          <Field label="Yığılmış amortizasiya"><Select value={f.accum_account_id} onChange={up('accum_account_id')} options={opts} /></Field>
          <Field label="Amortizasiya xərci"><Select value={f.expense_account_id} onChange={up('expense_account_id')} options={opts} /></Field>
        </div>
      </div>
    </Modal>
  )
}

function HistoryModal({ asset, onClose }) {
  const { data } = useList('/fixed-assets/' + asset.id + '/depreciation', [asset.id])
  const cols = [{ h: 'Period', render: (r) => fmtDate(r.period) }, { h: 'Məbləğ', right: true, render: (r) => money(r.amount) }]
  return (
    <Modal title={'Amortizasiya cədvəli — ' + asset.name} onClose={onClose}>
      {!data ? <Spinner /> : <Table columns={cols} rows={data} empty="Hələ amortizasiya hesablanmayıb" />}
    </Modal>
  )
}

function RunModal({ onClose, onDone }) {
  const toast = useToast()
  const [asOf, setAsOf] = useState(today())
  const [busy, setBusy] = useState(false)
  async function run() {
    setBusy(true)
    try {
      const r = await api.post('/depreciation/run?as_of=' + asOf)
      toast.ok(`${r.assets_processed} vəsait · ${r.months_posted} ay · ${money(r.total_amount)} ₼ kitablaşdırıldı`)
      onDone()
    } catch (e) { toast.err(e.message) } finally { setBusy(false) }
  }
  return (
    <Modal title="Amortizasiya hesabla" onClose={onClose}
      footer={<Btn variant="primary" disabled={busy} onClick={run}>{busy ? 'Hesablanır...' : 'Hesabla və kitablaşdır'}</Btn>}>
      <p className="mb-3 text-sm text-slate-400">Seçilmiş tarixə qədər bütün aktiv əsas vəsaitlər üzrə çatışmayan aylar üçün amortizasiya avtomatik hesablanıb mühasibat jurnalına yazılacaq (Dr amortizasiya xərci / Cr yığılmış amortizasiya).</p>
      <Field label="Tarixə qədər"><Input type="date" value={asOf} onChange={(e) => setAsOf(e.target.value)} /></Field>
    </Modal>
  )
}
