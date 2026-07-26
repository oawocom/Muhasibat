import { useState } from 'react'
import { api } from '../api.js'
import { Modal, Field, Input, Select, Textarea, Btn, useToast } from '../ui.jsx'

// Generic create/update modal driven by a fields config.
// field: { k, label, type: 'text'|'number'|'select'|'textarea'|'password', options, step, def }
export function CrudModal({ title, fields, item, path, onClose, onSaved }) {
  const toast = useToast()
  const isNew = !item?.id
  const [f, setF] = useState(() => {
    const init = {}
    fields.forEach((fl) => { init[fl.k] = item?.[fl.k] ?? fl.def ?? (fl.type === 'checkbox' ? false : '') })
    return init
  })
  const up = (fl) => (e) => {
    const v = fl.type === 'checkbox' ? e.target.checked : e.target.value
    setF((s) => ({ ...s, [fl.k]: v }))
  }
  async function save() {
    const body = {}
    fields.forEach((fl) => {
      let v = f[fl.k]
      if (fl.type === 'number') v = v === '' || v == null ? null : Number(v)
      if (fl.type === 'select' && fl.numeric) v = v === '' ? null : Number(v)
      body[fl.k] = v
    })
    try {
      if (isNew) await api.post(path, body)
      else await api.put(path + '/' + item.id, body)
      onSaved()
    } catch (e) { toast.err(e.message) }
  }
  return (
    <Modal title={title} onClose={onClose} footer={<Btn variant="primary" onClick={save}>Yadda saxla</Btn>}>
      {fields.map((fl) => (
        <Field key={fl.k} label={fl.label}>
          {fl.type === 'select' ? (
            <Select value={f[fl.k] ?? ''} onChange={up(fl)} options={fl.options || []} />
          ) : fl.type === 'textarea' ? (
            <Textarea value={f[fl.k] ?? ''} onChange={up(fl)} />
          ) : fl.type === 'checkbox' ? (
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={!!f[fl.k]} onChange={up(fl)} /> {fl.hint || ''}</label>
          ) : (
            <Input type={fl.type || 'text'} step={fl.step} value={f[fl.k] ?? ''} onChange={up(fl)} />
          )}
        </Field>
      ))}
    </Modal>
  )
}
