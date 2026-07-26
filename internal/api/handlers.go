package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/apolinario0x21/students-api/internal/models"
)

func jsonError(c echo.Context, status int, message string) error {
	return c.JSON(status, errorResponse{Error: message})
}

func parseIDParam(c echo.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return 0, err
	}

	return uint(id), nil
}

// healthCheck godoc
//
//	@Summary		Health check
//	@Description	Sinal de vida da aplicação.
//	@Tags			observabilidade
//	@Produce		json
//	@Success		200	{object}	map[string]string	"{\"status\":\"ok\"}"
//	@Router			/healthz [get]
func (s *Server) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Limites de paginação da listagem de estudantes.
const (
	defaultLimit = 20  // page size padrão quando 'limit' não é informado
	maxLimit     = 100 // teto para impedir páginas absurdamente grandes
)

// listStudents godoc
//
//	@Summary		Lista estudantes (paginada)
//	@Description	Retorna uma página de estudantes com metadados de paginação
//	@Description	(total, limit, offset). Aceita filtro opcional por `active`.
//	@Tags			students
//	@Produce		json
//	@Param			active	query		bool	false	"Filtra por ativo (true) ou inativo (false)"
//	@Param			limit	query		int		false	"Itens por página (1–100)"	default(20)
//	@Param			offset	query		int		false	"Itens a pular"				default(0)
//	@Success		200		{object}	listStudentsResponse
//	@Failure		400		{object}	errorResponse	"parâmetro inválido"
//	@Router			/students [get]
func (s *Server) listStudents(c echo.Context) error {
	ctx := c.Request().Context()

	params := models.ListParams{Limit: defaultLimit}

	if raw := c.QueryParam("active"); raw != "" {
		active, err := strconv.ParseBool(raw)
		if err != nil {
			return jsonError(c, http.StatusBadRequest, "query param 'active' must be a boolean")
		}
		params.Active = &active
	}

	limit, offset, err := parsePagination(c)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, err.Error())
	}
	params.Limit = limit
	params.Offset = offset

	page, err := s.students.List(ctx, params)
	if err != nil {
		log.Error().Err(err).Msg("[api] failed to list students")
		return jsonError(c, http.StatusInternalServerError, "failed to list students")
	}

	return c.JSON(http.StatusOK, listStudentsResponse{
		Students: newStudentResponses(page.Students),
		Total:    page.Total,
		Limit:    limit,
		Offset:   offset,
	})
}

// parsePagination lê e valida os query params 'limit' e 'offset', aplicando
// valores padrão e um teto máximo. Retorna erro (para 400) quando os valores
// não são numéricos ou estão fora da faixa permitida.
func parsePagination(c echo.Context) (limit, offset int, err error) {
	limit = defaultLimit
	offset = 0

	if raw := c.QueryParam("limit"); raw != "" {
		v, convErr := strconv.Atoi(raw)
		if convErr != nil {
			return 0, 0, fmt.Errorf("query param 'limit' must be an integer")
		}
		if v < 1 || v > maxLimit {
			return 0, 0, fmt.Errorf("query param 'limit' must be between 1 and %d", maxLimit)
		}
		limit = v
	}

	if raw := c.QueryParam("offset"); raw != "" {
		v, convErr := strconv.Atoi(raw)
		if convErr != nil {
			return 0, 0, fmt.Errorf("query param 'offset' must be an integer")
		}
		if v < 0 {
			return 0, 0, fmt.Errorf("query param 'offset' must be zero or positive")
		}
		offset = v
	}

	return limit, offset, nil
}

// createStudent godoc
//
//	@Summary		Cria um estudante
//	@Description	Cria um estudante. O CPF deve ter 11 dígitos (só números), ser
//	@Description	válido pelos dígitos verificadores e único; o e-mail deve ter
//	@Description	formato válido. `active` é obrigatório.
//	@Tags			students
//	@Accept			json
//	@Produce		json
//	@Param			student	body		CreateStudentRequest	true	"Dados do estudante"
//	@Success		201		{object}	StudentResponse
//	@Failure		400		{object}	errorResponse	"corpo/validação inválidos"
//	@Failure		409		{object}	errorResponse	"CPF já cadastrado"
//	@Router			/students [post]
func (s *Server) createStudent(c echo.Context) error {
	ctx := c.Request().Context()

	request := CreateStudentRequest{}
	if err := c.Bind(&request); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid request body")
	}

	if err := request.Validate(); err != nil {
		return jsonError(c, http.StatusBadRequest, err.Error())
	}

	exists, err := s.students.ExistsWithCPF(ctx, request.CPF)
	if err != nil {
		log.Error().Err(err).Msg("[api] failed to check CPF uniqueness")
		return jsonError(c, http.StatusInternalServerError, "failed to create student")
	}
	if exists {
		return jsonError(c, http.StatusConflict, "a student with this CPF already exists")
	}

	student := models.Student{
		Name:   request.Name,
		CPF:    request.CPF,
		Email:  request.Email,
		Age:    request.Age,
		Active: *request.Active,
	}

	if err := s.students.Create(ctx, &student); err != nil {
		log.Error().Err(err).Msg("[api] failed to create student")
		return jsonError(c, http.StatusInternalServerError, "failed to create student")
	}

	return c.JSON(http.StatusCreated, newStudentResponse(student))
}

// getStudent godoc
//
//	@Summary		Busca um estudante por ID
//	@Tags			students
//	@Produce		json
//	@Param			id	path		int	true	"ID do estudante"
//	@Success		200	{object}	StudentResponse
//	@Failure		400	{object}	errorResponse	"ID inválido"
//	@Failure		404	{object}	errorResponse	"não encontrado"
//	@Router			/students/{id} [get]
func (s *Server) getStudent(c echo.Context) error {
	id, err := parseIDParam(c)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "path param 'id' must be a positive integer")
	}

	student, err := s.students.FindByID(c.Request().Context(), id)
	if errors.Is(err, models.ErrStudentNotFound) {
		return jsonError(c, http.StatusNotFound, "student not found")
	}
	if err != nil {
		log.Error().Err(err).Msg("[api] failed to get student")
		return jsonError(c, http.StatusInternalServerError, "failed to get student")
	}

	return c.JSON(http.StatusOK, newStudentResponse(*student))
}

// updateStudent godoc
//
//	@Summary		Atualiza um estudante (parcial)
//	@Description	Atualização parcial: apenas os campos enviados são alterados.
//	@Description	As mesmas regras de validação de CPF e e-mail se aplicam.
//	@Tags			students
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"ID do estudante"
//	@Param			student	body		UpdateStudentRequest	true	"Campos a atualizar"
//	@Success		200		{object}	StudentResponse
//	@Failure		400		{object}	errorResponse	"corpo/validação inválidos"
//	@Failure		404		{object}	errorResponse	"não encontrado"
//	@Failure		409		{object}	errorResponse	"CPF já cadastrado"
//	@Router			/students/{id} [put]
func (s *Server) updateStudent(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parseIDParam(c)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "path param 'id' must be a positive integer")
	}

	request := UpdateStudentRequest{}
	if err := c.Bind(&request); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid request body")
	}

	if err := request.Validate(); err != nil {
		return jsonError(c, http.StatusBadRequest, err.Error())
	}

	student, err := s.students.FindByID(ctx, id)
	if errors.Is(err, models.ErrStudentNotFound) {
		return jsonError(c, http.StatusNotFound, "student not found")
	}
	if err != nil {
		log.Error().Err(err).Msg("[api] failed to get student")
		return jsonError(c, http.StatusInternalServerError, "failed to update student")
	}

	if request.CPF != nil && *request.CPF != student.CPF {
		exists, err := s.students.ExistsWithCPF(ctx, *request.CPF)
		if err != nil {
			log.Error().Err(err).Msg("[api] failed to check CPF uniqueness")
			return jsonError(c, http.StatusInternalServerError, "failed to update student")
		}
		if exists {
			return jsonError(c, http.StatusConflict, "a student with this CPF already exists")
		}
	}

	request.apply(student)

	if err := s.students.Update(ctx, student); err != nil {
		log.Error().Err(err).Msg("[api] failed to update student")
		return jsonError(c, http.StatusInternalServerError, "failed to update student")
	}

	return c.JSON(http.StatusOK, newStudentResponse(*student))
}

// deleteStudent godoc
//
//	@Summary		Remove um estudante
//	@Tags			students
//	@Param			id	path	int	true	"ID do estudante"
//	@Success		204	"sem conteúdo"
//	@Failure		400	{object}	errorResponse	"ID inválido"
//	@Failure		404	{object}	errorResponse	"não encontrado"
//	@Router			/students/{id} [delete]
func (s *Server) deleteStudent(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parseIDParam(c)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "path param 'id' must be a positive integer")
	}

	student, err := s.students.FindByID(ctx, id)
	if errors.Is(err, models.ErrStudentNotFound) {
		return jsonError(c, http.StatusNotFound, "student not found")
	}
	if err != nil {
		log.Error().Err(err).Msg("[api] failed to get student")
		return jsonError(c, http.StatusInternalServerError, "failed to delete student")
	}

	if err := s.students.Delete(ctx, student); err != nil {
		log.Error().Err(err).Msg("[api] failed to delete student")
		return jsonError(c, http.StatusInternalServerError, "failed to delete student")
	}

	return c.NoContent(http.StatusNoContent)
}

// apply copia para o estudante apenas os campos presentes na requisição.
func (r *UpdateStudentRequest) apply(student *models.Student) {
	if r.Name != nil {
		student.Name = *r.Name
	}

	if r.CPF != nil {
		student.CPF = *r.CPF
	}

	if r.Email != nil {
		student.Email = *r.Email
	}

	if r.Age != nil {
		student.Age = *r.Age
	}

	if r.Active != nil {
		student.Active = *r.Active
	}
}
