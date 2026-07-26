package api

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/apolinario0x21/students-api/internal/models"
)

// Chaves usadas para propagar a identidade autenticada pelo contexto do Echo.
const (
	ctxUserID = "auth_user_id"
	ctxRole   = "auth_role"
)

// requireAuth valida o access token (Bearer) e injeta o ID e o papel do usuário
// no contexto. Responde 401 quando o token está ausente, malformado, expirado ou
// com assinatura inválida.
func (s *Server) requireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		raw, err := bearerToken(c)
		if err != nil {
			return jsonError(c, http.StatusUnauthorized, err.Error())
		}

		claims, err := s.tokens.ParseAccessToken(raw)
		if err != nil {
			return jsonError(c, http.StatusUnauthorized, "invalid or expired token")
		}

		userID, err := claims.UserID()
		if err != nil {
			return jsonError(c, http.StatusUnauthorized, "invalid token subject")
		}

		c.Set(ctxUserID, userID)
		c.Set(ctxRole, claims.Role)
		return next(c)
	}
}

// requireAdmin autoriza apenas usuários com papel admin. Deve ser encadeado
// depois de requireAuth. Responde 403 quando o papel é insuficiente.
func (s *Server) requireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		role, _ := c.Get(ctxRole).(string)
		if role != models.RoleAdmin {
			return jsonError(c, http.StatusForbidden, "admin role required")
		}
		return next(c)
	}
}

// bearerToken extrai o token do cabeçalho Authorization: Bearer <token>.
func bearerToken(c echo.Context) (string, error) {
	header := c.Request().Header.Get(echo.HeaderAuthorization)
	if header == "" {
		return "", echoError("authorization header is required")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", echoError("authorization header must be a Bearer token")
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", echoError("bearer token is empty")
	}
	return token, nil
}

// echoError é um erro simples com mensagem, usado para respostas 401.
type echoError string

func (e echoError) Error() string { return string(e) }
