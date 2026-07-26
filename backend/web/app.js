/* OAWO Mühasibat — SPA (vanilla JS, no build step) */
(function () {
  'use strict';

  // ---------------- API ----------------
  var TOKEN = localStorage.getItem('oawo_token') || '';
  var USER = null;
  var CACHE = {}; // reference data cache

  function api(method, path, body) {
    var opts = { method: method, headers: { 'Content-Type': 'application/json' } };
    if (TOKEN) opts.headers['Authorization'] = 'Bearer ' + TOKEN;
    if (body !== undefined) opts.body = JSON.stringify(body);
    return fetch('/api' + path, opts).then(function (r) {
      if (r.status === 401) { logout(); throw new Error('Sessiya bitib'); }
      return r.text().then(function (t) {
        var d = t ? JSON.parse(t) : null;
        if (!r.ok) throw new Error((d && d.detail) || ('Xəta ' + r.status));
        return d;
      });
    });
  }
  var GET = function (p) { return api('GET', p); };
  var POST = function (p, b) { return api('POST', p, b); };
  var PUT = function (p, b) { return api('PUT', p, b); };
  var DEL = function (p) { return api('DELETE', p); };

  // ---------------- helpers ----------------
  function $(s, r) { return (r || document).querySelector(s); }
  function el(html) { var t = document.createElement('template'); t.innerHTML = html.trim(); return t.content.firstChild; }
  function esc(s) { return (s == null ? '' : String(s)).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
  function money(v) { v = Number(v || 0); return v.toLocaleString('az-AZ', { minimumFractionDigits: 2, maximumFractionDigits: 2 }); }
  function today() { return new Date().toISOString().slice(0, 10); }
  function fmtDate(d) { return d ? String(d).slice(0, 10) : ''; }

  function toast(msg, kind) {
    var t = el('<div class="toast ' + (kind || '') + '">' + esc(msg) + '</div>');
    $('#toast').appendChild(t);
    setTimeout(function () { t.style.opacity = '0'; setTimeout(function () { t.remove(); }, 300); }, 3200);
  }
  var ok = function (m) { toast(m, 'ok'); };
  var err = function (m) { toast(m, 'err'); };

  function modal(title, bodyNode, footNode, wide) {
    var ov = el('<div class="overlay"></div>');
    var m = el('<div class="modal' + (wide ? ' wide' : '') + '"><div class="mhead"><h3>' + esc(title) + '</h3><button class="x">&times;</button></div><div class="mbody"></div><div class="mfoot"></div></div>');
    $('.mbody', m).appendChild(bodyNode);
    if (footNode) $('.mfoot', m).appendChild(footNode); else $('.mfoot', m).remove();
    ov.appendChild(m); document.body.appendChild(ov);
    function close() { ov.remove(); }
    $('.x', m).onclick = close;
    ov.onclick = function (e) { if (e.target === ov) close(); };
    return { close: close, node: m };
  }

  function confirmDo(msg, fn) {
    var body = el('<div><p style="margin:0 0 4px">' + esc(msg) + '</p></div>');
    var foot = el('<div style="display:flex;gap:10px"></div>');
    var no = el('<button class="btn ghost">İmtina</button>');
    var yes = el('<button class="btn danger">Təsdiq</button>');
    foot.appendChild(no); foot.appendChild(yes);
    var mo = modal('Təsdiq', body, foot);
    no.onclick = mo.close;
    yes.onclick = function () { mo.close(); fn(); };
  }

  // form builder: fields = [{k,label,type,options,required,value,col,step}]
  function form(fields, values) {
    values = values || {};
    var wrap = el('<div></div>');
    var rows = {};
    fields.forEach(function (f) {
      var v = values[f.k] != null ? values[f.k] : (f.value != null ? f.value : '');
      var field = el('<div class="field"></div>');
      field.appendChild(el('<label class="f">' + esc(f.label) + (f.required ? ' *' : '') + '</label>'));
      var input;
      if (f.type === 'select') {
        input = el('<select></select>');
        (f.options || []).forEach(function (o) {
          var op = el('<option value="' + esc(o.value) + '">' + esc(o.label) + '</option>');
          if (String(o.value) === String(v)) op.selected = true;
          input.appendChild(op);
        });
      } else if (f.type === 'textarea') {
        input = el('<textarea rows="3"></textarea>'); input.value = v;
      } else if (f.type === 'checkbox') {
        input = el('<input type="checkbox" style="width:auto">'); input.checked = !!v;
      } else {
        input = el('<input type="' + (f.type || 'text') + '">');
        if (f.step) input.step = f.step;
        input.value = v;
      }
      input.dataset.k = f.k; input.dataset.type = f.type || 'text';
      field.appendChild(input);
      wrap.appendChild(field);
      rows[f.k] = field;
    });
    // group into grid columns where col:2
    return {
      node: wrap,
      collect: function () {
        var out = {};
        fields.forEach(function (f) {
          var input = wrap.querySelector('[data-k="' + f.k + '"]');
          if (!input) return;
          if (f.type === 'checkbox') out[f.k] = input.checked;
          else if (f.type === 'number') out[f.k] = input.value === '' ? null : Number(input.value);
          else if (f.type === 'select' && f.numeric) out[f.k] = input.value === '' ? null : Number(input.value);
          else out[f.k] = input.value;
        });
        return out;
      }
    };
  }

  // ---------------- reference data ----------------
  function loadRefs() {
    return Promise.all([
      GET('/accounts'), GET('/currencies'), GET('/tax-rates'), GET('/warehouses'), GET('/partners'), GET('/products')
    ]).then(function (r) {
      CACHE.accounts = r[0]; CACHE.currencies = r[1]; CACHE.taxRates = r[2];
      CACHE.warehouses = r[3]; CACHE.partners = r[4]; CACHE.products = r[5];
    });
  }
  function accName(id) { var a = (CACHE.accounts || []).find(function (x) { return x.id === id; }); return a ? a.code + ' — ' + a.name : id; }
  function partnerName(id) { var p = (CACHE.partners || []).find(function (x) { return x.id === id; }); return p ? p.name : '—'; }
  function accountOptions(postableOnly) {
    return (CACHE.accounts || []).filter(function (a) { return !postableOnly || !a.is_group; })
      .map(function (a) { return { value: a.id, label: a.code + ' — ' + a.name }; });
  }

  // ---------------- navigation ----------------
  var NAV = [
    { g: 'Əsas' },
    { id: 'dashboard', t: 'İdarə paneli', ic: '▚' },
    { id: 'journal', t: 'Mühasibat jurnalı', ic: '≣' },
    { g: 'Ticarət' },
    { id: 'sales', t: 'Satış fakturaları', ic: '↗' },
    { id: 'purchases', t: 'Alış fakturaları', ic: '↙' },
    { id: 'money', t: 'Kassa / Bank', ic: '₼' },
    { g: 'Kataloq' },
    { id: 'partners', t: 'Tərəfdaşlar', ic: '☺' },
    { id: 'products', t: 'Məhsul / Xidmət', ic: '▤' },
    { id: 'accounts', t: 'Hesablar planı', ic: '❏' },
    { g: 'Hesabatlar' },
    { id: 'trial', t: 'Dövriyyə balansı', ic: '∑' },
    { id: 'balance', t: 'Balans hesabatı', ic: '⚖' },
    { id: 'pl', t: 'Mənfəət və zərər', ic: '📈' },
    { id: 'partnerbal', t: 'Debitor / Kreditor', ic: '⇄' },
    { id: 'stock', t: 'Anbar qalıqları', ic: '▦' },
    { g: 'Sistem' },
    { id: 'settings', t: 'Parametrlər', ic: '⚙' }
  ];

  function renderNav() {
    var nav = $('#nav'); nav.innerHTML = '';
    NAV.forEach(function (n) {
      if (n.g) { nav.appendChild(el('<div class="lbl">' + esc(n.g) + '</div>')); return; }
      var a = el('<a href="#' + n.id + '" class="nav-item" data-id="' + n.id + '"><span class="ic">' + n.ic + '</span>' + esc(n.t) + '</a>');
      nav.appendChild(a);
    });
  }

  var ROUTES = {};
  function route() {
    var id = (location.hash || '#dashboard').slice(1).split('/')[0];
    if (!ROUTES[id]) id = 'dashboard';
    document.querySelectorAll('.nav-item').forEach(function (a) { a.classList.toggle('active', a.dataset.id === id); });
    var nav = NAV.find(function (n) { return n.id === id; });
    $('#pageTitle').textContent = nav ? nav.t : '';
    $('#pageActions').innerHTML = '';
    $('#view').innerHTML = '<div class="spin"></div>';
    if (window.innerWidth < 860) $('#sidebar').classList.remove('open');
    Promise.resolve().then(function () { return ROUTES[id](); }).catch(function (e) {
      $('#view').innerHTML = '<div class="empty">' + esc(e.message) + '</div>';
    });
  }

  function setActions(nodes) { var pa = $('#pageActions'); pa.innerHTML = ''; nodes.forEach(function (n) { pa.appendChild(n); }); }
  function view(node) { var v = $('#view'); v.innerHTML = ''; v.appendChild(node); }

  // ---------------- generic table ----------------
  function tablePanel(title, cols, rows, opts) {
    opts = opts || {};
    var p = el('<div class="panel"><div class="head"><h3>' + esc(title) + '</h3><div class="tools"></div></div><div class="body"></div></div>');
    if (opts.actions) opts.actions.forEach(function (a) { $('.tools', p).appendChild(a); });
    var body = $('.body', p);
    if (!rows.length) { body.appendChild(el('<div class="empty">Məlumat yoxdur</div>')); return p; }
    var t = el('<table></table>');
    var thead = el('<thead><tr></tr></thead>');
    cols.forEach(function (c) { $('tr', thead).appendChild(el('<th class="' + (c.cls || '') + '">' + esc(c.h) + '</th>')); });
    t.appendChild(thead);
    var tb = el('<tbody></tbody>');
    rows.forEach(function (row) {
      var tr = el('<tr></tr>');
      cols.forEach(function (c) {
        var td = el('<td class="' + (c.cls || '') + '"></td>');
        var val = c.render ? c.render(row) : row[c.k];
        if (val instanceof Node) td.appendChild(val); else td.innerHTML = val == null ? '' : val;
        tr.appendChild(td);
      });
      if (opts.onRow) { tr.style.cursor = 'pointer'; tr.onclick = function (e) { if (!e.target.closest('button,a')) opts.onRow(row); }; }
      tb.appendChild(tr);
    });
    t.appendChild(tb);
    body.appendChild(t);
    return p;
  }
  function statusBadge(s) { return '<span class="badge ' + esc(s) + '">' + esc({ draft: 'Layihə', posted: 'Təsdiqli', void: 'Ləğv', paid: 'Ödənilib' }[s] || s) + '</span>'; }
  function iconBtn(label, cls, fn) { var b = el('<button class="btn sm ' + (cls || 'ghost') + '">' + esc(label) + '</button>'); b.onclick = function (e) { e.stopPropagation(); fn(); }; return b; }

  // ================= VIEWS =================

  // ---- Dashboard ----
  ROUTES.dashboard = function () {
    return GET('/dashboard').then(function (d) {
      var wrap = el('<div></div>');
      function kpi(k, v, sub, cls, icon) {
        return '<div class="card kpi"><div class="k"><span>' + icon + '</span>' + esc(k) + '</div><div class="v mono">' + v + '</div>' + (sub ? '<div class="s ' + (cls || 'muted') + '">' + sub + '</div>' : '') + '</div>';
      }
      var cards = el('<div class="cards"></div>');
      cards.innerHTML =
        kpi('Kassa', money(d.cash) + ' ₼', 'Nağd vəsait', 'muted', '💵') +
        kpi('Bank', money(d.bank) + ' ₼', 'Hesablaşma hesabı', 'muted', '🏦') +
        kpi('Debitor borcu', money(d.receivable) + ' ₼', 'Müştərilərdən alacaq', 'pos', '↗') +
        kpi('Kreditor borcu', money(d.payable) + ' ₼', 'Təchizatçılara borc', 'neg', '↙') +
        kpi('Bu ay gəlir', money(d.income_this_month) + ' ₼', '', 'pos', '📈') +
        kpi('Bu ay xərc', money(d.expense_this_month) + ' ₼', '', 'neg', '📉') +
        kpi('Bu ay xalis', money(d.net_this_month) + ' ₼', d.net_this_month >= 0 ? 'Mənfəət' : 'Zərər', d.net_this_month >= 0 ? 'pos' : 'neg', '💰') +
        kpi('ƏDV (ödəniləcək)', money(d.vat_payable) + ' ₼', 'Çıxış − Giriş', 'muted', '🧾') +
        kpi('Anbar dəyəri', money(d.stock_value) + ' ₼', d.products + ' məhsul', 'muted', '📦') +
        kpi('Açıq fakturalar', d.open_invoices, 'Ödənilməmiş', 'amber', '⏳') +
        kpi('Tərəfdaşlar', d.partners, 'Müştəri/təchizatçı', 'muted', '👥');
      wrap.appendChild(cards);
      view(wrap);
    });
  };

  // ---- Accounts (chart) ----
  ROUTES.accounts = function () {
    return GET('/accounts').then(function (list) {
      var typeName = { asset: 'Aktiv', liability: 'Öhdəlik', equity: 'Kapital', income: 'Gəlir', expense: 'Xərc' };
      var cols = [
        { h: 'Kod', k: 'code', cls: 'mono' },
        { h: 'Ad', render: function (r) { return (r.is_group ? '<b>' : '') + esc(r.name) + (r.is_group ? '</b>' : '') + (r.system_key ? ' <span class="chip">sistem</span>' : ''); } },
        { h: 'Növ', render: function (r) { return esc(typeName[r.type] || r.type); } },
        { h: '', cls: 'right', render: function (r) {
          var w = el('<div class="tools" style="justify-content:flex-end"></div>');
          w.appendChild(iconBtn('Baş kitab', 'ghost', function () { location.hash = 'ledger/' + r.id; }));
          if (!r.is_group) w.appendChild(iconBtn('Düzəliş', 'ghost', function () { accountForm(r); }));
          return w;
        } }
      ];
      setActions([iconBtn('+ Yeni hesab', 'primary', function () { accountForm(); })]);
      view(tablePanel('Hesablar Planı (' + list.length + ')', cols, list));
    });
  };
  function accountForm(a) {
    var f = form([
      { k: 'code', label: 'Hesab kodu', required: true },
      { k: 'name', label: 'Adı', required: true },
      { k: 'type', label: 'Növ', type: 'select', options: [
        { value: 'asset', label: 'Aktiv' }, { value: 'liability', label: 'Öhdəlik' },
        { value: 'equity', label: 'Kapital' }, { value: 'income', label: 'Gəlir' }, { value: 'expense', label: 'Xərc' }] },
      { k: 'is_group', label: 'Qrup hesabı (yazılış qəbul etmir)', type: 'checkbox' },
      { k: 'description', label: 'Qeyd', type: 'textarea' }
    ], a || {});
    var foot = el('<div></div>'); var save = el('<button class="btn primary">Yadda saxla</button>'); foot.appendChild(save);
    var mo = modal(a ? 'Hesab düzəlişi' : 'Yeni hesab', f.node, foot);
    save.onclick = function () {
      var data = f.collect();
      var pr = a ? PUT('/accounts/' + a.id, data) : POST('/accounts', data);
      pr.then(function () { mo.close(); ok('Yadda saxlanıldı'); loadRefs().then(route); }).catch(function (e) { err(e.message); });
    };
  }

  // ---- Partners ----
  ROUTES.partners = function () {
    return GET('/partners').then(function (list) {
      var typeName = { customer: 'Müştəri', supplier: 'Təchizatçı', both: 'Hər ikisi' };
      var cols = [
        { h: 'Ad', render: function (r) { return '<b>' + esc(r.name) + '</b>' + (r.code ? ' <span class="muted">' + esc(r.code) + '</span>' : ''); } },
        { h: 'Növ', render: function (r) { return esc(typeName[r.type] || r.type); } },
        { h: 'VÖEN', k: 'tax_id', cls: 'mono' },
        { h: 'Telefon', k: 'phone' },
        { h: '', cls: 'right', render: function (r) { return iconBtn('Düzəliş', 'ghost', function () { partnerForm(r); }); } }
      ];
      setActions([iconBtn('+ Yeni tərəfdaş', 'primary', function () { partnerForm(); })]);
      view(tablePanel('Tərəfdaşlar (' + list.length + ')', cols, list));
    });
  };
  function partnerForm(p) {
    var f = form([
      { k: 'name', label: 'Ad / Şirkət', required: true },
      { k: 'type', label: 'Növ', type: 'select', options: [{ value: 'both', label: 'Hər ikisi' }, { value: 'customer', label: 'Müştəri' }, { value: 'supplier', label: 'Təchizatçı' }] },
      { k: 'tax_id', label: 'VÖEN' }, { k: 'phone', label: 'Telefon' },
      { k: 'email', label: 'Email' }, { k: 'contact_name', label: 'Əlaqələndirici şəxs' },
      { k: 'bank_name', label: 'Bank' }, { k: 'bank_account', label: 'IBAN' },
      { k: 'address', label: 'Ünvan', type: 'textarea' }, { k: 'notes', label: 'Qeyd', type: 'textarea' }
    ], p || {});
    var foot = el('<div></div>'); var save = el('<button class="btn primary">Yadda saxla</button>');
    if (p) { var d = el('<button class="btn danger">Sil</button>'); foot.appendChild(d); d.onclick = function () { confirmDo('Tərəfdaş silinsin?', function () { DEL('/partners/' + p.id).then(function () { mo.close(); loadRefs().then(route); }).catch(function (e) { err(e.message); }); }); }; }
    foot.appendChild(save);
    var mo = modal(p ? 'Tərəfdaş düzəlişi' : 'Yeni tərəfdaş', f.node, foot);
    save.onclick = function () { var data = f.collect(); (p ? PUT('/partners/' + p.id, data) : POST('/partners', data)).then(function () { mo.close(); ok('Yadda saxlanıldı'); loadRefs().then(route); }).catch(function (e) { err(e.message); }); };
  }

  // ---- Products ----
  ROUTES.products = function () {
    return GET('/products').then(function (list) {
      var cols = [
        { h: 'Kod', k: 'code', cls: 'mono' },
        { h: 'Ad', render: function (r) { return '<b>' + esc(r.name) + '</b>'; } },
        { h: 'Növ', render: function (r) { return r.type === 'service' ? 'Xidmət' : 'Məhsul'; } },
        { h: 'Vahid', k: 'unit' },
        { h: 'Satış qiyməti', cls: 'right mono', render: function (r) { return money(r.sale_price); } },
        { h: 'Maya', cls: 'right mono', render: function (r) { return money(r.cost_price); } },
        { h: '', cls: 'right', render: function (r) { return iconBtn('Düzəliş', 'ghost', function () { productForm(r); }); } }
      ];
      setActions([iconBtn('+ Yeni məhsul', 'primary', function () { productForm(); })]);
      view(tablePanel('Məhsul və xidmətlər (' + list.length + ')', cols, list));
    });
  };
  function productForm(p) {
    var taxOpts = [{ value: '', label: '—' }].concat((CACHE.taxRates || []).map(function (t) { return { value: t.id, label: t.name }; }));
    var f = form([
      { k: 'name', label: 'Ad', required: true }, { k: 'code', label: 'Kod / SKU' },
      { k: 'barcode', label: 'Barkod' },
      { k: 'type', label: 'Növ', type: 'select', options: [{ value: 'product', label: 'Məhsul' }, { value: 'service', label: 'Xidmət' }] },
      { k: 'unit', label: 'Ölçü vahidi', value: 'ədəd' },
      { k: 'sale_price', label: 'Satış qiyməti', type: 'number', step: '0.01' },
      { k: 'cost_price', label: 'Maya dəyəri', type: 'number', step: '0.01' },
      { k: 'tax_rate_id', label: 'ƏDV dərəcəsi', type: 'select', numeric: true, options: taxOpts },
      { k: 'track_stock', label: 'Anbar uçotu aparılsın', type: 'checkbox', value: true },
      { k: 'description', label: 'Təsvir', type: 'textarea' }
    ], p || {});
    var foot = el('<div></div>'); var save = el('<button class="btn primary">Yadda saxla</button>');
    if (p) { var d = el('<button class="btn danger">Sil</button>'); foot.appendChild(d); d.onclick = function () { confirmDo('Məhsul silinsin?', function () { DEL('/products/' + p.id).then(function () { mo.close(); loadRefs().then(route); }).catch(function (e) { err(e.message); }); }); }; }
    foot.appendChild(save);
    var mo = modal(p ? 'Məhsul düzəlişi' : 'Yeni məhsul', f.node, foot);
    save.onclick = function () { var data = f.collect(); (p ? PUT('/products/' + p.id, data) : POST('/products', data)).then(function () { mo.close(); ok('Yadda saxlanıldı'); loadRefs().then(route); }).catch(function (e) { err(e.message); }); };
  }

  // ---- Invoices (sales / purchase) ----
  ROUTES.sales = function () { return invoiceList('sales_invoice', 'Satış fakturaları', 'customer'); };
  ROUTES.purchases = function () { return invoiceList('purchase_invoice', 'Alış fakturaları', 'supplier'); };

  function invoiceList(type, title, partnerType) {
    return GET('/documents?type=' + type).then(function (list) {
      var cols = [
        { h: 'Nömrə', k: 'number', cls: 'mono' },
        { h: 'Tarix', render: function (r) { return fmtDate(r.date); } },
        { h: 'Tərəfdaş', render: function (r) { return esc(partnerName(r.partner_id)); } },
        { h: 'Cəmi', cls: 'right mono', render: function (r) { return money(r.total) + ' ₼'; } },
        { h: 'ƏDV', cls: 'right mono', render: function (r) { return money(r.tax_total); } },
        { h: 'Status', render: function (r) { return statusBadge(r.status); } }
      ];
      setActions([iconBtn('+ Yeni faktura', 'primary', function () { invoiceForm(type, partnerType); })]);
      view(tablePanel(title + ' (' + list.length + ')', cols, list, { onRow: function (r) { invoiceForm(type, partnerType, r); } }));
    });
  }

  function invoiceForm(type, partnerType, doc) {
    var isNew = !doc;
    var lines = (doc && doc.lines) ? doc.lines.map(function (l) { return Object.assign({}, l); }) : [{ description: '', quantity: 1, unit_price: 0, tax_rate: 18, product_id: null }];
    var body = el('<div></div>');
    var head = el('<div class="grid3"></div>');
    var partnerOpts = [{ value: '', label: '— seçin —' }].concat((CACHE.partners || []).filter(function (p) { return p.type === partnerType || p.type === 'both'; }).map(function (p) { return { value: p.id, label: p.name }; }));
    var whOpts = [{ value: '', label: '—' }].concat((CACHE.warehouses || []).map(function (w) { return { value: w.id, label: w.name }; }));
    var hf = form([
      { k: 'partner_id', label: partnerType === 'customer' ? 'Müştəri' : 'Təchizatçı', type: 'select', numeric: true, options: partnerOpts, value: doc && doc.partner_id },
      { k: 'date', label: 'Tarix', type: 'date', value: doc ? fmtDate(doc.date) : today() },
      { k: 'warehouse_id', label: 'Anbar', type: 'select', numeric: true, options: whOpts, value: doc && doc.warehouse_id }
    ], {});
    head.appendChild(hf.node);
    body.appendChild(head);

    var linesPanel = el('<div class="panel"><div class="head"><h3>Sətirlər</h3><div class="tools"></div></div><div class="body"></div></div>');
    var addBtn = iconBtn('+ Sətir', 'ghost', function () { lines.push({ description: '', quantity: 1, unit_price: 0, tax_rate: 18, product_id: null }); renderLines(); });
    $('.tools', linesPanel).appendChild(addBtn);
    body.appendChild(linesPanel);
    var totals = el('<div class="totrow"></div>');
    body.appendChild(totals);

    function renderLines() {
      var lb = $('.body', linesPanel); lb.innerHTML = '';
      var t = el('<table class="lines"></table>');
      t.appendChild(el('<thead><tr><th style="width:34%">Məhsul / təsvir</th><th style="width:12%">Say</th><th style="width:16%">Qiymət</th><th style="width:12%">ƏDV %</th><th class="right" style="width:16%">Məbləğ</th><th></th></tr></thead>'));
      var tb = el('<tbody></tbody>');
      lines.forEach(function (ln, i) {
        var tr = el('<tr></tr>');
        // product select
        var pTd = el('<td></td>');
        var sel = el('<select><option value="">— sərbəst —</option></select>');
        (CACHE.products || []).forEach(function (p) { var o = el('<option value="' + p.id + '">' + esc(p.name) + '</option>'); if (ln.product_id == p.id) o.selected = true; sel.appendChild(o); });
        var desc = el('<input placeholder="Təsvir" style="margin-top:5px">'); desc.value = ln.description || '';
        sel.onchange = function () {
          ln.product_id = sel.value ? Number(sel.value) : null;
          var p = (CACHE.products || []).find(function (x) { return x.id == sel.value; });
          if (p) { ln.unit_price = type === 'sales_invoice' ? p.sale_price : p.cost_price; ln.description = p.name; var tr2 = (CACHE.taxRates || []).find(function (t) { return t.id === p.tax_rate_id; }); if (tr2) ln.tax_rate = tr2.rate; renderLines(); }
        };
        desc.oninput = function () { ln.description = desc.value; };
        pTd.appendChild(sel); pTd.appendChild(desc);
        tr.appendChild(pTd);
        // qty
        tr.appendChild(numCell(ln, 'quantity', recalc));
        tr.appendChild(numCell(ln, 'unit_price', recalc));
        tr.appendChild(numCell(ln, 'tax_rate', recalc));
        var amt = el('<td class="right mono"></td>'); amt.textContent = money((ln.quantity || 0) * (ln.unit_price || 0)); tr.appendChild(amt);
        var del = el('<td class="right"></td>'); del.appendChild(iconBtn('✕', 'ghost', function () { lines.splice(i, 1); renderLines(); })); tr.appendChild(del);
        tb.appendChild(tr);
      });
      t.appendChild(tb); lb.appendChild(t); recalc();
    }
    function numCell(ln, key, cb) {
      var td = el('<td></td>'); var inp = el('<input type="number" step="0.01" class="mono right">'); inp.value = ln[key] != null ? ln[key] : 0;
      inp.oninput = function () { ln[key] = inp.value === '' ? 0 : Number(inp.value); cb(); };
      td.appendChild(inp); return td;
    }
    function recalc() {
      var sub = 0, tax = 0;
      lines.forEach(function (l) { var net = (l.quantity || 0) * (l.unit_price || 0); sub += net; tax += net * (l.tax_rate || 0) / 100; });
      totals.innerHTML = '<div><span class="lbl">Ara cəm:</span> <b class="mono">' + money(sub) + ' ₼</b></div>' +
        '<div><span class="lbl">ƏDV:</span> <b class="mono">' + money(tax) + ' ₼</b></div>' +
        '<div><span class="lbl">Yekun:</span> <b class="mono">' + money(sub + tax) + ' ₼</b></div>';
      // update per-row amounts
      var rows = linesPanel.querySelectorAll('tbody tr');
      rows.forEach(function (tr, i) { var l = lines[i]; if (l) tr.children[4].textContent = money((l.quantity || 0) * (l.unit_price || 0)); });
    }
    renderLines();

    var foot = el('<div style="display:flex;gap:10px"></div>');
    var titleTxt = (type === 'sales_invoice' ? 'Satış fakturası' : 'Alış fakturası');
    if (doc) { titleTxt += ' ' + doc.number; }
    if (doc && (doc.status === 'draft')) {
      var delB = el('<button class="btn danger">Sil</button>'); foot.appendChild(delB);
      delB.onclick = function () { confirmDo('Faktura silinsin?', function () { DEL('/documents/' + doc.id).then(function () { mo.close(); route(); }).catch(function (e) { err(e.message); }); }); };
    }
    var saveDraft, postBtn;
    if (!doc || doc.status === 'draft') {
      saveDraft = el('<button class="btn">Layihə saxla</button>');
      postBtn = el('<button class="btn primary">Təsdiqlə (kitablaşdır)</button>');
      foot.appendChild(saveDraft); foot.appendChild(postBtn);
    } else {
      foot.appendChild(el('<span class="chip">' + statusBadge(doc.status) + '</span>'));
    }

    var mo = modal(titleTxt, body, foot, true);

    function payload() {
      var h = hf.collect();
      return { type: type, partner_id: h.partner_id, warehouse_id: h.warehouse_id, date: h.date, fx_rate: 1, lines: lines.map(function (l, idx) { return { product_id: l.product_id || null, description: l.description || '', quantity: Number(l.quantity) || 0, unit_price: Number(l.unit_price) || 0, tax_rate: Number(l.tax_rate) || 0, sort_order: idx }; }) };
    }
    function persist(post) {
      var data = payload();
      if (!data.partner_id) { err('Tərəfdaş seçin'); return Promise.reject(); }
      if (!data.lines.length) { err('Ən azı bir sətir əlavə edin'); return Promise.reject(); }
      var pr;
      if (doc) { pr = PUT('/documents/' + doc.id, data).then(function () { return post ? POST('/documents/' + doc.id + '/post') : true; }); }
      else { pr = POST('/documents' + (post ? '?post=1' : ''), data); }
      return pr;
    }
    if (saveDraft) saveDraft.onclick = function () { persist(false).then(function () { mo.close(); ok('Yadda saxlanıldı'); route(); }).catch(function (e) { if (e) err(e.message); }); };
    if (postBtn) postBtn.onclick = function () { persist(true).then(function () { mo.close(); ok('Faktura təsdiqləndi'); route(); }).catch(function (e) { if (e) err(e.message); }); };
  }

  // ---- Money (payments / receipts) ----
  ROUTES.money = function () {
    return GET('/documents').then(function (all) {
      var list = all.filter(function (d) { return d.type === 'payment' || d.type === 'receipt'; });
      var cols = [
        { h: 'Nömrə', k: 'number', cls: 'mono' },
        { h: 'Tarix', render: function (r) { return fmtDate(r.date); } },
        { h: 'Növ', render: function (r) { return r.type === 'receipt' ? '<span class="pos">Mədaxil</span>' : '<span class="neg">Məxaric</span>'; } },
        { h: 'Tərəfdaş', render: function (r) { return esc(partnerName(r.partner_id)); } },
        { h: 'Məbləğ', cls: 'right mono', render: function (r) { return money(r.total) + ' ₼'; } },
        { h: 'Status', render: function (r) { return statusBadge(r.status); } }
      ];
      setActions([
        iconBtn('+ Mədaxil (müştəridən)', 'primary', function () { moneyForm('receipt'); }),
        iconBtn('+ Məxaric (təchizatçıya)', 'ghost', function () { moneyForm('payment'); })
      ]);
      view(tablePanel('Kassa / Bank əməliyyatları (' + list.length + ')', cols, list, { onRow: function (r) { if (r.status === 'draft') moneyForm(r.type, r); } }));
    });
  };
  function moneyForm(type, doc) {
    var partnerType = type === 'receipt' ? 'customer' : 'supplier';
    var partnerOpts = [{ value: '', label: '— seçin —' }].concat((CACHE.partners || []).filter(function (p) { return p.type === partnerType || p.type === 'both'; }).map(function (p) { return { value: p.id, label: p.name }; }));
    var cashOpts = (CACHE.accounts || []).filter(function (a) { return a.system_key === 'cash' || a.system_key === 'bank'; }).map(function (a) { return { value: a.id, label: a.code + ' — ' + a.name }; });
    var f = form([
      { k: 'partner_id', label: partnerType === 'customer' ? 'Müştəri' : 'Təchizatçı', type: 'select', numeric: true, options: partnerOpts, value: doc && doc.partner_id },
      { k: 'cash_account_id', label: 'Kassa / Bank', type: 'select', numeric: true, options: cashOpts, value: doc && doc.cash_account_id },
      { k: 'date', label: 'Tarix', type: 'date', value: doc ? fmtDate(doc.date) : today() },
      { k: 'amount', label: 'Məbləğ (₼)', type: 'number', step: '0.01', value: doc ? doc.total : '' },
      { k: 'notes', label: 'Təyinat', type: 'textarea', value: doc && doc.notes }
    ], {});
    var foot = el('<div style="display:flex;gap:10px"></div>');
    var post = el('<button class="btn primary">Təsdiqlə</button>'); foot.appendChild(post);
    var mo = modal(type === 'receipt' ? 'Mədaxil order' : 'Məxaric order', f.node, foot);
    post.onclick = function () {
      var d = f.collect();
      if (!d.partner_id || !d.cash_account_id || !d.amount) { err('Bütün sahələri doldurun'); return; }
      var data = { type: type, partner_id: d.partner_id, cash_account_id: d.cash_account_id, date: d.date, notes: d.notes, fx_rate: 1, lines: [{ description: type === 'receipt' ? 'Müştəridən mədaxil' : 'Təchizatçıya ödəniş', quantity: 1, unit_price: Number(d.amount), tax_rate: 0 }] };
      POST('/documents?post=1', data).then(function () { mo.close(); ok('Əməliyyat təsdiqləndi'); route(); }).catch(function (e) { err(e.message); });
    };
  }

  // ---- Journal ----
  ROUTES.journal = function () {
    return GET('/journal').then(function (list) {
      var cols = [
        { h: 'Nömrə', k: 'number', cls: 'mono' },
        { h: 'Tarix', render: function (r) { return fmtDate(r.date); } },
        { h: 'Təsvir', render: function (r) { return esc(r.description || '') + (r.reference ? ' <span class="muted">/ ' + esc(r.reference) + '</span>' : ''); } },
        { h: 'Debet', cls: 'right mono', render: function (r) { return money(r.total_debit); } },
        { h: 'Kredit', cls: 'right mono', render: function (r) { return money(r.total_credit); } },
        { h: 'Status', render: function (r) { return statusBadge(r.status); } }
      ];
      setActions([iconBtn('+ Yeni yazılış', 'primary', function () { journalForm(); })]);
      view(tablePanel('Mühasibat jurnalı (' + list.length + ')', cols, list, { onRow: function (r) { journalForm(r); } }));
    });
  };
  function journalForm(entry) {
    var lines = (entry && entry.lines) ? entry.lines.map(function (l) { return Object.assign({}, l); }) : [{ account_id: null, debit: 0, credit: 0, description: '' }, { account_id: null, debit: 0, credit: 0, description: '' }];
    var readonly = entry && entry.status === 'posted';
    var body = el('<div></div>');
    var hf = form([
      { k: 'date', label: 'Tarix', type: 'date', value: entry ? fmtDate(entry.date) : today() },
      { k: 'description', label: 'Təsvir', value: entry && entry.description },
      { k: 'reference', label: 'İstinad', value: entry && entry.reference }
    ], {});
    body.appendChild(el('<div class="grid3"></div>')).appendChild(hf.node);
    var lp = el('<div class="panel"><div class="head"><h3>Yazılış sətirləri</h3><div class="tools"></div></div><div class="body"></div></div>');
    if (!readonly) $('.tools', lp).appendChild(iconBtn('+ Sətir', 'ghost', function () { lines.push({ account_id: null, debit: 0, credit: 0, description: '' }); renderLines(); }));
    body.appendChild(lp);
    var tot = el('<div class="totrow"></div>'); body.appendChild(tot);

    function renderLines() {
      var lb = $('.body', lp); lb.innerHTML = '';
      var t = el('<table class="lines"></table>');
      t.appendChild(el('<thead><tr><th style="width:38%">Hesab</th><th style="width:24%">Təsvir</th><th class="right" style="width:16%">Debet</th><th class="right" style="width:16%">Kredit</th><th></th></tr></thead>'));
      var tb = el('<tbody></tbody>');
      lines.forEach(function (ln, i) {
        var tr = el('<tr></tr>');
        var aTd = el('<td></td>');
        if (readonly) { aTd.innerHTML = esc(accName(ln.account_id)); }
        else {
          var sel = el('<select><option value="">— hesab —</option></select>');
          accountOptions(true).forEach(function (o) { var op = el('<option value="' + o.value + '">' + esc(o.label) + '</option>'); if (ln.account_id == o.value) op.selected = true; sel.appendChild(op); });
          sel.onchange = function () { ln.account_id = sel.value ? Number(sel.value) : null; };
          aTd.appendChild(sel);
        }
        tr.appendChild(aTd);
        var dTd = el('<td></td>');
        if (readonly) dTd.textContent = ln.description || ''; else { var di = el('<input>'); di.value = ln.description || ''; di.oninput = function () { ln.description = di.value; }; dTd.appendChild(di); }
        tr.appendChild(dTd);
        tr.appendChild(amtCell(ln, 'debit', readonly));
        tr.appendChild(amtCell(ln, 'credit', readonly));
        var del = el('<td class="right"></td>'); if (!readonly) del.appendChild(iconBtn('✕', 'ghost', function () { lines.splice(i, 1); renderLines(); })); tr.appendChild(del);
        tb.appendChild(tr);
      });
      t.appendChild(tb); lb.appendChild(t); recalc();
    }
    function amtCell(ln, key, ro) {
      var td = el('<td class="right"></td>');
      if (ro) { td.className = 'right mono'; td.textContent = money(ln[key]); return td; }
      var inp = el('<input type="number" step="0.01" class="mono right">'); inp.value = ln[key] || 0;
      inp.oninput = function () { ln[key] = inp.value === '' ? 0 : Number(inp.value); if (key === 'debit' && ln[key]) ln.credit = 0; if (key === 'credit' && ln[key]) ln.debit = 0; recalc(); };
      td.appendChild(inp); return td;
    }
    function recalc() {
      var d = 0, c = 0; lines.forEach(function (l) { d += Number(l.debit) || 0; c += Number(l.credit) || 0; });
      var bal = Math.abs(d - c) < 0.005;
      tot.innerHTML = '<div><span class="lbl">Debet:</span> <b class="mono">' + money(d) + '</b></div>' +
        '<div><span class="lbl">Kredit:</span> <b class="mono">' + money(c) + '</b></div>' +
        '<div><span class="lbl">Fərq:</span> <b class="mono ' + (bal ? 'pos' : 'neg') + '">' + money(d - c) + '</b></div>';
    }
    renderLines();

    var foot = el('<div style="display:flex;gap:10px"></div>');
    var t = 'Mühasibat yazılışı' + (entry ? ' ' + entry.number : '');
    if (readonly) {
      var unp = el('<button class="btn">Geri qaytar (draft)</button>'); foot.appendChild(unp);
      unp.onclick = function () { POST('/journal/' + entry.id + '/unpost').then(function () { mo.close(); ok('Geri qaytarıldı'); route(); }).catch(function (e) { err(e.message); }); };
      foot.appendChild(el('<span class="chip">' + statusBadge('posted') + '</span>'));
    } else {
      if (entry) { var db = el('<button class="btn danger">Sil</button>'); foot.appendChild(db); db.onclick = function () { confirmDo('Silinsin?', function () { DEL('/journal/' + entry.id).then(function () { mo.close(); route(); }).catch(function (e) { err(e.message); }); }); }; }
      var sd = el('<button class="btn">Layihə saxla</button>');
      var pb = el('<button class="btn primary">Təsdiqlə</button>');
      foot.appendChild(sd); foot.appendChild(pb);
      var save = function (post) {
        var h = hf.collect();
        var data = { date: h.date, description: h.description, reference: h.reference, lines: lines.filter(function (l) { return l.account_id; }).map(function (l, i) { return { account_id: l.account_id, description: l.description || '', debit: Number(l.debit) || 0, credit: Number(l.credit) || 0, sort_order: i }; }) };
        var pr = entry ? PUT('/journal/' + entry.id, data).then(function () { return post ? POST('/journal/' + entry.id + '/post') : true; }) : POST('/journal' + (post ? '?post=1' : ''), data);
        pr.then(function () { mo.close(); ok('Yadda saxlanıldı'); route(); }).catch(function (e) { err(e.message); });
      };
      sd.onclick = function () { save(false); };
      pb.onclick = function () { save(true); };
    }
    var mo = modal(t, body, foot, true);
  }

  // ---- Reports ----
  function dateFilterBar(onChange, withFrom) {
    var bar = el('<div class="tools"></div>');
    var from, to;
    if (withFrom) { from = el('<input type="date" style="width:150px">'); from.value = today().slice(0, 8) + '01'; bar.appendChild(el('<span class="muted">Tarix:</span>')); bar.appendChild(from); bar.appendChild(el('<span class="muted">—</span>')); }
    to = el('<input type="date" style="width:150px">'); to.value = today(); bar.appendChild(to);
    var b = el('<button class="btn sm primary">Yenilə</button>'); bar.appendChild(b);
    b.onclick = function () { onChange(from ? from.value : null, to.value); };
    return bar;
  }

  ROUTES.trial = function () {
    var render = function (from, to) {
      GET('/reports/trial-balance' + (to ? '?to=' + to : '')).then(function (rep) {
        var cols = [
          { h: 'Kod', render: function (r) { return '<span class="mono">' + esc(r.code) + '</span>'; } },
          { h: 'Hesab', k: 'name' },
          { h: 'Debet', cls: 'right mono', render: function (r) { return money(r.debit); } },
          { h: 'Kredit', cls: 'right mono', render: function (r) { return money(r.credit); } }
        ];
        var p = tablePanel('Dövriyyə balansı', cols, rep.rows);
        var tb = $('tbody', p);
        if (tb) tb.appendChild(el('<tr style="font-weight:800;background:var(--surface2)"><td colspan="2">CƏMİ</td><td class="right mono">' + money(rep.total_debit) + '</td><td class="right mono">' + money(rep.total_credit) + '</td></tr>'));
        var wrap = el('<div></div>');
        wrap.appendChild(el('<div class="card" style="margin-bottom:16px">' + (rep.balanced ? '<span class="pos">✓ Balans bərabərdir</span>' : '<span class="neg">⚠ Balanssızlıq var</span>') + ' &nbsp;•&nbsp; <span class="muted">Debet ' + money(rep.total_debit) + ' = Kredit ' + money(rep.total_credit) + '</span></div>'));
        wrap.appendChild(p);
        view(wrap);
      }).catch(function (e) { err(e.message); });
    };
    setActions([dateFilterBar(render, false)]);
    render(null, today());
    return Promise.resolve();
  };

  ROUTES.balance = function () {
    var render = function (from, to) {
      GET('/reports/balance-sheet' + (to ? '?to=' + to : '')).then(function (rep) {
        function section(title, sec, tt) {
          var rows = (sec.rows || []).map(function (r) { return { code: r.code, name: r.name, bal: r.balance }; });
          var cols = [{ h: 'Kod', render: function (r) { return '<span class="mono">' + esc(r.code) + '</span>'; } }, { h: 'Hesab', k: 'name' }, { h: 'Məbləğ', cls: 'right mono', render: function (r) { return money(r.bal); } }];
          var p = tablePanel(title, cols, rows);
          var tb = $('tbody', p); if (tb) tb.appendChild(el('<tr style="font-weight:800;background:var(--surface2)"><td colspan="2">' + esc(tt) + '</td><td class="right mono">' + money(sec.total) + '</td></tr>'));
          return p;
        }
        var wrap = el('<div class="grid2"></div>');
        var left = el('<div></div>'); left.appendChild(section('AKTİVLƏR', rep.assets, 'Cəmi aktivlər'));
        var right = el('<div></div>');
        right.appendChild(section('ÖHDƏLİKLƏR', rep.liabilities, 'Cəmi öhdəliklər'));
        right.appendChild(section('KAPİTAL', rep.equity, 'Cəmi kapital'));
        right.appendChild(el('<div class="card"><div class="k muted">Dövrün mənfəəti (zərər)</div><div class="v mono ' + (rep.net_income >= 0 ? 'pos' : 'neg') + '">' + money(rep.net_income) + ' ₼</div></div>'));
        right.appendChild(el('<div class="card" style="margin-top:12px">' + (rep.balanced ? '<span class="pos">✓ Aktiv = Passiv (' + money(rep.assets.total) + ')</span>' : '<span class="neg">⚠ Aktiv ' + money(rep.assets.total) + ' ≠ Passiv ' + money(rep.total_liabilities_equity) + '</span>') + '</div>'));
        wrap.appendChild(left); wrap.appendChild(right);
        view(wrap);
      }).catch(function (e) { err(e.message); });
    };
    setActions([dateFilterBar(render, false)]);
    render(null, today());
    return Promise.resolve();
  };

  ROUTES.pl = function () {
    var render = function (from, to) {
      var qs = []; if (from) qs.push('from=' + from); if (to) qs.push('to=' + to);
      GET('/reports/profit-loss' + (qs.length ? '?' + qs.join('&') : '')).then(function (rep) {
        function section(title, sec, tt) {
          var cols = [{ h: 'Kod', render: function (r) { return '<span class="mono">' + esc(r.code) + '</span>'; } }, { h: 'Hesab', k: 'name' }, { h: 'Məbləğ', cls: 'right mono', render: function (r) { return money(r.balance); } }];
          var p = tablePanel(title, cols, sec.rows || []);
          var tb = $('tbody', p); if (tb) tb.appendChild(el('<tr style="font-weight:800;background:var(--surface2)"><td colspan="2">' + esc(tt) + '</td><td class="right mono">' + money(sec.total) + '</td></tr>'));
          return p;
        }
        var wrap = el('<div></div>');
        wrap.appendChild(el('<div class="cards" style="margin-bottom:16px"><div class="card kpi"><div class="k">Gəlirlər</div><div class="v mono pos">' + money(rep.income.total) + ' ₼</div></div><div class="card kpi"><div class="k">Xərclər</div><div class="v mono neg">' + money(rep.expense.total) + ' ₼</div></div><div class="card kpi"><div class="k">Xalis mənfəət</div><div class="v mono ' + (rep.net_profit >= 0 ? 'pos' : 'neg') + '">' + money(rep.net_profit) + ' ₼</div></div></div>'));
        wrap.appendChild(section('GƏLİRLƏR', rep.income, 'Cəmi gəlir'));
        wrap.appendChild(section('XƏRCLƏR', rep.expense, 'Cəmi xərc'));
        view(wrap);
      }).catch(function (e) { err(e.message); });
    };
    setActions([dateFilterBar(render, true)]);
    render(today().slice(0, 8) + '01', today());
    return Promise.resolve();
  };

  ROUTES.partnerbal = function () {
    return GET('/reports/partner-balances').then(function (rows) {
      rows.sort(function (a, b) { return Math.abs(b.net) - Math.abs(a.net); });
      var cols = [
        { h: 'Tərəfdaş', k: 'name' },
        { h: 'Debitor (bizə borc)', cls: 'right mono', render: function (r) { return money(r.receivable); } },
        { h: 'Kreditor (bizim borc)', cls: 'right mono', render: function (r) { return money(r.payable); } },
        { h: 'Xalis', cls: 'right mono', render: function (r) { return '<span class="' + (r.net >= 0 ? 'pos' : 'neg') + '">' + money(r.net) + '</span>'; } }
      ];
      view(tablePanel('Debitor / Kreditor borcları', cols, rows));
    });
  };

  ROUTES.stock = function () {
    return GET('/reports/stock').then(function (rows) {
      var cols = [
        { h: 'Kod', k: 'code', cls: 'mono' }, { h: 'Məhsul', k: 'name' }, { h: 'Vahid', k: 'unit' },
        { h: 'Qalıq', cls: 'right mono', render: function (r) { return money(r.quantity); } },
        { h: 'Dəyər', cls: 'right mono', render: function (r) { return money(r.value) + ' ₼'; } }
      ];
      view(tablePanel('Anbar qalıqları', cols, rows));
    });
  };

  // Ledger route (hash: #ledger/:id but nav id is 'ledger')
  ROUTES.ledger = function () {
    var accId = (location.hash.split('/')[1]) || '';
    if (!accId) { view(el('<div class="empty">Hesablar planından hesab seçin</div>')); return Promise.resolve(); }
    return GET('/reports/ledger/' + accId).then(function (rep) {
      var cols = [
        { h: 'Tarix', render: function (r) { return fmtDate(r.date); } },
        { h: 'Sənəd', k: 'number', cls: 'mono' },
        { h: 'Təsvir', k: 'description' },
        { h: 'Debet', cls: 'right mono', render: function (r) { return money(r.debit); } },
        { h: 'Kredit', cls: 'right mono', render: function (r) { return money(r.credit); } },
        { h: 'Qalıq', cls: 'right mono', render: function (r) { return money(r.balance); } }
      ];
      var wrap = el('<div></div>');
      wrap.appendChild(el('<div class="card" style="margin-bottom:16px"><b>' + esc(rep.account.code + ' — ' + rep.account.name) + '</b> &nbsp;•&nbsp; <span class="muted">Açılış: ' + money(rep.opening) + ' / Bağlanış: <b>' + money(rep.closing) + '</b></span></div>'));
      wrap.appendChild(tablePanel('Baş kitab (hesab üzrə hərəkət)', cols, rep.lines));
      $('#pageTitle').textContent = 'Baş kitab';
      view(wrap);
    });
  };

  // ---- Settings ----
  ROUTES.settings = function () {
    return Promise.all([GET('/settings'), GET('/currencies'), GET('/tax-rates'), GET('/warehouses')]).then(function (r) {
      var st = r[0], curr = r[1], tax = r[2], wh = r[3];
      var wrap = el('<div></div>');
      // company
      var cf = form([{ k: 'company_name', label: 'Şirkət adı', value: st.company_name || '' }], {});
      var cp = el('<div class="panel"><div class="head"><h3>Şirkət</h3></div><div class="body" style="padding:18px"></div></div>');
      $('.body', cp).appendChild(cf.node);
      var cbtn = el('<button class="btn primary">Yadda saxla</button>'); $('.body', cp).appendChild(cbtn);
      cbtn.onclick = function () { PUT('/settings', cf.collect()).then(function () { ok('Yadda saxlanıldı'); }).catch(function (e) { err(e.message); }); };
      wrap.appendChild(cp);
      // currencies
      var ccols = [{ h: 'Kod', k: 'code', cls: 'mono' }, { h: 'Ad', k: 'name' }, { h: 'Məzənnə', cls: 'right mono', render: function (r) { return r.is_base ? 'Baza' : money(r.rate); } }, { h: '', cls: 'right', render: function (r) { return r.is_base ? '' : iconBtn('Düzəliş', 'ghost', function () { simpleEdit('/currencies/' + r.id, [{ k: 'rate', label: 'Məzənnə', type: 'number', step: '0.000001' }], r, 'Valyuta məzənnəsi'); }); } }];
      wrap.appendChild(tablePanel('Valyutalar', ccols, curr, { actions: [iconBtn('+ Valyuta', 'ghost', function () { simpleEdit('/currencies', [{ k: 'code', label: 'Kod' }, { k: 'name', label: 'Ad' }, { k: 'symbol', label: 'Simvol' }, { k: 'rate', label: 'Məzənnə', type: 'number', step: '0.000001' }], { rate: 1, enabled: true }, 'Yeni valyuta'); })] }));
      // tax
      var tcols = [{ h: 'Ad', k: 'name' }, { h: 'Dərəcə %', cls: 'right mono', render: function (r) { return money(r.rate); } }, { h: '', cls: 'right', render: function (r) { return iconBtn('Düzəliş', 'ghost', function () { simpleEdit('/tax-rates/' + r.id, [{ k: 'name', label: 'Ad' }, { k: 'rate', label: 'Dərəcə', type: 'number', step: '0.01' }], r, 'ƏDV dərəcəsi'); }); } }];
      wrap.appendChild(tablePanel('ƏDV dərəcələri', tcols, tax, { actions: [iconBtn('+ Dərəcə', 'ghost', function () { simpleEdit('/tax-rates', [{ k: 'name', label: 'Ad' }, { k: 'rate', label: 'Dərəcə', type: 'number', step: '0.01' }], { enabled: true }, 'Yeni dərəcə'); })] }));
      // warehouses
      var wcols = [{ h: 'Kod', k: 'code', cls: 'mono' }, { h: 'Ad', k: 'name' }, { h: '', cls: 'right', render: function (r) { return iconBtn('Düzəliş', 'ghost', function () { simpleEdit('/warehouses/' + r.id, [{ k: 'code', label: 'Kod' }, { k: 'name', label: 'Ad' }, { k: 'address', label: 'Ünvan', type: 'textarea' }], r, 'Anbar'); }); } }];
      wrap.appendChild(tablePanel('Anbarlar', wcols, wh, { actions: [iconBtn('+ Anbar', 'ghost', function () { simpleEdit('/warehouses', [{ k: 'code', label: 'Kod' }, { k: 'name', label: 'Ad' }, { k: 'address', label: 'Ünvan', type: 'textarea' }], { enabled: true }, 'Yeni anbar'); })] }));
      // password
      var pf = form([{ k: 'old_password', label: 'Köhnə şifrə', type: 'password' }, { k: 'new_password', label: 'Yeni şifrə', type: 'password' }], {});
      var pp = el('<div class="panel"><div class="head"><h3>Şifrə dəyişdir</h3></div><div class="body" style="padding:18px"></div></div>');
      $('.body', pp).appendChild(pf.node);
      var pbtn = el('<button class="btn primary">Dəyiş</button>'); $('.body', pp).appendChild(pbtn);
      pbtn.onclick = function () { POST('/auth/change-password', pf.collect()).then(function () { ok('Şifrə dəyişdirildi'); }).catch(function (e) { err(e.message); }); };
      wrap.appendChild(pp);
      view(wrap);
    });
  };
  function simpleEdit(path, fields, values, title) {
    var isNew = path.indexOf('/', 1) === -1 || !/\d$/.test(path);
    var f = form(fields, values || {});
    var foot = el('<div></div>'); var save = el('<button class="btn primary">Yadda saxla</button>'); foot.appendChild(save);
    var mo = modal(title, f.node, foot);
    save.onclick = function () {
      var data = Object.assign({}, values, f.collect());
      var method = /\/\d+$/.test(path) ? PUT : POST;
      method(path, data).then(function () { mo.close(); ok('Yadda saxlanıldı'); loadRefs().then(route); }).catch(function (e) { err(e.message); });
    };
  }

  // ---------------- auth / boot ----------------
  function showApp() {
    $('#login').classList.add('hidden');
    $('#app').classList.remove('hidden');
    $('#userName').textContent = USER ? USER.name || USER.email : '';
    renderNav();
    if (window.innerWidth < 860) $('#menuBtn').style.display = '';
    loadRefs().then(route);
  }
  function logout() {
    TOKEN = ''; localStorage.removeItem('oawo_token');
    $('#app').classList.add('hidden'); $('#login').classList.remove('hidden');
  }

  $('#loginForm').addEventListener('submit', function (e) {
    e.preventDefault();
    POST('/auth/login', { email: $('#li_email').value, password: $('#li_pass').value })
      .then(function (r) { TOKEN = r.token; USER = r.user; localStorage.setItem('oawo_token', TOKEN); ok('Xoş gəlmisiniz!'); showApp(); })
      .catch(function (e) { err(e.message); });
  });
  $('#logout').addEventListener('click', function (e) { e.preventDefault(); logout(); });
  $('#menuBtn').addEventListener('click', function () { $('#sidebar').classList.toggle('open'); });
  window.addEventListener('hashchange', route);

  // auto-login if token exists
  if (TOKEN) {
    GET('/auth/me').then(function (u) { USER = u; showApp(); }).catch(function () { logout(); });
  }
})();
