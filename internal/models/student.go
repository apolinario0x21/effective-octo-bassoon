package models

import (
	"errors"

	"gorm.io/gorm"
)

// ErrStudentNotFound é retornado pelos repositórios quando o estudante não existe.
var ErrStudentNotFound = errors.New("student not found")

// Student é a entidade persistida no banco de dados.
// CPF é armazenado como string para preservar zeros à esquerda.
type Student struct {
	gorm.Model
	Name   string `gorm:"not null"`
	CPF    string `gorm:"size:11;index;not null"`
	Email  string `gorm:"not null"`
	Age    int    `gorm:"not null"`
	Active bool   `gorm:"not null"`
}

// ListParams controla o filtro e a paginação da listagem de estudantes.
// Active nulo significa "todos"; caso contrário, filtra por ativo/inativo.
type ListParams struct {
	Active *bool
	Limit  int
	Offset int
}

// StudentPage é uma página de estudantes acompanhada do total de registros
// que satisfazem o filtro (ignorando limit/offset).
type StudentPage struct {
	Students []Student `json:"students"`
	Total    int64     `json:"total"`
}
