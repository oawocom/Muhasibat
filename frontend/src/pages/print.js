import { money, fmtDate } from '../api.js'

// printInvoice opens a clean printable window; the browser's print dialog can
// "Save as PDF". No external dependency required.
export function printInvoice({ type, company, partner, number, date, lines, sub, tax, total }) {
  const isSales = type === 'sales_invoice'
  const title = isSales ? 'SATIŞ QAİMƏ-FAKTURASI' : 'ALIŞ QAİMƏ-FAKTURASI'
  const sellerName = isSales ? (company?.name || 'Şirkət') : (partner?.name || '—')
  const buyerName = isSales ? (partner?.name || '—') : (company?.name || 'Şirkət')
  const rows = lines.map((l, i) => {
    const net = (Number(l.quantity) || 0) * (Number(l.unit_price) || 0)
    const t = (net * (Number(l.tax_rate) || 0)) / 100
    return `<tr>
      <td>${i + 1}</td>
      <td>${esc(l.description || '')}</td>
      <td class="r">${money(l.quantity)}</td>
      <td class="r">${money(l.unit_price)}</td>
      <td class="r">${money(l.tax_rate)}%</td>
      <td class="r">${money(net + t)}</td>
    </tr>`
  }).join('')

  const html = `<!doctype html><html lang="az"><head><meta charset="utf-8"><title>${esc(number)}</title>
  <style>
    *{box-sizing:border-box} body{font-family:Arial,Helvetica,sans-serif;color:#111;margin:0;padding:32px;font-size:13px}
    h1{font-size:20px;margin:0 0 4px} .muted{color:#666}
    .top{display:flex;justify-content:space-between;align-items:flex-start;border-bottom:2px solid #111;padding-bottom:12px;margin-bottom:18px}
    .parties{display:flex;gap:24px;margin-bottom:18px}
    .party{flex:1;border:1px solid #ccc;border-radius:8px;padding:12px}
    .party h3{margin:0 0 6px;font-size:12px;text-transform:uppercase;color:#666}
    table{width:100%;border-collapse:collapse;margin-top:6px}
    th,td{border:1px solid #ccc;padding:8px;text-align:left} th{background:#f3f4f6;font-size:11px;text-transform:uppercase}
    td.r,th.r{text-align:right} tfoot td{font-weight:bold;background:#fafafa}
    .totals{margin-top:14px;margin-left:auto;width:280px}
    .totals div{display:flex;justify-content:space-between;padding:4px 0}
    .totals .grand{border-top:2px solid #111;font-size:16px;font-weight:bold;padding-top:8px}
    .sign{display:flex;justify-content:space-between;margin-top:48px}
    .sign div{width:40%;border-top:1px solid #111;padding-top:6px;text-align:center;color:#666}
    @media print{body{padding:0}}
  </style></head><body>
    <div class="top">
      <div><h1>${title}</h1><div class="muted">№ ${esc(number)} &nbsp;•&nbsp; ${esc(fmtDate(date))}</div></div>
      <div class="muted" style="text-align:right"><b>${esc(company?.name || '')}</b></div>
    </div>
    <div class="parties">
      <div class="party"><h3>Satıcı</h3><b>${esc(sellerName)}</b></div>
      <div class="party"><h3>Alıcı</h3><b>${esc(buyerName)}</b>${partner?.tax_id ? `<div class="muted">VÖEN: ${esc(partner.tax_id)}</div>` : ''}</div>
    </div>
    <table>
      <thead><tr><th>№</th><th>Adı</th><th class="r">Say</th><th class="r">Qiymət</th><th class="r">ƏDV</th><th class="r">Məbləğ</th></tr></thead>
      <tbody>${rows}</tbody>
    </table>
    <div class="totals">
      <div><span>Ara cəm:</span><span>${money(sub)} ₼</span></div>
      <div><span>ƏDV:</span><span>${money(tax)} ₼</span></div>
      <div class="grand"><span>Yekun:</span><span>${money(total)} ₼</span></div>
    </div>
    <div class="sign"><div>Təhvil verdi</div><div>Təhvil aldı</div></div>
    <script>window.onload=function(){window.print()}</script>
  </body></html>`

  const w = window.open('', '_blank')
  if (!w) { alert('Pop-up bloklandı. Zəhmət olmasa icazə verin.'); return }
  w.document.write(html)
  w.document.close()
}

function esc(s) {
  return String(s == null ? '' : s).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]))
}
