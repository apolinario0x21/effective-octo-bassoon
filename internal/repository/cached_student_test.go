package repository_test

import (
	"context"
	"testing"

	"github.com/apolinario0x21/effective-octo-bassoon/internal/models"
	"github.com/apolinario0x21/effective-octo-bassoon/internal/repository"
)

// fakeCache é um cache em memória para os testes.
type fakeCache struct {
	data map[string][]byte
}

func newFakeCache() *fakeCache { return &fakeCache{data: map[string][]byte{}} }

func (c *fakeCache) Get(_ context.Context, key string) ([]byte, error) {
	return c.data[key], nil
}

func (c *fakeCache) Set(_ context.Context, key string, value []byte) error {
	c.data[key] = value
	return nil
}

func (c *fakeCache) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(c.data, key)
	}
	return nil
}

// countingRepo conta chamadas a FindByID para provar hits/misses de cache.
type countingRepo struct {
	student   models.Student
	findCalls int
}

func (r *countingRepo) FindByID(_ context.Context, _ uint) (*models.Student, error) {
	r.findCalls++
	s := r.student
	return &s, nil
}

func (r *countingRepo) Create(context.Context, *models.Student) error { return nil }
func (r *countingRepo) FindAll(context.Context) ([]models.Student, error) {
	return nil, nil
}
func (r *countingRepo) FindByActive(context.Context, bool) ([]models.Student, error) {
	return nil, nil
}
func (r *countingRepo) ExistsWithCPF(context.Context, string) (bool, error) { return false, nil }
func (r *countingRepo) Update(context.Context, *models.Student) error       { return nil }
func (r *countingRepo) Delete(context.Context, *models.Student) error       { return nil }

func TestCachedRepositoryCachesReads(t *testing.T) {
	inner := &countingRepo{student: models.Student{Name: "Maria"}}
	inner.student.ID = 1
	repo := repository.NewCachedStudentRepository(inner, newFakeCache())
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		student, err := repo.FindByID(ctx, 1)
		if err != nil || student.Name != "Maria" {
			t.Fatalf("FindByID = %+v, %v", student, err)
		}
	}

	if inner.findCalls != 1 {
		t.Errorf("inner FindByID called %d times, want 1 (cache miss then hits)", inner.findCalls)
	}
}

func TestCachedRepositoryInvalidatesOnUpdate(t *testing.T) {
	inner := &countingRepo{student: models.Student{Name: "Maria"}}
	inner.student.ID = 1
	repo := repository.NewCachedStudentRepository(inner, newFakeCache())
	ctx := context.Background()

	if _, err := repo.FindByID(ctx, 1); err != nil { // popula o cache
		t.Fatal(err)
	}
	if err := repo.Update(ctx, &inner.student); err != nil { // deve invalidar
		t.Fatal(err)
	}
	if _, err := repo.FindByID(ctx, 1); err != nil { // volta ao banco
		t.Fatal(err)
	}

	if inner.findCalls != 2 {
		t.Errorf("inner FindByID called %d times, want 2 (cache invalidated by update)", inner.findCalls)
	}
}
