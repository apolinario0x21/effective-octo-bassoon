// Package auth emite e valida os tokens da API: um access token JWT de curta
// duração e um refresh token opaco de longa duração (cujo hash é persistido para
// permitir revogação).
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken indica um access token ausente, malformado, expirado ou com
// assinatura inválida.
var ErrInvalidToken = errors.New("invalid or expired token")

// Claims é o conteúdo do access token JWT.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// UserID devolve o "subject" do token como uint (o ID do usuário).
func (c Claims) UserID() (uint, error) {
	id, err := c.GetSubject()
	if err != nil {
		return 0, err
	}
	var uid uint64
	if _, err := fmt.Sscan(id, &uid); err != nil {
		return 0, err
	}
	return uint(uid), nil
}

// Manager emite e valida tokens usando um segredo HMAC.
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewManager cria um Manager. accessTTL é a validade do JWT; refreshTTL, a do
// refresh token.
func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// RefreshTTL devolve a validade configurada do refresh token.
func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }

// GenerateAccessToken emite um JWT assinado para o usuário/papel informados.
func (m *Manager) GenerateAccessToken(userID uint, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// ParseAccessToken valida a assinatura e a expiração do JWT e devolve as claims.
func (m *Manager) ParseAccessToken(raw string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// GenerateRefreshToken cria um refresh token opaco aleatório e devolve o valor
// bruto (para o cliente), seu hash (para persistir) e a data de expiração.
func (m *Manager) GenerateRefreshToken() (raw, hash string, expiresAt time.Time, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", time.Time{}, err
	}
	raw = hex.EncodeToString(buf)
	return raw, HashRefreshToken(raw), time.Now().Add(m.refreshTTL), nil
}

// HashRefreshToken devolve o SHA-256 (hex) de um refresh token. Como o token é
// aleatório e de alta entropia, SHA-256 é suficiente para o lookup — não é senha.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
