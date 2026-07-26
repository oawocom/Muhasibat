import { useState, useEffect, useMemo } from 'react'
import { useList } from '../hooks.js'
import { api, money } from '../api.js'
import { Spinner, PageHeader, Btn, Modal, Field, Input, Select, useToast } from '../ui.jsx'
import { useAuth } from '../store.jsx'

export default function Pos() {
  const auth = useAuth()
  const toast = useToast()
  const registers = useList('/pos/registers', [auth.company?.id])
  const [regId, setRegId] = useState(null)
  const [session, setSession] = useState(null)
  const [loadingSess, setLoadingSess] = useState(false)
  const [manage, setManage] = useState(false)

  // Pick the first register by default.
  useEffect(() => {
    if (registers.data && registers.data.length && !regId) setRegId(registers.data[0].id)
  }, [registers.data]) // eslint-disable-line

  // Load the open session for the chosen register.
  useEffect(() => {
    if (!regId) return
    setLoadingSess(true)
    api.get(`/pos/registers/${regId}/open-session`).then((s) => setSession(s)).catch(() => setSession(null)).finally(() => setLoadingSess(false))
  }, [regId])

  if (registers.error) return (
    <>
      <PageHeader title="Kassa (POS)" />
      <div className="card text-danger">Kassa məlumatı yüklənmədi: {registers.error}</div>
    </>
  )
  if (!registers.data) return <Spinner />

  const register = registers.data.find((r) => r.id === regId)

  if (registers.data.length === 0) {
    return (
      <>
        <PageHeader title="Kassa (POS)" />
        <div className="card">
          <p className="mb-3 text-muted">Hələ kassa yoxdur. Satışa başlamaq üçün bir kassa yaradın.</p>
          {auth.canWrite() && <Btn variant="primary" onClick={() => setManage(true)}>+ Kassa yarat</Btn>}
        </div>
        {manage && <Registers onClose={() => { setManage(false); registers.reload() }} />}
      </>
    )
  }

  // regId is set by an effect after the first render with data; guard the
  // brief window where it is still null so OpenShift never gets an undefined register.
  if (!register) return <Spinner />

  return (
    <>
      <PageHeader title="Kassa (POS)">
        <Select value={regId || ''} onChange={(e) => setRegId(Number(e.target.value))}
          options={registers.data.map((r) => ({ value: r.id, label: r.name }))} />
        {auth.canWrite() && <Btn sm variant="ghost" onClick={() => setManage(true)}>Kassalar</Btn>}
      </PageHeader>

      {loadingSess ? <Spinner /> : !session ? (
        <OpenShift register={register} onOpened={setSession} />
      ) : (
        <Terminal session={session} onSessionChange={setSession} />
      )}
      {manage && <Registers onClose={() => { setManage(false); registers.reload() }} />}
    </>
  )
}

function OpenShift({ register, onOpened }) {
  const toast = useToast()
  const auth = useAuth()
  const [f, setF] = useState({ cashier_name: auth.user?.name || '', opening_cash: 0 })
  const [busy, setBusy] = useState(false)
  async function open() {
    setBusy(true)
    try {
      const s = await api.post('/pos/open-session', { register_id: register.id, cashier_name: f.cashier_name, opening_cash: Number(f.opening_cash) || 0 })
      toast.ok('Növbə açıldı'); onOpened(s)
    } catch (e) { toast.err(e.message) } finally { setBusy(false) }
  }
  return (
    <div className="card max-w-md">
      <h3 className="mb-1 text-[15px] font-semibold">{register.name} — növbə aç</h3>
      <p className="mb-4 text-sm text-muted">Satışa başlamaq üçün növbəni açın.</p>
      <Field label="Kassir"><Input value={f.cashier_name} onChange={(e) => setF({ ...f, cashier_name: e.target.value })} /></Field>
      <Field label="Kassada başlanğıc məbləğ (₼)"><Input type="number" step="0.01" value={f.opening_cash} onChange={(e) => setF({ ...f, opening_cash: e.target.value })} /></Field>
      <Btn variant="primary" disabled={busy} onClick={open}>{busy ? 'Açılır...' : 'Növbəni aç'}</Btn>
    </div>
  )
}

function Terminal({ session, onSessionChange }) {
  const toast = useToast()
  const products = useList('/products', [])
  const taxes = useList('/tax-rates', [])
  const [cart, setCart] = useState([])
  const [q, setQ] = useState('')
  const [pay, setPay] = useState(null) // payment modal
  const [receipt, setReceipt] = useState(null)
  const [closing, setClosing] = useState(false)
  const [sess, setSess] = useState(session)

  const items = products.data || []
  const filtered = useMemo(() => {
    const s = q.trim().toLowerCase()
    const active = items.filter((p) => p.enabled !== false)
    if (!s) return active.slice(0, 60)
    return active.filter((p) => (p.name + ' ' + (p.code || '') + ' ' + (p.barcode || '')).toLowerCase().includes(s)).slice(0, 60)
  }, [items, q])

  function add(p) {
    setCart((c) => {
      const i = c.findIndex((x) => x.product_id === p.id)
      if (i >= 0) { const n = [...c]; n[i] = { ...n[i], quantity: n[i].quantity + 1 }; return n }
      const rate = taxRate(p)
      return [...c, { product_id: p.id, name: p.name, unit_price: p.sale_price, tax_rate: rate, quantity: 1 }]
    })
  }
  // A barcode scanner types the code then presses Enter. Match by barcode /
  // code exactly (or the only visible result), drop it in the cart, clear.
  function onScanKey(e) {
    if (e.key !== 'Enter') return
    e.preventDefault()
    const code = q.trim()
    if (!code) return
    let p = items.find((x) => x.barcode && x.barcode === code)
    if (!p) p = items.find((x) => (x.code || '').toLowerCase() === code.toLowerCase())
    if (!p && filtered.length === 1) p = filtered[0]
    if (p) { add(p); setQ('') } else { toast.err('Məhsul tapılmadı: ' + code) }
  }
  function taxRate(p) {
    const t = (taxes.data || []).find((x) => x.id === p.tax_rate_id)
    return t ? t.rate : 0
  }
  function setQty(i, d) { setCart((c) => c.map((x, j) => j === i ? { ...x, quantity: Math.max(0.001, x.quantity + d) } : x)) }
  function removeLine(i) { setCart((c) => c.filter((_, j) => j !== i)) }

  const subtotal = cart.reduce((s, l) => s + l.quantity * l.unit_price, 0)
  const tax = cart.reduce((s, l) => s + (l.quantity * l.unit_price * l.tax_rate) / 100, 0)
  const total = subtotal + tax

  if (!products.data || !taxes.data) return <Spinner />

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2 rounded-xl border border-line bg-surface2 px-4 py-2.5 text-sm">
        <span><b>{sess.cashier_name || 'Kassir'}</b> · Satış: <b>{sess.sales_count}</b> · Cəm: <b className="mono">{money(sess.sales_total)} ₼</b></span>
        <Btn sm variant="ghost" onClick={() => setClosing(true)}>Növbəni bağla (Z)</Btn>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_360px]">
        {/* products */}
        <div>
          <Input placeholder="Barkodu skan et və ya məhsul axtar (ad / kod)…" value={q} onChange={(e) => setQ(e.target.value)} onKeyDown={onScanKey} className="mb-3" autoFocus />
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-4">
            {filtered.map((p) => (
              <button key={p.id} onClick={() => add(p)}
                className="flex flex-col items-start rounded-xl border border-line bg-surface p-3 text-left shadow-soft hover:border-brand">
                <span className="line-clamp-2 text-sm font-semibold">{p.name}</span>
                <span className="mt-1 mono text-brand">{money(p.sale_price)} ₼</span>
              </button>
            ))}
            {filtered.length === 0 && <div className="col-span-full py-8 text-center text-muted">Məhsul tapılmadı</div>}
          </div>
        </div>

        {/* cart */}
        <div className="rounded-2xl border border-line bg-surface shadow-soft">
          <div className="border-b border-line px-4 py-3 font-semibold">Səbət ({cart.length})</div>
          <div className="max-h-[46vh] overflow-y-auto">
            {cart.length === 0 && <div className="py-10 text-center text-muted">Boşdur — məhsula toxunun</div>}
            {cart.map((l, i) => (
              <div key={i} className="flex items-center gap-2 border-b border-line px-3 py-2">
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-medium">{l.name}</div>
                  <div className="text-xs text-muted mono">{money(l.unit_price)} × {l.quantity} = {money(l.quantity * l.unit_price)} ₼</div>
                </div>
                <button className="h-7 w-7 rounded-lg border border-line" onClick={() => setQty(i, -1)}>−</button>
                <span className="w-8 text-center mono">{l.quantity}</span>
                <button className="h-7 w-7 rounded-lg border border-line" onClick={() => setQty(i, +1)}>+</button>
                <button className="ml-1 text-danger" onClick={() => removeLine(i)}>✕</button>
              </div>
            ))}
          </div>
          <div className="space-y-1 px-4 py-3 text-sm">
            <div className="flex justify-between text-muted"><span>Ara cəm</span><span className="mono">{money(subtotal)} ₼</span></div>
            <div className="flex justify-between text-muted"><span>ƏDV</span><span className="mono">{money(tax)} ₼</span></div>
            <div className="flex justify-between text-lg font-bold"><span>Yekun</span><span className="mono">{money(total)} ₼</span></div>
          </div>
          <div className="grid grid-cols-2 gap-2 p-3">
            <Btn variant="ghost" onClick={() => setCart([])} disabled={!cart.length}>Təmizlə</Btn>
            <Btn variant="primary" disabled={!cart.length} onClick={() => setPay({ total })}>Ödə</Btn>
          </div>
        </div>
      </div>

      {pay && <Payment total={total} onClose={() => setPay(null)} onDone={async (method, paid) => {
        try {
          const sale = await api.post('/pos/sale', {
            session_id: sess.id, payment_method: method, paid,
            lines: cart.map((l) => ({ product_id: l.product_id, name: l.name, quantity: l.quantity, unit_price: l.unit_price, tax_rate: l.tax_rate })),
          })
          setPay(null); setCart([]); setReceipt(sale)
          // refresh session totals
          const os = await api.get(`/pos/registers/${sess.register_id}/open-session`); if (os) { setSess(os); onSessionChange(os) }
        } catch (e) { toast.err(e.message) }
      }} />}

      {receipt && <ReceiptModal sale={receipt} onClose={() => setReceipt(null)} />}
      {closing && <CloseShift session={sess} onClose={() => setClosing(false)} onClosed={() => { setClosing(false); onSessionChange(null) }} />}
    </>
  )
}

function Payment({ total, onClose, onDone }) {
  const [method, setMethod] = useState('cash')
  const [tendered, setTendered] = useState(total.toFixed(2))
  const change = Math.max(0, (Number(tendered) || 0) - total)
  return (
    <Modal title="Ödəniş" onClose={onClose}
      footer={<Btn variant="primary" onClick={() => onDone(method, method === 'cash' ? (Number(tendered) || total) : total)}>Təsdiqlə — {money(total)} ₼</Btn>}>
      <div className="mb-4 text-center">
        <div className="text-sm text-muted">Ödəniləcək</div>
        <div className="mono text-3xl font-extrabold">{money(total)} ₼</div>
      </div>
      <div className="mb-3 grid grid-cols-2 gap-2">
        <button className={`btn ${method === 'cash' ? 'btn-primary' : ''}`} onClick={() => setMethod('cash')}>💵 Nağd</button>
        <button className={`btn ${method === 'card' ? 'btn-primary' : ''}`} onClick={() => setMethod('card')}>💳 Kart</button>
      </div>
      {method === 'cash' && (
        <>
          <Field label="Verilən məbləğ (₼)"><Input type="number" step="0.01" value={tendered} onChange={(e) => setTendered(e.target.value)} autoFocus /></Field>
          <div className="flex justify-between rounded-xl bg-surface2 px-4 py-2 text-lg font-bold"><span>Qaytarılacaq</span><span className="mono">{money(change)} ₼</span></div>
        </>
      )}
    </Modal>
  )
}

function ReceiptModal({ sale, onClose }) {
  function printIt() {
    const w = window.open('', '_blank'); if (!w) return
    const rows = (sale.lines || []).map((l) => `<tr><td>${esc(l.name)}</td><td style="text-align:right">${l.quantity} × ${money(l.unit_price)}</td><td style="text-align:right">${money(l.line_total + l.tax_amount)}</td></tr>`).join('')
    w.document.write(`<div style="font-family:monospace;width:300px;padding:12px">
      <h3 style="text-align:center;margin:0">OAWO Kassa</h3>
      <div style="text-align:center;font-size:12px">Qaimə: ${esc(sale.number)}</div><hr>
      <table style="width:100%;font-size:12px">${rows}</table><hr>
      <div style="display:flex;justify-content:space-between"><b>YEKUN</b><b>${money(sale.total)} ₼</b></div>
      <div style="display:flex;justify-content:space-between">Ödəniş (${sale.payment_method === 'card' ? 'kart' : 'nağd'})<span>${money(sale.paid)} ₼</span></div>
      <div style="display:flex;justify-content:space-between">Qaytarıldı<span>${money(sale.change)} ₼</span></div>
      <p style="text-align:center;font-size:11px">Təşəkkür edirik!</p></div><script>window.print()</script>`)
    w.document.close()
  }
  return (
    <Modal title={'Satış tamamlandı — ' + sale.number} onClose={onClose}
      footer={<><Btn variant="ghost" onClick={printIt}>🖨 Çek</Btn><Btn variant="primary" onClick={onClose}>Yeni satış</Btn></>}>
      <div className="mb-3 text-center">
        <div className="mono text-3xl font-extrabold text-ok">{money(sale.total)} ₼</div>
        <div className="text-sm text-muted">{sale.payment_method === 'card' ? 'Kart' : 'Nağd'} · Qaytarıldı: {money(sale.change)} ₼</div>
      </div>
      <div className="rounded-xl border border-line">
        {(sale.lines || []).map((l, i) => (
          <div key={i} className="flex justify-between border-b border-line px-3 py-2 text-sm last:border-0">
            <span>{l.name} <span className="text-muted">× {l.quantity}</span></span>
            <span className="mono">{money(l.line_total + l.tax_amount)} ₼</span>
          </div>
        ))}
      </div>
    </Modal>
  )
}

function CloseShift({ session, onClose, onClosed }) {
  const toast = useToast()
  const [closingCash, setClosingCash] = useState('')
  const expected = (session.opening_cash || 0) + (session.cash_total || 0)
  async function close() {
    try { await api.post(`/pos/sessions/${session.id}/close`, { closing_cash: Number(closingCash) || 0 }); toast.ok('Növbə bağlandı'); onClosed() }
    catch (e) { toast.err(e.message) }
  }
  return (
    <Modal title="Növbənin bağlanması (Z-hesabat)" onClose={onClose}
      footer={<Btn variant="primary" onClick={close}>Növbəni bağla</Btn>}>
      <div className="mb-3 grid grid-cols-2 gap-3 text-sm">
        <Info label="Satış sayı" v={session.sales_count} />
        <Info label="Satış cəmi" v={money(session.sales_total) + ' ₼'} />
        <Info label="Nağd satış" v={money(session.cash_total) + ' ₼'} />
        <Info label="Kart satış" v={money(session.card_total) + ' ₼'} />
        <Info label="Başlanğıc kassa" v={money(session.opening_cash) + ' ₼'} />
        <Info label="Gözlənilən nağd" v={money(expected) + ' ₼'} />
      </div>
      <Field label="Kassada faktiki nağd (₼)"><Input type="number" step="0.01" value={closingCash} onChange={(e) => setClosingCash(e.target.value)} /></Field>
      {closingCash !== '' && <div className="text-sm text-muted">Fərq: <b className="mono">{money((Number(closingCash) || 0) - expected)} ₼</b></div>}
    </Modal>
  )
}
function Info({ label, v }) { return <div className="rounded-xl bg-surface2 px-3 py-2"><div className="text-xs text-muted">{label}</div><div className="mono font-bold">{v}</div></div> }

function Registers({ onClose }) {
  const toast = useToast()
  const { data, reload } = useList('/pos/registers', [])
  const [f, setF] = useState({ name: '', code: '' })
  async function create() {
    if (!f.name) { toast.err('Ad tələb olunur'); return }
    try { await api.post('/pos/registers', f); toast.ok('Kassa yaradıldı'); setF({ name: '', code: '' }); reload() } catch (e) { toast.err(e.message) }
  }
  return (
    <Modal title="Kassalar" onClose={onClose}>
      <div className="mb-3 space-y-1">
        {(data || []).map((r) => <div key={r.id} className="rounded-lg border border-line px-3 py-2 text-sm">{r.name} {r.code && <span className="text-muted">· {r.code}</span>}</div>)}
        {data && data.length === 0 && <div className="text-muted text-sm">Hələ kassa yoxdur</div>}
      </div>
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
        <Field label="Ad"><Input value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} /></Field>
        <Field label="Kod"><Input value={f.code} onChange={(e) => setF({ ...f, code: e.target.value })} /></Field>
      </div>
      <Btn variant="primary" onClick={create}>+ Kassa yarat</Btn>
    </Modal>
  )
}

function esc(s) { return String(s == null ? '' : s).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c])) }
