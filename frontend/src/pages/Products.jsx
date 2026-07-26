import { useState } from 'react'
import { useList } from '../hooks.js'
import { Spinner, PageHeader, Table, Btn, useToast } from '../ui.jsx'
import { CrudModal } from './crud.jsx'
import { api, money } from '../api.js'
import { useAuth } from '../store.jsx'

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
  ]
  return (
    <>
      <PageHeader title="Məhsul / Xidmət">
        {auth.canWrite() && <Btn variant="primary" onClick={() => setEdit({})}>+ Yeni məhsul</Btn>}
      </PageHeader>
      <Table title={`Məhsullar (${data.length})`} columns={columns} rows={data} onRow={auth.canWrite() ? setEdit : undefined} />
      {edit && <CrudModal title={edit.id ? 'Məhsul' : 'Yeni məhsul'} fields={fields} item={edit} path="/products"
        onClose={() => setEdit(null)} onSaved={() => { setEdit(null); reload(); toast.ok('Yadda saxlanıldı') }} />}
    </>
  )
}
