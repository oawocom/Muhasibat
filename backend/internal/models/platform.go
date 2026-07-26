package models

import "time"

// ============================================================
// Platform (control-plane) models — live in the main platform DB.
// Tenancy model: DATABASE-PER-COMPANY. Each company's accounting data
// lives in its own PostgreSQL database (oawo_company_<id>), giving full
// isolation. The platform DB only holds companies, users and memberships.
// ============================================================

// Company is a tenant. Its accounting tables live in DBName.
type Company struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Name           string    `json:"name" gorm:"size:255;not null"`
	TaxID          string    `json:"tax_id" gorm:"size:50"`
	DBName         string    `json:"-" gorm:"size:128;uniqueIndex"`
	Enabled        bool      `json:"enabled" gorm:"default:true"`
	Provisioned    bool      `json:"provisioned" gorm:"default:false"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// User is a global account. A user reaches company data through a Membership.
type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"size:255;uniqueIndex;not null"`
	Name         string    `json:"name" gorm:"size:255"`
	Password     string    `json:"-" gorm:"size:255"`
	IsSuperadmin bool      `json:"is_superadmin" gorm:"default:false"`
	Enabled      bool      `json:"enabled" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Membership links a user to a company with a role.
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
		&Company{}, &User{}, &Membership{},
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

// ModuleCatalog is the fixed list of toggleable tenant modules.
// Core modules are always available and cannot be turned off.
type ModuleDef struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Core  bool   `json:"core"`
}

func ModuleCatalog() []ModuleDef {
	return []ModuleDef{
		{Key: "dashboard", Label: "İdarə paneli", Core: true},
		{Key: "accounts", Label: "Hesablar planı", Core: true},
		{Key: "journal", Label: "Mühasibat jurnalı", Core: true},
		{Key: "partners", Label: "Tərəfdaşlar", Core: false},
		{Key: "products", Label: "Məhsul / Xidmət", Core: false},
		{Key: "sales", Label: "Satış fakturaları", Core: false},
		{Key: "purchases", Label: "Alış fakturaları", Core: false},
		{Key: "money", Label: "Kassa / Bank", Core: false},
		{Key: "inventory", Label: "Anbar", Core: false},
		{Key: "reports", Label: "Maliyyə hesabatları", Core: false},
		{Key: "settings", Label: "Parametrlər", Core: true},
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
