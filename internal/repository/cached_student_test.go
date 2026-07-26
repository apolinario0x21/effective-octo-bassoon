package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/apolinario0x21/students-api/internal/models"
	"github.com/apolinario0x21/students-api/internal/repository"
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

// countingRepo conta chamadas a FindByID e List para provar hits/misses de cache.
// List devolve dados que codificam os parâmetros recebidos, permitindo detectar
// colisão entre páginas diferentes.
type countingRepo struct {
	student   models.Student
	findCalls int
	listCalls int
}

func (r *countingRepo) FindByID(_ context.Context, _ uint) (*models.Student, error) {
	r.findCalls++
	s := r.student
	return &s, nil
}

func (r *countingRepo) List(_ context.Context, params models.ListParams) (models.StudentPage, error) {
	r.listCalls++
	// O nome codifica os parâmetros: assim um teste consegue provar que a página
	// devolvida corresponde exatamente ao que foi pedido (sem colisão de cache).
	name := fmt.Sprintf("limit=%d offset=%d", params.Limit, params.Offset)
	return models.StudentPage{
		Students: []models.Student{{Name: name}},
		Total:    100,
	}, nil
}

func (r *countingRepo) Create(context.Context, *models.Student) error       { return nil }
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

// TestCachedRepositoryListDoesNotCollideBetweenPages é o teste que protege
// contra a armadilha crítica: páginas com offsets diferentes NÃO podem
// compartilhar a mesma entrada de cache.
func TestCachedRepositoryListDoesNotCollideBetweenPages(t *testing.T) {
	inner := &countingRepo{}
	repo := repository.NewCachedStudentRepository(inner, newFakeCache())
	ctx := context.Background()

	page1 := models.ListParams{Limit: 10, Offset: 0}
	page2 := models.ListParams{Limit: 10, Offset: 10}

	// Primeira leitura de cada página: dois misses, dois acessos ao inner.
	first, err := repo.List(ctx, page1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.List(ctx, page2)
	if err != nil {
		t.Fatal(err)
	}

	if first.Students[0].Name == second.Students[0].Name {
		t.Fatalf("páginas diferentes retornaram os mesmos dados: %q", first.Students[0].Name)
	}
	if want := "limit=10 offset=0"; first.Students[0].Name != want {
		t.Errorf("page1 = %q, want %q", first.Students[0].Name, want)
	}
	if want := "limit=10 offset=10"; second.Students[0].Name != want {
		t.Errorf("page2 = %q, want %q", second.Students[0].Name, want)
	}

	// Segunda leitura de cada página: ambas devem vir do cache (nenhum acesso
	// novo ao inner) e ainda devolver o conteúdo correto de cada página.
	cachedFirst, err := repo.List(ctx, page1)
	if err != nil {
		t.Fatal(err)
	}
	cachedSecond, err := repo.List(ctx, page2)
	if err != nil {
		t.Fatal(err)
	}

	if cachedFirst.Students[0].Name != first.Students[0].Name {
		t.Errorf("page1 cacheada = %q, want %q", cachedFirst.Students[0].Name, first.Students[0].Name)
	}
	if cachedSecond.Students[0].Name != second.Students[0].Name {
		t.Errorf("page2 cacheada = %q, want %q", cachedSecond.Students[0].Name, second.Students[0].Name)
	}

	if inner.listCalls != 2 {
		t.Errorf("inner List chamado %d vezes, want 2 (uma por página distinta; demais são hits)", inner.listCalls)
	}
}

// TestCachedRepositoryListInvalidatesOnWrite garante que qualquer escrita
// invalida TODAS as páginas cacheadas, não apenas uma.
func TestCachedRepositoryListInvalidatesOnWrite(t *testing.T) {
	writes := map[string]func(repo *repository.CachedStudentRepository, ctx context.Context) error{
		"create": func(repo *repository.CachedStudentRepository, ctx context.Context) error {
			return repo.Create(ctx, &models.Student{})
		},
		"update": func(repo *repository.CachedStudentRepository, ctx context.Context) error {
			return repo.Update(ctx, &models.Student{})
		},
		"delete": func(repo *repository.CachedStudentRepository, ctx context.Context) error {
			return repo.Delete(ctx, &models.Student{})
		},
	}

	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			inner := &countingRepo{}
			repo := repository.NewCachedStudentRepository(inner, newFakeCache())
			ctx := context.Background()
			params := models.ListParams{Limit: 10, Offset: 0}

			if _, err := repo.List(ctx, params); err != nil { // popula o cache
				t.Fatal(err)
			}
			if err := write(repo, ctx); err != nil { // deve invalidar todas as páginas
				t.Fatal(err)
			}
			if _, err := repo.List(ctx, params); err != nil { // volta ao banco
				t.Fatal(err)
			}

			if inner.listCalls != 2 {
				t.Errorf("inner List chamado %d vezes, want 2 (cache invalidado por %s)", inner.listCalls, name)
			}
		})
	}
}
