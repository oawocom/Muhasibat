import { useState } from 'react'
import { useList } from '../hooks.js'
import { Spinner, PageHeader, Table, Btn, useToast } from '../ui.jsx'
import { CrudModal } from './crud.jsx'
import { api, money } from '../api.js'
import { useAuth } from '../store.jsx'
import { code128SVG } from '../barcode.js'

function escHtml(s) { return String(s == null ? '' : s).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c])) }

// Open a print window with barcode labels for the given products.
function printLabels(list, qty = 1) {
  const items = []
  for (const p of list) {
    const code = p.barcode || p.code
    if (!code) continue
    for (let i = 0; i < qty; i++) items.push({ ...p, _code: code })
  }
  if (!items.length) return false
  const label = (p) => `<div class="label">
      <div class="name">${escHtml(p.name)}</div>
      <div class="price">${money(p.sale_price)} ₼</div>
      <div class="bc">${code128SVG(p._code, { barWidth: 2, height: 46 })}</div>
      <div class="num">${escHtml(p._code)}</div>
    </div>`
  const w = window.open('', '_blank')
  if (!w) return true
  w.document.write(`<!doctype html><html><head><meta charset="utf-8"><title>Barkod etiketləri</title><style>
    body{font-family:system-ui,sans-serif;margin:0;padding:8px}
    .bar{margin-bottom:8px}
    .grid{display:flex;flex-wrap:wrap;gap:6px}
    .label{width:5cm;border:1px solid #e5e7eb;border-radius:6px;padding:8px 6px;text-align:center;page-break-inside:avoid}
    .name{font-size:12px;font-weight:600;line-height:1.15;height:30px;overflow:hidden}
    .price{font-size:17px;font-weight:800;margin:2px 0}
    .bc svg{max-width:100%;height:46px}
    .num{font-family:monospace;font-size:11px;letter-spacing:2px}
    @media print{.noprint{display:none}}
  </style></head><body>
    <div class="bar noprint"><button onclick="window.print()">🖨 Çap et</button></div>
    <div class="grid">${items.map(label).join('')}</div>
  </body></html>`)
  w.document.close()
  return true
}

export default function Products() {
  const auth = useAuth()
  const toast = useToast()
  const { data, reload } = useList('/products', [auth.company?.id])
  const { data: taxes } = useList('/tax-rates', [auth.company?.id])
  const [edit, setEdit] = useState(null)
  if (!data || !taxes) return <Spinner />

  const fields = [
    { k: 'name', label: 'Ad' }, { k: 'code', label: 'Kod / SKU' }, { k: 'barcode', label: 'Barkod' },
    { k: 'type', label: 'Növ', type: 'select', def: 'product', options: [{ value: 'product', label: 'Məhsul' }, { value: 'service', label: 'Xidmət' }] },
    { k: 'unit', label: 'Ölçü vahidi', def: 'ədəd' },
    { k: 'sale_price', label: 'Satış qiyməti', type: 'number', step: '0.01' },
    { k: 'cost_price', label: 'Maya dəyəri', type: 'number', step: '0.01' },
    { k: 'tax_rate_id', label: 'ƏDV dərəcəsi', type: 'select', numeric: true, options: [{ value: '', label: '—' }, ...taxes.map((t) => ({ value: t.id, label: t.name }))] },
    { k: 'track_stock', label: 'Anbar uçotu', type: 'checkbox', def: true, hint: 'Anbar uçotu aparılsın' },
    { k: 'description', label: 'Təsvir', type: 'textarea' },
  ]
  const columns = [
    { h: 'Kod', mono: true, k: 'code' },
    { h: 'Ad', render: (r) => <b>{r.name}</b> },
    { h: 'Növ', render: (r) => (r.type === 'service' ? 'Xidmət' : 'Məhsul') },
    { h: 'Vahid', k: 'unit' },
    { h: 'Satış', right: true, render: (r) => money(r.sale_price) },
    { h: 'Maya', right: true, render: (r) => money(r.cost_price) },
    { h: 'Barkod', right: true, render: (r) => (r.barcode || r.code)
      ? <button title="Etiket çap et" className="rounded-lg border border-line px-2 py-1 text-sm hover:border-brand"
          onClick={(e) => { e.stopPropagation(); if (!printLabels([r])) toast.err('Barkod yoxdur') }}>🏷</button>
      : <span className="text-muted">—</span> },
  ]
  const printable = data.filter((p) => p.barcode || p.code)
  return (
    <>
      <PageHeader title="Məhsul / Xidmət">
        {printable.length > 0 && (
          <Btn variant="ghost" onClick={() => { if (!printLabels(printable)) toast.err('Barkodu olan məhsul yoxdur') }}>🏷 Etiketləri çap et</Btn>
        )}
        {auth.canWrite() && <Btn variant="primary" onClick={() => setEdit({})}>+ Yeni məhsul</Btn>}
      </PageHeader>
      <Table title={`Məhsullar (${data.length})`} columns={columns} rows={data} onRow={auth.canWrite() ? setEdit : undefined} />
      {edit && <CrudModal title={edit.id ? 'Məhsul' : 'Yeni məhsul'} fields={fields} item={edit} path="/products"
        onClose={() => setEdit(null)} onSaved={() => { setEdit(null); reload(); toast.ok('Yadda saxlanıldı') }} />}
    </>
  )
}
