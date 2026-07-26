// Lightweight API client. Token + active company are injected via setters so
// the client stays framework-agnostic.
let TOKEN = localStorage.getItem('oawo_token') || ''
let COMPANY_ID = null
let onUnauthorized = () => {}

export function setToken(t) {
  TOKEN = t || ''
  if (t) localStorage.setItem('oawo_token', t)
  else localStorage.removeItem('oawo_token')
}
export function getToken() { return TOKEN }
export function setCompanyId(id) { COMPANY_ID = id || null }
export function setUnauthorizedHandler(fn) { onUnauthorized = fn }

async function req(method, path, body) {
  const headers = { 'Content-Type': 'application/json' }
  if (TOKEN) headers['Authorization'] = 'Bearer ' + TOKEN
  if (COMPANY_ID) headers['X-Company-ID'] = String(COMPANY_ID)
  const res = await fetch('/api' + path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (res.status === 401) { onUnauthorized(); throw new Error('Sessiya bitib') }
  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) throw new Error((data && data.detail) || 'Xəta ' + res.status)
  return data
}

export const api = {
  get: (p) => req('GET', p),
  post: (p, b) => req('POST', p, b),
  put: (p, b) => req('PUT', p, b),
  del: (p) => req('DELETE', p),
}

// ---- formatting helpers ----
export const money = (v) =>
  Number(v || 0).toLocaleString('az-AZ', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
export const fmtDate = (d) => (d ? String(d).slice(0, 10) : '')
export const today = () => new Date().toISOString().slice(0, 10)
export const monthStart = () => today().slice(0, 8) + '01'

export const roleLabel = (r) =>
  ({ owner: 'Sahib', admin: 'Admin', accountant: 'Mühasib', warehouse: 'Anbardar', viewer: 'Baxış' }[r] || r || '')
