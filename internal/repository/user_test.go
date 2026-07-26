package repository_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/apolinario0x21/students-api/internal/config"
	"github.com/apolinario0x21/students-api/internal/db"
	"github.com/apolinario0x21/students-api/internal/models"
	"github.com/apolinario0x21/students-api/internal/repository"
)

func newUserRepo(t *testing.T) *repository.GormUserRepository {
	t.Helper()

	database, err := db.Connect(config.DBConfig{
		Driver: "sqlite",
		DSN:    filepath.Join(t.TempDir(), "users.db"),
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return repository.NewUserRepository(database)
}

func TestUserRepositoryCreateAndFind(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	user := models.User{Username: "maria", PasswordHash: "hash", Role: models.RoleAdmin}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	found, err := repo.FindUserByUsername(ctx, "maria")
	if err != nil {
		t.Fatalf("FindUserByUsername: %v", err)
	}
	if found.ID != user.ID || found.Role != models.RoleAdmin {
		t.Errorf("found = %+v, want id %d role admin", found, user.ID)
	}

	if _, err := repo.FindUserByUsername(ctx, "ghost"); err != models.ErrUserNotFound {
		t.Errorf("err = %v, want ErrUserNotFound", err)
	}
}

func TestPurgeExpiredRefreshTokens(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()
	now := time.Now()

	tokens := []models.RefreshToken{
		{UserID: 1, TokenHash: "valid", ExpiresAt: now.Add(time.Hour), Revoked: false},
		{UserID: 1, TokenHash: "expired", ExpiresAt: now.Add(-time.Hour), Revoked: false},
		{UserID: 1, TokenHash: "revoked", ExpiresAt: now.Add(time.Hour), Revoked: true},
	}
	for i := range tokens {
		if err := repo.SaveRefreshToken(ctx, &tokens[i]); err != nil {
			t.Fatalf("SaveRefreshToken: %v", err)
		}
	}

	removed, err := repo.PurgeExpiredRefreshTokens(ctx, now)
	if err != nil {
		t.Fatalf("PurgeExpiredRefreshTokens: %v", err)
	}
	if removed != 2 {
		t.Errorf("removed = %d, want 2 (expired + revoked)", removed)
	}

	// O token válido continua consultável; os demais foram apagados.
	if _, err := repo.FindRefreshToken(ctx, "valid"); err != nil {
		t.Errorf("token válido sumiu: %v", err)
	}
	if _, err := repo.FindRefreshToken(ctx, "expired"); err != models.ErrRefreshTokenNotFound {
		t.Errorf("token expirado ainda presente: %v", err)
	}
}
