package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/apolinario0x21/effective-octo-bassoon/internal/models"
)

// GormStudentRepository implementa o acesso a dados de estudantes usando GORM.
type GormStudentRepository struct {
	db *gorm.DB
}

func NewStudentRepository(db *gorm.DB) *GormStudentRepository {
	return &GormStudentRepository{db: db}
}

func (r *GormStudentRepository) Create(ctx context.Context, student *models.Student) error {
	return r.db.WithContext(ctx).Create(student).Error
}

// List devolve uma página de estudantes (aplicando limit/offset e o filtro
// opcional por active) junto do total de registros que satisfazem o filtro.
func (r *GormStudentRepository) List(ctx context.Context, params models.ListParams) (models.StudentPage, error) {
	// filtered devolve uma query nova já com o filtro aplicado, evitando que a
	// contagem e a busca compartilhem o mesmo statement do GORM.
	filtered := func() *gorm.DB {
		q := r.db.WithContext(ctx).Model(&models.Student{})
		if params.Active != nil {
			q = q.Where("active = ?", *params.Active)
		}
		return q
	}

	var total int64
	if err := filtered().Count(&total).Error; err != nil {
		return models.StudentPage{}, err
	}

	students := []models.Student{}
	err := filtered().
		Order("id ASC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&students).Error
	if err != nil {
		return models.StudentPage{}, err
	}

	return models.StudentPage{Students: students, Total: total}, nil
}

func (r *GormStudentRepository) FindByID(ctx context.Context, id uint) (*models.Student, error) {
	student := models.Student{}

	err := r.db.WithContext(ctx).First(&student, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, models.ErrStudentNotFound
	}
	if err != nil {
		return nil, err
	}

	return &student, nil
}

func (r *GormStudentRepository) ExistsWithCPF(ctx context.Context, cpf string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Student{}).Where("cpf = ?", cpf).Count(&count).Error

	return count > 0, err
}

func (r *GormStudentRepository) Update(ctx context.Context, student *models.Student) error {
	return r.db.WithContext(ctx).Save(student).Error
}

func (r *GormStudentRepository) Delete(ctx context.Context, student *models.Student) error {
	return r.db.WithContext(ctx).Delete(student).Error
}
