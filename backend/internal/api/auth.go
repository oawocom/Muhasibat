package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"oawo-muhasibat/internal/models"
)

// Self-contained token: base64(payload).base64(hmac). No external JWT dep.
type tokenPayload struct {
	UserID     uint   `json:"uid"`
	Email      string `json:"email"`
	Superadmin bool   `json:"sa"`
	Exp        int64  `json:"exp"`
}

func secret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "oawo-muhasibat-dev-secret-change-me"
	}
	return []byte(s)
}

func signToken(p tokenPayload) string {
	body, _ := json.Marshal(p)
	b64 := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, secret())
	mac.Write([]byte(b64))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return b64 + "." + sig
}

func parseToken(tok string) (*tokenPayload, error) {
	parts := strings.SplitN(tok, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("format")
	}
	mac := hmac.New(sha256.New, secret())
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, errors.New("imza yanlışdır")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var p tokenPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, err
	}
	if time.Now().Unix() > p.Exp {
		return nil, errors.New("token vaxtı bitib")
	}
	return &p, nil
}

// Login authenticates a platform user.
func (s *Server) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış sorğu"})
		return
	}
	var u models.User
	if err := s.DB.Where("email = ? AND enabled = ?", strings.ToLower(strings.TrimSpace(req.Email)), true).First(&u).Error; err != nil {
		c.JSON(401, gin.H{"detail": "Email və ya şifrə yanlışdır"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
		c.JSON(401, gin.H{"detail": "Email və ya şifrə yanlışdır"})
		return
	}
	tok := signToken(tokenPayload{UserID: u.ID, Email: u.Email, Superadmin: u.IsSuperadmin,
		Exp: time.Now().Add(30 * 24 * time.Hour).Unix()})
	resp := s.meView(u)
	resp["token"] = tok
	c.JSON(200, resp)
}

// companyView is a company plus the caller's role and the tenant's subscribed
// modules (which gate the UI). Subscription price is never included here.
type companyView struct {
	models.Company
	Role    string   `json:"role"`
	Modules []string `json:"modules"`
}

// tenantModules returns a tenant's subscribed module keys (cached-free lookup).
func (s *Server) tenantModules(tenantID uint) []string {
	var t models.Tenant
	if err := s.DB.First(&t, tenantID).Error; err != nil {
		return models.DefaultEnabledModules()
	}
	return t.Modules()
}

func (s *Server) userCompanies(u models.User) []companyView {
	out := []companyView{}
	add := func(cmp models.Company, role string) {
		if !cmp.Enabled {
			return
		}
		out = append(out, companyView{Company: cmp, Role: role, Modules: s.tenantModules(cmp.TenantID)})
	}
	if u.IsSuperadmin {
		var companies []models.Company
		s.DB.Where("enabled = ?", true).Order("tenant_id asc, name asc").Find(&companies)
		for _, cmp := range companies {
			add(cmp, models.RoleOwner)
		}
		return out
	}
	if u.IsTenantAdmin && u.TenantID != nil {
		var companies []models.Company
		s.DB.Where("tenant_id = ? AND enabled = ?", *u.TenantID, true).Order("name asc").Find(&companies)
		for _, cmp := range companies {
			add(cmp, models.RoleOwner)
		}
		return out
	}
	var memberships []models.Membership
	s.DB.Where("user_id = ? AND enabled = ?", u.ID, true).Find(&memberships)
	for _, m := range memberships {
		var cmp models.Company
		if err := s.DB.First(&cmp, m.CompanyID).Error; err == nil {
			add(cmp, m.Role)
		}
	}
	return out
}

// meView returns the user plus role flags and (for tenant users) their tenant.
func (s *Server) meView(u models.User) gin.H {
	h := gin.H{"user": u, "companies": s.userCompanies(u), "is_superadmin": u.IsSuperadmin, "is_tenant_admin": u.IsTenantAdmin}
	if u.TenantID != nil {
		var t models.Tenant
		if s.DB.First(&t, *u.TenantID).Error == nil {
			// Tenant users never see the subscription amount.
			h["tenant"] = gin.H{"id": t.ID, "name": t.Name, "plan": t.Plan, "modules": t.Modules()}
		}
	}
	return h
}

// Me returns the current user with their companies.
func (s *Server) Me(c *gin.Context) {
	var u models.User
	if err := s.DB.First(&u, c.GetUint("userID")).Error; err != nil {
		c.JSON(404, gin.H{"detail": "İstifadəçi tapılmadı"})
		return
	}
	c.JSON(200, s.meView(u))
}

// ChangePassword updates the current user's password.
func (s *Server) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.NewPassword) < 6 {
		c.JSON(400, gin.H{"detail": "Yeni şifrə ən azı 6 simvol olmalıdır"})
		return
	}
	var u models.User
	if err := s.DB.First(&u, c.GetUint("userID")).Error; err != nil {
		c.JSON(404, gin.H{"detail": "İstifadəçi tapılmadı"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.OldPassword)) != nil {
		c.JSON(401, gin.H{"detail": "Köhnə şifrə yanlışdır"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	s.DB.Model(&u).Update("password", string(hash))
	c.JSON(200, gin.H{"message": "Şifrə yeniləndi"})
}

// AuthMiddleware validates the bearer token (platform-level).
func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.JSON(401, gin.H{"detail": "Avtorizasiya tələb olunur"})
			c.Abort()
			return
		}
		p, err := parseToken(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			c.JSON(401, gin.H{"detail": fmt.Sprintf("Token yanlışdır: %v", err)})
			c.Abort()
			return
		}
		c.Set("userID", p.UserID)
		c.Set("isSuperadmin", p.Superadmin)
		c.Next()
	}
}

// TenantMiddleware resolves the active company from the X-Company-ID header,
// verifies the caller's membership/role and attaches the tenant DB.
func (s *Server) TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		companyID, err := strconv.ParseUint(c.GetHeader("X-Company-ID"), 10, 64)
		if err != nil || companyID == 0 {
			c.JSON(400, gin.H{"detail": "Şirkət seçilməyib (X-Company-ID)"})
			c.Abort()
			return
		}
		var company models.Company
		if err := s.DB.First(&company, uint(companyID)).Error; err != nil || !company.Enabled {
			c.JSON(404, gin.H{"detail": "Şirkət tapılmadı"})
			c.Abort()
			return
		}

		role := ""
		if c.GetBool("isSuperadmin") {
			role = models.RoleOwner
		} else {
			var u models.User
			if err := s.DB.First(&u, c.GetUint("userID")).Error; err != nil {
				c.JSON(401, gin.H{"detail": "İstifadəçi tapılmadı"})
				c.Abort()
				return
			}
			if u.IsTenantAdmin && u.TenantID != nil && company.TenantID == *u.TenantID {
				role = models.RoleOwner // tenant admin: full access to all tenant companies
			} else {
				var m models.Membership
				if err := s.DB.Where("user_id = ? AND company_id = ? AND enabled = ?",
					u.ID, company.ID, true).First(&m).Error; err != nil {
					c.JSON(403, gin.H{"detail": "Bu şirkətə girişiniz yoxdur"})
					c.Abort()
					return
				}
				role = m.Role
			}
		}

		if !company.Provisioned {
			c.JSON(409, gin.H{"detail": "Şirkət hələ hazırlanır"})
			c.Abort()
			return
		}
		tenantDB, err := s.Mgr.Get(company.ID, company.DBName)
		if err != nil {
			c.JSON(500, gin.H{"detail": "Şirkət bazasına qoşulmaq mümkün olmadı"})
			c.Abort()
			return
		}

		// Enforce read-only for viewers on mutating requests.
		if c.Request.Method != "GET" && c.Request.Method != "OPTIONS" && !models.CanWrite(role) {
			c.JSON(403, gin.H{"detail": "Bu əməliyyat üçün icazəniz yoxdur (yalnız baxış)"})
			c.Abort()
			return
		}

		c.Set("tdb", tenantDB)
		c.Set("role", role)
		c.Set("companyID", company.ID)
		c.Next()
	}
}

// tdb returns the request-scoped tenant database.
func tdb(c *gin.Context) *gorm.DB {
	return c.MustGet("tdb").(*gorm.DB)
}
