package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"oawo-muhasibat/internal/models"
)

// EInvoiceConfig is stored in tenant settings (key "einvoice").
type EInvoiceConfig struct {
	SellerVoen  string `json:"seller_voen"`
	CompanyName string `json:"company_name"`
	Mode        string `json:"mode"`     // manual | test | live
	Endpoint    string `json:"endpoint"` // rəsmi e-taxes API ünvanı (live)
	Token       string `json:"token"`
}

func (s *Server) loadEInvoiceConfig(c *gin.Context) EInvoiceConfig {
	var cfg EInvoiceConfig
	var st models.Setting
	if tdb(c).Where("key = ?", "einvoice").First(&st).Error == nil && st.Value != "" {
		json.Unmarshal([]byte(st.Value), &cfg)
	}
	return cfg
}

func (s *Server) GetEInvoiceConfig(c *gin.Context) { c.JSON(200, s.loadEInvoiceConfig(c)) }

func (s *Server) UpdateEInvoiceConfig(c *gin.Context) {
	var cfg EInvoiceConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(400, gin.H{"detail": "Yanlış məlumat"})
		return
	}
	b, _ := json.Marshal(cfg)
	tdb(c).Save(&models.Setting{Key: "einvoice", Value: string(b), UpdatedAt: time.Now()})
	c.JSON(200, cfg)
}

// ListEInvoices returns e-invoices enriched with their invoice number.
func (s *Server) ListEInvoices(c *gin.Context) {
	var items []models.EInvoice
	tdb(c).Order("id desc").Find(&items)
	// Attach the source document number for display.
	docNums := map[uint]string{}
	var docs []models.Document
	tdb(c).Where("type = ?", "sales_invoice").Find(&docs)
	partners := map[uint]string{}
	var ps []models.Partner
	tdb(c).Find(&ps)
	for _, p := range ps {
		partners[p.ID] = p.Name
	}
	docMeta := map[uint]models.Document{}
	for _, d := range docs {
		docNums[d.ID] = d.Number
		docMeta[d.ID] = d
	}
	out := make([]gin.H, 0, len(items))
	for _, e := range items {
		d := docMeta[e.DocumentID]
		buyer := ""
		if d.PartnerID != nil {
			buyer = partners[*d.PartnerID]
		}
		out = append(out, gin.H{"einvoice": e, "invoice_number": docNums[e.DocumentID], "buyer": buyer, "date": d.Date})
	}
	c.JSON(200, out)
}

// ListEligible returns posted sales invoices that have no e-invoice yet.
func (s *Server) ListEligibleInvoices(c *gin.Context) {
	var docs []models.Document
	tdb(c).Preload("Lines").
		Where("type = ? AND status IN ?", "sales_invoice", []string{"posted", "paid"}).
		Where("id NOT IN (?)", tdb(c).Model(&models.EInvoice{}).Select("document_id")).
		Order("date desc").Find(&docs)
	c.JSON(200, docs)
}

// buildPayload assembles the official e-invoice data from a sales invoice.
func (s *Server) buildPayload(c *gin.Context, doc models.Document, cfg EInvoiceConfig) (gin.H, string) {
	var partner models.Partner
	buyerVoen, buyerName := "", ""
	if doc.PartnerID != nil {
		if tdb(c).First(&partner, *doc.PartnerID).Error == nil {
			buyerVoen, buyerName = partner.TaxID, partner.Name
		}
	}
	items := make([]gin.H, 0, len(doc.Lines))
	for _, l := range doc.Lines {
		items = append(items, gin.H{
			"name": l.Description, "quantity": l.Quantity, "unit_price": l.UnitPrice,
			"vat_rate": l.TaxRate, "net": l.LineTotal, "vat": l.TaxAmount, "total": l.LineTotal + l.TaxAmount,
		})
	}
	payload := gin.H{
		"seller": gin.H{"voen": cfg.SellerVoen, "name": cfg.CompanyName},
		"buyer":  gin.H{"voen": buyerVoen, "name": buyerName},
		"invoice": gin.H{
			"internal_number": doc.Number,
			"date":            doc.Date.Time.Format("2006-01-02"),
		},
		"items":    items,
		"subtotal": doc.Subtotal,
		"vat":      doc.TaxTotal,
		"total":    doc.Total,
	}
	return payload, buyerVoen
}

// CreateEInvoice builds a draft e-invoice from a sales invoice.
func (s *Server) CreateEInvoice(c *gin.Context) {
	var req struct {
		DocumentID uint `json:"document_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.DocumentID == 0 {
		c.JSON(400, gin.H{"detail": "Sənəd seçilməlidir"})
		return
	}
	var doc models.Document
	if err := tdb(c).Preload("Lines").First(&doc, req.DocumentID).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Faktura tapılmadı"})
		return
	}
	if doc.Type != "sales_invoice" {
		c.JSON(400, gin.H{"detail": "Yalnız satış fakturası üçün e-qaimə yaradıla bilər"})
		return
	}
	var exists int64
	tdb(c).Model(&models.EInvoice{}).Where("document_id = ?", doc.ID).Count(&exists)
	if exists > 0 {
		c.JSON(400, gin.H{"detail": "Bu faktura üçün e-qaimə artıq var"})
		return
	}
	cfg := s.loadEInvoiceConfig(c)
	payload, buyerVoen := s.buildPayload(c, doc, cfg)
	pj, _ := json.Marshal(payload)
	e := models.EInvoice{
		DocumentID: doc.ID, Status: "draft", SellerVoen: cfg.SellerVoen, BuyerVoen: buyerVoen,
		Total: doc.Total, TaxTotal: doc.TaxTotal, Payload: pj,
	}
	if err := tdb(c).Create(&e).Error; err != nil {
		c.JSON(500, gin.H{"detail": "Yaradıla bilmədi"})
		return
	}
	c.JSON(201, e)
}

func (s *Server) GetEInvoice(c *gin.Context) { s.tenantGet(c, &models.EInvoice{}) }

// UpdateEInvoice records the official series/number/status (manual or after send).
func (s *Server) UpdateEInvoice(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var e models.EInvoice
	if err := tdb(c).First(&e, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	var in struct {
		Series, Number, Status, ExternalID, Note string
	}
	c.ShouldBindJSON(&in)
	if in.Series != "" {
		e.Series = in.Series
	}
	if in.Number != "" {
		e.Number = in.Number
	}
	if in.Status != "" {
		e.Status = in.Status
	}
	if in.ExternalID != "" {
		e.ExternalID = in.ExternalID
	}
	e.Note = in.Note
	tdb(c).Save(&e)
	c.JSON(200, e)
}

// SendEInvoice submits the e-invoice. In "live" mode it POSTs the payload to
// the configured e-taxes endpoint and records the portal's response. In "test"
// mode it simulates a successful registration so the whole flow can be
// demonstrated/trained without state credentials.
func (s *Server) SendEInvoice(c *gin.Context) {
	id, ok := s.bindID(c)
	if !ok {
		return
	}
	var e models.EInvoice
	if err := tdb(c).First(&e, id).Error; err != nil {
		c.JSON(404, gin.H{"detail": "Tapılmadı"})
		return
	}
	cfg := s.loadEInvoiceConfig(c)
	now := models.Date{Time: time.Now()}

	// Live: real submission to the portal.
	if cfg.Mode == "live" || (cfg.Mode == "" && cfg.Endpoint != "") {
		if cfg.Endpoint == "" {
			c.JSON(400, gin.H{"detail": "Real inteqrasiya üçün endpoint təyin edilməlidir."})
			return
		}
		httpReq, _ := http.NewRequest("POST", cfg.Endpoint, bytes.NewReader(e.Payload))
		httpReq.Header.Set("Content-Type", "application/json")
		if cfg.Token != "" {
			httpReq.Header.Set("Authorization", "Bearer "+cfg.Token)
		}
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(httpReq)
		if err != nil {
			c.JSON(502, gin.H{"detail": "Göndərilmə alınmadı: " + err.Error()})
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		e.Status = "sent"
		e.SentAt = &now
		e.Note = string(body)
		// Try to pick up the portal's registration id/series/number.
		var pr struct {
			ID, ExternalID, Number, Series, RegistrationNumber string
		}
		if json.Unmarshal(body, &pr) == nil {
			if pr.ExternalID != "" {
				e.ExternalID = pr.ExternalID
			} else if pr.ID != "" {
				e.ExternalID = pr.ID
			}
			if pr.Series != "" {
				e.Series = pr.Series
			}
			if pr.RegistrationNumber != "" {
				e.Number = pr.RegistrationNumber
			} else if pr.Number != "" {
				e.Number = pr.Number
			}
			if resp.StatusCode < 300 && e.Number != "" {
				e.Status = "registered"
			}
		}
		tdb(c).Save(&e)
		c.JSON(resp.StatusCode, gin.H{"status": e.Status, "response": string(body), "einvoice": e})
		return
	}

	// Test / simulation: mint a registration reference locally.
	if cfg.Mode == "test" {
		e.Status = "registered"
		e.SentAt = &now
		if e.Series == "" {
			e.Series = "TEST"
		}
		if e.Number == "" {
			e.Number = fmt.Sprintf("%s%06d", now.Time.Format("060102"), e.ID)
		}
		e.ExternalID = fmt.Sprintf("SIM-%d", e.ID)
		e.Note = "Test rejimi — simulyasiya (rəsmi göndəriş deyil)"
		tdb(c).Save(&e)
		c.JSON(200, gin.H{"status": "registered", "test": true, "einvoice": e})
		return
	}

	// Manual mode: no automatic sending configured.
	c.JSON(400, gin.H{"detail": "Göndəriş rejimi seçilməyib. Parametrlərdən 'Test' və ya 'Real inteqrasiya' seçin, yaxud rəsmi nömrəni əl ilə daxil edin."})
}
