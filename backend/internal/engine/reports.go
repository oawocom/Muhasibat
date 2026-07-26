package engine

import (
	"time"

	"gorm.io/gorm"

	"oawo-muhasibat/internal/models"
)

// ============================================================
// Reporting — all figures derive from POSTED journal lines only.
// ============================================================

type AccountBalance struct {
	AccountID uint    `json:"account_id"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Debit     float64 `json:"debit"`
	Credit    float64 `json:"credit"`
	Balance   float64 `json:"balance"` // signed by normal side
}

type sumRow struct {
	AccountID uint
	Debit     float64
	Credit    float64
}

// accountSums returns posted debit/credit totals per account up to `to`
// (inclusive) and optionally from `from`.
func accountSums(db *gorm.DB, from, to *time.Time) (map[uint]sumRow, error) {
	q := db.Model(&models.JournalLine{}).
		Select("account_id, COALESCE(SUM(debit),0) as debit, COALESCE(SUM(credit),0) as credit").
		Where("posted = ?", true).
		Group("account_id")
	if from != nil {
		q = q.Where("date >= ?", *from)
	}
	if to != nil {
		q = q.Where("date <= ?", *to)
	}
	var rows []sumRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	m := make(map[uint]sumRow, len(rows))
	for _, r := range rows {
		m[r.AccountID] = r
	}
	return m, nil
}

// normalBalance returns the signed balance for an account type.
func normalBalance(t string, debit, credit float64) float64 {
	switch t {
	case "asset", "expense":
		return round2(debit - credit)
	default: // liability, equity, income
		return round2(credit - debit)
	}
}

// TrialBalance — every non-group account with debit/credit/balance.
type TrialBalanceReport struct {
	Rows        []AccountBalance `json:"rows"`
	TotalDebit  float64          `json:"total_debit"`
	TotalCredit float64          `json:"total_credit"`
	Balanced    bool             `json:"balanced"`
	AsOf        time.Time        `json:"as_of"`
}

func TrialBalance(db *gorm.DB, to *time.Time) (*TrialBalanceReport, error) {
	sums, err := accountSums(db, nil, to)
	if err != nil {
		return nil, err
	}
	var accounts []models.Account
	db.Where("is_group = ?", false).Order("code asc").Find(&accounts)

	rep := &TrialBalanceReport{}
	if to != nil {
		rep.AsOf = *to
	} else {
		rep.AsOf = time.Now()
	}
	for _, a := range accounts {
		s := sums[a.ID]
		if s.Debit == 0 && s.Credit == 0 {
			continue
		}
		bal := normalBalance(a.Type, s.Debit, s.Credit)
		rep.Rows = append(rep.Rows, AccountBalance{
			AccountID: a.ID, Code: a.Code, Name: a.Name, Type: a.Type,
			Debit: round2(s.Debit), Credit: round2(s.Credit), Balance: bal,
		})
		rep.TotalDebit += s.Debit
		rep.TotalCredit += s.Credit
	}
	rep.TotalDebit = round2(rep.TotalDebit)
	rep.TotalCredit = round2(rep.TotalCredit)
	rep.Balanced = round2(rep.TotalDebit-rep.TotalCredit) == 0
	return rep, nil
}

// Ledger — running-balance statement for a single account.
type LedgerLine struct {
	EntryID     uint      `json:"entry_id"`
	Number      string    `json:"number"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	Debit       float64   `json:"debit"`
	Credit      float64   `json:"credit"`
	Balance     float64   `json:"balance"`
}

type LedgerReport struct {
	Account   models.Account `json:"account"`
	Opening   float64        `json:"opening"`
	Closing   float64        `json:"closing"`
	Lines     []LedgerLine   `json:"lines"`
}

func Ledger(db *gorm.DB, accountID uint, from, to *time.Time) (*LedgerReport, error) {
	var acc models.Account
	if err := db.First(&acc, accountID).Error; err != nil {
		return nil, err
	}
	rep := &LedgerReport{Account: acc}

	// Opening balance = signed balance strictly before `from`.
	if from != nil {
		before := from.Add(-time.Nanosecond)
		sums, _ := accountSums(db.Where("account_id = ?", accountID), nil, &before)
		s := sums[accountID]
		rep.Opening = normalBalance(acc.Type, s.Debit, s.Credit)
	}

	type row struct {
		EntryID     uint
		Number      string
		Date        time.Time
		Description string
		Debit       float64
		Credit      float64
	}
	q := db.Table("journal_lines as jl").
		Select("jl.entry_id, je.number, jl.date, COALESCE(NULLIF(jl.description,''), je.description) as description, jl.debit, jl.credit").
		Joins("JOIN journal_entries je ON je.id = jl.entry_id").
		Where("jl.account_id = ? AND jl.posted = ?", accountID, true)
	if from != nil {
		q = q.Where("jl.date >= ?", *from)
	}
	if to != nil {
		q = q.Where("jl.date <= ?", *to)
	}
	var rows []row
	q.Order("jl.date asc, jl.entry_id asc").Scan(&rows)

	running := rep.Opening
	sign := 1.0
	if acc.Type == "liability" || acc.Type == "equity" || acc.Type == "income" {
		sign = -1.0
	}
	for _, r := range rows {
		running = round2(running + sign*(r.Debit-r.Credit))
		rep.Lines = append(rep.Lines, LedgerLine{
			EntryID: r.EntryID, Number: r.Number, Date: r.Date, Description: r.Description,
			Debit: round2(r.Debit), Credit: round2(r.Credit), Balance: running,
		})
	}
	rep.Closing = running
	return rep, nil
}

// Balance Sheet & P&L
type Section struct {
	Rows  []AccountBalance `json:"rows"`
	Total float64          `json:"total"`
}

type BalanceSheetReport struct {
	Assets      Section   `json:"assets"`
	Liabilities Section   `json:"liabilities"`
	Equity      Section   `json:"equity"`
	NetIncome   float64   `json:"net_income"`
	TotalLiabEq float64   `json:"total_liabilities_equity"`
	Balanced    bool      `json:"balanced"`
	AsOf        time.Time `json:"as_of"`
}

func BalanceSheet(db *gorm.DB, to *time.Time) (*BalanceSheetReport, error) {
	sums, err := accountSums(db, nil, to)
	if err != nil {
		return nil, err
	}
	var accounts []models.Account
	db.Where("is_group = ?", false).Order("code asc").Find(&accounts)

	rep := &BalanceSheetReport{}
	if to != nil {
		rep.AsOf = *to
	} else {
		rep.AsOf = time.Now()
	}
	var income, expense float64
	for _, a := range accounts {
		s := sums[a.ID]
		if s.Debit == 0 && s.Credit == 0 {
			continue
		}
		bal := normalBalance(a.Type, s.Debit, s.Credit)
		row := AccountBalance{AccountID: a.ID, Code: a.Code, Name: a.Name, Type: a.Type, Balance: bal}
		switch a.Type {
		case "asset":
			rep.Assets.Rows = append(rep.Assets.Rows, row)
			rep.Assets.Total += bal
		case "liability":
			rep.Liabilities.Rows = append(rep.Liabilities.Rows, row)
			rep.Liabilities.Total += bal
		case "equity":
			rep.Equity.Rows = append(rep.Equity.Rows, row)
			rep.Equity.Total += bal
		case "income":
			income += bal
		case "expense":
			expense += bal
		}
	}
	rep.NetIncome = round2(income - expense)
	rep.Assets.Total = round2(rep.Assets.Total)
	rep.Liabilities.Total = round2(rep.Liabilities.Total)
	rep.Equity.Total = round2(rep.Equity.Total)
	rep.TotalLiabEq = round2(rep.Liabilities.Total + rep.Equity.Total + rep.NetIncome)
	rep.Balanced = round2(rep.Assets.Total-rep.TotalLiabEq) == 0
	return rep, nil
}

type ProfitLossReport struct {
	Income    Section   `json:"income"`
	Expense   Section   `json:"expense"`
	NetProfit float64   `json:"net_profit"`
	From      *time.Time `json:"from"`
	To        *time.Time `json:"to"`
}

func ProfitLoss(db *gorm.DB, from, to *time.Time) (*ProfitLossReport, error) {
	sums, err := accountSums(db, from, to)
	if err != nil {
		return nil, err
	}
	var accounts []models.Account
	db.Where("is_group = ? AND type IN ?", false, []string{"income", "expense"}).Order("code asc").Find(&accounts)

	rep := &ProfitLossReport{From: from, To: to}
	for _, a := range accounts {
		s := sums[a.ID]
		if s.Debit == 0 && s.Credit == 0 {
			continue
		}
		bal := normalBalance(a.Type, s.Debit, s.Credit)
		row := AccountBalance{AccountID: a.ID, Code: a.Code, Name: a.Name, Type: a.Type, Balance: bal}
		if a.Type == "income" {
			rep.Income.Rows = append(rep.Income.Rows, row)
			rep.Income.Total += bal
		} else {
			rep.Expense.Rows = append(rep.Expense.Rows, row)
			rep.Expense.Total += bal
		}
	}
	rep.Income.Total = round2(rep.Income.Total)
	rep.Expense.Total = round2(rep.Expense.Total)
	rep.NetProfit = round2(rep.Income.Total - rep.Expense.Total)
	return rep, nil
}

// PartnerBalance — receivable/payable per partner from AR/AP accounts.
type PartnerBalanceRow struct {
	PartnerID   uint    `json:"partner_id"`
	Name        string  `json:"name"`
	Receivable  float64 `json:"receivable"`
	Payable     float64 `json:"payable"`
	Net         float64 `json:"net"`
}

func PartnerBalances(db *gorm.DB) ([]PartnerBalanceRow, error) {
	type row struct {
		PartnerID uint
		Type      string
		Debit     float64
		Credit    float64
	}
	var rows []row
	db.Table("journal_lines as jl").
		Select("jl.partner_id, a.system_key as type, COALESCE(SUM(jl.debit),0) as debit, COALESCE(SUM(jl.credit),0) as credit").
		Joins("JOIN accounts a ON a.id = jl.account_id").
		Where("jl.posted = ? AND jl.partner_id IS NOT NULL AND a.system_key IN ?", true, []string{"ar", "ap"}).
		Group("jl.partner_id, a.system_key").
		Scan(&rows)

	agg := map[uint]*PartnerBalanceRow{}
	for _, r := range rows {
		if _, ok := agg[r.PartnerID]; !ok {
			agg[r.PartnerID] = &PartnerBalanceRow{PartnerID: r.PartnerID}
		}
		if r.Type == "ar" {
			agg[r.PartnerID].Receivable += round2(r.Debit - r.Credit)
		} else {
			agg[r.PartnerID].Payable += round2(r.Credit - r.Debit)
		}
	}
	var partners []models.Partner
	db.Find(&partners)
	names := map[uint]string{}
	for _, p := range partners {
		names[p.ID] = p.Name
	}
	out := make([]PartnerBalanceRow, 0, len(agg))
	for id, v := range agg {
		v.Receivable = round2(v.Receivable)
		v.Payable = round2(v.Payable)
		v.Net = round2(v.Receivable - v.Payable)
		v.Name = names[id]
		out = append(out, *v)
	}
	return out, nil
}

// StockLevel — on-hand quantity and value per product.
type StockLevelRow struct {
	ProductID uint    `json:"product_id"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	Quantity  float64 `json:"quantity"`
	Value     float64 `json:"value"`
}

func StockLevels(db *gorm.DB) ([]StockLevelRow, error) {
	type row struct {
		ProductID uint
		Quantity  float64
		Value     float64
	}
	var rows []row
	db.Table("stock_moves").
		Select("product_id, COALESCE(SUM(quantity),0) as quantity, COALESCE(SUM(quantity*unit_cost),0) as value").
		Group("product_id").Scan(&rows)
	var products []models.Product
	db.Find(&products)
	pm := map[uint]models.Product{}
	for _, p := range products {
		pm[p.ID] = p
	}
	out := make([]StockLevelRow, 0, len(rows))
	for _, r := range rows {
		p := pm[r.ProductID]
		out = append(out, StockLevelRow{
			ProductID: r.ProductID, Code: p.Code, Name: p.Name, Unit: p.Unit,
			Quantity: round2(r.Quantity), Value: round2(r.Value),
		})
	}
	return out, nil
}

// VATReport — ƏDV (VAT) declaration for a period.
type VATReport struct {
	From         *time.Time `json:"from"`
	To           *time.Time `json:"to"`
	SalesBase    float64    `json:"sales_base"`    // vergitutma bazası (satış)
	OutputVAT    float64    `json:"output_vat"`    // hesablanmış ƏDV (satışdan)
	PurchaseBase float64    `json:"purchase_base"` // vergitutma bazası (alış)
	InputVAT     float64    `json:"input_vat"`     // əvəzləşdirilən ƏDV (alışdan)
	Payable      float64    `json:"payable"`       // ödəniləcək (+) / gələcəyə keçən (-)
}

// VATDeclaration sums VAT from posted sales/purchase invoices in the period.
// Output VAT (from sales) minus input VAT (from purchases) = amount payable.
func VATDeclaration(db *gorm.DB, from, to *time.Time) (*VATReport, error) {
	type agg struct {
		Base float64
		Tax  float64
	}
	sum := func(typ string) agg {
		var a agg
		q := db.Model(&models.Document{}).
			Select("COALESCE(SUM(subtotal),0) as base, COALESCE(SUM(tax_total),0) as tax").
			Where("type = ? AND status IN ?", typ, []string{"posted", "paid"})
		if from != nil {
			q = q.Where("date >= ?", *from)
		}
		if to != nil {
			q = q.Where("date <= ?", *to)
		}
		q.Scan(&a)
		return a
	}
	s := sum("sales_invoice")
	p := sum("purchase_invoice")
	return &VATReport{
		From: from, To: to,
		SalesBase: round2(s.Base), OutputVAT: round2(s.Tax),
		PurchaseBase: round2(p.Base), InputVAT: round2(p.Tax),
		Payable: round2(s.Tax - p.Tax),
	}, nil
}

// Dashboard — headline KPIs for the home screen.
type Dashboard struct {
	Cash            float64 `json:"cash"`
	Bank            float64 `json:"bank"`
	Receivable      float64 `json:"receivable"`
	Payable         float64 `json:"payable"`
	IncomeThisMonth float64 `json:"income_this_month"`
	ExpenseThisMonth float64 `json:"expense_this_month"`
	NetThisMonth    float64 `json:"net_this_month"`
	VatPayable      float64 `json:"vat_payable"`
	Partners        int64   `json:"partners"`
	Products        int64   `json:"products"`
	OpenInvoices    int64   `json:"open_invoices"`
	StockValue      float64 `json:"stock_value"`
}

func BuildDashboard(db *gorm.DB, now time.Time) (*Dashboard, error) {
	d := &Dashboard{}
	sysBal := func(key string) float64 {
		var acc models.Account
		if err := db.Where("system_key = ?", key).First(&acc).Error; err != nil {
			return 0
		}
		sums, _ := accountSums(db, nil, nil)
		s := sums[acc.ID]
		return normalBalance(acc.Type, s.Debit, s.Credit)
	}
	d.Cash = sysBal("cash")
	d.Bank = sysBal("bank")
	d.Receivable = sysBal("ar")
	d.Payable = sysBal("ap")
	vatOut := sysBal("vat_out")
	vatIn := sysBal("vat_in")
	d.VatPayable = round2(vatOut - vatIn)

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	pl, _ := ProfitLoss(db, &monthStart, &now)
	if pl != nil {
		d.IncomeThisMonth = pl.Income.Total
		d.ExpenseThisMonth = pl.Expense.Total
		d.NetThisMonth = pl.NetProfit
	}
	db.Model(&models.Partner{}).Count(&d.Partners)
	db.Model(&models.Product{}).Count(&d.Products)
	db.Model(&models.Document{}).
		Where("type IN ? AND status IN ?", []string{"sales_invoice", "purchase_invoice"}, []string{"posted"}).
		Where("paid_total < total").Count(&d.OpenInvoices)
	levels, _ := StockLevels(db)
	for _, l := range levels {
		d.StockValue += l.Value
	}
	d.StockValue = round2(d.StockValue)
	return d, nil
}
