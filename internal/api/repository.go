package api

import (
	"context"

	"github.com/apolinario0x21/effective-octo-bassoon/internal/models"
)

// StudentRepository abstrai o acesso a dados de estudantes, permitindo
// substituir a implementação (ex.: mocks em testes).
type StudentRepository interface {
	Create(ctx context.Context, student *models.Student) error
	FindAll(ctx context.Context) ([]models.Student, error)
	FindByActive(ctx context.Context, active bool) ([]models.Student, error)
	FindByID(ctx context.Context, id uint) (*models.Student, error)
	ExistsWithCPF(ctx context.Context, cpf string) (bool, error)
	Update(ctx context.Context, student *models.Student) error
	Delete(ctx context.Context, student *models.Student) error
}
