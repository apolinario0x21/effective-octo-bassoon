package api

import (
	"time"

	"github.com/apolinario0x21/effective-octo-bassoon/internal/models"
)

// StudentResponse é a representação pública de um estudante, sem expor
// campos internos do GORM (ex.: DeletedAt).
type StudentResponse struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Name      string    `json:"name"`
	CPF       string    `json:"cpf"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	Active    bool      `json:"active"`
}

type listStudentsResponse struct {
	Students []StudentResponse `json:"students"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func newStudentResponse(student models.Student) StudentResponse {
	return StudentResponse{
		ID:        student.ID,
		CreatedAt: student.CreatedAt,
		UpdatedAt: student.UpdatedAt,
		Name:      student.Name,
		CPF:       student.CPF,
		Email:     student.Email,
		Age:       student.Age,
		Active:    student.Active,
	}
}

func newStudentResponses(students []models.Student) []StudentResponse {
	responses := make([]StudentResponse, 0, len(students))

	for _, student := range students {
		responses = append(responses, newStudentResponse(student))
	}

	return responses
}
