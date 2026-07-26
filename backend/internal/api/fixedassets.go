package api

import (
	"time"

	"github.com/gin-gonic/gin"

	"oawo-muhasibat/internal/engine"
	"oawo-muhasibat/internal/models"
)

// ==================== FIXED ASSETS ====================

func (s *Server) ListFixedAssets(c *gin.Context) {
	var items []models.FixedAsset
	q := tdb(c).Order("code asc, id asc")
	if st := c.Query("status"); st != "" {
		q = q.Where("status = ?", st)
	}
	q.Find(&items)
	c.JSON(200, items)
}

func (s *Server) GetFixedAsset(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var a models.FixedAsset
	if err := tdb(c).First(&a, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Əsas vəsait tapılmadı"})
		return
	}
	c.JSON(200, a)
}

func (s *Server) CreateFixedAsset(c *gin.Context) {
	var a models.FixedAsset
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	if a.AcquisitionDate.Time.IsZero() {
		a.AcquisitionDate = models.Date{Time: time.Now()}
	}
	a.ID = 0
	a.MonthsDepreciated = 0
	a.AccumulatedDepreciation = 0
	a.BookValue = a.Cost
	a.Status = "active"
	if a.Method == "" {
		a.Method = "straight_line"
	}
	if err := tdb(c).Create(&a).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Yaradıla bilmədi: " + err.Error()})
		return
	}
	c.JSON(201, a)
}

func (s *Server) UpdateFixedAsset(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var a models.FixedAsset
	if err := tdb(c).First(&a, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	var in models.FixedAsset
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	// Metadata + accounts are always editable; cost/life only before depreciation.
	a.Code, a.Name, a.Category, a.Notes = in.Code, in.Name, in.Category, in.Notes
	a.AssetAccountID, a.AccumAccountID, a.ExpenseAccountID = in.AssetAccountID, in.AccumAccountID, in.ExpenseAccountID
	if a.MonthsDepreciated == 0 {
		a.AcquisitionDate = in.AcquisitionDate
		a.Cost = in.Cost
		a.SalvageValue = in.SalvageValue
		a.UsefulLifeMonths = in.UsefulLifeMonths
		a.BookValue = in.Cost
	}
	tdb(c).Save(&a)
	c.JSON(200, a)
}

func (s *Server) DeleteFixedAsset(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var n int64
	tdb(c).Model(&models.DepreciationEntry{}).Where("asset_id = ?", id).Count(&n)
	if n > 0 {
		c.JSON(400, gin.H{"detail": "Amortizasiya yazılışları var, silinə bilməz"})
		return
	}
	tdb(c).Delete(&models.FixedAsset{}, id)
	c.JSON(200, gin.H{"message": "Silindi"})
}

func (s *Server) ListDepreciation(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var items []models.DepreciationEntry
	tdb(c).Where("asset_id = ?", id).Order("period asc").Find(&items)
	c.JSON(200, items)
}

// RunDepreciationHandler posts depreciation due up to the given date (default now).
func (s *Server) RunDepreciationHandler(c *gin.Context) {
	asOf := time.Now()
	if d := parseDate(c.Query("as_of")); d != nil {
		asOf = *d
	}
	res, err := engine.RunDepreciation(tdb(c), asOf)
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, res)
}
