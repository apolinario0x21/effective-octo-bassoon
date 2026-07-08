package db

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/apolinario0x21/effective-octo-bassoon/internal/models"
)

// Connect abre o banco SQLite e executa as migrações.
func Connect(path string) (*gorm.DB, error) {
	database, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("opening sqlite database: %w", err)
	}

	if err := database.AutoMigrate(&models.Student{}); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return database, nil
}
