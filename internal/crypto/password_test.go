package crypto_test

import (
	"testing"

	"github.com/apolinario0x21/students-api/internal/crypto"
)

func TestHashAndCheckPassword(t *testing.T) {
	const password = "s3cr3t-password"

	hash, err := crypto.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == password {
		t.Fatal("hash não pode ser igual à senha em texto puro")
	}

	if err := crypto.CheckPassword(hash, password); err != nil {
		t.Errorf("CheckPassword com senha correta: %v", err)
	}
	if err := crypto.CheckPassword(hash, "senha-errada"); err == nil {
		t.Error("CheckPassword com senha errada: esperava erro, veio nil")
	}
}
