package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// ErrUserNotFound é retornado pelos repositórios quando o usuário não existe.
var ErrUserNotFound = errors.New("user not found")

// ErrRefreshTokenNotFound é retornado quando o refresh token não existe (ou já
// foi revogado/removido).
var ErrRefreshTokenNotFound = errors.New("refresh token not found")

// Papéis de autorização (RBAC). São poucos e estáticos, então ficam como um
// campo em users em vez de uma tabela de junção.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// ValidRole indica se s é um papel conhecido.
func ValidRole(s string) bool {
	return s == RoleAdmin || s == RoleUser
}

// User é uma conta que autentica na API. A senha nunca é armazenada em texto
// puro — apenas seu hash (bcrypt).
type User struct {
	gorm.Model
	Username     string `gorm:"uniqueIndex;size:255;not null"`
	PasswordHash string `gorm:"not null"`
	Role         string `gorm:"not null"`
}

// RefreshToken representa um refresh token emitido a um usuário. Guardamos apenas
// o hash (SHA-256) do token; o valor bruto só existe do lado do cliente.
type RefreshToken struct {
	gorm.Model
	UserID    uint      `gorm:"index;not null"`
	TokenHash string    `gorm:"uniqueIndex;size:64;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	Revoked   bool      `gorm:"not null"`
}
