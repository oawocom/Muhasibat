package api

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"oawo-muhasibat/internal/models"
)

// ============================================================
// Tenant (subscription/contract) management — SUPERADMIN ONLY.
// The subscription amount and payment data are visible here only.
// ============================================================

func (s *Server) requireSuperadmin(c *gin.Context) bool {
	if !c.GetBool("isSuperadmin") {
		c.JSON(403, gin.H{"detail": "Yalnız sistem administratoru"})
		return false
	}
	return true
}

type tenantView struct {
	models.Tenant
	Modules      []string `json:"modules"`
	CompanyCount int64    `json:"company_count"`
	UserCount    int64    `json:"user_count"`
	AdminEmail   string   `json:"admin_email"`
}

func (s *Server) tenantToView(t models.Tenant) tenantView {
	v := tenantView{Tenant: t, Modules: t.Modules()}
	s.DB.Model(&models.Company{}).Where("tenant_id = ?", t.ID).Count(&v.CompanyCount)
	s.DB.Model(&models.User{}).Where("tenant_id = ?", t.ID).Count(&v.UserCount)
	var admin models.User
	if s.DB.Where("tenant_id = ? AND is_tenant_admin = ?", t.ID, true).Order("id asc").First(&admin).Error == nil {
		v.AdminEmail = admin.Email
	}
	return v
}

// ListTenants returns all tenants with subscription + counts (superadmin).
func (s *Server) ListTenants(c *gin.Context) {
	if !s.requireSuperadmin(c) {
		return
	}
	var tenants []models.Tenant
	s.DB.Order("name asc").Find(&tenants)
	out := []tenantView{}
	for _, t := range tenants {
		out = append(out, s.tenantToView(t))
	}
	c.JSON(200, out)
}

// GetTenant returns one tenant (superadmin).
func (s *Server) GetTenant(c *gin.Context) {
	if !s.requireSuperadmin(c) {
		return
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var t models.Tenant
	if err := s.DB.First(&t, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tenant tapılmadı"})
		return
	}
	c.JSON(200, s.tenantToView(t))
}

// CreateTenant onboards a new customer: creates the tenant (subscription),
// its single main admin user, and a first company. Superadmin only.
func (s *Server) CreateTenant(c *gin.Context) {
	if !s.requireSuperadmin(c) {
		return
	}
	var req struct {
		Name               string   `json:"name"`
		ContactEmail       string   `json:"contact_email"`
		ContactPhone       string   `json:"contact_phone"`
		Plan               string   `json:"plan"`
		Modules            []string `json:"modules"`
		SubscriptionAmount float64  `json:"subscription_amount"`
		BillingCycle       string   `json:"billing_cycle"`
		AdminEmail         string   `json:"admin_email"`
		AdminName          string   `json:"admin_name"`
		AdminPassword      string   `json:"admin_password"`
		CompanyName        string   `json:"company_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(400, gin.H{"detail": "Tenant adı tələb olunur"})
		return
	}
	req.AdminEmail = strings.ToLower(strings.TrimSpace(req.AdminEmail))
	if req.AdminEmail == "" || len(req.AdminPassword) < 6 {
		c.JSON(400, gin.H{"detail": "Admin email və şifrə (min 6 simvol) tələb olunur"})
		return
	}
	var exists int64
	s.DB.Model(&models.User{}).Where("email = ?", req.AdminEmail).Count(&exists)
	if exists > 0 {
		c.JSON(400, gin.H{"detail": "Bu email artıq istifadə olunur"})
		return
	}

	if len(req.Modules) == 0 {
		req.Modules = models.DefaultEnabledModules()
	}
	modules := ensureCoreModules(req.Modules)
	modJSON, _ := json.Marshal(modules)
	if req.BillingCycle == "" {
		req.BillingCycle = "monthly"
	}
	if req.Plan == "" {
		req.Plan = "standard"
	}

	t := models.Tenant{
		Name: strings.TrimSpace(req.Name), ContactEmail: req.ContactEmail, ContactPhone: req.ContactPhone,
		Plan: req.Plan, EnabledModules: string(modJSON), SubscriptionAmount: ComputeSubscription(modules),
		BillingCycle: req.BillingCycle, Enabled: true,
	}
	if err := s.DB.Create(&t).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Tenant yaradıla bilmədi"})
		return
	}

	// The single tenant admin.
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	admin := models.User{
		Email: req.AdminEmail, Name: req.AdminName, Password: string(hash),
		TenantID: &t.ID, IsTenantAdmin: true, Enabled: true,
	}
	if err := s.DB.Create(&admin).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Tenant admini yaradıla bilmədi"})
		return
	}

	// First company (so the tenant admin has somewhere to work immediately).
	companyName := req.CompanyName
	if strings.TrimSpace(companyName) == "" {
		companyName = req.Name
	}
	if _, err := s.provisionCompany(t.ID, companyName, ""); err != nil {
		c.JSON(500, gin.H{"detail": "İlk şirkət hazırlana bilmədi: " + err.Error()})
		return
	}

	c.JSON(201, s.tenantToView(t))
}

// UpdateTenant changes subscription: modules, price, plan, enabled (superadmin).
func (s *Server) UpdateTenant(c *gin.Context) {
	if !s.requireSuperadmin(c) {
		return
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var t models.Tenant
	if err := s.DB.First(&t, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tenant tapılmadı"})
		return
	}
	var req struct {
		Name               *string   `json:"name"`
		ContactEmail       *string   `json:"contact_email"`
		ContactPhone       *string   `json:"contact_phone"`
		Plan               *string   `json:"plan"`
		Modules            *[]string `json:"modules"`
		SubscriptionAmount *float64  `json:"subscription_amount"`
		BillingCycle       *string   `json:"billing_cycle"`
		Enabled            *bool     `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.ContactEmail != nil {
		t.ContactEmail = *req.ContactEmail
	}
	if req.ContactPhone != nil {
		t.ContactPhone = *req.ContactPhone
	}
	if req.Plan != nil {
		t.Plan = *req.Plan
	}
	if req.Modules != nil {
		mods := ensureCoreModules(*req.Modules)
		mj, _ := json.Marshal(mods)
		t.EnabledModules = string(mj)
		t.SubscriptionAmount = ComputeSubscription(mods) // price follows modules
	}
	if req.SubscriptionAmount != nil {
		t.SubscriptionAmount = *req.SubscriptionAmount // explicit override
	}
	if req.BillingCycle != nil {
		t.BillingCycle = *req.BillingCycle
	}
	if req.Enabled != nil {
		t.Enabled = *req.Enabled
	}
	s.DB.Save(&t)
	c.JSON(200, s.tenantToView(t))
}

// ComputeSubscription is the pricing formula: sum of the prices of the
// enabled non-core modules. Adding a module raises the amount, removing it
// lowers it.
func ComputeSubscription(modules []string) float64 {
	set := map[string]bool{}
	for _, m := range modules {
		set[m] = true
	}
	var total float64
	for _, def := range models.ModuleCatalog() {
		if !def.Core && set[def.Key] {
			total += def.Price
		}
	}
	return total
}

// GetMyTenant returns the current tenant admin's own tenant (name, modules,
// amount) — so they can see and adjust their subscription.
func (s *Server) GetMyTenant(c *gin.Context) {
	u, ok := s.currentUser(c)
	if !ok || u.TenantID == nil {
		c.JSON(404, gin.H{"detail": "Tenant tapılmadı"})
		return
	}
	var t models.Tenant
	if err := s.DB.First(&t, *u.TenantID).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tenant tapılmadı"})
		return
	}
	c.JSON(200, gin.H{
		"id": t.ID, "name": t.Name, "plan": t.Plan, "modules": t.Modules(),
		"subscription_amount": t.SubscriptionAmount, "billing_cycle": t.BillingCycle,
	})
}

// UpdateMyTenantModules lets a tenant admin self-select modules; the amount is
// recomputed from the formula.
func (s *Server) UpdateMyTenantModules(c *gin.Context) {
	u, ok := s.currentUser(c)
	if !ok || u.TenantID == nil || !u.IsTenantAdmin {
		c.JSON(403, gin.H{"detail": "Yalnız tenant admini modulları dəyişə bilər"})
		return
	}
	var req struct {
		Modules []string `json:"modules"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	var t models.Tenant
	if err := s.DB.First(&t, *u.TenantID).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tenant tapılmadı"})
		return
	}
	mods := ensureCoreModules(req.Modules)
	mj, _ := json.Marshal(mods)
	t.EnabledModules = string(mj)
	t.SubscriptionAmount = ComputeSubscription(mods)
	s.DB.Save(&t)
	c.JSON(200, gin.H{"modules": mods, "subscription_amount": t.SubscriptionAmount})
}

// ensureCoreModules guarantees core modules are always present.
func ensureCoreModules(mods []string) []string {
	set := map[string]bool{}
	for _, m := range mods {
		set[m] = true
	}
	for _, def := range models.ModuleCatalog() {
		if def.Core {
			set[def.Key] = true
		}
	}
	out := []string{}
	for _, def := range models.ModuleCatalog() { // stable order
		if set[def.Key] {
			out = append(out, def.Key)
		}
	}
	return out
}
