import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { api, money, today, monthStart } from '../api.js'
import { Spinner, PageHeader, Table, Btn, Input } from '../ui.jsx'
import { useAuth } from '../store.jsx'

const TITLES = {
  trial: 'Dövriyyə balansı', balance: 'Balans hesabatı', pl: 'Mənfəət və zərər',
  partners: 'Debitor / Kreditor', stock: 'Anbar qalıqları',
}

export default function Reports() {
  const { kind } = useParams()
  const auth = useAuth()
  const [from, setFrom] = useState(monthStart())
  const [to, setTo] = useState(today())
  const [data, setData] = useState(null)
  const [err, setErr] = useState(null)

  // The route /reports/:kind reuses this one component for every report.
  // When kind changes, `data` still holds the previous report (a different
  // shape); reset it during render so a renderer never receives stale,
  // wrong-shaped data before the effect refetches.
  const [shownKind, setShownKind] = useState(kind)
  if (shownKind !== kind) { setShownKind(kind); setData(null); setErr(null) }

  function load() {
    setData(null); setErr(null)
    let path
    if (kind === 'trial') path = '/reports/trial-balance?to=' + to
    else if (kind === 'balance') path = '/reports/balance-sheet?to=' + to
    else if (kind === 'pl') path = '/reports/profit-loss?from=' + from + '&to=' + to
    else if (kind === 'partners') path = '/reports/partner-balances'
    else if (kind === 'stock') path = '/reports/stock'
    else path = '/reports/trial-balance'
    api.get(path).then(setData).catch((e) => setErr(e.message))
  }
  useEffect(() => { load() }, [kind, auth.company?.id]) // eslint-disable-line

  const showFrom = kind === 'pl'
  const showTo = kind === 'trial' || kind === 'balance' || kind === 'pl'
  return (
    <>
      <PageHeader title={TITLES[kind] || 'Hesabat'}>
        {showFrom && <Input type="date" className="w-40" value={from} onChange={(e) => setFrom(e.target.value)} />}
        {showTo && <Input type="date" className="w-40" value={to} onChange={(e) => setTo(e.target.value)} />}
        {(showFrom || showTo) && <Btn variant="primary" sm onClick={load}>Yenilə</Btn>}
      </PageHeader>
      {err && <div className="py-12 text-center text-danger">{err}</div>}
      {!err && !data && <Spinner />}
      {data && kind === 'trial' && <Trial d={data} />}
      {data && kind === 'balance' && <BalanceSheet d={data} />}
      {data && kind === 'pl' && <ProfitLoss d={data} />}
      {data && kind === 'partners' && <PartnerBalances d={data} />}
      {data && kind === 'stock' && <Stock d={data} />}
    </>
  )
}

function Trial({ d }) {
  const cols = [
    { h: 'Kod', mono: true, render: (r) => r.code },
    { h: 'Hesab', k: 'name' },
    { h: 'Debet', right: true, render: (r) => money(r.debit) },
    { h: 'Kredit', right: true, render: (r) => money(r.credit) },
  ]
  return (
    <>
      <div className="card mb-4">
        {d.balanced ? <span className="text-ok">✓ Balans bərabərdir</span> : <span className="text-danger">⚠ Balanssızlıq</span>}
        <span className="ml-2 text-muted">Debet {money(d.total_debit)} = Kredit {money(d.total_credit)}</span>
      </div>
      <Table title="Dövriyyə balansı" columns={cols} rows={[...d.rows, { id: 'tot', code: '', name: 'CƏMİ', debit: d.total_debit, credit: d.total_credit }]} />
    </>
  )
}

function Section({ title, sec, totalLabel }) {
  const cols = [{ h: 'Kod', mono: true, render: (r) => r.code }, { h: 'Hesab', k: 'name' }, { h: 'Məbləğ', right: true, render: (r) => money(r.balance) }]
  const rows = [...(sec.rows || []), { id: 't', code: '', name: totalLabel, balance: sec.total }]
  return <Table title={title} columns={cols} rows={rows} />
}

function BalanceSheet({ d }) {
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <div><Section title="AKTİVLƏR" sec={d.assets} totalLabel="Cəmi aktivlər" /></div>
      <div>
        <Section title="ÖHDƏLİKLƏR" sec={d.liabilities} totalLabel="Cəmi öhdəliklər" />
        <Section title="KAPİTAL" sec={d.equity} totalLabel="Cəmi kapital" />
        <div className="card mb-4">
          <div className="text-muted">Dövrün mənfəəti (zərər)</div>
          <div className={`mono text-2xl font-extrabold ${d.net_income >= 0 ? 'text-ok' : 'text-danger'}`}>{money(d.net_income)} ₼</div>
        </div>
        <div className="card">{d.balanced ? <span className="text-ok">✓ Aktiv = Passiv ({money(d.assets.total)})</span> : <span className="text-danger">⚠ Aktiv ≠ Passiv</span>}</div>
      </div>
    </div>
  )
}

function ProfitLoss({ d }) {
  return (
    <>
      <div className="mb-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div className="card"><div className="text-muted">Gəlirlər</div><div className="mono text-2xl font-extrabold text-ok">{money(d.income.total)} ₼</div></div>
        <div className="card"><div className="text-muted">Xərclər</div><div className="mono text-2xl font-extrabold text-danger">{money(d.expense.total)} ₼</div></div>
        <div className="card"><div className="text-muted">Xalis mənfəət</div><div className={`mono text-2xl font-extrabold ${d.net_profit >= 0 ? 'text-ok' : 'text-danger'}`}>{money(d.net_profit)} ₼</div></div>
      </div>
      <Section title="GƏLİRLƏR" sec={d.income} totalLabel="Cəmi gəlir" />
      <Section title="XƏRCLƏR" sec={d.expense} totalLabel="Cəmi xərc" />
    </>
  )
}

function PartnerBalances({ d }) {
  const rows = [...d].sort((a, b) => Math.abs(b.net) - Math.abs(a.net))
  const cols = [
    { h: 'Tərəfdaş', k: 'name' },
    { h: 'Debitor', right: true, render: (r) => money(r.receivable) },
    { h: 'Kreditor', right: true, render: (r) => money(r.payable) },
    { h: 'Xalis', right: true, render: (r) => <span className={r.net >= 0 ? 'text-ok' : 'text-danger'}>{money(r.net)}</span> },
  ]
  return <Table title="Debitor / Kreditor borcları" columns={cols} rows={rows} />
}

function Stock({ d }) {
  const cols = [
    { h: 'Kod', mono: true, k: 'code' }, { h: 'Məhsul', k: 'name' }, { h: 'Vahid', k: 'unit' },
    { h: 'Qalıq', right: true, render: (r) => money(r.quantity) },
    { h: 'Dəyər', right: true, render: (r) => money(r.value) + ' ₼' },
  ]
  return <Table title="Anbar qalıqları" columns={cols} rows={d} />
}
