package api

import (
	"context"

	"github.com/apolinario0x21/students-api/internal/models"
)

// StudentRepository abstrai o acesso a dados de estudantes, permitindo
// substituir a implementação (ex.: mocks em testes).
type StudentRepository interface {
	Create(ctx context.Context, student *models.Student) error
	List(ctx context.Context, params models.ListParams) (models.StudentPage, error)
	FindByID(ctx context.Context, id uint) (*models.Student, error)
	ExistsWithCPF(ctx context.Context, cpf string) (bool, error)
	Update(ctx context.Context, student *models.Student) error
	Delete(ctx context.Context, student *models.Student) error
}

// UserRepository abstrai o acesso a dados de usuários e refresh tokens.
type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	FindUserByUsername(ctx context.Context, username string) (*models.User, error)
	FindUserByID(ctx context.Context, id uint) (*models.User, error)
	ExistsUserWithUsername(ctx context.Context, username string) (bool, error)
	SaveRefreshToken(ctx context.Context, token *models.RefreshToken) error
	FindRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
}
