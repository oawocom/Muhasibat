package engine

import (
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"

	"oawo-muhasibat/internal/models"
)

// round2 rounds money to 2 decimals (half away from zero).
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// ============================================================
// Numbering — atomic per-key document/journal sequences.
// ============================================================

func NextNumber(db *gorm.DB, key, prefix string) (string, error) {
	var number string
	err := db.Transaction(func(tx *gorm.DB) error {
		var seq models.Sequence
		if err := tx.Where("key = ?", key).First(&seq).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				seq = models.Sequence{Key: key, Prefix: prefix, Next: 1, Padding: 5}
				if err := tx.Create(&seq).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
		n := seq.Next
		number = fmt.Sprintf("%s%0*d", seq.Prefix, seq.Padding, n)
		seq.Next = n + 1
		seq.UpdatedAt = time.Now()
		return tx.Save(&seq).Error
	})
	return number, err
}

// ============================================================
// Journal posting — the double-entry guarantee.
// ============================================================

// ValidateBalanced ensures total debit == total credit and there is at
// least one non-zero line. Returns totals rounded to 2 dp.
func ValidateBalanced(lines []models.JournalLine) (debit, credit float64, err error) {
	for _, l := range lines {
		if l.Debit < 0 || l.Credit < 0 {
			return 0, 0, errors.New("mənfi məbləğə icazə yoxdur")
		}
		if l.Debit > 0 && l.Credit > 0 {
			return 0, 0, errors.New("bir sətir eyni anda həm debet, həm kredit ola bilməz")
		}
		if l.AccountID == 0 {
			return 0, 0, errors.New("hesab seçilməlidir")
		}
		debit += l.Debit
		credit += l.Credit
	}
	debit, credit = round2(debit), round2(credit)
	if debit == 0 && credit == 0 {
		return 0, 0, errors.New("boş yazılış: ən azı bir sətir olmalıdır")
	}
	if math.Abs(debit-credit) > 0.005 {
		return 0, 0, fmt.Errorf("balanssız yazılış: debet %.2f ≠ kredit %.2f", debit, credit)
	}
	return debit, credit, nil
}

// PostEntry validates and posts a draft journal entry inside a transaction.
func PostEntry(db *gorm.DB, entryID uint) (*models.JournalEntry, error) {
	var entry models.JournalEntry
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Lines").First(&entry, entryID).Error; err != nil {
			return err
		}
		if entry.Status == "posted" {
			return errors.New("yazılış artıq təsdiqlənib")
		}
		if entry.Status == "void" {
			return errors.New("ləğv edilmiş yazılış təsdiqlənə bilməz")
		}
		debit, credit, verr := ValidateBalanced(entry.Lines)
		if verr != nil {
			return verr
		}
		now := time.Now()
		entry.Status = "posted"
		entry.TotalDebit = debit
		entry.TotalCredit = credit
		entry.PostedAt = &now
		if err := tx.Save(&entry).Error; err != nil {
			return err
		}
		// Denormalise date/posted onto lines for fast reporting.
		if err := tx.Model(&models.JournalLine{}).Where("entry_id = ?", entry.ID).
			Updates(map[string]interface{}{"posted": true, "date": entry.Date}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	db.Preload("Lines").First(&entry, entryID)
	return &entry, nil
}

// UnpostEntry reverses a posting back to draft (e.g. correction before close).
func UnpostEntry(db *gorm.DB, entryID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var entry models.JournalEntry
		if err := tx.First(&entry, entryID).Error; err != nil {
			return err
		}
		if entry.Status != "posted" {
			return errors.New("yalnız təsdiqlənmiş yazılış geri qaytarıla bilər")
		}
		entry.Status = "draft"
		entry.PostedAt = nil
		if err := tx.Save(&entry).Error; err != nil {
			return err
		}
		return tx.Model(&models.JournalLine{}).Where("entry_id = ?", entry.ID).
			Update("posted", false).Error
	})
}

// ============================================================
// System accounts — resolved by SystemKey for document posting.
// ============================================================

func systemAccount(db *gorm.DB, key string) (*models.Account, error) {
	var acc models.Account
	if err := db.Where("system_key = ?", key).First(&acc).Error; err != nil {
		return nil, fmt.Errorf("sistem hesabı tapılmadı: %s (hesablar planını seed edin)", key)
	}
	return &acc, nil
}

// ============================================================
// Document posting — turns a business document into a balanced entry.
// ============================================================

func PostDocument(db *gorm.DB, docID uint) (*models.Document, error) {
	var doc models.Document
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Lines").First(&doc, docID).Error; err != nil {
			return err
		}
		if doc.Status == "posted" || doc.Status == "paid" {
			return errors.New("sənəd artıq təsdiqlənib")
		}
		if doc.Status == "void" {
			return errors.New("ləğv edilmiş sənəd təsdiqlənə bilməz")
		}

		recompute(&doc)

		lines, err := buildDocumentLines(tx, &doc)
		if err != nil {
			return err
		}

		number, err := NextNumber(tx, "journal", "J-")
		if err != nil {
			return err
		}
		now := time.Now()
		entry := models.JournalEntry{
			Number:      number,
			Date:        doc.Date,
			Description: docDescription(&doc),
			Reference:   doc.Number,
			Status:      "posted",
			Source:      "document",
			DocumentID:  &doc.ID,
			PostedAt:    &now,
			Lines:       lines,
		}
		debit, credit, verr := ValidateBalanced(entry.Lines)
		if verr != nil {
			return verr
		}
		entry.TotalDebit, entry.TotalCredit = debit, credit
		for i := range entry.Lines {
			entry.Lines[i].Posted = true
			entry.Lines[i].Date = doc.Date
		}
		if err := tx.Create(&entry).Error; err != nil {
			return err
		}

		// Inventory movements for stock-tracked documents.
		if err := applyStock(tx, &doc); err != nil {
			return err
		}

		doc.JournalID = &entry.ID
		doc.Status = "posted"
		if err := tx.Save(&doc).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	db.Preload("Lines").First(&doc, docID)
	return &doc, nil
}

// recompute recalculates document totals from its lines.
func recompute(doc *models.Document) {
	var sub, tax float64
	for i := range doc.Lines {
		l := &doc.Lines[i]
		net := round2(l.Quantity * l.UnitPrice)
		l.LineTotal = net
		l.TaxAmount = round2(net * l.TaxRate / 100)
		sub += net
		tax += l.TaxAmount
	}
	doc.Subtotal = round2(sub)
	doc.TaxTotal = round2(tax)
	doc.Total = round2(sub + tax)
}

func docDescription(doc *models.Document) string {
	switch doc.Type {
	case "sales_invoice":
		return "Satış qaimə-fakturası " + doc.Number
	case "purchase_invoice":
		return "Alış qaimə-fakturası " + doc.Number
	case "receipt":
		return "Müştəridən mədaxil " + doc.Number
	case "payment":
		return "Təchizatçıya ödəniş " + doc.Number
	}
	return doc.Number
}

// buildDocumentLines encodes the posting rules per document type.
func buildDocumentLines(db *gorm.DB, doc *models.Document) ([]models.JournalLine, error) {
	line := func(accID uint, partner *uint, desc string, dr, cr float64) models.JournalLine {
		return models.JournalLine{AccountID: accID, PartnerID: partner, Description: desc,
			Debit: round2(dr), Credit: round2(cr)}
	}

	switch doc.Type {
	case "sales_invoice":
		ar, err := systemAccount(db, "ar")
		if err != nil {
			return nil, err
		}
		sales, err := systemAccount(db, "sales")
		if err != nil {
			return nil, err
		}
		var lines []models.JournalLine
		lines = append(lines, line(ar.ID, doc.PartnerID, "Müştəri borcu", doc.Total, 0))
		lines = append(lines, line(sales.ID, doc.PartnerID, "Satışdan gəlir", 0, doc.Subtotal))
		if doc.TaxTotal > 0 {
			vat, err := systemAccount(db, "vat_out")
			if err != nil {
				return nil, err
			}
			lines = append(lines, line(vat.ID, doc.PartnerID, "ƏDV (satış)", 0, doc.TaxTotal))
		}
		// Cost of goods sold for stock-tracked lines.
		if cogsAmt, invAmt := stockCost(db, doc); cogsAmt > 0 {
			cogs, e1 := systemAccount(db, "cogs")
			inv, e2 := systemAccount(db, "inventory")
			if e1 == nil && e2 == nil {
				lines = append(lines, line(cogs.ID, nil, "Satılan malın maya dəyəri", cogsAmt, 0))
				lines = append(lines, line(inv.ID, nil, "Anbardan silinmə", 0, invAmt))
			}
		}
		return lines, nil

	case "purchase_invoice":
		ap, err := systemAccount(db, "ap")
		if err != nil {
			return nil, err
		}
		var lines []models.JournalLine
		// Debit inventory (stock) or expense (services).
		targetKey := "inventory"
		if !anyStockLine(db, doc) {
			targetKey = "expense"
		}
		target, err := systemAccount(db, targetKey)
		if err != nil {
			return nil, err
		}
		lines = append(lines, line(target.ID, nil, "Alış (mal/xidmət)", doc.Subtotal, 0))
		if doc.TaxTotal > 0 {
			vat, err := systemAccount(db, "vat_in")
			if err != nil {
				return nil, err
			}
			lines = append(lines, line(vat.ID, nil, "ƏDV (alış, əvəzləşdirilən)", doc.TaxTotal, 0))
		}
		lines = append(lines, line(ap.ID, doc.PartnerID, "Təchizatçı borcu", 0, doc.Total))
		return lines, nil

	case "receipt": // customer pays us
		cashID, err := cashAccount(db, doc)
		if err != nil {
			return nil, err
		}
		ar, err := systemAccount(db, "ar")
		if err != nil {
			return nil, err
		}
		return []models.JournalLine{
			line(cashID, doc.PartnerID, "Müştəridən mədaxil", doc.Total, 0),
			line(ar.ID, doc.PartnerID, "Müştəri borcunun bağlanması", 0, doc.Total),
		}, nil

	case "payment": // we pay supplier
		cashID, err := cashAccount(db, doc)
		if err != nil {
			return nil, err
		}
		ap, err := systemAccount(db, "ap")
		if err != nil {
			return nil, err
		}
		return []models.JournalLine{
			line(ap.ID, doc.PartnerID, "Təchizatçı borcunun ödənişi", doc.Total, 0),
			line(cashID, doc.PartnerID, "Kassadan/bankdan çıxış", 0, doc.Total),
		}, nil
	}
	return nil, fmt.Errorf("naməlum sənəd tipi: %s", doc.Type)
}

func cashAccount(db *gorm.DB, doc *models.Document) (uint, error) {
	if doc.CashAccountID != nil && *doc.CashAccountID != 0 {
		return *doc.CashAccountID, nil
	}
	acc, err := systemAccount(db, "cash")
	if err != nil {
		return 0, err
	}
	return acc.ID, nil
}

func anyStockLine(db *gorm.DB, doc *models.Document) bool {
	for _, l := range doc.Lines {
		if l.ProductID != nil {
			var p models.Product
			if err := db.First(&p, *l.ProductID).Error; err == nil && p.Type == "product" && p.TrackStock {
				return true
			}
		}
	}
	return false
}

// stockCost returns total COGS/inventory reduction for a sales document.
func stockCost(db *gorm.DB, doc *models.Document) (cogs float64, inv float64) {
	for _, l := range doc.Lines {
		if l.ProductID == nil {
			continue
		}
		var p models.Product
		if err := db.First(&p, *l.ProductID).Error; err != nil || p.Type != "product" || !p.TrackStock {
			continue
		}
		cogs += round2(l.Quantity * p.CostPrice)
	}
	cogs = round2(cogs)
	return cogs, cogs
}

func applyStock(db *gorm.DB, doc *models.Document) error {
	if doc.WarehouseID == nil {
		return nil
	}
	sign := 0.0
	switch doc.Type {
	case "purchase_invoice":
		sign = 1
	case "sales_invoice":
		sign = -1
	default:
		return nil
	}
	for _, l := range doc.Lines {
		if l.ProductID == nil {
			continue
		}
		var p models.Product
		if err := db.First(&p, *l.ProductID).Error; err != nil || p.Type != "product" || !p.TrackStock {
			continue
		}
		cost := p.CostPrice
		if doc.Type == "purchase_invoice" {
			cost = l.UnitPrice
		}
		mv := models.StockMove{
			ProductID:   *l.ProductID,
			WarehouseID: *doc.WarehouseID,
			Date:        doc.Date,
			Quantity:    round2(sign * l.Quantity),
			UnitCost:    round2(cost),
			DocumentID:  &doc.ID,
			DocType:     doc.Type,
		}
		if err := db.Create(&mv).Error; err != nil {
			return err
		}
	}
	return nil
}
