import { useParams } from 'react-router-dom'
import { useList } from '../hooks.js'
import { Spinner, PageHeader, Table } from '../ui.jsx'
import { money, fmtDate } from '../api.js'

export default function Ledger() {
  const { id } = useParams()
  const { data } = useList('/reports/ledger/' + id, [id])
  if (!data) return <Spinner />
  const cols = [
    { h: 'Tarix', render: (r) => fmtDate(r.date) },
    { h: 'Sənəd', mono: true, k: 'number' },
    { h: 'Təsvir', k: 'description' },
    { h: 'Debet', right: true, render: (r) => money(r.debit) },
    { h: 'Kredit', right: true, render: (r) => money(r.credit) },
    { h: 'Qalıq', right: true, render: (r) => money(r.balance) },
  ]
  return (
    <>
      <PageHeader title="Baş kitab" />
      <div className="card mb-4">
        <b>{data.account.code} — {data.account.name}</b>
        <span className="ml-2 text-muted">Açılış: {money(data.opening)} / Bağlanış: <b>{money(data.closing)}</b></span>
      </div>
      <Table title="Hesab üzrə hərəkət" columns={cols} rows={data.lines || []} />
    </>
  )
}
