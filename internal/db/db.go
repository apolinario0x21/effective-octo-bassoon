package db

import (
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/apolinario0x21/effective-octo-bassoon/internal/config"
	"github.com/apolinario0x21/effective-octo-bassoon/internal/models"
)

// Connect abre a conexão com o banco conforme o driver configurado
// ("postgres" ou "sqlite"), ajusta o pool e executa as migrações.
func Connect(cfg config.DBConfig) (*gorm.DB, error) {
	dialector, err := dialectorFor(cfg)
	if err != nil {
		return nil, err
	}

	database, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("opening %s database: %w", cfg.Driver, err)
	}

	if err := configurePool(database); err != nil {
		return nil, err
	}

	if err := database.AutoMigrate(&models.Student{}); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return database, nil
}

func dialectorFor(cfg config.DBConfig) (gorm.Dialector, error) {
	dsn := cfg.DSNString()

	switch cfg.Driver {
	case "postgres":
		return postgres.Open(dsn), nil
	case "sqlite":
		return sqlite.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported db driver %q (want postgres or sqlite)", cfg.Driver)
	}
}

func configurePool(database *gorm.DB) error {
	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("accessing sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return nil
}
