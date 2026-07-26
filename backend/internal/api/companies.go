package api

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"oawo-muhasibat/internal/models"
	"oawo-muhasibat/internal/tenant"
)

var validRoles = map[string]bool{
	models.RoleOwner: true, models.RoleAdmin: true, models.RoleAccountant: true,
	models.RoleWarehouse: true, models.RoleViewer: true,
}

// currentUser loads the authenticated user.
func (s *Server) currentUser(c *gin.Context) (models.User, bool) {
	var u models.User
	if err := s.DB.First(&u, c.GetUint("userID")).Error; err != nil {
		return u, false
	}
	return u, true
}

// canManage reports whether the caller may manage the given company
// (superadmin, the tenant's admin, or a company owner/admin).
func (s *Server) canManage(c *gin.Context, companyID uint) bool {
	if c.GetBool("isSuperadmin") {
		return true
	}
	u, ok := s.currentUser(c)
	if !ok {
		return false
	}
	var cmp models.Company
	if err := s.DB.First(&cmp, companyID).Error; err != nil {
		return false
	}
	if u.IsTenantAdmin && u.TenantID != nil && cmp.TenantID == *u.TenantID {
		return true
	}
	var m models.Membership
	if err := s.DB.Where("user_id = ? AND company_id = ? AND enabled = ?", u.ID, companyID, true).First(&m).Error; err != nil {
		return false
	}
	return models.CanManageCompany(m.Role)
}

// ListCompanies returns the companies the caller can access.
func (s *Server) ListCompanies(c *gin.Context) {
	u, ok := s.currentUser(c)
	if !ok {
		c.JSON(404, gin.H{"detail": "İstifadəçi tapılmadı"})
		return
	}
	c.JSON(200, s.userCompanies(u))
}

// CreateCompany creates a company under a tenant, provisions its database and
// seeds the chart of accounts. Allowed for the superadmin (any tenant) or a
// tenant admin (their own tenant only). Plain users cannot.
func (s *Server) CreateCompany(c *gin.Context) {
	u, ok := s.currentUser(c)
	if !ok {
		c.JSON(404, gin.H{"detail": "İstifadəçi tapılmadı"})
		return
	}
	var req struct {
		Name     string `json:"name"`
		TaxID    string `json:"tax_id"`
		TenantID uint   `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(400, gin.H{"detail": "Şirkət adı tələb olunur"})
		return
	}

	tenantID := req.TenantID
	if !u.IsSuperadmin {
		if !u.IsTenantAdmin || u.TenantID == nil {
			c.JSON(403, gin.H{"detail": "Şirkət yaratmaq üçün icazəniz yoxdur"})
			return
		}
		tenantID = *u.TenantID // tenant admins may only create within their tenant
	}
	if tenantID == 0 {
		c.JSON(400, gin.H{"detail": "Tenant seçilməlidir"})
		return
	}
	var t models.Tenant
	if err := s.DB.First(&t, tenantID).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tenant tapılmadı"})
		return
	}

	cmp, err := s.provisionCompany(tenantID, req.Name, req.TaxID)
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(201, companyView{Company: *cmp, Role: models.RoleOwner, Modules: t.Modules()})
}

// provisionCompany creates + provisions a company under a tenant.
func (s *Server) provisionCompany(tenantID uint, name, taxID string) (*models.Company, error) {
	cmp := models.Company{TenantID: tenantID, Name: strings.TrimSpace(name), TaxID: taxID, Enabled: true}
	if err := s.DB.Create(&cmp).Error; err != nil {
		return nil, err
	}
	cmp.DBName = tenant.DBNameFor(cmp.ID)
	s.DB.Save(&cmp)
	if _, err := s.Mgr.Provision(cmp.ID, cmp.DBName); err != nil {
		return nil, err
	}
	s.DB.Model(&models.Company{}).Where("id = ?", cmp.ID).Update("provisioned", true)
	cmp.Provisioned = true
	return &cmp, nil
}

// UpdateCompany updates company metadata.
func (s *Server) UpdateCompany(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	if !s.canManage(c, id) {
		c.JSON(403, gin.H{"detail": "İcazəniz yoxdur"})
		return
	}
	var company models.Company
	if err := s.DB.First(&company, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	var req struct {
		Name    string `json:"name"`
		TaxID   string `json:"tax_id"`
		Enabled *bool  `json:"enabled"`
	}
	c.ShouldBindJSON(&req)
	if req.Name != "" {
		company.Name = req.Name
	}
	company.TaxID = req.TaxID
	if req.Enabled != nil {
		company.Enabled = *req.Enabled
	}
	s.DB.Save(&company)
	c.JSON(200, company)
}

// membershipView joins a membership with its user details.
type membershipView struct {
	ID      uint   `json:"id"`
	UserID  uint   `json:"user_id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	Enabled bool   `json:"enabled"`
}

// ListCompanyUsers lists the users (memberships) of a company.
func (s *Server) ListCompanyUsers(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış ID"})
		return
	}
	if !s.canManage(c, uint(id)) {
		c.JSON(403, gin.H{"detail": "İcazəniz yoxdur"})
		return
	}
	var memberships []models.Membership
	s.DB.Where("company_id = ?", id).Find(&memberships)
	out := []membershipView{}
	for _, m := range memberships {
		var u models.User
		if s.DB.First(&u, m.UserID).Error == nil {
			out = append(out, membershipView{ID: m.ID, UserID: u.ID, Email: u.Email, Name: u.Name, Role: m.Role, Enabled: m.Enabled})
		}
	}
	c.JSON(200, out)
}

// AddCompanyUser attaches an existing user or creates a new one, with a role.
// New users are bound to the company's tenant.
func (s *Server) AddCompanyUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış ID"})
		return
	}
	if !s.canManage(c, uint(id)) {
		c.JSON(403, gin.H{"detail": "İcazəniz yoxdur"})
		return
	}
	var company models.Company
	if err := s.DB.First(&company, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Şirkət tapılmadı"})
		return
	}
	var req struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || !validRoles[req.Role] {
		c.JSON(400, gin.H{"detail": "Email və düzgün rol tələb olunur"})
		return
	}
	var u models.User
	if err := s.DB.Where("email = ?", req.Email).First(&u).Error; err != nil {
		if len(req.Password) < 6 {
			c.JSON(400, gin.H{"detail": "Yeni istifadəçi üçün şifrə (min 6 simvol) tələb olunur"})
			return
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		tid := company.TenantID
		u = models.User{Email: req.Email, Name: req.Name, Password: string(hash), TenantID: &tid, Enabled: true}
		if err := s.DB.Create(&u).Error; err != nil {
			c.JSON(500, gin.H{"detail": "İstifadəçi yaradıla bilmədi"})
			return
		}
	} else if u.TenantID != nil && *u.TenantID != company.TenantID && !u.IsSuperadmin {
		c.JSON(400, gin.H{"detail": "Bu email başqa tenant-a aiddir"})
		return
	}
	var m models.Membership
	if err := s.DB.Where("user_id = ? AND company_id = ?", u.ID, id).First(&m).Error; err == nil {
		m.Role = req.Role
		m.Enabled = true
		s.DB.Save(&m)
	} else {
		m = models.Membership{UserID: u.ID, CompanyID: uint(id), Role: req.Role, Enabled: true}
		s.DB.Create(&m)
	}
	c.JSON(201, membershipView{ID: m.ID, UserID: u.ID, Email: u.Email, Name: u.Name, Role: m.Role, Enabled: m.Enabled})
}

// UpdateCompanyUser changes a member's role or enabled state.
func (s *Server) UpdateCompanyUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	mid, _ := strconv.ParseUint(c.Param("mid"), 10, 64)
	if !s.canManage(c, uint(id)) {
		c.JSON(403, gin.H{"detail": "İcazəniz yoxdur"})
		return
	}
	var m models.Membership
	if err := s.DB.Where("id = ? AND company_id = ?", mid, id).First(&m).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	var req struct {
		Role    string `json:"role"`
		Enabled *bool  `json:"enabled"`
	}
	c.ShouldBindJSON(&req)
	if req.Role != "" && validRoles[req.Role] {
		m.Role = req.Role
	}
	if req.Enabled != nil {
		m.Enabled = *req.Enabled
	}
	s.DB.Save(&m)
	c.JSON(200, m)
}

// RemoveCompanyUser detaches a user from the company.
func (s *Server) RemoveCompanyUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	mid, _ := strconv.ParseUint(c.Param("mid"), 10, 64)
	if !s.canManage(c, uint(id)) {
		c.JSON(403, gin.H{"detail": "İcazəniz yoxdur"})
		return
	}
	s.DB.Where("id = ? AND company_id = ?", mid, id).Delete(&models.Membership{})
	c.JSON(200, gin.H{"message": "Silindi"})
}

// Roles lists the available company roles (for UI dropdowns).
func (s *Server) Roles(c *gin.Context) {
	c.JSON(200, []gin.H{
		{"key": models.RoleOwner, "label": "Sahib"},
		{"key": models.RoleAdmin, "label": "Admin"},
		{"key": models.RoleAccountant, "label": "Mühasib"},
		{"key": models.RoleWarehouse, "label": "Anbardar"},
		{"key": models.RoleViewer, "label": "Baxış (yalnız oxu)"},
	})
}
