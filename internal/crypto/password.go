// Package crypto concentra utilidades de criptografia da aplicação: hashing de
// senhas (bcrypt) e de tokens.
package crypto

import "golang.org/x/crypto/bcrypt"

// HashPassword devolve o hash bcrypt de uma senha em texto puro.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compara uma senha em texto puro com um hash bcrypt.
// Devolve nil quando conferem.
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
