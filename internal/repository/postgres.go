package repository

import (
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

	db, err := gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
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
	)
	if err != nil {
		return nil, err
	}

	return &DB{DB: db}, nil
}
