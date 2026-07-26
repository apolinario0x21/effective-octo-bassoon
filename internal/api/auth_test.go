package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// tokens espelha o corpo devolvido por /auth/login e /auth/refresh.
type tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

func decodeTokens(t *testing.T, body string) tokens {
	t.Helper()
	tk := tokens{}
	if err := json.Unmarshal([]byte(body), &tk); err != nil {
		t.Fatalf("failed to decode tokens %q: %v", body, err)
	}
	return tk
}

func TestRegisterAndLoginFlow(t *testing.T) {
	server := newTestServer(t)

	// Registro cria um usuário com papel "user".
	rec := doRequestNoAuth(t, server, http.MethodPost, "/auth/register",
		`{"username":"alice","password":"password123"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if body := rec.Body.String(); strings.Contains(body, "password") || strings.Contains(body, "hash") {
		t.Errorf("register response vazou senha/hash: %s", body)
	}

	// Login devolve os tokens.
	rec = doRequestNoAuth(t, server, http.MethodPost, "/auth/login",
		`{"username":"alice","password":"password123"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	tk := decodeTokens(t, rec.Body.String())
	if tk.AccessToken == "" || tk.RefreshToken == "" || tk.TokenType != "Bearer" {
		t.Fatalf("tokens incompletos: %+v", tk)
	}

	// O access token de um "user" lê, mas não escreve.
	if rec := doRequestToken(t, server, http.MethodGet, "/students", "", tk.AccessToken); rec.Code != http.StatusOK {
		t.Errorf("GET /students como user: status = %d, want 200", rec.Code)
	}
	if rec := doRequestToken(t, server, http.MethodPost, "/students",
		`{"name":"X","cpf":"`+cpfMaria+`","email":"x@e.com","age":20,"active":true}`, tk.AccessToken); rec.Code != http.StatusForbidden {
		t.Errorf("POST /students como user: status = %d, want 403", rec.Code)
	}

	// Refresh troca por um novo par e revoga o refresh usado.
	rec = doRequestNoAuth(t, server, http.MethodPost, "/auth/refresh",
		`{"refresh_token":"`+tk.RefreshToken+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	newTk := decodeTokens(t, rec.Body.String())
	if newTk.AccessToken == "" || newTk.RefreshToken == "" {
		t.Fatalf("refresh não devolveu novos tokens: %+v", newTk)
	}

	// O refresh token antigo agora está revogado.
	if rec := doRequestNoAuth(t, server, http.MethodPost, "/auth/refresh",
		`{"refresh_token":"`+tk.RefreshToken+`"}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("refresh com token revogado: status = %d, want 401", rec.Code)
	}
}

func TestRegisterValidation(t *testing.T) {
	server := newTestServer(t)

	// Senha curta.
	if rec := doRequestNoAuth(t, server, http.MethodPost, "/auth/register",
		`{"username":"bob","password":"short"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("senha curta: status = %d, want 400", rec.Code)
	}

	// Usuário duplicado.
	doRequestNoAuth(t, server, http.MethodPost, "/auth/register", `{"username":"carol","password":"password123"}`)
	if rec := doRequestNoAuth(t, server, http.MethodPost, "/auth/register",
		`{"username":"carol","password":"password123"}`); rec.Code != http.StatusConflict {
		t.Errorf("usuário duplicado: status = %d, want 409", rec.Code)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	server := newTestServer(t)
	doRequestNoAuth(t, server, http.MethodPost, "/auth/register", `{"username":"dave","password":"password123"}`)

	tests := map[string]string{
		"senha errada":        `{"username":"dave","password":"wrongpass1"}`,
		"usuário inexistente": `{"username":"ghost","password":"password123"}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			if rec := doRequestNoAuth(t, server, http.MethodPost, "/auth/login", body); rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
		})
	}
}

func TestWriteRoutesRequireAdmin(t *testing.T) {
	server := newTestServer(t)
	userToken := server.token(t, "user")
	body := `{"name":"X","cpf":"` + cpfMaria + `","email":"x@e.com","age":20,"active":true}`

	writes := []struct {
		method, target, body string
	}{
		{http.MethodPost, "/students", body},
		{http.MethodPut, "/students/1", `{"age":30}`},
		{http.MethodDelete, "/students/1", ""},
	}

	for _, w := range writes {
		t.Run("sem token "+w.method, func(t *testing.T) {
			if rec := doRequestNoAuth(t, server, w.method, w.target, w.body); rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
		})
		t.Run("como user "+w.method, func(t *testing.T) {
			if rec := doRequestToken(t, server, w.method, w.target, w.body, userToken); rec.Code != http.StatusForbidden {
				t.Errorf("status = %d, want 403", rec.Code)
			}
		})
	}
}

func TestReadRoutesRequireAuth(t *testing.T) {
	server := newTestServer(t)

	for _, target := range []string{"/students", "/students/1"} {
		if rec := doRequestNoAuth(t, server, http.MethodGet, target, ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("GET %s sem token: status = %d, want 401", target, rec.Code)
		}
	}
}

func TestOpenRoutesNeedNoAuth(t *testing.T) {
	server := newTestServer(t)

	for _, target := range []string{"/healthz", "/metrics"} {
		if rec := doRequestNoAuth(t, server, http.MethodGet, target, ""); rec.Code != http.StatusOK {
			t.Errorf("GET %s sem token: status = %d, want 200", target, rec.Code)
		}
	}
}
