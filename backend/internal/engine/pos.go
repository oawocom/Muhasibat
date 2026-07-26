package engine

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"oawo-muhasibat/internal/models"
)

// ============================================================
// POS (kassa aparatı). A completed sale reuses the accounting engine:
// it posts a sales invoice (revenue + VAT + COGS + stock-out) and a receipt
// (cash/bank in, closing the receivable). So POS sales also appear in the
// sales-invoice list and all reports, fully double-entry.
// ============================================================

// OpenSession starts a shift on a register (one open shift per register).
func OpenSession(db *gorm.DB, registerID uint, cashier string, openingCash float64) (*models.PosSession, error) {
	var reg models.Register
	if err := db.First(&reg, registerID).Error; err != nil {
		return nil, errors.New("kassa tapılmadı")
	}
	var open int64
	db.Model(&models.PosSession{}).Where("register_id = ? AND status = ?", registerID, "open").Count(&open)
	if open > 0 {
		return nil, errors.New("bu kassada artıq açıq növbə var")
	}
	s := models.PosSession{RegisterID: registerID, CashierName: cashier, Status: "open",
		OpeningCash: round2(openingCash), OpenedAt: time.Now()}
	if err := db.Create(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// CloseSession ends a shift and finalises its Z-report totals.
func CloseSession(db *gorm.DB, sessionID uint, closingCash float64) (*models.PosSession, error) {
	var s models.PosSession
	if err := db.First(&s, sessionID).Error; err != nil {
		return nil, err
	}
	if s.Status == "closed" {
		return nil, errors.New("növbə artıq bağlıdır")
	}
	now := time.Now()
	s.Status = "closed"
	s.ClosingCash = round2(closingCash)
	s.ClosedAt = &now
	db.Save(&s)
	return &s, nil
}

type SaleLineInput struct {
	ProductID *uint   `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	TaxRate   float64 `json:"tax_rate"`
}

// CompleteSale posts a POS sale and returns the recorded PosSale.
func CompleteSale(db *gorm.DB, sessionID uint, method string, paid float64, lines []SaleLineInput) (*models.PosSale, error) {
	var session models.PosSession
	if err := db.First(&session, sessionID).Error; err != nil {
		return nil, errors.New("növbə tapılmadı")
	}
	if session.Status != "open" {
		return nil, errors.New("növbə bağlıdır — əvvəlcə yeni növbə açın")
	}
	if len(lines) == 0 {
		return nil, errors.New("səbət boşdur")
	}
	var reg models.Register
	if err := db.First(&reg, session.RegisterID).Error; err != nil {
		return nil, errors.New("kassa tapılmadı")
	}

	// Cash goes to the register's cash account; card to the bank account.
	cashAccID := reg.CashAccountID
	if method == "card" {
		if bank := accountBySystemKey(db, "bank"); bank != 0 {
			cashAccID = bank
		}
	}
	if cashAccID == 0 {
		cashAccID = accountBySystemKey(db, "cash")
	}

	// 1) Sales invoice (revenue/VAT/COGS/stock).
	docLines := make([]models.DocumentLine, 0, len(lines))
	for i, l := range lines {
		docLines = append(docLines, models.DocumentLine{
			ProductID: l.ProductID, Description: l.Name, Quantity: l.Quantity,
			UnitPrice: l.UnitPrice, TaxRate: l.TaxRate, SortOrder: i,
		})
	}
	now := models.Date{Time: time.Now()}
	whID := reg.WarehouseID
	inv := models.Document{Type: "sales_invoice", Date: now, FxRate: 1, Status: "draft", Lines: docLines}
	if whID != 0 {
		inv.WarehouseID = &whID
	}
	num, _ := NextNumber(db, "sales_invoice", "SF-")
	inv.Number = num
	if err := db.Create(&inv).Error; err != nil {
		return nil, err
	}
	if _, err := PostDocument(db, inv.ID); err != nil {
		return nil, err
	}
	db.First(&inv, inv.ID) // reload totals

	// 2) Receipt for the payment (closes the receivable).
	rNum, _ := NextNumber(db, "receipt", "MD-")
	rcpt := models.Document{
		Type: "receipt", Date: now, FxRate: 1, Status: "draft", Number: rNum,
		CashAccountID: &cashAccID,
		Lines:         []models.DocumentLine{{Description: "POS satış " + inv.Number, Quantity: 1, UnitPrice: inv.Total, TaxRate: 0}},
	}
	if err := db.Create(&rcpt).Error; err != nil {
		return nil, err
	}
	if _, err := PostDocument(db, rcpt.ID); err != nil {
		return nil, err
	}

	// 3) POS sale record.
	saleLines := make([]models.PosSaleLine, 0, len(lines))
	for _, l := range lines {
		net := round2(l.Quantity * l.UnitPrice)
		saleLines = append(saleLines, models.PosSaleLine{
			ProductID: l.ProductID, Name: l.Name, Quantity: l.Quantity, UnitPrice: l.UnitPrice,
			TaxRate: l.TaxRate, LineTotal: net, TaxAmount: round2(net * l.TaxRate / 100),
		})
	}
	if paid < inv.Total {
		paid = inv.Total
	}
	saleNum, _ := NextNumber(db, "pos_sale", "POS-")
	sale := models.PosSale{
		SessionID: session.ID, RegisterID: reg.ID, Number: saleNum, Date: now,
		Subtotal: inv.Subtotal, TaxTotal: inv.TaxTotal, Total: inv.Total,
		Paid: round2(paid), Change: round2(paid - inv.Total), PaymentMethod: method,
		InvoiceDocID: &inv.ID, ReceiptDocID: &rcpt.ID, Lines: saleLines,
	}
	if err := db.Create(&sale).Error; err != nil {
		return nil, err
	}

	// 4) Update the session running totals.
	session.SalesCount++
	session.SalesTotal = round2(session.SalesTotal + inv.Total)
	if method == "card" {
		session.CardTotal = round2(session.CardTotal + inv.Total)
	} else {
		session.CashTotal = round2(session.CashTotal + inv.Total)
	}
	db.Save(&session)

	db.Preload("Lines").First(&sale, sale.ID)
	return &sale, nil
}
