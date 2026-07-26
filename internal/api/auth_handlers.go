package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/apolinario0x21/students-api/internal/auth"
	"github.com/apolinario0x21/students-api/internal/crypto"
	"github.com/apolinario0x21/students-api/internal/models"
)

const minPasswordLen = 8

// registerRequest é o corpo de POST /auth/register.
type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginRequest é o corpo de POST /auth/login.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// refreshRequest é o corpo de POST /auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// tokenResponse é devolvido por login e refresh.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

// userResponse é a representação pública de um usuário (sem o hash da senha).
type userResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (r registerRequest) validate() error {
	if strings.TrimSpace(r.Username) == "" {
		return errParamRequired("username", "string")
	}
	if len(r.Password) < minPasswordLen {
		return errParamInvalid("password", "must have at least 8 characters")
	}
	return nil
}

// register godoc
//
//	@Summary		Registra um novo usuário
//	@Description	Cria uma conta com papel `user`. A senha é armazenada apenas como
//	@Description	hash bcrypt.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		registerRequest	true	"Usuário e senha (mín. 8 caracteres)"
//	@Success		201			{object}	userResponse
//	@Failure		400			{object}	errorResponse	"validação inválida"
//	@Failure		409			{object}	errorResponse	"usuário já existe"
//	@Router			/auth/register [post]
func (s *Server) register(c echo.Context) error {
	ctx := c.Request().Context()

	request := registerRequest{}
	if err := c.Bind(&request); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid request body")
	}
	if err := request.validate(); err != nil {
		return jsonError(c, http.StatusBadRequest, err.Error())
	}

	exists, err := s.users.ExistsUserWithUsername(ctx, request.Username)
	if err != nil {
		log.Error().Err(err).Msg("[api] failed to check username uniqueness")
		return jsonError(c, http.StatusInternalServerError, "failed to register user")
	}
	if exists {
		return jsonError(c, http.StatusConflict, "a user with this username already exists")
	}

	hash, err := crypto.HashPassword(request.Password)
	if err != nil {
		log.Error().Err(err).Msg("[api] failed to hash password")
		return jsonError(c, http.StatusInternalServerError, "failed to register user")
	}

	user := models.User{Username: request.Username, PasswordHash: hash, Role: models.RoleUser}
	if err := s.users.CreateUser(ctx, &user); err != nil {
		log.Error().Err(err).Msg("[api] failed to create user")
		return jsonError(c, http.StatusInternalServerError, "failed to register user")
	}

	return c.JSON(http.StatusCreated, userResponse{ID: user.ID, Username: user.Username, Role: user.Role})
}

// login godoc
//
//	@Summary		Autentica e emite tokens
//	@Description	Valida as credenciais e devolve um access token (JWT, curto) e um
//	@Description	refresh token (longo).
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		loginRequest	true	"Usuário e senha"
//	@Success		200			{object}	tokenResponse
//	@Failure		400			{object}	errorResponse	"corpo inválido"
//	@Failure		401			{object}	errorResponse	"credenciais inválidas"
//	@Router			/auth/login [post]
func (s *Server) login(c echo.Context) error {
	ctx := c.Request().Context()

	request := loginRequest{}
	if err := c.Bind(&request); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid request body")
	}

	user, err := s.users.FindUserByUsername(ctx, request.Username)
	if errors.Is(err, models.ErrUserNotFound) {
		// Mesma resposta de senha errada: não revela se o usuário existe.
		return jsonError(c, http.StatusUnauthorized, "invalid username or password")
	}
	if err != nil {
		log.Error().Err(err).Msg("[api] failed to find user")
		return jsonError(c, http.StatusInternalServerError, "failed to login")
	}

	if err := crypto.CheckPassword(user.PasswordHash, request.Password); err != nil {
		return jsonError(c, http.StatusUnauthorized, "invalid username or password")
	}

	tokens, err := s.issueTokens(ctx, user)
	if err != nil {
		log.Error().Err(err).Msg("[api] failed to issue tokens")
		return jsonError(c, http.StatusInternalServerError, "failed to login")
	}

	return c.JSON(http.StatusOK, tokens)
}

// refresh godoc
//
//	@Summary		Renova o access token
//	@Description	Troca um refresh token válido por um novo par de tokens. O refresh
//	@Description	usado é revogado (rotação).
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			token	body		refreshRequest	true	"Refresh token"
//	@Success		200		{object}	tokenResponse
//	@Failure		400		{object}	errorResponse	"corpo inválido"
//	@Failure		401		{object}	errorResponse	"refresh token inválido ou expirado"
//	@Router			/auth/refresh [post]
func (s *Server) refresh(c echo.Context) error {
	ctx := c.Request().Context()

	request := refreshRequest{}
	if err := c.Bind(&request); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid request body")
	}
	if request.RefreshToken == "" {
		return jsonError(c, http.StatusBadRequest, "refresh_token is required")
	}

	hash := auth.HashRefreshToken(request.RefreshToken)
	stored, err := s.users.FindRefreshToken(ctx, hash)
	if errors.Is(err, models.ErrRefreshTokenNotFound) {
		return jsonError(c, http.StatusUnauthorized, "invalid or expired refresh token")
	}
	if err != nil {
		log.Error().Err(err).Msg("[api] failed to find refresh token")
		return jsonError(c, http.StatusInternalServerError, "failed to refresh")
	}

	if time.Now().After(stored.ExpiresAt) {
		_ = s.users.RevokeRefreshToken(ctx, hash)
		return jsonError(c, http.StatusUnauthorized, "invalid or expired refresh token")
	}

	user, err := s.users.FindUserByID(ctx, stored.UserID)
	if err != nil {
		log.Error().Err(err).Msg("[api] failed to find user for refresh")
		return jsonError(c, http.StatusUnauthorized, "invalid or expired refresh token")
	}

	// Rotação: revoga o refresh usado antes de emitir um novo par.
	if err := s.users.RevokeRefreshToken(ctx, hash); err != nil {
		log.Error().Err(err).Msg("[api] failed to revoke refresh token")
		return jsonError(c, http.StatusInternalServerError, "failed to refresh")
	}

	tokens, err := s.issueTokens(ctx, user)
	if err != nil {
		log.Error().Err(err).Msg("[api] failed to issue tokens")
		return jsonError(c, http.StatusInternalServerError, "failed to refresh")
	}

	return c.JSON(http.StatusOK, tokens)
}

// logout godoc
//
//	@Summary		Encerra a sessão
//	@Description	Revoga o refresh token informado (encerra a sessão atual). É
//	@Description	idempotente: um token inexistente também devolve 204.
//	@Tags			auth
//	@Accept			json
//	@Param			token	body	refreshRequest	true	"Refresh token a revogar"
//	@Success		204		"sessão encerrada"
//	@Failure		400		{object}	errorResponse	"corpo inválido"
//	@Router			/auth/logout [post]
func (s *Server) logout(c echo.Context) error {
	ctx := c.Request().Context()

	request := refreshRequest{}
	if err := c.Bind(&request); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid request body")
	}
	if request.RefreshToken == "" {
		return jsonError(c, http.StatusBadRequest, "refresh_token is required")
	}

	if err := s.users.RevokeRefreshToken(ctx, auth.HashRefreshToken(request.RefreshToken)); err != nil {
		log.Error().Err(err).Msg("[api] failed to revoke refresh token on logout")
		return jsonError(c, http.StatusInternalServerError, "failed to logout")
	}

	return c.NoContent(http.StatusNoContent)
}

// issueTokens gera um access token e um refresh token novos, persistindo o hash
// do refresh para permitir revogação.
func (s *Server) issueTokens(ctx context.Context, user *models.User) (tokenResponse, error) {
	access, err := s.tokens.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return tokenResponse{}, err
	}

	raw, hash, expiresAt, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return tokenResponse{}, err
	}

	refreshToken := models.RefreshToken{UserID: user.ID, TokenHash: hash, ExpiresAt: expiresAt, Revoked: false}
	if err := s.users.SaveRefreshToken(ctx, &refreshToken); err != nil {
		return tokenResponse{}, err
	}

	return tokenResponse{AccessToken: access, RefreshToken: raw, TokenType: "Bearer"}, nil
}
