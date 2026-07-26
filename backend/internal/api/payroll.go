package api

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"oawo-muhasibat/internal/engine"
	"oawo-muhasibat/internal/models"
)

// ==================== EMPLOYEES ====================

func (s *Server) ListEmployees(c *gin.Context) {
	var items []models.Employee
	tdb(c).Order("full_name asc").Find(&items)
	c.JSON(200, items)
}
func (s *Server) GetEmployee(c *gin.Context)    { s.tenantGet(c, &models.Employee{}) }
func (s *Server) CreateEmployee(c *gin.Context) { s.tenantCreate(c, &models.Employee{}) }
func (s *Server) UpdateEmployee(c *gin.Context) { s.tenantUpdate(c, &models.Employee{}) }
func (s *Server) DeleteEmployee(c *gin.Context) { s.tenantDelete(c, &models.Employee{}) }

// ==================== PAYROLL CONFIG ====================

func (s *Server) GetPayrollConfig(c *gin.Context) {
	c.JSON(200, engine.LoadPayrollConfig(tdb(c)))
}

func (s *Server) UpdatePayrollConfig(c *gin.Context) {
	var cfg engine.PayrollConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	b, _ := json.Marshal(cfg)
	tdb(c).Save(&models.Setting{Key: "payroll", Value: string(b), UpdatedAt: time.Now()})
	c.JSON(200, cfg)
}

// ==================== PAYROLL RUNS ====================

func (s *Server) ListPayrollRuns(c *gin.Context) {
	var items []models.PayrollRun
	tdb(c).Order("period desc, id desc").Find(&items)
	c.JSON(200, items)
}

func (s *Server) GetPayrollRun(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var run models.PayrollRun
	if err := tdb(c).Preload("Lines").First(&run, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Hesablama tapılmadı"})
		return
	}
	c.JSON(200, run)
}

func (s *Server) CreatePayrollRun(c *gin.Context) {
	period := time.Now()
	if d := parseDate(c.Query("period")); d != nil {
		period = *d
	}
	// Normalise to the first day of the month.
	period = time.Date(period.Year(), period.Month(), 1, 0, 0, 0, 0, time.UTC)
	run, err := engine.CreatePayrollRun(tdb(c), period)
	if err != nil {
		c.JSON(400, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(201, run)
}

func (s *Server) PostPayrollRun(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	run, err := engine.PostPayrollRun(tdb(c), id)
	if err != nil {
		c.JSON(400, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, run)
}

func (s *Server) DeletePayrollRun(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var run models.PayrollRun
	if err := tdb(c).First(&run, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	if run.Status == "posted" {
		c.JSON(400, gin.H{"detail": "Kitablaşdırılmış hesablama silinə bilməz"})
		return
	}
	tdb(c).Where("run_id = ?", id).Delete(&models.PayrollLine{})
	tdb(c).Delete(&models.PayrollRun{}, id)
	c.JSON(200, gin.H{"message": "Silindi"})
}

// ---- small tenant-scoped generic helpers (mirror the platform ones) ----

func (s *Server) tenantGet(c *gin.Context, model interface{}) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	if err := tdb(c).First(model, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	c.JSON(200, model)
}
func (s *Server) tenantCreate(c *gin.Context, model interface{}) {
	if err := c.ShouldBindJSON(model); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	if err := tdb(c).Create(model).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Yaradıla bilmədi: " + err.Error()})
		return
	}
	c.JSON(201, model)
}
func (s *Server) tenantUpdate(c *gin.Context, model interface{}) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	if err := tdb(c).First(model, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)
	delete(updates, "id")
	delete(updates, "created_at")
	tdb(c).Model(model).Updates(updates)
	tdb(c).First(model, id)
	c.JSON(200, model)
}
func (s *Server) tenantDelete(c *gin.Context, model interface{}) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	if err := tdb(c).Delete(model, id).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Silinə bilmədi"})
		return
	}
	c.JSON(200, gin.H{"message": "Silindi"})
}

var _ = strconv.Atoi
