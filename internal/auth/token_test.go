package auth_test

import (
	"testing"
	"time"

	"github.com/apolinario0x21/students-api/internal/auth"
	"github.com/apolinario0x21/students-api/internal/models"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	m := auth.NewManager("secret", 15*time.Minute, time.Hour)

	tok, err := m.GenerateAccessToken(42, models.RoleAdmin)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := m.ParseAccessToken(tok)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims.Role != models.RoleAdmin {
		t.Errorf("role = %q, want %q", claims.Role, models.RoleAdmin)
	}
	id, err := claims.UserID()
	if err != nil {
		t.Fatalf("UserID: %v", err)
	}
	if id != 42 {
		t.Errorf("UserID = %d, want 42", id)
	}
}

func TestExpiredAccessTokenIsRejected(t *testing.T) {
	m := auth.NewManager("secret", -time.Minute, time.Hour) // já expirado

	tok, err := m.GenerateAccessToken(1, models.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.ParseAccessToken(tok); err == nil {
		t.Error("esperava erro para token expirado, veio nil")
	}
}

func TestTokenFromAnotherSecretIsRejected(t *testing.T) {
	issuer := auth.NewManager("secret-a", time.Hour, time.Hour)
	verifier := auth.NewManager("secret-b", time.Hour, time.Hour)

	tok, err := issuer.GenerateAccessToken(1, models.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.ParseAccessToken(tok); err == nil {
		t.Error("esperava erro para assinatura inválida, veio nil")
	}
}

func TestRefreshTokenGeneration(t *testing.T) {
	m := auth.NewManager("secret", time.Hour, 48*time.Hour)

	raw, hash, expiresAt, err := m.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if raw == "" || hash == "" {
		t.Fatal("raw/hash vazios")
	}
	if raw == hash {
		t.Error("hash não pode ser igual ao token bruto")
	}
	if auth.HashRefreshToken(raw) != hash {
		t.Error("HashRefreshToken não é determinístico com o valor bruto")
	}
	if time.Until(expiresAt) <= 47*time.Hour {
		t.Errorf("expiração muito curta: %v", expiresAt)
	}
}
