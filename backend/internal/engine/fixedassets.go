package engine

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"oawo-muhasibat/internal/models"
)

// ============================================================
// Fixed-asset depreciation (straight-line).
//   monthly = (cost - salvage) / useful_life_months
// Each run posts, per asset, a single balanced journal entry for all
// months due up to `asOf`:  Dr amortizasiya xərci  /  Cr yığılmış amortizasiya
// and records one DepreciationEntry per month.
// ============================================================

func monthIndex(t time.Time) int { return t.Year()*12 + int(t.Month()) - 1 }

// addMonthsStart returns the first day of the month `n` months after t's month.
func addMonthsStart(t time.Time, n int) time.Time {
	base := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	return base.AddDate(0, n, 0)
}

type DepreciationResult struct {
	AssetsProcessed int     `json:"assets_processed"`
	MonthsPosted    int     `json:"months_posted"`
	TotalAmount     float64 `json:"total_amount"`
	JournalsCreated int     `json:"journals_created"`
}

// RunDepreciation posts all depreciation due up to asOf across active assets.
func RunDepreciation(db *gorm.DB, asOf time.Time) (*DepreciationResult, error) {
	res := &DepreciationResult{}
	err := db.Transaction(func(tx *gorm.DB) error {
		var assets []models.FixedAsset
		tx.Where("status = ?", "active").Find(&assets)
		for i := range assets {
			a := &assets[i]
			if a.UsefulLifeMonths <= 0 || a.ExpenseAccountID == 0 || a.AccumAccountID == 0 {
				continue
			}
			base := round2(a.Cost - a.SalvageValue)
			if base <= 0 {
				continue
			}
			monthly := round2(base / float64(a.UsefulLifeMonths))

			// How many months have elapsed since acquisition (inclusive of the
			// acquisition month), capped at the useful life.
			elapsed := monthIndex(asOf) - monthIndex(a.AcquisitionDate.Time) + 1
			if elapsed < 0 {
				elapsed = 0
			}
			target := elapsed
			if target > a.UsefulLifeMonths {
				target = a.UsefulLifeMonths
			}
			pending := target - a.MonthsDepreciated
			if pending <= 0 {
				continue
			}

			var runTotal float64
			var entries []models.DepreciationEntry
			for k := 0; k < pending; k++ {
				monthNo := a.MonthsDepreciated + k // 0-based
				period := addMonthsStart(a.AcquisitionDate.Time, monthNo)
				amt := monthly
				// Final month absorbs rounding so the asset lands exactly on salvage.
				if monthNo == a.UsefulLifeMonths-1 {
					amt = round2(base - (a.AccumulatedDepreciation + runTotal))
				}
				if amt < 0 {
					amt = 0
				}
				runTotal = round2(runTotal + amt)
				entries = append(entries, models.DepreciationEntry{AssetID: a.ID, Period: models.Date{Time: period}, Amount: amt})
			}
			if runTotal <= 0 {
				continue
			}

			number, err := NextNumber(tx, "journal", "J-")
			if err != nil {
				return err
			}
			now := time.Now()
			entry := models.JournalEntry{
				Number:      number,
				Date:        models.Date{Time: asOf},
				Description: fmt.Sprintf("Amortizasiya: %s", a.Name),
				Reference:   a.Code,
				Status:      "posted",
				Source:      "depreciation",
				PostedAt:    &now,
				TotalDebit:  runTotal,
				TotalCredit: runTotal,
				Lines: []models.JournalLine{
					{AccountID: a.ExpenseAccountID, Description: "Amortizasiya xərci", Debit: runTotal, Posted: true, Date: models.Date{Time: asOf}},
					{AccountID: a.AccumAccountID, Description: "Yığılmış amortizasiya", Credit: runTotal, Posted: true, Date: models.Date{Time: asOf}},
				},
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
			for j := range entries {
				entries[j].JournalID = &entry.ID
				if err := tx.Create(&entries[j]).Error; err != nil {
					return err
				}
			}

			a.MonthsDepreciated = target
			a.AccumulatedDepreciation = round2(a.AccumulatedDepreciation + runTotal)
			a.BookValue = round2(a.Cost - a.AccumulatedDepreciation)
			if a.MonthsDepreciated >= a.UsefulLifeMonths {
				a.Status = "fully_depreciated"
			}
			if err := tx.Save(a).Error; err != nil {
				return err
			}

			res.AssetsProcessed++
			res.MonthsPosted += pending
			res.TotalAmount = round2(res.TotalAmount + runTotal)
			res.JournalsCreated++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}
