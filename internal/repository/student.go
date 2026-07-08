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

func (r *GormStudentRepository) FindAll(ctx context.Context) ([]models.Student, error) {
	students := []models.Student{}
	err := r.db.WithContext(ctx).Find(&students).Error

	return students, err
}

func (r *GormStudentRepository) FindByActive(ctx context.Context, active bool) ([]models.Student, error) {
	students := []models.Student{}
	err := r.db.WithContext(ctx).Where("active = ?", active).Find(&students).Error

	return students, err
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
