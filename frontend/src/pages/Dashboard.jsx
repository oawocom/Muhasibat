import { useList } from '../hooks.js'
import { Spinner, PageHeader } from '../ui.jsx'
import { money } from '../api.js'
import { useAuth } from '../store.jsx'

function Kpi({ icon, label, value, sub, tone }) {
  const toneCls = tone === 'pos' ? 'text-ok' : tone === 'neg' ? 'text-danger' : 'text-slate-400'
  return (
    <div className="card">
      <div className="flex items-center gap-2 text-[13px] text-slate-400"><span>{icon}</span>{label}</div>
      <div className="mono mt-2 text-2xl font-extrabold tracking-tight">{value}</div>
      {sub && <div className={`mt-1 text-xs ${toneCls}`}>{sub}</div>}
    </div>
  )
}

export default function Dashboard() {
  const auth = useAuth()
  const { data, error } = useList('/dashboard', [auth.company?.id])
  if (error) return <div className="py-12 text-center text-danger">{error}</div>
  if (!data) return <Spinner />
  const d = data
  return (
    <>
      <PageHeader title="İdarə paneli" />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        <Kpi icon="💵" label="Kassa" value={money(d.cash) + ' ₼'} sub="Nağd vəsait" />
        <Kpi icon="🏦" label="Bank" value={money(d.bank) + ' ₼'} sub="Hesablaşma hesabı" />
        <Kpi icon="↗" label="Debitor borcu" value={money(d.receivable) + ' ₼'} sub="Müştərilərdən alacaq" tone="pos" />
        <Kpi icon="↙" label="Kreditor borcu" value={money(d.payable) + ' ₼'} sub="Təchizatçılara borc" tone="neg" />
        <Kpi icon="📈" label="Bu ay gəlir" value={money(d.income_this_month) + ' ₼'} tone="pos" />
        <Kpi icon="📉" label="Bu ay xərc" value={money(d.expense_this_month) + ' ₼'} tone="neg" />
        <Kpi icon="💰" label="Bu ay xalis" value={money(d.net_this_month) + ' ₼'} sub={d.net_this_month >= 0 ? 'Mənfəət' : 'Zərər'} tone={d.net_this_month >= 0 ? 'pos' : 'neg'} />
        <Kpi icon="🧾" label="ƏDV (ödəniləcək)" value={money(d.vat_payable) + ' ₼'} sub="Çıxış − Giriş" />
        <Kpi icon="📦" label="Anbar dəyəri" value={money(d.stock_value) + ' ₼'} sub={d.products + ' məhsul'} />
        <Kpi icon="⏳" label="Açıq fakturalar" value={d.open_invoices} sub="Ödənilməmiş" tone="warn" />
        <Kpi icon="👥" label="Tərəfdaşlar" value={d.partners} sub="Müştəri / təchizatçı" />
      </div>
    </>
  )
}
