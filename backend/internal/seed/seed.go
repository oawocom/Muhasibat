package seed

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"gorm.io/gorm"

	"oawo-muhasibat/internal/models"
)

// SeedTenant seeds a fresh company database once (guarded by a Setting flag):
// chart of accounts, currencies, tax rates, default warehouse, numbering
// sequences and the enabled-modules list.
func SeedTenant(db *gorm.DB) error {
	var flag models.Setting
	if err := db.Where("key = ?", "seeded").First(&flag).Error; err == nil && flag.Value == "1" {
		return nil // already seeded
	}

	log.Println("🌱 Tenant başlanğıc məlumatları seed olunur...")
	if err := seedCurrencies(db); err != nil {
		return err
	}
	if err := seedTaxRates(db); err != nil {
		return err
	}
	if err := seedWarehouse(db); err != nil {
		return err
	}
	if err := seedAccounts(db); err != nil {
		return err
	}
	if err := seedSequences(db); err != nil {
		return err
	}

	modules, _ := json.Marshal(models.DefaultEnabledModules())
	db.Save(&models.Setting{Key: "seeded", Value: "1", UpdatedAt: time.Now()})
	db.Save(&models.Setting{Key: "base_currency", Value: "AZN", UpdatedAt: time.Now()})
	db.Save(&models.Setting{Key: "enabled_modules", Value: string(modules), UpdatedAt: time.Now()})
	log.Println("✅ Tenant seed tamamlandı.")
	return nil
}

func seedCurrencies(db *gorm.DB) error {
	currencies := []models.Currency{
		{Code: "AZN", Name: "Azərbaycan manatı", Symbol: "₼", Rate: 1, IsBase: true, Enabled: true},
		{Code: "USD", Name: "ABŞ dolları", Symbol: "$", Rate: 1.70, Enabled: true},
		{Code: "EUR", Name: "Avro", Symbol: "€", Rate: 1.85, Enabled: true},
		{Code: "RUB", Name: "Rusiya rublu", Symbol: "₽", Rate: 0.018, Enabled: true},
		{Code: "TRY", Name: "Türk lirəsi", Symbol: "₺", Rate: 0.05, Enabled: true},
	}
	for _, c := range currencies {
		var n int64
		db.Model(&models.Currency{}).Where("code = ?", c.Code).Count(&n)
		if n == 0 {
			if err := db.Create(&c).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedTaxRates(db *gorm.DB) error {
	rates := []models.TaxRate{
		{Name: "ƏDV 18%", Rate: 18, IsDefault: true, Enabled: true},
		{Name: "ƏDV 0% (ixrac)", Rate: 0, Enabled: true},
		{Name: "ƏDV-dən azad", Rate: 0, Enabled: true},
	}
	for _, r := range rates {
		var n int64
		db.Model(&models.TaxRate{}).Where("name = ?", r.Name).Count(&n)
		if n == 0 {
			if err := db.Create(&r).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedWarehouse(db *gorm.DB) error {
	var n int64
	db.Model(&models.Warehouse{}).Count(&n)
	if n == 0 {
		return db.Create(&models.Warehouse{Code: "ANB-1", Name: "Əsas anbar", IsDefault: true, Enabled: true}).Error
	}
	return nil
}

type acctDef struct {
	Code, Name, Type, SystemKey, Parent string
	Group                               bool
}

// Azerbaijani-style chart of accounts (Mühasibat uçotunun Hesablar Planı).
func seedAccounts(db *gorm.DB) error {
	var n int64
	db.Model(&models.Account{}).Count(&n)
	if n > 0 {
		return nil
	}

	defs := []acctDef{
		// ---- AKTİVLƏR (Assets) ----
		{Code: "1", Name: "AKTİVLƏR", Type: "asset", Group: true},
		{Code: "11", Name: "Uzunmüddətli aktivlər", Type: "asset", Parent: "1", Group: true},
		{Code: "111", Name: "Torpaq, tikili və avadanlıqlar", Type: "asset", Parent: "11"},
		{Code: "112", Name: "Yığılmış amortizasiya", Type: "asset", Parent: "11"},
		{Code: "13", Name: "Qeyri-maddi aktivlər", Type: "asset", Parent: "1"},

		{Code: "20", Name: "Qısamüddətli aktivlər", Type: "asset", Parent: "1", Group: true},
		{Code: "201", Name: "Mallar (əmtəə-material ehtiyatları)", Type: "asset", SystemKey: "inventory", Parent: "20"},
		{Code: "205", Name: "Hazır məhsul", Type: "asset", Parent: "20"},
		{Code: "211", Name: "Alıcılar və sifarişçilər (debitorlar)", Type: "asset", SystemKey: "ar", Parent: "20"},
		{Code: "217", Name: "Verilmiş avanslar", Type: "asset", Parent: "20"},
		{Code: "221", Name: "Kassa", Type: "asset", SystemKey: "cash", Parent: "20"},
		{Code: "223", Name: "Bank hesablaşma hesabı", Type: "asset", SystemKey: "bank", Parent: "20"},
		{Code: "241", Name: "ƏDV üzrə əvəzləşdirilən (giriş)", Type: "asset", SystemKey: "vat_in", Parent: "20"},

		// ---- KAPİTAL (Equity) ----
		{Code: "3", Name: "KAPİTAL", Type: "equity", Group: true},
		{Code: "301", Name: "Nizamnamə kapitalı", Type: "equity", Parent: "3"},
		{Code: "341", Name: "Bölüşdürülməmiş mənfəət (zərər)", Type: "equity", SystemKey: "retained", Parent: "3"},

		// ---- ÖHDƏLİKLƏR (Liabilities) ----
		{Code: "5", Name: "ÖHDƏLİKLƏR", Type: "liability", Group: true},
		{Code: "501", Name: "Qısamüddətli bank kreditləri", Type: "liability", Parent: "5"},
		{Code: "531", Name: "Malsatan və podratçılara borclar (kreditorlar)", Type: "liability", SystemKey: "ap", Parent: "5"},
		{Code: "537", Name: "Alınmış avanslar", Type: "liability", Parent: "5"},
		{Code: "521", Name: "ƏDV üzrə ödəniləcək (çıxış)", Type: "liability", SystemKey: "vat_out", Parent: "5"},
		{Code: "522", Name: "Digər vergi öhdəlikləri", Type: "liability", Parent: "5"},
		{Code: "533", Name: "Əməyin ödənişi üzrə öhdəliklər", Type: "liability", Parent: "5"},

		// ---- GƏLİRLƏR (Income) ----
		{Code: "6", Name: "GƏLİRLƏR", Type: "income", Group: true},
		{Code: "601", Name: "Satışdan gəlir (məhsul və xidmət)", Type: "income", SystemKey: "sales", Parent: "6"},
		{Code: "611", Name: "Sair əməliyyat gəlirləri", Type: "income", Parent: "6"},
		{Code: "631", Name: "Maliyyə gəlirləri", Type: "income", Parent: "6"},

		// ---- XƏRCLƏR (Expenses) ----
		{Code: "7", Name: "XƏRCLƏR", Type: "expense", Group: true},
		{Code: "701", Name: "Satılan malların maya dəyəri", Type: "expense", SystemKey: "cogs", Parent: "7"},
		{Code: "711", Name: "Kommersiya (satış) xərcləri", Type: "expense", SystemKey: "expense", Parent: "7"},
		{Code: "721", Name: "İnzibati xərclər", Type: "expense", Parent: "7"},
		{Code: "731", Name: "Sair əməliyyat xərcləri", Type: "expense", Parent: "7"},
		{Code: "741", Name: "Maliyyə xərcləri", Type: "expense", Parent: "7"},
	}

	idByCode := map[string]uint{}
	for i, d := range defs {
		acc := models.Account{
			Code: d.Code, Name: d.Name, Type: d.Type, IsGroup: d.Group,
			SystemKey: d.SystemKey, Enabled: true, SortOrder: i,
		}
		if d.Parent != "" {
			if pid, ok := idByCode[d.Parent]; ok {
				acc.ParentID = &pid
			}
		}
		if err := db.Create(&acc).Error; err != nil {
			return err
		}
		idByCode[d.Code] = acc.ID
	}
	return nil
}

func seedSequences(db *gorm.DB) error {
	seqs := []models.Sequence{
		{Key: "journal", Prefix: "J-", Next: 1, Padding: 5},
		{Key: "sales_invoice", Prefix: "SF-", Next: 1, Padding: 5},
		{Key: "purchase_invoice", Prefix: "AF-", Next: 1, Padding: 5},
		{Key: "payment", Prefix: "ÖD-", Next: 1, Padding: 5},
		{Key: "receipt", Prefix: "MD-", Next: 1, Padding: 5},
	}
	for _, s := range seqs {
		var n int64
		db.Model(&models.Sequence{}).Where("key = ?", s.Key).Count(&n)
		if n == 0 {
			if err := db.Create(&s).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
