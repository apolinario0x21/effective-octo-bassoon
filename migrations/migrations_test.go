package migrations_test

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/apolinario0x21/students-api/internal/models"
	"github.com/apolinario0x21/students-api/migrations"
)

func openSQLite(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "migrate.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func TestRunCreatesUsableSchema(t *testing.T) {
	db := openSQLite(t)

	if err := migrations.Run(db, "sqlite"); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// A tabela students precisa estar utilizável via GORM (incluindo o
	// autoincremento do id e o soft-delete de gorm.Model).
	student := models.Student{Name: "Maria", CPF: "52998224725", Email: "m@e.com", Age: 20, Active: true}
	if err := db.Create(&student).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	if student.ID == 0 {
		t.Fatal("expected autoincremented id, got 0")
	}

	if err := db.Delete(&student).Error; err != nil { // exercita deleted_at
		t.Fatalf("soft delete: %v", err)
	}
	var count int64
	if err := db.Model(&models.Student{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("count after soft delete = %d, want 0", count)
	}
}

func TestRunIsIdempotent(t *testing.T) {
	db := openSQLite(t)

	for i := 0; i < 3; i++ {
		if err := migrations.Run(db, "sqlite"); err != nil {
			t.Fatalf("Run #%d: %v", i+1, err)
		}
	}

	// A migração inicial deve constar exatamente uma vez em schema_migrations.
	var applied int64
	if err := db.Raw(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, "001_create_students").
		Scan(&applied).Error; err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if applied != 1 {
		t.Errorf("migração 001 registrada %d vezes, want 1", applied)
	}
}

func TestRunRejectsUnknownDriver(t *testing.T) {
	db := openSQLite(t)

	if err := migrations.Run(db, "mysql"); err == nil {
		t.Fatal("expected error for unsupported driver, got nil")
	}
}
