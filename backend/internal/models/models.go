package models

import (
	"encoding/json"
	"time"
)

// ============================================================
// OAWO Mühasibat — professional double-entry accounting engine.
// Standalone product. Money: decimal(18,2); FX rates: decimal(18,6).
// Account types drive reports and normal-balance side:
//   asset, liability, equity, income, expense
// ============================================================

// ==================== CHART OF ACCOUNTS ====================
type Account struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Code        string    `json:"code" gorm:"size:32;index;not null"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Type        string    `json:"type" gorm:"size:20;index;not null"` // asset|liability|equity|income|expense
	ParentID    *uint     `json:"parent_id" gorm:"index"`
	IsGroup     bool      `json:"is_group" gorm:"default:false"`
	CurrencyID  *uint     `json:"currency_id" gorm:"index"`
	Description string    `json:"description" gorm:"type:text"`
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	SortOrder   int       `json:"sort_order" gorm:"default:0"`
	SystemKey   string    `json:"system_key" gorm:"size:64;index"` // ar, ap, vat_out, vat_in, sales, cogs, cash, bank, retained
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ==================== CURRENCIES ====================
type Currency struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Code      string    `json:"code" gorm:"size:8;uniqueIndex;not null"`
	Name      string    `json:"name" gorm:"size:100"`
	Symbol    string    `json:"symbol" gorm:"size:8"`
	Rate      float64   `json:"rate" gorm:"type:decimal(18,6);default:1"` // base units per 1 unit
	IsBase    bool      `json:"is_base" gorm:"default:false"`
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ==================== PARTNERS ====================
type Partner struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Code        string    `json:"code" gorm:"size:32;index"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Type        string    `json:"type" gorm:"size:20;default:both"` // customer|supplier|both
	TaxID       string    `json:"tax_id" gorm:"size:50;index"`      // VÖEN
	Email       string    `json:"email" gorm:"size:255"`
	Phone       string    `json:"phone" gorm:"size:50"`
	Address     string    `json:"address" gorm:"type:text"`
	ContactName string    `json:"contact_name" gorm:"size:255"`
	BankName    string    `json:"bank_name" gorm:"size:255"`
	BankAccount string    `json:"bank_account" gorm:"size:64"`
	Notes       string    `json:"notes" gorm:"type:text"`
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ==================== TAX RATES ====================
type TaxRate struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	Rate      float64   `json:"rate" gorm:"type:decimal(9,4);default:0"`
	IsDefault bool      `json:"is_default" gorm:"default:false"`
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ==================== WAREHOUSES ====================
type Warehouse struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Code      string    `json:"code" gorm:"size:32;index"`
	Name      string    `json:"name" gorm:"size:255;not null"`
	Address   string    `json:"address" gorm:"type:text"`
	IsDefault bool      `json:"is_default" gorm:"default:false"`
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ==================== PRODUCTS / SERVICES ====================
type Product struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Code        string    `json:"code" gorm:"size:64;index"`
	Barcode     string    `json:"barcode" gorm:"size:64;index"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Type        string    `json:"type" gorm:"size:20;default:product"` // product|service
	Unit        string    `json:"unit" gorm:"size:20;default:ədəd"`
	SalePrice   float64   `json:"sale_price" gorm:"type:decimal(18,2);default:0"`
	CostPrice   float64   `json:"cost_price" gorm:"type:decimal(18,2);default:0"`
	TaxRateID   *uint     `json:"tax_rate_id" gorm:"index"`
	TrackStock  bool      `json:"track_stock" gorm:"default:true"`
	Description string    `json:"description" gorm:"type:text"`
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// StockMove — physical stock movement line (+ in, - out).
type StockMove struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ProductID   uint      `json:"product_id" gorm:"index;not null"`
	WarehouseID uint      `json:"warehouse_id" gorm:"index;not null"`
	Date        Date      `json:"date" gorm:"index"`
	Quantity    float64   `json:"quantity" gorm:"type:decimal(18,3)"`
	UnitCost    float64   `json:"unit_cost" gorm:"type:decimal(18,2)"`
	DocumentID  *uint     `json:"document_id" gorm:"index"`
	DocType     string    `json:"doc_type" gorm:"size:32"`
	CreatedAt   time.Time `json:"created_at"`
}

// ==================== JOURNAL (double-entry core) ====================
type JournalEntry struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Number      string         `json:"number" gorm:"size:32;index"`
	Date        Date           `json:"date" gorm:"index;not null"`
	Description string         `json:"description" gorm:"type:text"`
	Reference   string         `json:"reference" gorm:"size:128"`
	Status      string         `json:"status" gorm:"size:16;index;default:draft"` // draft|posted|void
	Source      string         `json:"source" gorm:"size:32;default:manual"`
	DocumentID  *uint          `json:"document_id" gorm:"index"`
	TotalDebit  float64        `json:"total_debit" gorm:"type:decimal(18,2);default:0"`
	TotalCredit float64        `json:"total_credit" gorm:"type:decimal(18,2);default:0"`
	Lines       []JournalLine  `json:"lines" gorm:"foreignKey:EntryID;constraint:OnDelete:CASCADE"`
	Meta        json.RawMessage `json:"meta" gorm:"type:jsonb"`
	PostedAt    *time.Time     `json:"posted_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type JournalLine struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	EntryID     uint      `json:"entry_id" gorm:"index;not null"`
	AccountID   uint      `json:"account_id" gorm:"index;not null"`
	PartnerID   *uint     `json:"partner_id" gorm:"index"`
	Description string    `json:"description" gorm:"size:512"`
	Debit       float64   `json:"debit" gorm:"type:decimal(18,2);default:0"`
	Credit      float64   `json:"credit" gorm:"type:decimal(18,2);default:0"`
	CurrencyID  *uint     `json:"currency_id" gorm:"index"`
	FxRate      float64   `json:"fx_rate" gorm:"type:decimal(18,6);default:1"`
	SortOrder   int       `json:"sort_order" gorm:"default:0"`
	Date        Date      `json:"date" gorm:"index"`
	Posted      bool      `json:"posted" gorm:"index;default:false"`
}

// ==================== BUSINESS DOCUMENTS ====================
type Document struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Type          string         `json:"type" gorm:"size:24;index;not null"` // sales_invoice|purchase_invoice|payment|receipt
	Number        string         `json:"number" gorm:"size:32;index"`
	Date          Date           `json:"date" gorm:"index;not null"`
	DueDate       *Date          `json:"due_date"`
	PartnerID     *uint          `json:"partner_id" gorm:"index"`
	WarehouseID   *uint          `json:"warehouse_id" gorm:"index"`
	CurrencyID    *uint          `json:"currency_id" gorm:"index"`
	FxRate        float64        `json:"fx_rate" gorm:"type:decimal(18,6);default:1"`
	Status        string         `json:"status" gorm:"size:16;index;default:draft"` // draft|posted|paid|void
	Notes         string         `json:"notes" gorm:"type:text"`
	Subtotal      float64        `json:"subtotal" gorm:"type:decimal(18,2);default:0"`
	TaxTotal      float64        `json:"tax_total" gorm:"type:decimal(18,2);default:0"`
	Total         float64        `json:"total" gorm:"type:decimal(18,2);default:0"`
	PaidTotal     float64        `json:"paid_total" gorm:"type:decimal(18,2);default:0"`
	CashAccountID *uint          `json:"cash_account_id" gorm:"index"`
	Lines         []DocumentLine `json:"lines" gorm:"foreignKey:DocumentID;constraint:OnDelete:CASCADE"`
	JournalID     *uint          `json:"journal_id" gorm:"index"`
	Meta          json.RawMessage `json:"meta" gorm:"type:jsonb"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type DocumentLine struct {
	ID          uint    `json:"id" gorm:"primaryKey"`
	DocumentID  uint    `json:"document_id" gorm:"index;not null"`
	ProductID   *uint   `json:"product_id" gorm:"index"`
	Description string  `json:"description" gorm:"size:512"`
	Quantity    float64 `json:"quantity" gorm:"type:decimal(18,3);default:1"`
	UnitPrice   float64 `json:"unit_price" gorm:"type:decimal(18,2);default:0"`
	TaxRateID   *uint   `json:"tax_rate_id" gorm:"index"`
	TaxRate     float64 `json:"tax_rate" gorm:"type:decimal(9,4);default:0"`
	LineTotal   float64 `json:"line_total" gorm:"type:decimal(18,2);default:0"` // net (excl tax)
	TaxAmount   float64 `json:"tax_amount" gorm:"type:decimal(18,2);default:0"`
	SortOrder   int     `json:"sort_order" gorm:"default:0"`
}

// ==================== SETTINGS / NUMBERING ====================
type Sequence struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"size:48;uniqueIndex;not null"`
	Prefix    string    `json:"prefix" gorm:"size:16"`
	Next      int       `json:"next" gorm:"default:1"`
	Padding   int       `json:"padding" gorm:"default:5"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Setting struct {
	Key       string    `json:"key" gorm:"primaryKey;size:64"`
	Value     string    `json:"value" gorm:"type:text"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TenantModels returns every accounting model — migrated into each
// company's own database. (Users live in the platform DB, not here.)
func TenantModels() []interface{} {
	return []interface{}{
		&Currency{}, &Account{}, &Partner{}, &TaxRate{}, &Warehouse{},
		&Product{}, &StockMove{}, &JournalEntry{}, &JournalLine{},
		&Document{}, &DocumentLine{}, &Sequence{}, &Setting{},
	}
}
