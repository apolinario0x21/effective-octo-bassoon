package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/apolinario0x21/effective-octo-bassoon/internal/models"
)

// StudentRepository é o conjunto de operações de persistência de estudantes.
// É satisfeito tanto pelo repositório GORM quanto pelo cache que o decora.
type StudentRepository interface {
	Create(ctx context.Context, student *models.Student) error
	FindAll(ctx context.Context) ([]models.Student, error)
	FindByActive(ctx context.Context, active bool) ([]models.Student, error)
	FindByID(ctx context.Context, id uint) (*models.Student, error)
	ExistsWithCPF(ctx context.Context, cpf string) (bool, error)
	Update(ctx context.Context, student *models.Student) error
	Delete(ctx context.Context, student *models.Student) error
}

// Cache abstrai o armazenamento de chave-valor usado pelo cache de estudantes.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, keys ...string) error
}

// CachedStudentRepository decora um StudentRepository com um cache de leitura
// por ID. Falhas no cache nunca quebram a requisição: são registradas e a
// operação recorre ao repositório subjacente.
type CachedStudentRepository struct {
	inner StudentRepository
	cache Cache
}

// NewCachedStudentRepository envolve inner com um cache.
func NewCachedStudentRepository(inner StudentRepository, cache Cache) *CachedStudentRepository {
	return &CachedStudentRepository{inner: inner, cache: cache}
}

func studentKey(id uint) string {
	return fmt.Sprintf("student:%d", id)
}

func (r *CachedStudentRepository) FindByID(ctx context.Context, id uint) (*models.Student, error) {
	key := studentKey(id)

	if cached, err := r.cache.Get(ctx, key); err != nil {
		log.Warn().Err(err).Str("key", key).Msg("[cache] get failed, falling back to db")
	} else if cached != nil {
		student := models.Student{}
		if err := json.Unmarshal(cached, &student); err == nil {
			return &student, nil
		}
		log.Warn().Str("key", key).Msg("[cache] corrupt entry, falling back to db")
	}

	student, err := r.inner.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if encoded, err := json.Marshal(student); err == nil {
		if err := r.cache.Set(ctx, key, encoded); err != nil {
			log.Warn().Err(err).Str("key", key).Msg("[cache] set failed")
		}
	}

	return student, nil
}

func (r *CachedStudentRepository) Update(ctx context.Context, student *models.Student) error {
	if err := r.inner.Update(ctx, student); err != nil {
		return err
	}
	r.invalidate(ctx, student.ID)
	return nil
}

func (r *CachedStudentRepository) Delete(ctx context.Context, student *models.Student) error {
	if err := r.inner.Delete(ctx, student); err != nil {
		return err
	}
	r.invalidate(ctx, student.ID)
	return nil
}

func (r *CachedStudentRepository) invalidate(ctx context.Context, id uint) {
	if err := r.cache.Delete(ctx, studentKey(id)); err != nil {
		log.Warn().Err(err).Uint("id", id).Msg("[cache] invalidation failed")
	}
}

// As operações abaixo não são cacheadas e apenas delegam ao repositório.

func (r *CachedStudentRepository) Create(ctx context.Context, student *models.Student) error {
	return r.inner.Create(ctx, student)
}

func (r *CachedStudentRepository) FindAll(ctx context.Context) ([]models.Student, error) {
	return r.inner.FindAll(ctx)
}

func (r *CachedStudentRepository) FindByActive(ctx context.Context, active bool) ([]models.Student, error) {
	return r.inner.FindByActive(ctx, active)
}

func (r *CachedStudentRepository) ExistsWithCPF(ctx context.Context, cpf string) (bool, error) {
	return r.inner.ExistsWithCPF(ctx, cpf)
}
