package tenant

import (
	"fmt"
	"log"
	"os"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"oawo-muhasibat/internal/models"
	"oawo-muhasibat/internal/seed"
)

// Config holds the PostgreSQL connection parameters used to build both the
// platform DSN and per-tenant DSNs.
type Config struct {
	Host, Port, User, Password, SSLMode string
}

func ConfigFromEnv() Config {
	return Config{
		Host:     envOr("DB_HOST", "localhost"),
		Port:     envOr("DB_PORT", "5432"),
		User:     envOr("DB_USER", "oawo"),
		Password: envOr("DB_PASSWORD", "oawo"),
		SSLMode:  envOr("DB_SSLMODE", "disable"),
	}
}

func (c Config) DSN(dbName string) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Baku",
		c.Host, c.Port, c.User, c.Password, dbName, c.SSLMode)
}

// Manager provisions and caches per-company database connections.
type Manager struct {
	platform *gorm.DB
	cfg      Config
	mu       sync.RWMutex
	cache    map[uint]*gorm.DB
}

func NewManager(platform *gorm.DB, cfg Config) *Manager {
	return &Manager{platform: platform, cfg: cfg, cache: map[uint]*gorm.DB{}}
}

// DBNameFor returns the deterministic database name for a company.
func DBNameFor(companyID uint) string {
	return fmt.Sprintf("oawo_company_%d", companyID)
}

// Get returns a cached connection to the company's database, opening it on
// first use. The database must already be provisioned.
func (m *Manager) Get(companyID uint, dbName string) (*gorm.DB, error) {
	m.mu.RLock()
	if db, ok := m.cache[companyID]; ok {
		m.mu.RUnlock()
		return db, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if db, ok := m.cache[companyID]; ok { // re-check under write lock
		return db, nil
	}
	db, err := gorm.Open(postgres.Open(m.cfg.DSN(dbName)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("tenant DB açıla bilmədi (%s): %w", dbName, err)
	}
	m.cache[companyID] = db
	return db, nil
}

// Provision creates the company's database (if missing), migrates the tenant
// schema and seeds the chart of accounts / defaults.
func (m *Manager) Provision(companyID uint, dbName string) (*gorm.DB, error) {
	if err := m.createDatabaseIfNotExists(dbName); err != nil {
		return nil, err
	}
	db, err := m.Get(companyID, dbName)
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(models.TenantModels()...); err != nil {
		return nil, fmt.Errorf("tenant migrate: %w", err)
	}
	if err := seed.SeedTenant(db); err != nil {
		return nil, fmt.Errorf("tenant seed: %w", err)
	}
	log.Printf("🏢 Tenant provisioned: %s", dbName)
	return db, nil
}

// createDatabaseIfNotExists issues CREATE DATABASE via the platform connection
// (runs in autocommit — CREATE DATABASE cannot run inside a transaction).
func (m *Manager) createDatabaseIfNotExists(dbName string) error {
	var exists bool
	m.platform.Raw("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = ?)", dbName).Scan(&exists)
	if exists {
		return nil
	}
	// dbName is derived from an integer ID, so it is safe to interpolate.
	if err := m.platform.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)).Error; err != nil {
		return fmt.Errorf("CREATE DATABASE %s: %w", dbName, err)
	}
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
