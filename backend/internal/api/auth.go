package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"oawo-muhasibat/internal/models"
)

// Lightweight self-contained token: base64(payload).base64(hmac).
// Avoids an external JWT dependency while staying tamper-proof.

type tokenPayload struct {
	UserID uint   `json:"uid"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp"`
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

// Login authenticates and returns a token.
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
	tok := signToken(tokenPayload{UserID: u.ID, Email: u.Email, Role: u.Role,
		Exp: time.Now().Add(30 * 24 * time.Hour).Unix()})
	c.JSON(200, gin.H{"token": tok, "user": u})
}

// Me returns the current user.
func (s *Server) Me(c *gin.Context) {
	uid := c.GetUint("userID")
	var u models.User
	if err := s.DB.First(&u, uid).Error; err != nil {
		c.JSON(404, gin.H{"detail": "İstifadəçi tapılmadı"})
		return
	}
	c.JSON(200, u)
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

// AuthMiddleware validates the bearer token.
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
		c.Set("role", p.Role)
		c.Next()
	}
}
