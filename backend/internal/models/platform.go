package models

import (
	"encoding/json"
	"time"
)

// ============================================================
// Platform (control-plane) models — live in the main platform DB.
//
// Three-level hierarchy:
//   Superadmin (OAWO)  →  Tenant (a contract / customer, a subscription)
//                         →  Company (an accounting entity, own database)
//
// - The superadmin onboards Tenants and their main admin, and sets which
//   modules the tenant subscribes to (which drives the subscription price).
// - A Tenant's admin self-manages: creates companies and users within the
//   tenant, limited to the subscribed modules.
// - Each Company's accounting data lives in its own database
//   (oawo_company_<id>), giving full isolation.
// ============================================================

// Tenant is the contracting customer / subscription.
type Tenant struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	Name               string    `json:"name" gorm:"size:255;not null"`
	ContactEmail       string    `json:"contact_email" gorm:"size:255"`
	ContactPhone       string    `json:"contact_phone" gorm:"size:50"`
	Plan               string    `json:"plan" gorm:"size:32;default:standard"`
	EnabledModules     string    `json:"enabled_modules" gorm:"type:text"` // JSON array of module keys (subscription scope)
	SubscriptionAmount float64   `json:"subscription_amount" gorm:"type:decimal(18,2);default:0"`
	BillingCycle       string    `json:"billing_cycle" gorm:"size:16;default:monthly"` // monthly|yearly
	Enabled            bool      `json:"enabled" gorm:"default:true"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Modules returns the tenant's subscribed module keys.
func (t Tenant) Modules() []string {
	var out []string
	if t.EnabledModules == "" {
		return DefaultEnabledModules()
	}
	if err := json.Unmarshal([]byte(t.EnabledModules), &out); err != nil {
		return DefaultEnabledModules()
	}
	return out
}

// User is a global account. Superadmins have no tenant; every other user
// belongs to exactly one tenant. Tenant admins self-manage their tenant.
type User struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Email         string    `json:"email" gorm:"size:255;uniqueIndex;not null"`
	Name          string    `json:"name" gorm:"size:255"`
	Password      string    `json:"-" gorm:"size:255"`
	IsSuperadmin  bool      `json:"is_superadmin" gorm:"default:false"`
	TenantID      *uint     `json:"tenant_id" gorm:"index"`
	IsTenantAdmin bool      `json:"is_tenant_admin" gorm:"default:false"`
	Enabled       bool      `json:"enabled" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Company belongs to a tenant. Its accounting tables live in DBName.
type Company struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TenantID    uint      `json:"tenant_id" gorm:"index;not null"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	TaxID       string    `json:"tax_id" gorm:"size:50"`
	DBName      string    `json:"-" gorm:"size:128;uniqueIndex"`
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	Provisioned bool      `json:"provisioned" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Membership links a regular user to a company with a role.
// Roles: owner | admin | accountant | warehouse | viewer
type Membership struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null;uniqueIndex:ux_user_company"`
	CompanyID uint      `json:"company_id" gorm:"index;not null;uniqueIndex:ux_user_company"`
	Role      string    `json:"role" gorm:"size:32;default:viewer"`
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PlatformModels are migrated into the platform (control-plane) DB.
func PlatformModels() []interface{} {
	return []interface{}{
		&Tenant{}, &Company{}, &User{}, &Membership{},
	}
}

// Role hierarchy helpers.
const (
	RoleOwner      = "owner"
	RoleAdmin      = "admin"
	RoleAccountant = "accountant"
	RoleWarehouse  = "warehouse"
	RoleViewer     = "viewer"
)

// CanWrite reports whether a role may create/modify tenant data.
func CanWrite(role string) bool {
	return role != RoleViewer && role != ""
}

// CanManageCompany reports whether a role may manage users/company settings.
func CanManageCompany(role string) bool {
	return role == RoleOwner || role == RoleAdmin
}

// ModuleCatalog is the fixed list of modules a tenant can subscribe to.
type ModuleDef struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Core  bool    `json:"core"`
	Price float64 `json:"price"` // suggested monthly price contribution (AZN)
}

func ModuleCatalog() []ModuleDef {
	return []ModuleDef{
		{Key: "dashboard", Label: "İdarə paneli", Core: true, Price: 0},
		{Key: "accounts", Label: "Hesablar planı", Core: true, Price: 0},
		{Key: "journal", Label: "Mühasibat jurnalı", Core: true, Price: 0},
		{Key: "partners", Label: "Tərəfdaşlar", Core: false, Price: 5},
		{Key: "products", Label: "Məhsul / Xidmət", Core: false, Price: 5},
		{Key: "sales", Label: "Satış fakturaları", Core: false, Price: 10},
		{Key: "purchases", Label: "Alış fakturaları", Core: false, Price: 10},
		{Key: "money", Label: "Kassa / Bank", Core: false, Price: 8},
		{Key: "inventory", Label: "Anbar", Core: false, Price: 12},
		{Key: "fixed_assets", Label: "Əsas vəsaitlər", Core: false, Price: 12},
		{Key: "reports", Label: "Maliyyə hesabatları", Core: false, Price: 15},
		{Key: "settings", Label: "Parametrlər", Core: true, Price: 0},
	}
}

// DefaultEnabledModules returns every module key (all on by default).
func DefaultEnabledModules() []string {
	cat := ModuleCatalog()
	keys := make([]string, 0, len(cat))
	for _, m := range cat {
		keys = append(keys, m.Key)
	}
	return keys
}
