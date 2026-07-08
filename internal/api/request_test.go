package api

import "testing"

// CPF válido usado nos testes (dígitos verificadores corretos).
const validCPF = "52998224725"

func boolPtr(v bool) *bool    { return &v }
func strPtr(v string) *string { return &v }
func intPtr(v int) *int       { return &v }

func TestIsValidCPF(t *testing.T) {
	tests := []struct {
		name string
		cpf  string
		want bool
	}{
		{"valid", "52998224725", true},
		{"wrong check digit", "52998224724", false},
		{"all digits equal", "11111111111", false},
		{"too short", "1234567890", false},
		{"too long", "123456789012", false},
		{"formatted", "529.982.247-25", false},
		{"letters", "5299822472a", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidCPF(tt.cpf); got != tt.want {
				t.Errorf("isValidCPF(%q) = %v, want %v", tt.cpf, got, tt.want)
			}
		})
	}
}

func TestCreateStudentRequestValidate(t *testing.T) {
	valid := func() CreateStudentRequest {
		return CreateStudentRequest{
			Name:   "Maria",
			CPF:    validCPF,
			Email:  "maria@example.com",
			Age:    22,
			Active: boolPtr(true),
		}
	}

	tests := []struct {
		name    string
		mutate  func(r *CreateStudentRequest)
		wantErr bool
	}{
		{"valid", func(r *CreateStudentRequest) {}, false},
		{"missing name", func(r *CreateStudentRequest) { r.Name = "" }, true},
		{"blank name", func(r *CreateStudentRequest) { r.Name = "   " }, true},
		{"missing cpf", func(r *CreateStudentRequest) { r.CPF = "" }, true},
		{"invalid cpf", func(r *CreateStudentRequest) { r.CPF = "12345678901" }, true},
		{"missing email", func(r *CreateStudentRequest) { r.Email = "" }, true},
		{"invalid email", func(r *CreateStudentRequest) { r.Email = "not-an-email" }, true},
		{"missing age", func(r *CreateStudentRequest) { r.Age = 0 }, true},
		{"negative age", func(r *CreateStudentRequest) { r.Age = -1 }, true},
		{"missing active", func(r *CreateStudentRequest) { r.Active = nil }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := valid()
			tt.mutate(&request)

			err := request.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateStudentRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request UpdateStudentRequest
		wantErr bool
	}{
		{"empty request", UpdateStudentRequest{}, false},
		{"valid partial", UpdateStudentRequest{Age: intPtr(30)}, false},
		{"valid full", UpdateStudentRequest{
			Name:   strPtr("Ana"),
			CPF:    strPtr(validCPF),
			Email:  strPtr("ana@example.com"),
			Age:    intPtr(25),
			Active: boolPtr(false),
		}, false},
		{"blank name", UpdateStudentRequest{Name: strPtr(" ")}, true},
		{"invalid cpf", UpdateStudentRequest{CPF: strPtr("123")}, true},
		{"invalid email", UpdateStudentRequest{Email: strPtr("nope")}, true},
		{"invalid age", UpdateStudentRequest{Age: intPtr(0)}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
