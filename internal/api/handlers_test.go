package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/apolinario0x21/effective-octo-bassoon/internal/api"
	"github.com/apolinario0x21/effective-octo-bassoon/internal/config"
	"github.com/apolinario0x21/effective-octo-bassoon/internal/db"
	"github.com/apolinario0x21/effective-octo-bassoon/internal/repository"
)

// CPFs válidos (dígitos verificadores corretos) para os testes.
const (
	cpfMaria = "52998224725"
	cpfJoao  = "15350946056"
)

func newTestServer(t *testing.T) *api.Server {
	t.Helper()

	database, err := db.Connect(config.DBConfig{
		Driver: "sqlite",
		DSN:    filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	return api.NewServer(repository.NewStudentRepository(database))
}

func doRequest(t *testing.T, server *api.Server, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	return recorder
}

func decodeStudent(t *testing.T, body string) api.StudentResponse {
	t.Helper()

	student := api.StudentResponse{}
	if err := json.Unmarshal([]byte(body), &student); err != nil {
		t.Fatalf("failed to decode student response %q: %v", body, err)
	}

	return student
}

func decodeStudentList(t *testing.T, body string) []api.StudentResponse {
	t.Helper()

	response := struct {
		Students []api.StudentResponse `json:"students"`
	}{}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("failed to decode student list %q: %v", body, err)
	}

	return response.Students
}

func createMaria(t *testing.T, server *api.Server) api.StudentResponse {
	t.Helper()

	recorder := doRequest(t, server, http.MethodPost, "/students",
		`{"name":"Maria","cpf":"`+cpfMaria+`","email":"maria@example.com","age":22,"active":true}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("failed to create student: status %d, body %s", recorder.Code, recorder.Body.String())
	}

	return decodeStudent(t, recorder.Body.String())
}

func TestHealthCheck(t *testing.T) {
	server := newTestServer(t)

	recorder := doRequest(t, server, http.MethodGet, "/healthz", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestCreateStudent(t *testing.T) {
	server := newTestServer(t)

	student := createMaria(t, server)
	if student.ID != 1 || student.Name != "Maria" || student.CPF != cpfMaria || !student.Active {
		t.Errorf("unexpected student response: %+v", student)
	}
}

func TestCreateStudentDuplicateCPF(t *testing.T) {
	server := newTestServer(t)
	createMaria(t, server)

	recorder := doRequest(t, server, http.MethodPost, "/students",
		`{"name":"Outra Maria","cpf":"`+cpfMaria+`","email":"outra@example.com","age":30,"active":false}`)
	if recorder.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
}

func TestCreateStudentInvalidPayload(t *testing.T) {
	server := newTestServer(t)

	tests := []struct {
		name string
		body string
	}{
		{"missing active", `{"name":"X","cpf":"` + cpfMaria + `","email":"x@example.com","age":20}`},
		{"invalid cpf", `{"name":"X","cpf":"12345678901","email":"x@example.com","age":20,"active":true}`},
		{"malformed json", `{"name":`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := doRequest(t, server, http.MethodPost, "/students", tt.body)
			if recorder.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestListStudents(t *testing.T) {
	server := newTestServer(t)
	createMaria(t, server)

	recorder := doRequest(t, server, http.MethodPost, "/students",
		`{"name":"Joao","cpf":"`+cpfJoao+`","email":"joao@example.com","age":30,"active":false}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("failed to create second student: %s", recorder.Body.String())
	}

	recorder = doRequest(t, server, http.MethodGet, "/students", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if students := decodeStudentList(t, recorder.Body.String()); len(students) != 2 {
		t.Errorf("len(students) = %d, want 2", len(students))
	}

	recorder = doRequest(t, server, http.MethodGet, "/students?active=true", "")
	students := decodeStudentList(t, recorder.Body.String())
	if len(students) != 1 || students[0].Name != "Maria" {
		t.Errorf("filtered students = %+v, want only Maria", students)
	}

	recorder = doRequest(t, server, http.MethodGet, "/students?active=banana", "")
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestGetStudent(t *testing.T) {
	server := newTestServer(t)
	created := createMaria(t, server)

	recorder := doRequest(t, server, http.MethodGet, "/students/1", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if student := decodeStudent(t, recorder.Body.String()); student.ID != created.ID {
		t.Errorf("student.ID = %d, want %d", student.ID, created.ID)
	}

	recorder = doRequest(t, server, http.MethodGet, "/students/999", "")
	if recorder.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}

	recorder = doRequest(t, server, http.MethodGet, "/students/abc", "")
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestUpdateStudentPartial(t *testing.T) {
	server := newTestServer(t)
	createMaria(t, server)

	recorder := doRequest(t, server, http.MethodPut, "/students/1", `{"age":23}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	student := decodeStudent(t, recorder.Body.String())
	if student.Age != 23 {
		t.Errorf("student.Age = %d, want 23", student.Age)
	}
	if !student.Active {
		t.Errorf("student.Active was clobbered by partial update, want true")
	}

	recorder = doRequest(t, server, http.MethodPut, "/students/1", `{"active":false}`)
	if student := decodeStudent(t, recorder.Body.String()); student.Active {
		t.Errorf("student.Active = true, want false")
	}
}

func TestUpdateStudentErrors(t *testing.T) {
	server := newTestServer(t)
	createMaria(t, server)

	recorder := doRequest(t, server, http.MethodPost, "/students",
		`{"name":"Joao","cpf":"`+cpfJoao+`","email":"joao@example.com","age":30,"active":false}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("failed to create second student: %s", recorder.Body.String())
	}

	recorder = doRequest(t, server, http.MethodPut, "/students/999", `{"age":23}`)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}

	recorder = doRequest(t, server, http.MethodPut, "/students/1", `{"email":"nope"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	recorder = doRequest(t, server, http.MethodPut, "/students/2", `{"cpf":"`+cpfMaria+`"}`)
	if recorder.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}

	recorder = doRequest(t, server, http.MethodPut, "/students/1", `{"cpf":"`+cpfMaria+`"}`)
	if recorder.Code != http.StatusOK {
		t.Errorf("updating student keeping its own CPF: status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestDeleteStudent(t *testing.T) {
	server := newTestServer(t)
	createMaria(t, server)

	recorder := doRequest(t, server, http.MethodDelete, "/students/1", "")
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}

	recorder = doRequest(t, server, http.MethodGet, "/students/1", "")
	if recorder.Code != http.StatusNotFound {
		t.Errorf("status after delete = %d, want %d", recorder.Code, http.StatusNotFound)
	}

	recorder = doRequest(t, server, http.MethodDelete, "/students/1", "")
	if recorder.Code != http.StatusNotFound {
		t.Errorf("second delete status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
