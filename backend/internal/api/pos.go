package api

import (
	"github.com/gin-gonic/gin"

	"oawo-muhasibat/internal/engine"
	"oawo-muhasibat/internal/models"
)

// ==================== REGISTERS (kassalar) ====================

func (s *Server) ListRegisters(c *gin.Context) {
	var items []models.Register
	tdb(c).Order("name asc").Find(&items)
	c.JSON(200, items)
}

func (s *Server) CreateRegister(c *gin.Context) {
	var r models.Register
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	// Sensible defaults: first warehouse + cash account.
	if r.WarehouseID == 0 {
		var w models.Warehouse
		if tdb(c).First(&w).Error == nil {
			r.WarehouseID = w.ID
		}
	}
	if r.CashAccountID == 0 {
		var a models.Account
		if tdb(c).Where("system_key = ?", "cash").First(&a).Error == nil {
			r.CashAccountID = a.ID
		}
	}
	r.Enabled = true
	if err := tdb(c).Create(&r).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Yaradıla bilmədi"})
		return
	}
	c.JSON(201, r)
}

func (s *Server) UpdateRegister(c *gin.Context) { s.tenantUpdate(c, &models.Register{}) }
func (s *Server) DeleteRegister(c *gin.Context) { s.tenantDelete(c, &models.Register{}) }

// GetOpenSession returns the register's current open shift (or null).
func (s *Server) GetOpenSession(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var sess models.PosSession
	if err := tdb(c).Where("register_id = ? AND status = ?", id, "open").First(&sess).Error; err != nil {
		c.JSON(200, nil)
		return
	}
	c.JSON(200, sess)
}

// ==================== SESSIONS (növbələr) ====================

func (s *Server) OpenPosSession(c *gin.Context) {
	var req struct {
		RegisterID  uint    `json:"register_id"`
		CashierName string  `json:"cashier_name"`
		OpeningCash float64 `json:"opening_cash"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.RegisterID == 0 {
		c.JSON(400, gin.H{"detail": "Kassa seçilməlidir"})
		return
	}
	sess, err := engine.OpenSession(tdb(c), req.RegisterID, req.CashierName, req.OpeningCash)
	if err != nil {
		c.JSON(400, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(201, sess)
}

func (s *Server) ClosePosSession(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var req struct {
		ClosingCash float64 `json:"closing_cash"`
	}
	c.ShouldBindJSON(&req)
	sess, err := engine.CloseSession(tdb(c), id, req.ClosingCash)
	if err != nil {
		c.JSON(400, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, sess)
}

func (s *Server) ListPosSessions(c *gin.Context) {
	var items []models.PosSession
	tdb(c).Order("id desc").Limit(200).Find(&items)
	c.JSON(200, items)
}

func (s *Server) GetPosSession(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var sess models.PosSession
	if err := tdb(c).First(&sess, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Növbə tapılmadı"})
		return
	}
	var sales []models.PosSale
	tdb(c).Preload("Lines").Where("session_id = ?", id).Order("id desc").Find(&sales)
	c.JSON(200, gin.H{"session": sess, "sales": sales})
}

// ==================== SALE ====================

func (s *Server) CompletePosSale(c *gin.Context) {
	var req struct {
		SessionID     uint                   `json:"session_id"`
		PaymentMethod string                 `json:"payment_method"`
		Paid          float64                `json:"paid"`
		Lines         []engine.SaleLineInput `json:"lines"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = "cash"
	}
	sale, err := engine.CompleteSale(tdb(c), req.SessionID, req.PaymentMethod, req.Paid, req.Lines)
	if err != nil {
		c.JSON(400, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(201, sale)
}
