package api

import (
	"context"

	"github.com/apolinario0x21/effective-octo-bassoon/internal/models"
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
