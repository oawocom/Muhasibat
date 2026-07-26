package store

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"oawo-muhasibat/internal/models"
	"oawo-muhasibat/internal/seed"
)

// Connect opens the PostgreSQL connection, auto-migrates and seeds.
func Connect() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := envOr("DB_HOST", "localhost")
		port := envOr("DB_PORT", "5432")
		user := envOr("DB_USER", "oawo")
		pass := envOr("DB_PASSWORD", "oawo")
		name := envOr("DB_NAME", "oawo_muhasibat")
		ssl := envOr("DB_SSLMODE", "disable")
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Baku",
			host, port, user, pass, name, ssl)
	}

	var db *gorm.DB
	var err error
	// Retry — the DB container may still be starting.
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

	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if err := seed.Run(db); err != nil {
		return nil, fmt.Errorf("seed: %w", err)
	}
	log.Println("✅ Verilənlər bazası hazırdır.")
	return db, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
