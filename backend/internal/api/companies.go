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

// canManage reports whether the caller may manage the given company.
func (s *Server) canManage(c *gin.Context, companyID uint) bool {
	if c.GetBool("isSuperadmin") {
		return true
	}
	var m models.Membership
	if err := s.DB.Where("user_id = ? AND company_id = ? AND enabled = ?",
		c.GetUint("userID"), companyID, true).First(&m).Error; err != nil {
		return false
	}
	return models.CanManageCompany(m.Role)
}

// ListCompanies returns the companies the caller can access.
func (s *Server) ListCompanies(c *gin.Context) {
	var u models.User
	if err := s.DB.First(&u, c.GetUint("userID")).Error; err != nil {
		c.JSON(404, gin.H{"detail": "İstifadəçi tapılmadı"})
		return
	}
	c.JSON(200, s.userCompanies(u))
}

// CreateCompany creates a new tenant, provisions its database and makes the
// caller its owner.
func (s *Server) CreateCompany(c *gin.Context) {
	var req struct {
		Name  string `json:"name"`
		TaxID string `json:"tax_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(400, gin.H{"detail": "Şirkət adı tələb olunur"})
		return
	}
	company := models.Company{Name: strings.TrimSpace(req.Name), TaxID: req.TaxID, Enabled: true}
	if err := s.DB.Create(&company).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Şirkət yaradıla bilmədi"})
		return
	}
	company.DBName = tenant.DBNameFor(company.ID)
	s.DB.Save(&company)
	if _, err := s.Mgr.Provision(company.ID, company.DBName); err != nil {
		c.JSON(500, gin.H{"detail": "Şirkət bazası hazırlana bilmədi: " + err.Error()})
		return
	}
	s.DB.Model(&models.Company{}).Where("id = ?", company.ID).Update("provisioned", true)
	company.Provisioned = true

	// Creator becomes owner (superadmin already has implicit access).
	if !c.GetBool("isSuperadmin") {
		s.DB.Create(&models.Membership{UserID: c.GetUint("userID"), CompanyID: company.ID, Role: models.RoleOwner, Enabled: true})
	}
	c.JSON(201, companyView{Company: company, Role: models.RoleOwner})
}

// UpdateCompany updates company metadata (owner/admin/superadmin).
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
	if req.Enabled != nil && c.GetBool("isSuperadmin") {
		company.Enabled = *req.Enabled
	}
	s.DB.Save(&company)
	c.JSON(200, company)
}

// membershipView joins a membership with its user details.
type membershipView struct {
	ID     uint   `json:"id"`
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Enabled bool  `json:"enabled"`
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
		// New user — password required.
		if len(req.Password) < 6 {
			c.JSON(400, gin.H{"detail": "Yeni istifadəçi üçün şifrə (min 6 simvol) tələb olunur"})
			return
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		u = models.User{Email: req.Email, Name: req.Name, Password: string(hash), Enabled: true}
		if err := s.DB.Create(&u).Error; err != nil {
			c.JSON(500, gin.H{"detail": "İstifadəçi yaradıla bilmədi"})
			return
		}
	}
	// Upsert membership.
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

// Roles lists the available roles (for UI dropdowns).
func (s *Server) Roles(c *gin.Context) {
	c.JSON(200, []gin.H{
		{"key": models.RoleOwner, "label": "Sahib"},
		{"key": models.RoleAdmin, "label": "Admin"},
		{"key": models.RoleAccountant, "label": "Mühasib"},
		{"key": models.RoleWarehouse, "label": "Anbardar"},
		{"key": models.RoleViewer, "label": "Baxış (yalnız oxu)"},
	})
}
