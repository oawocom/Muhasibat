import { createContext, useContext, useState, useCallback } from 'react'

// ---------------- Toast ----------------
const ToastCtx = createContext(null)
export const useToast = () => useContext(ToastCtx)

export function ToastProvider({ children }) {
  const [toasts, setToasts] = useState([])
  const push = useCallback((msg, kind) => {
    const id = Math.random().toString(36).slice(2)
    setToasts((t) => [...t, { id, msg, kind }])
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 3400)
  }, [])
  const api = {
    ok: (m) => push(m, 'ok'),
    err: (m) => push(m, 'err'),
    info: (m) => push(m, ''),
  }
  return (
    <ToastCtx.Provider value={api}>
      {children}
      <div className="fixed bottom-5 right-5 z-[100] flex flex-col gap-2.5">
        {toasts.map((t) => (
          <div key={t.id}
            className={`min-w-[240px] rounded-xl border border-line bg-surface2 px-4 py-3 shadow-2xl border-l-4 ${
              t.kind === 'err' ? 'border-l-danger' : t.kind === 'ok' ? 'border-l-ok' : 'border-l-brand'}`}>
            {t.msg}
          </div>
        ))}
      </div>
    </ToastCtx.Provider>
  )
}

// ---------------- Spinner ----------------
export function Spinner() {
  return <div className="mx-auto my-10 h-7 w-7 animate-spin rounded-full border-[3px] border-line border-t-brand" />
}

// ---------------- Page header ----------------
export function PageHeader({ title, children }) {
  return (
    <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
      <h2 className="text-lg font-semibold">{title}</h2>
      <div className="flex flex-wrap items-center gap-2.5">{children}</div>
    </div>
  )
}

// ---------------- Buttons ----------------
export function Btn({ variant = '', sm, className = '', ...p }) {
  const v = { primary: 'btn-primary', danger: 'btn-danger', ghost: 'btn-ghost' }[variant] || ''
  return <button className={`btn ${v} ${sm ? 'btn-sm' : ''} ${className}`} {...p} />
}

// ---------------- Badges ----------------
export function Badge({ status }) {
  const map = {
    draft: ['Layihə', 'bg-warn/15 text-warn'],
    posted: ['Təsdiqli', 'bg-ok/15 text-ok'],
    paid: ['Ödənilib', 'bg-brand/20 text-brand'],
    void: ['Ləğv', 'bg-danger/15 text-danger'],
  }
  const [label, cls] = map[status] || [status, 'bg-surface2 text-slate-400']
  return <span className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${cls}`}>{label}</span>
}

// ---------------- Modal ----------------
export function Modal({ title, onClose, children, footer, wide }) {
  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/60 p-4 backdrop-blur-sm"
      onMouseDown={(e) => { if (e.target === e.currentTarget) onClose() }}>
      <div className={`mt-10 w-full ${wide ? 'max-w-4xl' : 'max-w-2xl'} rounded-2xl border border-line bg-surface shadow-2xl`}>
        <div className="flex items-center justify-between border-b border-line px-5 py-4">
          <h3 className="text-base font-semibold">{title}</h3>
          <button className="text-2xl leading-none text-slate-400 hover:text-slate-100" onClick={onClose}>&times;</button>
        </div>
        <div className="max-h-[70vh] overflow-y-auto p-5">{children}</div>
        {footer && <div className="flex justify-end gap-2.5 border-t border-line px-5 py-3.5">{footer}</div>}
      </div>
    </div>
  )
}

// ---------------- Confirm ----------------
export function useConfirm() {
  const [state, setState] = useState(null)
  const confirm = (msg, onYes) => setState({ msg, onYes })
  const node = state ? (
    <Modal title="Təsdiq" onClose={() => setState(null)}
      footer={<>
        <Btn variant="ghost" onClick={() => setState(null)}>İmtina</Btn>
        <Btn variant="danger" onClick={() => { state.onYes(); setState(null) }}>Təsdiq</Btn>
      </>}>
      <p>{state.msg}</p>
    </Modal>
  ) : null
  return [confirm, node]
}

// ---------------- Field ----------------
export function Field({ label, children }) {
  return (
    <div className="mb-3.5">
      {label && <label className="label">{label}</label>}
      {children}
    </div>
  )
}
export function Input({ className = '', ...props }) { return <input className={`input ${className}`} {...props} /> }
export function Select({ options = [], className = '', ...p }) {
  return (
    <select className={`input ${className}`} {...p}>
      {options.map((o) => <option key={String(o.value)} value={o.value}>{o.label}</option>)}
    </select>
  )
}
export function Textarea({ className = '', ...props }) { return <textarea className={`input ${className}`} rows={3} {...props} /> }

// ---------------- Table ----------------
export function Table({ title, actions, columns, rows, onRow, empty = 'Məlumat yoxdur' }) {
  return (
    <div className="panel mb-5">
      {(title || actions) && (
        <div className="flex items-center justify-between border-b border-line px-4.5 py-3.5 px-4">
          <h3 className="text-[15px] font-semibold">{title}</h3>
          <div className="flex flex-wrap items-center gap-2.5">{actions}</div>
        </div>
      )}
      {rows.length === 0 ? (
        <div className="py-12 text-center text-slate-400">{empty}</div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full border-collapse">
            <thead>
              <tr>{columns.map((c, i) => <th key={i} className={`th ${c.right ? 'text-right' : ''}`}>{c.h}</th>)}</tr>
            </thead>
            <tbody>
              {rows.map((row, ri) => (
                <tr key={row.id ?? ri}
                  className={`hover:bg-surface2 ${onRow ? 'cursor-pointer' : ''}`}
                  onClick={onRow ? (e) => { if (!e.target.closest('button,a')) onRow(row) } : undefined}>
                  {columns.map((c, ci) => (
                    <td key={ci} className={`td ${c.right ? 'text-right mono' : ''} ${c.mono ? 'mono' : ''}`}>
                      {c.render ? c.render(row) : row[c.k]}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
