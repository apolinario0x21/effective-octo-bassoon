package api_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// TestAuditLogOnWrites verifica que create/update/delete emitem uma linha de
// auditoria com quem fez (user_id + role), a ação e o student_id.
func TestAuditLogOnWrites(t *testing.T) {
	var buf bytes.Buffer
	original := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = original }()

	server := newTestServer(t) // doRequest autentica como admin (user_id=1)

	created := createMaria(t, server)
	doRequest(t, server, http.MethodPut, "/students/1", `{"age":23}`)
	doRequest(t, server, http.MethodDelete, "/students/1", "")

	out := buf.String()

	// Identidade e correlação comuns a todas as escritas.
	for _, want := range []string{`"audit":"student"`, `"user_id":1`, `"role":"admin"`} {
		if !strings.Contains(out, want) {
			t.Errorf("log de auditoria não contém %s\n%s", want, out)
		}
	}

	// Uma linha por ação, com o student_id afetado.
	for _, action := range []string{"create", "update", "delete"} {
		if !strings.Contains(out, `"action":"`+action+`"`) {
			t.Errorf("faltou auditoria da ação %q", action)
		}
	}
	if !strings.Contains(out, `"student_id":1`) {
		t.Errorf("auditoria não registrou o student_id=%d", created.ID)
	}
}

// TestAuditLogNotEmittedOnForbiddenWrite garante que uma escrita barrada por
// autorização (403) não gera linha de auditoria.
func TestAuditLogNotEmittedOnForbiddenWrite(t *testing.T) {
	var buf bytes.Buffer
	original := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = original }()

	server := newTestServer(t)
	userToken := server.token(t, "user")

	rec := doRequestToken(t, server, http.MethodPost, "/students",
		`{"name":"X","cpf":"`+cpfMaria+`","email":"x@e.com","age":20,"active":true}`, userToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}

	if strings.Contains(buf.String(), `"audit":"student"`) {
		t.Error("escrita negada (403) não deveria gerar auditoria")
	}
}
