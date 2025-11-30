package repository

import (
	"fmt"
	"log"
	"time"

	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
}

// Ping verifies the database connection is still alive
// Useful for health checks and recovering from connection failures
func (d *DB) Ping() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// NewDB creates a new database connection with retry logic for cold starts
func NewDB(cfg *config.Config) (*DB, error) {
	var logLevel logger.LogLevel
	switch cfg.LogLevel {
	case "debug":
		logLevel = logger.Info
	case "error":
		logLevel = logger.Error
	default:
		logLevel = logger.Warn
	}

	// Retry connection up to 5 times with exponential backoff for cold starts
	var db *gorm.DB
	var err error
	maxRetries := 5
	retryDelay := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{
			Logger: logger.Default.LogMode(logLevel),
		})
		if err == nil {
			// Test the connection immediately
			sqlDB, pingErr := db.DB()
			if pingErr == nil {
				if pingErr = sqlDB.Ping(); pingErr == nil {
					// Connection successful
					break
				}
			}
			if pingErr != nil {
				err = pingErr
			}
		}

		if attempt < maxRetries {
			log.Printf("Database connection attempt %d/%d failed: %v. Retrying in %v...", attempt, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff: 1s, 2s, 4s, 8s
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Set maximum number of open connections to the database
	sqlDB.SetMaxOpenConns(25)

	// Set maximum number of idle connections in the pool
	sqlDB.SetMaxIdleConns(5)

	// Set maximum lifetime of a connection (1 hour)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Set connection timeout (5 seconds) - how long to wait for a connection from the pool
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	// Verify connection is working after setup
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed after connection: %w", err)
	}

	// Auto-migrate all models
	err = db.AutoMigrate(
		&models.User{},
		&models.APIKey{},
		&models.JWTToken{},
		&models.Quest{},
		&models.Item{},
		&models.SkillNode{},
		&models.HideoutModule{},
		&models.EnemyType{},
		&models.Alert{},
		&models.AuditLog{},
		&models.UserQuestProgress{},
		&models.UserHideoutModuleProgress{},
		&models.UserSkillNodeProgress{},
		&models.UserBlueprintProgress{},
		&models.AuthorizationCode{},
		&models.RefreshToken{},
		&models.Bot{},
		&models.Map{},
		&models.Trader{},
		&models.Project{},
	)
	if err != nil {
		return nil, err
	}

	return &DB{DB: db}, nil
}
