package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"oawo-muhasibat/internal/models"
	"oawo-muhasibat/internal/tenant"
)

// Connect opens the platform (control-plane) database and migrates the
// platform models (companies, users, memberships).
func Connect() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		cfg := tenant.ConfigFromEnv()
		dsn = cfg.DSN(envOr("DB_NAME", "oawo_muhasibat"))
	}

	var db *gorm.DB
	var err error
	for i := 0; i < 20; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			break
		}
		log.Printf("DB bağlantısı gözlənilir (%d)... %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(models.PlatformModels()...); err != nil {
		return nil, fmt.Errorf("platform migrate: %w", err)
	}
	log.Println("✅ Platform bazası hazırdır.")
	return db, nil
}

// Bootstrap ensures a superadmin user exists and, on first boot, creates a
// default company (provisioning its tenant database) so the app is usable
// immediately.
func Bootstrap(db *gorm.DB, mgr *tenant.Manager) error {
	// Superadmin.
	var admin models.User
	err := db.Where("email = ?", adminEmail()).First(&admin).Error
	if err != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte(adminPassword()), bcrypt.DefaultCost)
		admin = models.User{
			Email: adminEmail(), Name: "Administrator", Password: string(hash),
			IsSuperadmin: true, Enabled: true,
		}
		if err := db.Create(&admin).Error; err != nil {
			return fmt.Errorf("superadmin: %w", err)
		}
		log.Printf("👤 Superadmin yaradıldı: %s", admin.Email)
	}

	// Default tenant + company on first boot (so the app is usable and the
	// superadmin can demo bookkeeping immediately).
	var count int64
	db.Model(&models.Tenant{}).Count(&count)
	if count == 0 {
		modules, _ := json.Marshal(models.DefaultEnabledModules())
		t := models.Tenant{
			Name: envOr("COMPANY_NAME", "OAWO"), Plan: "standard",
			EnabledModules: string(modules), BillingCycle: "monthly", Enabled: true,
		}
		if err := db.Create(&t).Error; err != nil {
			return fmt.Errorf("default tenant: %w", err)
		}
		company := models.Company{TenantID: t.ID, Name: envOr("COMPANY_NAME", "OAWO MMC"), Enabled: true}
		if err := db.Create(&company).Error; err != nil {
			return fmt.Errorf("default company: %w", err)
		}
		company.DBName = tenant.DBNameFor(company.ID)
		db.Save(&company)
		if _, err := mgr.Provision(company.ID, company.DBName); err != nil {
			return fmt.Errorf("provision default company: %w", err)
		}
		db.Model(&models.Company{}).Where("id = ?", company.ID).Update("provisioned", true)
		log.Printf("🏢 Default tenant + şirkət yaradıldı: %s", company.Name)
	}
	return nil
}

func adminEmail() string    { return envOr("ADMIN_EMAIL", "admin@oawo.com") }
func adminPassword() string { return envOr("ADMIN_PASSWORD", "admin123") }

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
