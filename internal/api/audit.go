package api

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// auditWrite registra, em log estruturado, uma operação de escrita sobre um
// estudante: quem fez (user_id + papel, vindos do access token), a ação e o id
// do estudante afetado. Correlaciona pelo request_id.
func (s *Server) auditWrite(c echo.Context, action string, studentID uint) {
	userID, _ := c.Get(ctxUserID).(uint)
	role, _ := c.Get(ctxRole).(string)

	log.Info().
		Str("audit", "student").
		Str("action", action).
		Uint("student_id", studentID).
		Uint("user_id", userID).
		Str("role", role).
		Str("request_id", c.Response().Header().Get(echo.HeaderXRequestID)).
		Msg("student write")
}
