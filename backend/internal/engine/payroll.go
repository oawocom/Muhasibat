package engine

import (
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"oawo-muhasibat/internal/models"
)

// ============================================================
// Payroll — configurable rates (Azerbaijan defaults) so the calculation
// adapts when the law changes. All rates are percentages.
// ============================================================

type PayrollConfig struct {
	IncomeTaxExempt float64 `json:"income_tax_exempt"` // aylıq güzəşt (məs. 8000)
	IncomeTaxRate   float64 `json:"income_tax_rate"`   // güzəştdən yuxarı %
	DsmfEmp         float64 `json:"dsmf_emp"`
	DsmfEmpr        float64 `json:"dsmf_empr"`
	UnempEmp        float64 `json:"unemp_emp"`
	UnempEmpr       float64 `json:"unemp_empr"`
	MedicalEmp      float64 `json:"medical_emp"`
	MedicalEmpr     float64 `json:"medical_empr"`
	// Accounts.
	SalaryExpenseAccount    uint `json:"salary_expense_account"`
	WagesPayableAccount     uint `json:"wages_payable_account"`
	StatutoryPayableAccount uint `json:"statutory_payable_account"`
}

func defaultPayrollConfig() PayrollConfig {
	return PayrollConfig{
		IncomeTaxExempt: 8000, IncomeTaxRate: 14,
		DsmfEmp: 3, DsmfEmpr: 22,
		UnempEmp: 0.5, UnempEmpr: 0.5,
		MedicalEmp: 2, MedicalEmpr: 2,
	}
}

// LoadPayrollConfig reads saved config, fills defaults and resolves accounts.
func LoadPayrollConfig(db *gorm.DB) PayrollConfig {
	cfg := defaultPayrollConfig()
	var s models.Setting
	if err := db.Where("key = ?", "payroll").First(&s).Error; err == nil && s.Value != "" {
		var saved PayrollConfig
		if json.Unmarshal([]byte(s.Value), &saved) == nil {
			cfg = saved
		}
	}
	// Fallback to accounts by system key when not configured.
	if cfg.SalaryExpenseAccount == 0 {
		cfg.SalaryExpenseAccount = accountBySystemKey(db, "salary_expense")
	}
	if cfg.WagesPayableAccount == 0 {
		cfg.WagesPayableAccount = accountBySystemKey(db, "wages_payable")
	}
	if cfg.StatutoryPayableAccount == 0 {
		cfg.StatutoryPayableAccount = accountBySystemKey(db, "tax_payable")
	}
	return cfg
}

func accountBySystemKey(db *gorm.DB, key string) uint {
	var a models.Account
	if db.Where("system_key = ?", key).First(&a).Error == nil {
		return a.ID
	}
	return 0
}

// computeLine breaks a gross salary into deductions and employer contributions.
func computeLine(gross float64, cfg PayrollConfig) models.PayrollLine {
	taxable := gross - cfg.IncomeTaxExempt
	if taxable < 0 {
		taxable = 0
	}
	l := models.PayrollLine{
		Gross:       round2(gross),
		IncomeTax:   round2(taxable * cfg.IncomeTaxRate / 100),
		DsmfEmp:     round2(gross * cfg.DsmfEmp / 100),
		UnempEmp:    round2(gross * cfg.UnempEmp / 100),
		MedicalEmp:  round2(gross * cfg.MedicalEmp / 100),
		DsmfEmpr:    round2(gross * cfg.DsmfEmpr / 100),
		UnempEmpr:   round2(gross * cfg.UnempEmpr / 100),
		MedicalEmpr: round2(gross * cfg.MedicalEmpr / 100),
	}
	l.Net = round2(l.Gross - l.IncomeTax - l.DsmfEmp - l.UnempEmp - l.MedicalEmp)
	return l
}

// CreatePayrollRun computes a draft run for all active employees in a period.
func CreatePayrollRun(db *gorm.DB, period time.Time) (*models.PayrollRun, error) {
	cfg := LoadPayrollConfig(db)
	var employees []models.Employee
	db.Where("status = ?", "active").Order("full_name asc").Find(&employees)
	if len(employees) == 0 {
		return nil, errors.New("aktiv işçi yoxdur")
	}
	run := models.PayrollRun{Period: models.Date{Time: period}, Status: "draft"}
	for _, e := range employees {
		l := computeLine(e.Salary, cfg)
		l.EmployeeID = e.ID
		l.EmployeeName = e.FullName
		run.Lines = append(run.Lines, l)
		run.GrossTotal += l.Gross
		run.NetTotal += l.Net
		run.DeductionsTotal += round2(l.IncomeTax + l.DsmfEmp + l.UnempEmp + l.MedicalEmp)
		run.EmployerTotal += round2(l.DsmfEmpr + l.UnempEmpr + l.MedicalEmpr)
	}
	run.GrossTotal = round2(run.GrossTotal)
	run.NetTotal = round2(run.NetTotal)
	run.DeductionsTotal = round2(run.DeductionsTotal)
	run.EmployerTotal = round2(run.EmployerTotal)
	if err := db.Create(&run).Error; err != nil {
		return nil, err
	}
	db.Preload("Lines").First(&run, run.ID)
	return &run, nil
}

// PostPayrollRun posts a draft run to the journal:
//   Dr Salary expense (gross + employer contributions)
//   Cr Net wages payable
//   Cr Statutory payable (all taxes + social contributions)
func PostPayrollRun(db *gorm.DB, runID uint) (*models.PayrollRun, error) {
	var run models.PayrollRun
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Lines").First(&run, runID).Error; err != nil {
			return err
		}
		if run.Status == "posted" {
			return errors.New("bu hesablama artıq kitablaşdırılıb")
		}
		cfg := LoadPayrollConfig(tx)
		if cfg.SalaryExpenseAccount == 0 || cfg.WagesPayableAccount == 0 || cfg.StatutoryPayableAccount == 0 {
			return errors.New("əmək haqqı hesabları təyin edilməyib (Parametrlərdən seçin)")
		}
		expense := round2(run.GrossTotal + run.EmployerTotal)
		statutory := round2(run.DeductionsTotal + run.EmployerTotal)

		number, err := NextNumber(tx, "journal", "J-")
		if err != nil {
			return err
		}
		now := time.Now()
		entry := models.JournalEntry{
			Number: number, Date: run.Period, Status: "posted", Source: "payroll",
			Description: "Əmək haqqı hesablanması", PostedAt: &now,
			TotalDebit: expense, TotalCredit: round2(run.NetTotal + statutory),
			Lines: []models.JournalLine{
				{AccountID: cfg.SalaryExpenseAccount, Description: "Əmək haqqı xərci", Debit: expense, Posted: true, Date: run.Period},
				{AccountID: cfg.WagesPayableAccount, Description: "Ödəniləcək əmək haqqı (net)", Credit: round2(run.NetTotal), Posted: true, Date: run.Period},
				{AccountID: cfg.StatutoryPayableAccount, Description: "Vergi və sosial öhdəliklər", Credit: statutory, Posted: true, Date: run.Period},
			},
		}
		if _, _, verr := ValidateBalanced(entry.Lines); verr != nil {
			return verr
		}
		if err := tx.Create(&entry).Error; err != nil {
			return err
		}
		run.Status = "posted"
		run.JournalID = &entry.ID
		return tx.Save(&run).Error
	})
	if err != nil {
		return nil, err
	}
	db.Preload("Lines").First(&run, runID)
	return &run, nil
}
