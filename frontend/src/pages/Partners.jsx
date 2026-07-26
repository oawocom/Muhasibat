import { useState } from 'react'
import { useList } from '../hooks.js'
import { Spinner, PageHeader, Table, Btn, useToast } from '../ui.jsx'
import { CrudModal } from './crud.jsx'
import { useAuth } from '../store.jsx'

const typeName = { customer: 'Müştəri', supplier: 'Təchizatçı', both: 'Hər ikisi' }
const fields = [
  { k: 'name', label: 'Ad / Şirkət' },
  { k: 'type', label: 'Növ', type: 'select', def: 'both', options: [{ value: 'both', label: 'Hər ikisi' }, { value: 'customer', label: 'Müştəri' }, { value: 'supplier', label: 'Təchizatçı' }] },
  { k: 'tax_id', label: 'VÖEN' }, { k: 'phone', label: 'Telefon' }, { k: 'email', label: 'Email' },
  { k: 'contact_name', label: 'Əlaqələndirici' }, { k: 'bank_name', label: 'Bank' }, { k: 'bank_account', label: 'IBAN' },
  { k: 'address', label: 'Ünvan', type: 'textarea' }, { k: 'notes', label: 'Qeyd', type: 'textarea' },
]

export default function Partners() {
  const auth = useAuth()
  const toast = useToast()
  const { data, reload } = useList('/partners', [auth.company?.id])
  const [edit, setEdit] = useState(null)
  if (!data) return <Spinner />
  const columns = [
    { h: 'Ad', render: (r) => <span><b>{r.name}</b>{r.code && <span className="ml-1 text-muted">{r.code}</span>}</span> },
    { h: 'Növ', render: (r) => typeName[r.type] || r.type },
    { h: 'VÖEN', mono: true, k: 'tax_id' },
    { h: 'Telefon', k: 'phone' },
  ]
  return (
    <>
      <PageHeader title="Tərəfdaşlar">
        {auth.canWrite() && <Btn variant="primary" onClick={() => setEdit({})}>+ Yeni tərəfdaş</Btn>}
      </PageHeader>
      <Table title={`Tərəfdaşlar (${data.length})`} columns={columns} rows={data} onRow={auth.canWrite() ? setEdit : undefined} />
      {edit && <CrudModal title={edit.id ? 'Tərəfdaş' : 'Yeni tərəfdaş'} fields={fields} item={edit} path="/partners"
        onClose={() => setEdit(null)} onSaved={() => { setEdit(null); reload(); toast.ok('Yadda saxlanıldı') }} />}
    </>
  )
}
