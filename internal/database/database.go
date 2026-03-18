package database

import (
	"fmt"

	"github.com/statusy/statusy/config"
	"github.com/statusy/statusy/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a database connection based on config.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}

	var db *gorm.DB
	var err error

	switch cfg.Database.Type {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.Database.DSN), gormCfg)
	case "mysql":
		db, err = gorm.Open(mysql.Open(cfg.Database.DSN), gormCfg)
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.Database.DSN), gormCfg)
	default:
		return nil, fmt.Errorf("unsupported database type: %s (use sqlite, mysql, or postgres)", cfg.Database.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for SQLite (better concurrent read performance)
	if cfg.Database.Type == "sqlite" {
		sqlDB, _ := db.DB()
		sqlDB.Exec("PRAGMA journal_mode=WAL;")
		sqlDB.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes
	}

	return db, nil
}

// Migrate runs auto-migration for all models.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.JWTBlacklist{},
		&models.Monitor{},
		&models.CheckResult{},
		&models.Notification{},
		&models.StatusPage{},
	)
}
