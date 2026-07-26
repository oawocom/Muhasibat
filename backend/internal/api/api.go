package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"oawo-muhasibat/internal/engine"
	"oawo-muhasibat/internal/models"
	"oawo-muhasibat/internal/tenant"
)

type Server struct {
	DB  *gorm.DB        // platform (control-plane) database
	Mgr *tenant.Manager // per-company database manager
}

func New(db *gorm.DB, mgr *tenant.Manager) *Server {
	return &Server{DB: db, Mgr: mgr}
}

// ModuleCatalogHandler returns the fixed catalog of toggleable modules.
func (s *Server) ModuleCatalogHandler(c *gin.Context) {
	c.JSON(200, models.ModuleCatalog())
}

// RegisterRoutes wires the full REST API under /api.
func (s *Server) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")

	api.POST("/auth/login", s.Login)

	// Platform-scoped (authenticated, no company context required).
	auth := api.Group("")
	auth.Use(s.AuthMiddleware())
	{
		auth.GET("/auth/me", s.Me)
		auth.POST("/auth/change-password", s.ChangePassword)
		auth.GET("/module-catalog", s.ModuleCatalogHandler)
		auth.GET("/roles", s.Roles)

		// Tenants / subscriptions (superadmin only — enforced in handlers)
		auth.GET("/tenants", s.ListTenants)
		auth.POST("/tenants", s.CreateTenant)
		auth.GET("/tenants/:id", s.GetTenant)
		auth.PUT("/tenants/:id", s.UpdateTenant)

		// Tenant admin: self-service subscription (own tenant only)
		auth.GET("/my-tenant", s.GetMyTenant)
		auth.PUT("/my-tenant/modules", s.UpdateMyTenantModules)

		// Companies & members (control-plane)
		auth.GET("/companies", s.ListCompanies)
		auth.POST("/companies", s.CreateCompany)
		auth.PUT("/companies/:id", s.UpdateCompany)
		auth.GET("/companies/:id/users", s.ListCompanyUsers)
		auth.POST("/companies/:id/users", s.AddCompanyUser)
		auth.PUT("/companies/:id/users/:mid", s.UpdateCompanyUser)
		auth.DELETE("/companies/:id/users/:mid", s.RemoveCompanyUser)
	}

	// Tenant-scoped (authenticated + active company via X-Company-ID).
	t := api.Group("")
	t.Use(s.AuthMiddleware(), s.TenantMiddleware())
	{
		// Chart of accounts
		t.GET("/accounts", s.ListAccounts)
		t.GET("/accounts/:id", s.GetAccount)
		t.POST("/accounts", s.CreateAccount)
		t.PUT("/accounts/:id", s.UpdateAccount)
		t.DELETE("/accounts/:id", s.DeleteAccount)

		// Currencies
		t.GET("/currencies", s.ListCurrencies)
		t.POST("/currencies", s.CreateCurrency)
		t.PUT("/currencies/:id", s.UpdateCurrency)
		t.DELETE("/currencies/:id", s.DeleteCurrency)

		// Tax rates
		t.GET("/tax-rates", s.ListTaxRates)
		t.POST("/tax-rates", s.CreateTaxRate)
		t.PUT("/tax-rates/:id", s.UpdateTaxRate)
		t.DELETE("/tax-rates/:id", s.DeleteTaxRate)

		// Warehouses
		t.GET("/warehouses", s.ListWarehouses)
		t.POST("/warehouses", s.CreateWarehouse)
		t.PUT("/warehouses/:id", s.UpdateWarehouse)
		t.DELETE("/warehouses/:id", s.DeleteWarehouse)

		// Partners
		t.GET("/partners", s.ListPartners)
		t.GET("/partners/:id", s.GetPartner)
		t.POST("/partners", s.CreatePartner)
		t.PUT("/partners/:id", s.UpdatePartner)
		t.DELETE("/partners/:id", s.DeletePartner)

		// Products
		t.GET("/products", s.ListProducts)
		t.GET("/products/:id", s.GetProduct)
		t.POST("/products", s.CreateProduct)
		t.PUT("/products/:id", s.UpdateProduct)
		t.DELETE("/products/:id", s.DeleteProduct)

		// Journal entries
		t.GET("/journal", s.ListJournal)
		t.GET("/journal/:id", s.GetJournal)
		t.POST("/journal", s.CreateJournal)
		t.PUT("/journal/:id", s.UpdateJournal)
		t.POST("/journal/:id/post", s.PostJournal)
		t.POST("/journal/:id/unpost", s.UnpostJournal)
		t.DELETE("/journal/:id", s.DeleteJournal)

		// Documents (invoices, payments, receipts)
		t.GET("/documents", s.ListDocuments)
		t.GET("/documents/:id", s.GetDocument)
		t.POST("/documents", s.CreateDocument)
		t.PUT("/documents/:id", s.UpdateDocument)
		t.POST("/documents/:id/post", s.PostDocument)
		t.DELETE("/documents/:id", s.DeleteDocument)

		// Fixed assets + depreciation
		t.GET("/fixed-assets", s.ListFixedAssets)
		t.GET("/fixed-assets/:id", s.GetFixedAsset)
		t.POST("/fixed-assets", s.CreateFixedAsset)
		t.PUT("/fixed-assets/:id", s.UpdateFixedAsset)
		t.DELETE("/fixed-assets/:id", s.DeleteFixedAsset)
		t.GET("/fixed-assets/:id/depreciation", s.ListDepreciation)
		t.POST("/depreciation/run", s.RunDepreciationHandler)

		// Payroll
		t.GET("/employees", s.ListEmployees)
		t.GET("/employees/:id", s.GetEmployee)
		t.POST("/employees", s.CreateEmployee)
		t.PUT("/employees/:id", s.UpdateEmployee)
		t.DELETE("/employees/:id", s.DeleteEmployee)
		t.GET("/payroll/config", s.GetPayrollConfig)
		t.PUT("/payroll/config", s.UpdatePayrollConfig)
		t.GET("/payroll/runs", s.ListPayrollRuns)
		t.POST("/payroll/runs", s.CreatePayrollRun)
		t.GET("/payroll/runs/:id", s.GetPayrollRun)
		t.POST("/payroll/runs/:id/post", s.PostPayrollRun)
		t.DELETE("/payroll/runs/:id", s.DeletePayrollRun)

		// POS (kassa aparatı)
		t.GET("/pos/registers", s.ListRegisters)
		t.POST("/pos/registers", s.CreateRegister)
		t.PUT("/pos/registers/:id", s.UpdateRegister)
		t.DELETE("/pos/registers/:id", s.DeleteRegister)
		t.GET("/pos/registers/:id/open-session", s.GetOpenSession)
		t.POST("/pos/open-session", s.OpenPosSession)
		t.GET("/pos/sessions", s.ListPosSessions)
		t.GET("/pos/sessions/:id", s.GetPosSession)
		t.POST("/pos/sessions/:id/close", s.ClosePosSession)
		t.POST("/pos/sale", s.CompletePosSale)

		// e-Qaimə (e-invoice)
		t.GET("/einvoice/config", s.GetEInvoiceConfig)
		t.PUT("/einvoice/config", s.UpdateEInvoiceConfig)
		t.GET("/einvoices", s.ListEInvoices)
		t.GET("/einvoices/eligible", s.ListEligibleInvoices)
		t.POST("/einvoices", s.CreateEInvoice)
		t.GET("/einvoices/:id", s.GetEInvoice)
		t.PUT("/einvoices/:id", s.UpdateEInvoice)
		t.POST("/einvoices/:id/send", s.SendEInvoice)

		// Reports
		t.GET("/reports/trial-balance", s.TrialBalance)
		t.GET("/reports/ledger/:accountId", s.Ledger)
		t.GET("/reports/balance-sheet", s.BalanceSheet)
		t.GET("/reports/profit-loss", s.ProfitLoss)
		t.GET("/reports/vat", s.VATReport)
		t.GET("/reports/partner-balances", s.PartnerBalancesReport)
		t.GET("/reports/stock", s.StockReport)
		t.GET("/dashboard", s.DashboardHandler)

		// Settings (includes enabled_modules)
		t.GET("/settings", s.GetSettings)
		t.PUT("/settings", s.UpdateSettings)
	}
}

// ---------- helpers ----------

func (s *Server) bindID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış ID"})
		return 0, false
	}
	return uint(id), true
}

func parseDate(v string) *time.Time {
	if v == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, v); err == nil {
			return &t
		}
	}
	return nil
}

// endOfDay makes a date filter inclusive of the whole day.
func endOfDay(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	e := t.Add(24*time.Hour - time.Nanosecond)
	return &e
}

// ==================== ACCOUNTS ====================

func (s *Server) ListAccounts(c *gin.Context) {
	var items []models.Account
	q := tdb(c).Order("code asc")
	if t := c.Query("type"); t != "" {
		q = q.Where("type = ?", t)
	}
	if c.Query("postable") == "1" {
		q = q.Where("is_group = ?", false)
	}
	q.Find(&items)
	c.JSON(200, items)
}

func (s *Server) GetAccount(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var a models.Account
	if err := tdb(c).First(&a, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Hesab tapılmadı"})
		return
	}
	c.JSON(200, a)
}

func (s *Server) CreateAccount(c *gin.Context) {
	var a models.Account
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	if err := tdb(c).Create(&a).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Yaradıla bilmədi"})
		return
	}
	c.JSON(201, a)
}

func (s *Server) UpdateAccount(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var a models.Account
	if err := tdb(c).First(&a, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "system_key") // protect system accounts
	tdb(c).Model(&a).Updates(updates)
	tdb(c).First(&a, id)
	c.JSON(200, a)
}

func (s *Server) DeleteAccount(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var a models.Account
	if err := tdb(c).First(&a, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	if a.SystemKey != "" {
		c.JSON(400, gin.H{"detail": "Sistem hesabı silinə bilməz"})
		return
	}
	var used int64
	tdb(c).Model(&models.JournalLine{}).Where("account_id = ?", id).Count(&used)
	if used > 0 {
		c.JSON(400, gin.H{"detail": "Bu hesabda yazılışlar var, silinə bilməz"})
		return
	}
	tdb(c).Delete(&models.Account{}, id)
	c.JSON(200, gin.H{"message": "Silindi"})
}

// ==================== CURRENCIES ====================

func (s *Server) ListCurrencies(c *gin.Context) {
	var items []models.Currency
	tdb(c).Order("is_base desc, code asc").Find(&items)
	c.JSON(200, items)
}
func (s *Server) CreateCurrency(c *gin.Context) { s.genericCreate(c, &models.Currency{}) }
func (s *Server) UpdateCurrency(c *gin.Context) { s.genericUpdate(c, &models.Currency{}) }
func (s *Server) DeleteCurrency(c *gin.Context) { s.genericDelete(c, &models.Currency{}) }

// ==================== TAX RATES ====================

func (s *Server) ListTaxRates(c *gin.Context) {
	var items []models.TaxRate
	tdb(c).Order("is_default desc, rate desc").Find(&items)
	c.JSON(200, items)
}
func (s *Server) CreateTaxRate(c *gin.Context) { s.genericCreate(c, &models.TaxRate{}) }
func (s *Server) UpdateTaxRate(c *gin.Context) { s.genericUpdate(c, &models.TaxRate{}) }
func (s *Server) DeleteTaxRate(c *gin.Context) { s.genericDelete(c, &models.TaxRate{}) }

// ==================== WAREHOUSES ====================

func (s *Server) ListWarehouses(c *gin.Context) {
	var items []models.Warehouse
	tdb(c).Order("is_default desc, name asc").Find(&items)
	c.JSON(200, items)
}
func (s *Server) CreateWarehouse(c *gin.Context) { s.genericCreate(c, &models.Warehouse{}) }
func (s *Server) UpdateWarehouse(c *gin.Context) { s.genericUpdate(c, &models.Warehouse{}) }
func (s *Server) DeleteWarehouse(c *gin.Context) { s.genericDelete(c, &models.Warehouse{}) }

// ==================== PARTNERS ====================

func (s *Server) ListPartners(c *gin.Context) {
	var items []models.Partner
	q := tdb(c).Order("name asc")
	if t := c.Query("type"); t != "" {
		q = q.Where("type = ? OR type = ?", t, "both")
	}
	if search := c.Query("q"); search != "" {
		like := "%" + search + "%"
		q = q.Where("name ILIKE ? OR tax_id ILIKE ? OR phone ILIKE ?", like, like, like)
	}
	q.Find(&items)
	c.JSON(200, items)
}
func (s *Server) GetPartner(c *gin.Context)    { s.genericGet(c, &models.Partner{}) }
func (s *Server) CreatePartner(c *gin.Context) { s.genericCreate(c, &models.Partner{}) }
func (s *Server) UpdatePartner(c *gin.Context) { s.genericUpdate(c, &models.Partner{}) }
func (s *Server) DeletePartner(c *gin.Context) { s.genericDelete(c, &models.Partner{}) }

// ==================== PRODUCTS ====================

func (s *Server) ListProducts(c *gin.Context) {
	var items []models.Product
	q := tdb(c).Order("name asc")
	if search := c.Query("q"); search != "" {
		like := "%" + search + "%"
		q = q.Where("name ILIKE ? OR code ILIKE ? OR barcode ILIKE ?", like, like, like)
	}
	q.Find(&items)
	c.JSON(200, items)
}
func (s *Server) GetProduct(c *gin.Context)    { s.genericGet(c, &models.Product{}) }
func (s *Server) CreateProduct(c *gin.Context) { s.genericCreate(c, &models.Product{}) }
func (s *Server) UpdateProduct(c *gin.Context) { s.genericUpdate(c, &models.Product{}) }
func (s *Server) DeleteProduct(c *gin.Context) { s.genericDelete(c, &models.Product{}) }

// ==================== JOURNAL ====================

func (s *Server) ListJournal(c *gin.Context) {
	var items []models.JournalEntry
	q := tdb(c).Preload("Lines").Order("date desc, id desc")
	if st := c.Query("status"); st != "" {
		q = q.Where("status = ?", st)
	}
	if from := parseDate(c.Query("from")); from != nil {
		q = q.Where("date >= ?", *from)
	}
	if to := endOfDay(parseDate(c.Query("to"))); to != nil {
		q = q.Where("date <= ?", *to)
	}
	q.Limit(500).Find(&items)
	c.JSON(200, items)
}

func (s *Server) GetJournal(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var e models.JournalEntry
	if err := tdb(c).Preload("Lines").First(&e, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Yazılış tapılmadı"})
		return
	}
	c.JSON(200, e)
}

func (s *Server) CreateJournal(c *gin.Context) {
	var e models.JournalEntry
	if err := c.ShouldBindJSON(&e); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	if e.Date.IsZero() {
		e.Date = models.Date{Time: time.Now()}
	}
	if e.Number == "" {
		e.Number, _ = engine.NextNumber(tdb(c), "journal", "J-")
	}
	e.Status = "draft"
	e.ID = 0
	for i := range e.Lines {
		e.Lines[i].ID = 0
		e.Lines[i].Posted = false
	}
	if err := tdb(c).Create(&e).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Yaradıla bilmədi: " + err.Error()})
		return
	}
	if c.Query("post") == "1" {
		if _, err := engine.PostEntry(tdb(c), e.ID); err != nil {
			c.JSON(400, gin.H{"detail": err.Error(), "id": e.ID})
			return
		}
	}
	tdb(c).Preload("Lines").First(&e, e.ID)
	c.JSON(201, e)
}

func (s *Server) UpdateJournal(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var e models.JournalEntry
	if err := tdb(c).First(&e, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	if e.Status == "posted" {
		c.JSON(400, gin.H{"detail": "Təsdiqlənmiş yazılış redaktə edilə bilməz. Əvvəl geri qaytarın."})
		return
	}
	var in models.JournalEntry
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	err := tdb(c).Transaction(func(tx *gorm.DB) error {
		e.Date = in.Date
		e.Description = in.Description
		e.Reference = in.Reference
		if err := tx.Save(&e).Error; err != nil {
			return err
		}
		tx.Where("entry_id = ?", e.ID).Delete(&models.JournalLine{})
		for i := range in.Lines {
			in.Lines[i].ID = 0
			in.Lines[i].EntryID = e.ID
			in.Lines[i].Posted = false
			if err := tx.Create(&in.Lines[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	tdb(c).Preload("Lines").First(&e, id)
	c.JSON(200, e)
}

func (s *Server) PostJournal(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	e, err := engine.PostEntry(tdb(c), id)
	if err != nil {
		c.JSON(400, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, e)
}

func (s *Server) UnpostJournal(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	if err := engine.UnpostEntry(tdb(c), id); err != nil {
		c.JSON(400, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Geri qaytarıldı"})
}

func (s *Server) DeleteJournal(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var e models.JournalEntry
	if err := tdb(c).First(&e, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	if e.Status == "posted" {
		c.JSON(400, gin.H{"detail": "Təsdiqlənmiş yazılış silinə bilməz"})
		return
	}
	tdb(c).Where("entry_id = ?", id).Delete(&models.JournalLine{})
	tdb(c).Delete(&models.JournalEntry{}, id)
	c.JSON(200, gin.H{"message": "Silindi"})
}

// ==================== DOCUMENTS ====================

func (s *Server) ListDocuments(c *gin.Context) {
	var items []models.Document
	q := tdb(c).Preload("Lines").Order("date desc, id desc")
	if t := c.Query("type"); t != "" {
		q = q.Where("type = ?", t)
	}
	if st := c.Query("status"); st != "" {
		q = q.Where("status = ?", st)
	}
	q.Limit(500).Find(&items)
	c.JSON(200, items)
}

func (s *Server) GetDocument(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var d models.Document
	if err := tdb(c).Preload("Lines").First(&d, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Sənəd tapılmadı"})
		return
	}
	c.JSON(200, d)
}

func seqKeyForType(t string) (string, string) {
	switch t {
	case "sales_invoice":
		return "sales_invoice", "SF-"
	case "purchase_invoice":
		return "purchase_invoice", "AF-"
	case "payment":
		return "payment", "ÖD-"
	case "receipt":
		return "receipt", "MD-"
	}
	return "journal", "J-"
}

func (s *Server) CreateDocument(c *gin.Context) {
	var d models.Document
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	if d.Type == "" {
		c.JSON(400, gin.H{"detail": "Sənəd tipi tələb olunur"})
		return
	}
	if d.Date.IsZero() {
		d.Date = models.Date{Time: time.Now()}
	}
	if d.FxRate == 0 {
		d.FxRate = 1
	}
	if d.Number == "" {
		key, prefix := seqKeyForType(d.Type)
		d.Number, _ = engine.NextNumber(tdb(c), key, prefix)
	}
	d.Status = "draft"
	d.ID = 0
	for i := range d.Lines {
		d.Lines[i].ID = 0
	}
	if err := tdb(c).Create(&d).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Yaradıla bilmədi: " + err.Error()})
		return
	}
	// Recompute totals from lines and persist.
	tdb(c).Preload("Lines").First(&d, d.ID)
	s.recomputeAndSave(c, &d)
	if c.Query("post") == "1" {
		if _, err := engine.PostDocument(tdb(c), d.ID); err != nil {
			c.JSON(400, gin.H{"detail": err.Error(), "id": d.ID})
			return
		}
	}
	tdb(c).Preload("Lines").First(&d, d.ID)
	c.JSON(201, d)
}

func (s *Server) UpdateDocument(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var d models.Document
	if err := tdb(c).First(&d, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	if d.Status == "posted" || d.Status == "paid" {
		c.JSON(400, gin.H{"detail": "Təsdiqlənmiş sənəd redaktə edilə bilməz"})
		return
	}
	var in models.Document
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	err := tdb(c).Transaction(func(tx *gorm.DB) error {
		d.Date = in.Date
		d.DueDate = in.DueDate
		d.PartnerID = in.PartnerID
		d.WarehouseID = in.WarehouseID
		d.CurrencyID = in.CurrencyID
		d.CashAccountID = in.CashAccountID
		d.Notes = in.Notes
		if in.FxRate > 0 {
			d.FxRate = in.FxRate
		}
		if err := tx.Save(&d).Error; err != nil {
			return err
		}
		tx.Where("document_id = ?", d.ID).Delete(&models.DocumentLine{})
		for i := range in.Lines {
			in.Lines[i].ID = 0
			in.Lines[i].DocumentID = d.ID
			if err := tx.Create(&in.Lines[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	tdb(c).Preload("Lines").First(&d, id)
	s.recomputeAndSave(c, &d)
	tdb(c).Preload("Lines").First(&d, id)
	c.JSON(200, d)
}

// recomputeAndSave recalculates line/document totals and persists them.
func (s *Server) recomputeAndSave(c *gin.Context, d *models.Document) {
	var sub, tax float64
	for i := range d.Lines {
		l := &d.Lines[i]
		net := round2(l.Quantity * l.UnitPrice)
		l.LineTotal = net
		l.TaxAmount = round2(net * l.TaxRate / 100)
		sub += net
		tax += l.TaxAmount
		tdb(c).Model(&models.DocumentLine{}).Where("id = ?", l.ID).
			Updates(map[string]interface{}{"line_total": l.LineTotal, "tax_amount": l.TaxAmount})
	}
	d.Subtotal = round2(sub)
	d.TaxTotal = round2(tax)
	d.Total = round2(sub + tax)
	tdb(c).Model(&models.Document{}).Where("id = ?", d.ID).
		Updates(map[string]interface{}{"subtotal": d.Subtotal, "tax_total": d.TaxTotal, "total": d.Total})
}

func round2(v float64) float64 {
	return float64(int64(v*100+sign(v)*0.5)) / 100
}
func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

func (s *Server) PostDocument(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	d, err := engine.PostDocument(tdb(c), id)
	if err != nil {
		c.JSON(400, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, d)
}

func (s *Server) DeleteDocument(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var d models.Document
	if err := tdb(c).First(&d, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	if d.Status == "posted" || d.Status == "paid" {
		c.JSON(400, gin.H{"detail": "Təsdiqlənmiş sənəd silinə bilməz"})
		return
	}
	tdb(c).Where("document_id = ?", id).Delete(&models.DocumentLine{})
	tdb(c).Delete(&models.Document{}, id)
	c.JSON(200, gin.H{"message": "Silindi"})
}

// ==================== REPORTS ====================

func (s *Server) TrialBalance(c *gin.Context) {
	to := endOfDay(parseDate(c.Query("to")))
	rep, err := engine.TrialBalance(tdb(c), to)
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, rep)
}

func (s *Server) Ledger(c *gin.Context) {
	accID, err := strconv.ParseUint(c.Param("accountId"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış hesab ID"})
		return
	}
	from := parseDate(c.Query("from"))
	to := endOfDay(parseDate(c.Query("to")))
	rep, err := engine.Ledger(tdb(c), uint(accID), from, to)
	if err != nil {
		c.JSON(404, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, rep)
}

func (s *Server) BalanceSheet(c *gin.Context) {
	to := endOfDay(parseDate(c.Query("to")))
	rep, err := engine.BalanceSheet(tdb(c), to)
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, rep)
}

func (s *Server) ProfitLoss(c *gin.Context) {
	from := parseDate(c.Query("from"))
	to := endOfDay(parseDate(c.Query("to")))
	rep, err := engine.ProfitLoss(tdb(c), from, to)
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, rep)
}

func (s *Server) VATReport(c *gin.Context) {
	from := parseDate(c.Query("from"))
	to := endOfDay(parseDate(c.Query("to")))
	rep, err := engine.VATDeclaration(tdb(c), from, to)
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, rep)
}

func (s *Server) PartnerBalancesReport(c *gin.Context) {
	rep, err := engine.PartnerBalances(tdb(c))
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, rep)
}

func (s *Server) StockReport(c *gin.Context) {
	rep, err := engine.StockLevels(tdb(c))
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, rep)
}

func (s *Server) DashboardHandler(c *gin.Context) {
	d, err := engine.BuildDashboard(tdb(c), time.Now())
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, d)
}

// ==================== SETTINGS ====================

func (s *Server) GetSettings(c *gin.Context) {
	var items []models.Setting
	tdb(c).Find(&items)
	out := map[string]string{}
	for _, it := range items {
		out[it.Key] = it.Value
	}
	c.JSON(200, out)
}

func (s *Server) UpdateSettings(c *gin.Context) {
	var in map[string]string
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	for k, v := range in {
		if k == "seeded" {
			continue
		}
		tdb(c).Save(&models.Setting{Key: k, Value: v, UpdatedAt: time.Now()})
	}
	c.JSON(200, gin.H{"message": "Yadda saxlanıldı"})
}

// ==================== GENERIC CRUD ====================
// Small reflection-free helpers for the simple lookup entities.

func (s *Server) genericGet(c *gin.Context, model interface{}) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	if err := tdb(c).First(model, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	c.JSON(200, model)
}

func (s *Server) genericCreate(c *gin.Context, model interface{}) {
	if err := c.ShouldBindJSON(model); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	if err := tdb(c).Create(model).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Yaradıla bilmədi: " + err.Error()})
		return
	}
	c.JSON(201, model)
}

func (s *Server) genericUpdate(c *gin.Context, model interface{}) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	if err := tdb(c).First(model, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)
	delete(updates, "id")
	delete(updates, "created_at")
	tdb(c).Model(model).Updates(updates)
	tdb(c).First(model, id)
	c.JSON(200, model)
}

func (s *Server) genericDelete(c *gin.Context, model interface{}) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	if err := tdb(c).Delete(model, id).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Silinə bilmədi"})
		return
	}
	c.JSON(200, gin.H{"message": "Silindi"})
}

var _ = http.StatusOK
