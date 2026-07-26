package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/apolinario0x21/students-api/internal/models"
)

// StudentRepository é o conjunto de operações de persistência de estudantes.
// É satisfeito tanto pelo repositório GORM quanto pelo cache que o decora.
type StudentRepository interface {
	Create(ctx context.Context, student *models.Student) error
	List(ctx context.Context, params models.ListParams) (models.StudentPage, error)
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

// listGenKey guarda a "geração" atual das listagens. Como a interface Cache não
// oferece varredura por padrão (SCAN/DEL prefix), embutimos a geração na chave
// de cada página. Uma escrita bumpa a geração, tornando todas as páginas
// anteriores inalcançáveis (elas expiram sozinhas pelo TTL) — uma invalidação
// de todas as páginas de uma vez, sem enumerá-las.
const listGenKey = "students:list:gen"

// studentListKey compõe a chave de uma página incluindo geração, filtro active
// e paginação, garantindo que páginas diferentes nunca colidam.
func studentListKey(gen int64, params models.ListParams) string {
	active := "all"
	if params.Active != nil {
		if *params.Active {
			active = "true"
		} else {
			active = "false"
		}
	}
	return fmt.Sprintf("students:list:g%d:a%s:l%d:o%d", gen, active, params.Limit, params.Offset)
}

// listGeneration lê a geração atual; ausência ou erro equivalem à geração 0
// (ainda correto: escritas gravam sempre uma geração nova e única).
func (r *CachedStudentRepository) listGeneration(ctx context.Context) int64 {
	raw, err := r.cache.Get(ctx, listGenKey)
	if err != nil {
		log.Warn().Err(err).Msg("[cache] list generation get failed, assuming 0")
		return 0
	}
	if raw == nil {
		return 0
	}
	gen, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return 0
	}
	return gen
}

// bumpListGeneration grava uma geração nova e única, invalidando todas as
// páginas cacheadas na geração anterior.
func (r *CachedStudentRepository) bumpListGeneration(ctx context.Context) {
	gen := strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := r.cache.Set(ctx, listGenKey, []byte(gen)); err != nil {
		log.Warn().Err(err).Msg("[cache] list generation bump failed")
	}
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

func (r *CachedStudentRepository) List(ctx context.Context, params models.ListParams) (models.StudentPage, error) {
	key := studentListKey(r.listGeneration(ctx), params)

	if cached, err := r.cache.Get(ctx, key); err != nil {
		log.Warn().Err(err).Str("key", key).Msg("[cache] list get failed, falling back to db")
	} else if cached != nil {
		page := models.StudentPage{}
		if err := json.Unmarshal(cached, &page); err == nil {
			return page, nil
		}
		log.Warn().Str("key", key).Msg("[cache] corrupt list entry, falling back to db")
	}

	page, err := r.inner.List(ctx, params)
	if err != nil {
		return models.StudentPage{}, err
	}

	if encoded, err := json.Marshal(page); err == nil {
		if err := r.cache.Set(ctx, key, encoded); err != nil {
			log.Warn().Err(err).Str("key", key).Msg("[cache] list set failed")
		}
	}

	return page, nil
}

func (r *CachedStudentRepository) Update(ctx context.Context, student *models.Student) error {
	if err := r.inner.Update(ctx, student); err != nil {
		return err
	}
	r.invalidate(ctx, student.ID)
	r.bumpListGeneration(ctx)
	return nil
}

func (r *CachedStudentRepository) Delete(ctx context.Context, student *models.Student) error {
	if err := r.inner.Delete(ctx, student); err != nil {
		return err
	}
	r.invalidate(ctx, student.ID)
	r.bumpListGeneration(ctx)
	return nil
}

func (r *CachedStudentRepository) invalidate(ctx context.Context, id uint) {
	if err := r.cache.Delete(ctx, studentKey(id)); err != nil {
		log.Warn().Err(err).Uint("id", id).Msg("[cache] invalidation failed")
	}
}

func (r *CachedStudentRepository) Create(ctx context.Context, student *models.Student) error {
	if err := r.inner.Create(ctx, student); err != nil {
		return err
	}
	// Um novo estudante muda o conteúdo e o total das listagens.
	r.bumpListGeneration(ctx)
	return nil
}

// As operações abaixo não são cacheadas e apenas delegam ao repositório.

func (r *CachedStudentRepository) ExistsWithCPF(ctx context.Context, cpf string) (bool, error) {
	return r.inner.ExistsWithCPF(ctx, cpf)
}
