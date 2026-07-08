package api

import (
	"fmt"
	"net/mail"
	"strings"
)

// CreateStudentRequest é o corpo esperado em POST /students.
type CreateStudentRequest struct {
	Name   string `json:"name"`
	CPF    string `json:"cpf"`
	Email  string `json:"email"`
	Age    int    `json:"age"`
	Active *bool  `json:"active"` // ponteiro para diferenciar "false" de "campo ausente"
}

// UpdateStudentRequest é o corpo esperado em PUT /students/:id.
// Todos os campos são opcionais; apenas os enviados são atualizados.
type UpdateStudentRequest struct {
	Name   *string `json:"name"`
	CPF    *string `json:"cpf"`
	Email  *string `json:"email"`
	Age    *int    `json:"age"`
	Active *bool   `json:"active"`
}

func errParamRequired(param, typ string) error {
	return fmt.Errorf("param '%s' of type '%s' is required", param, typ)
}

func errParamInvalid(param, reason string) error {
	return fmt.Errorf("param '%s' is invalid: %s", param, reason)
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errParamInvalid("name", "must not be empty")
	}
	return nil
}

func validateCPF(cpf string) error {
	if !isValidCPF(cpf) {
		return errParamInvalid("cpf", "must be a valid CPF with 11 digits")
	}
	return nil
}

func validateEmail(email string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return errParamInvalid("email", "must be a valid email address")
	}
	return nil
}

func validateAge(age int) error {
	if age <= 0 {
		return errParamInvalid("age", "must be a positive integer")
	}
	return nil
}

func (r *CreateStudentRequest) Validate() error {
	if r.Name == "" {
		return errParamRequired("name", "string")
	}
	if err := validateName(r.Name); err != nil {
		return err
	}

	if r.CPF == "" {
		return errParamRequired("cpf", "string")
	}
	if err := validateCPF(r.CPF); err != nil {
		return err
	}

	if r.Email == "" {
		return errParamRequired("email", "string")
	}
	if err := validateEmail(r.Email); err != nil {
		return err
	}

	if r.Age == 0 {
		return errParamRequired("age", "int")
	}
	if err := validateAge(r.Age); err != nil {
		return err
	}

	if r.Active == nil {
		return errParamRequired("active", "bool")
	}

	return nil
}

func (r *UpdateStudentRequest) Validate() error {
	if r.Name != nil {
		if err := validateName(*r.Name); err != nil {
			return err
		}
	}

	if r.CPF != nil {
		if err := validateCPF(*r.CPF); err != nil {
			return err
		}
	}

	if r.Email != nil {
		if err := validateEmail(*r.Email); err != nil {
			return err
		}
	}

	if r.Age != nil {
		if err := validateAge(*r.Age); err != nil {
			return err
		}
	}

	return nil
}
