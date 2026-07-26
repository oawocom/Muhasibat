package demo

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"oawo-muhasibat/internal/engine"
	"oawo-muhasibat/internal/models"
	"oawo-muhasibat/internal/tenant"
)

// Seed creates a demo tenant with a fully populated company on first boot,
// so there is a realistic login + data to explore. Controlled by SEED_DEMO
// (default on). No-op if the demo tenant already exists.
func Seed(pdb *gorm.DB, mgr *tenant.Manager) error {
	if os.Getenv("SEED_DEMO") == "0" {
		return nil
	}
	email := envOr("DEMO_EMAIL", "demo@oawo.com")
	var existing int64
	pdb.Model(&models.User{}).Where("email = ?", email).Count(&existing)
	if existing > 0 {
		return nil
	}

	// Tenant + its admin.
	allMods := models.DefaultEnabledModules()
	modules, _ := json.Marshal(allMods)
	t := models.Tenant{Name: "Demo MMC", Plan: "pro", EnabledModules: string(modules),
		SubscriptionAmount: priceOf(allMods), BillingCycle: "monthly", Enabled: true, ContactEmail: email}
	if err := pdb.Create(&t).Error; err != nil {
		return err
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(envOr("DEMO_PASSWORD", "demo123")), bcrypt.DefaultCost)
	admin := models.User{Email: email, Name: "Demo Admin", Password: string(hash),
		TenantID: &t.ID, IsTenantAdmin: true, Enabled: true}
	if err := pdb.Create(&admin).Error; err != nil {
		return err
	}

	// Company + its database.
	company := models.Company{TenantID: t.ID, Name: "Demo Ticarət MMC", TaxID: "1234567890", Enabled: true}
	if err := pdb.Create(&company).Error; err != nil {
		return err
	}
	company.DBName = tenant.DBNameFor(company.ID)
	pdb.Save(&company)
	db, err := mgr.Provision(company.ID, company.DBName)
	if err != nil {
		return err
	}
	pdb.Model(&models.Company{}).Where("id = ?", company.ID).Update("provisioned", true)

	if err := populate(db); err != nil {
		log.Printf("demo: populate xətası: %v", err)
	}
	log.Printf("🎬 Demo tenant hazırdır: %s / (parol: demo123) — şirkət: %s", email, company.Name)

	// A couple more test tenants with different module sets → different amounts,
	// so the superadmin tenant list is realistic for a presentation.
	extraTenant(pdb, mgr, "Alfa Ticarət MMC", "alfa@oawo.com", []string{"partners", "products", "sales", "money", "reports"}, true)
	extraTenant(pdb, mgr, "Beta Xidmət MMC", "beta@oawo.com", []string{"partners", "sales", "reports", "einvoice"}, false)
	return nil
}

// extraTenant creates a lighter test tenant (admin + one company, a little data).
func extraTenant(pdb *gorm.DB, mgr *tenant.Manager, name, email string, mods []string, withData bool) {
	var n int64
	pdb.Model(&models.User{}).Where("email = ?", email).Count(&n)
	if n > 0 {
		return
	}
	full := ensureCore(mods)
	mj, _ := json.Marshal(full)
	t := models.Tenant{Name: name, Plan: "standard", EnabledModules: string(mj),
		SubscriptionAmount: priceOf(full), BillingCycle: "monthly", Enabled: true, ContactEmail: email}
	if pdb.Create(&t).Error != nil {
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
	pdb.Create(&models.User{Email: email, Name: name + " admin", Password: string(hash), TenantID: &t.ID, IsTenantAdmin: true, Enabled: true})
	company := models.Company{TenantID: t.ID, Name: name, Enabled: true}
	pdb.Create(&company)
	company.DBName = tenant.DBNameFor(company.ID)
	pdb.Save(&company)
	db, err := mgr.Provision(company.ID, company.DBName)
	if err != nil {
		return
	}
	pdb.Model(&models.Company{}).Where("id = ?", company.ID).Update("provisioned", true)
	if withData {
		var vat models.TaxRate
		db.Where("is_default = ?", true).First(&vat)
		vid := vat.ID
		cust := models.Partner{Name: "Müştəri A", Type: "customer", TaxID: "1400000001", Enabled: true}
		db.Create(&cust)
		p := models.Product{Name: "Xidmət paketi", Code: "SVC", Type: "service", Unit: "ədəd", SalePrice: 500, TaxRateID: &vid, Enabled: true}
		db.Create(&p)
		var wh models.Warehouse
		db.First(&wh)
		postDoc(db, models.Document{Type: "sales_invoice", PartnerID: &cust.ID, WarehouseID: &wh.ID, Date: models.Date{Time: time.Now().AddDate(0, 0, -5)}, FxRate: 1,
			Lines: []models.DocumentLine{{ProductID: &p.ID, Description: p.Name, Quantity: 2, UnitPrice: 500, TaxRate: 18}}})
	}
	log.Printf("🎬 Test tenant: %s / demo123", email)
}

// ensureCore adds core module keys.
func ensureCore(mods []string) []string {
	set := map[string]bool{}
	for _, m := range mods {
		set[m] = true
	}
	for _, d := range models.ModuleCatalog() {
		if d.Core {
			set[d.Key] = true
		}
	}
	out := []string{}
	for _, d := range models.ModuleCatalog() {
		if set[d.Key] {
			out = append(out, d.Key)
		}
	}
	return out
}

// priceOf sums non-core module prices (mirrors api.ComputeSubscription).
func priceOf(mods []string) float64 {
	set := map[string]bool{}
	for _, m := range mods {
		set[m] = true
	}
	var total float64
	for _, d := range models.ModuleCatalog() {
		if !d.Core && set[d.Key] {
			total += d.Price
		}
	}
	return total
}

func populate(db *gorm.DB) error {
	now := time.Now()
	d := func(daysAgo int) models.Date { return models.Date{Time: now.AddDate(0, 0, -daysAgo)} }

	// Partners.
	cust1 := models.Partner{Name: "Azərsun Holdinq", Type: "customer", TaxID: "1300161271", Phone: "+994125551122", Enabled: true}
	cust2 := models.Partner{Name: "Bravo Supermarket", Type: "customer", TaxID: "1401234567", Phone: "+994125553344", Enabled: true}
	supp := models.Partner{Name: "Global Təchizat MMC", Type: "supplier", TaxID: "1509876543", Phone: "+994125557788", Enabled: true}
	db.Create(&cust1)
	db.Create(&cust2)
	db.Create(&supp)

	// Products.
	var vat models.TaxRate
	db.Where("is_default = ?", true).First(&vat)
	vatID := vat.ID
	p1 := models.Product{Name: "Noutbuk Dell", Code: "NB-001", Type: "product", Unit: "ədəd", SalePrice: 1500, CostPrice: 1100, TaxRateID: &vatID, TrackStock: true, Enabled: true}
	p2 := models.Product{Name: "Monitor 24\"", Code: "MN-024", Type: "product", Unit: "ədəd", SalePrice: 350, CostPrice: 240, TaxRateID: &vatID, TrackStock: true, Enabled: true}
	p3 := models.Product{Name: "Ofis kreslosu", Code: "KR-100", Type: "product", Unit: "ədəd", SalePrice: 220, CostPrice: 150, TaxRateID: &vatID, TrackStock: true, Enabled: true}
	p4 := models.Product{Name: "IT dəstək xidməti", Code: "SV-IT", Type: "service", Unit: "saat", SalePrice: 40, CostPrice: 0, TaxRateID: &vatID, TrackStock: false, Enabled: true}
	db.Create(&p1)
	db.Create(&p2)
	db.Create(&p3)
	db.Create(&p4)

	var wh models.Warehouse
	db.First(&wh)
	whID := wh.ID

	// Purchase invoice (stock in) — buy 20 laptops, 30 monitors, 25 chairs.
	postDoc(db, models.Document{
		Type: "purchase_invoice", PartnerID: &supp.ID, WarehouseID: &whID, Date: d(25), FxRate: 1,
		Lines: []models.DocumentLine{
			{ProductID: &p1.ID, Description: p1.Name, Quantity: 20, UnitPrice: 1100, TaxRate: 18},
			{ProductID: &p2.ID, Description: p2.Name, Quantity: 30, UnitPrice: 240, TaxRate: 18},
			{ProductID: &p3.ID, Description: p3.Name, Quantity: 25, UnitPrice: 150, TaxRate: 18},
		},
	})

	// Two sales invoices.
	s1 := postDoc(db, models.Document{
		Type: "sales_invoice", PartnerID: &cust1.ID, WarehouseID: &whID, Date: d(12), FxRate: 1,
		Lines: []models.DocumentLine{
			{ProductID: &p1.ID, Description: p1.Name, Quantity: 5, UnitPrice: 1500, TaxRate: 18},
			{ProductID: &p2.ID, Description: p2.Name, Quantity: 8, UnitPrice: 350, TaxRate: 18},
		},
	})
	postDoc(db, models.Document{
		Type: "sales_invoice", PartnerID: &cust2.ID, WarehouseID: &whID, Date: d(6), FxRate: 1,
		Lines: []models.DocumentLine{
			{ProductID: &p3.ID, Description: p3.Name, Quantity: 10, UnitPrice: 220, TaxRate: 18},
			{ProductID: &p4.ID, Description: p4.Name, Quantity: 12, UnitPrice: 40, TaxRate: 18},
		},
	})

	// A customer receipt and a supplier payment.
	cash := accByKey(db, "cash")
	bank := accByKey(db, "bank")
	if s1 != nil {
		postDoc(db, models.Document{Type: "receipt", PartnerID: &cust1.ID, CashAccountID: &cash, Date: d(4), FxRate: 1,
			Lines: []models.DocumentLine{{Description: "Müştəridən nağd mədaxil", Quantity: 1, UnitPrice: 6000, TaxRate: 0}}})
	}
	postDoc(db, models.Document{Type: "payment", PartnerID: &supp.ID, CashAccountID: &bank, Date: d(3), FxRate: 1,
		Lines: []models.DocumentLine{{Description: "Təchizatçıya bank köçürməsi", Quantity: 1, UnitPrice: 8000, TaxRate: 0}}})

	// Opening capital: Dr bank / Cr nizamnamə kapitalı.
	capital := accByCode(db, "301")
	if bank != 0 && capital != 0 {
		e := models.JournalEntry{Number: mustNum(db), Date: d(30), Description: "Nizamnamə kapitalının qoyuluşu", Status: "draft",
			Lines: []models.JournalLine{
				{AccountID: bank, Debit: 20000},
				{AccountID: capital, Credit: 20000},
			}}
		db.Create(&e)
		engine.PostEntry(db, e.ID)
	}

	// Fixed asset + depreciation.
	ppe := accByKey(db, "ppe")
	accum := accByKey(db, "accum_dep")
	depExp := accByKey(db, "dep_expense")
	if ppe != 0 && accum != 0 && depExp != 0 {
		fa := models.FixedAsset{Code: "AV-01", Name: "Xidməti avtomobil", Category: "Nəqliyyat",
			AcquisitionDate: d(180), Cost: 24000, SalvageValue: 0, UsefulLifeMonths: 48, Method: "straight_line",
			AssetAccountID: ppe, AccumAccountID: accum, ExpenseAccountID: depExp, Status: "active", BookValue: 24000}
		db.Create(&fa)
		engine.RunDepreciation(db, now)
	}

	// Employees + one posted payroll run.
	db.Create(&models.Employee{FullName: "Rəşad Quliyev", Position: "Direktor", Salary: 4000, Status: "active", TaxID: "AZE1234"})
	db.Create(&models.Employee{FullName: "Günel Əliyeva", Position: "Mühasib", Salary: 1800, Status: "active", TaxID: "AZE5678"})
	db.Create(&models.Employee{FullName: "Elvin Məmmədov", Position: "Satış meneceri", Salary: 1200, Status: "active", TaxID: "AZE9012"})
	period := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	if run, err := engine.CreatePayrollRun(db, period); err == nil {
		engine.PostPayrollRun(db, run.ID)
	}

	return nil
}

// postDoc creates a document with lines and posts it.
func postDoc(db *gorm.DB, doc models.Document) *models.Document {
	if doc.FxRate == 0 {
		doc.FxRate = 1
	}
	if err := db.Create(&doc).Error; err != nil {
		log.Printf("demo: doc create: %v", err)
		return nil
	}
	// Recompute totals from lines before posting.
	recompute(db, &doc)
	if _, err := engine.PostDocument(db, doc.ID); err != nil {
		log.Printf("demo: post doc %d: %v", doc.ID, err)
	}
	return &doc
}

func recompute(db *gorm.DB, doc *models.Document) {
	var lines []models.DocumentLine
	db.Where("document_id = ?", doc.ID).Find(&lines)
	var sub, tax float64
	for i := range lines {
		net := lines[i].Quantity * lines[i].UnitPrice
		lines[i].LineTotal = net
		lines[i].TaxAmount = net * lines[i].TaxRate / 100
		sub += net
		tax += lines[i].TaxAmount
		db.Model(&models.DocumentLine{}).Where("id = ?", lines[i].ID).
			Updates(map[string]interface{}{"line_total": lines[i].LineTotal, "tax_amount": lines[i].TaxAmount})
	}
	db.Model(&models.Document{}).Where("id = ?", doc.ID).
		Updates(map[string]interface{}{"subtotal": sub, "tax_total": tax, "total": sub + tax})
}

func accByKey(db *gorm.DB, key string) uint {
	var a models.Account
	if db.Where("system_key = ?", key).First(&a).Error == nil {
		return a.ID
	}
	return 0
}
func accByCode(db *gorm.DB, code string) uint {
	var a models.Account
	if db.Where("code = ?", code).First(&a).Error == nil {
		return a.ID
	}
	return 0
}
func mustNum(db *gorm.DB) string {
	n, _ := engine.NextNumber(db, "journal", "J-")
	return n
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
