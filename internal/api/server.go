package api

import (
	"context"
	"net/http"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"

	"github.com/apolinario0x21/students-api/internal/auth"
)

// Server encapsula o Echo e as dependências dos handlers.
type Server struct {
	echo     *echo.Echo
	students StudentRepository
	users    UserRepository
	tokens   *auth.Manager
	registry *prometheus.Registry
}

// Deps agrupa as dependências injetadas no servidor.
type Deps struct {
	Students StudentRepository
	Users    UserRepository
	Tokens   *auth.Manager
}

func NewServer(deps Deps) *Server {
	e := echo.New()
	e.HideBanner = true

	// Registry dedicado por servidor: evita colisões de registro (ex.: vários
	// servidores no mesmo processo durante os testes) no registry global.
	registry := prometheus.NewRegistry()

	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(requestLogger())
	// Coleta métricas Prometheus (contadores e histogramas) de todas as rotas.
	e.Use(echoprometheus.NewMiddlewareWithConfig(echoprometheus.MiddlewareConfig{
		Subsystem:  "students",
		Registerer: registry,
	}))

	server := &Server{
		echo:     e,
		students: deps.Students,
		users:    deps.Users,
		tokens:   deps.Tokens,
		registry: registry,
	}
	server.registerRoutes()

	return server
}

func (s *Server) registerRoutes() {
	// Rotas abertas (sem token): observabilidade, docs e autenticação.
	s.echo.GET("/healthz", s.healthCheck)
	s.echo.GET("/metrics", s.metrics())
	registerSwagger(s.echo)

	s.echo.POST("/auth/register", s.register)
	s.echo.POST("/auth/login", s.login)
	s.echo.POST("/auth/refresh", s.refresh)
	s.echo.POST("/auth/logout", s.logout)

	// Leitura: qualquer usuário autenticado (user ou admin).
	read := s.echo.Group("", s.requireAuth)
	read.GET("/students", s.listStudents)
	read.GET("/students/:id", s.getStudent)

	// Escrita: apenas admin.
	write := s.echo.Group("", s.requireAuth, s.requireAdmin)
	write.POST("/students", s.createStudent)
	write.PUT("/students/:id", s.updateStudent)
	write.DELETE("/students/:id", s.deleteStudent)
}

// metrics devolve o handler das métricas Prometheus.
//
//	@Summary		Métricas Prometheus
//	@Description	Expõe contadores e histogramas por rota/método/status no formato
//	@Description	de texto do Prometheus.
//	@Tags			observabilidade
//	@Produce		plain
//	@Success		200	{string}	string	"Métricas no formato Prometheus"
//	@Router			/metrics [get]
func (s *Server) metrics() echo.HandlerFunc {
	return echoprometheus.NewHandlerWithConfig(echoprometheus.HandlerConfig{
		Gatherer: s.registry,
	})
}

func (s *Server) Start(address string) error {
	return s.echo.Start(address)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

// ServeHTTP torna o Server um http.Handler, útil em testes.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.echo.ServeHTTP(w, r)
}

// requestLogger devolve um middleware que registra cada requisição em JSON
// estruturado via zerolog, incluindo o request ID. As rotas de observabilidade
// são ignoradas para não poluir os logs.
func requestLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		Skipper: func(c echo.Context) bool {
			path := c.Request().URL.Path
			return path == "/metrics" || path == "/healthz"
		},
		LogMethod:    true,
		LogURI:       true,
		LogStatus:    true,
		LogLatency:   true,
		LogRequestID: true,
		LogError:     true,
		HandleError:  true,
		LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
			event := log.Info()
			if v.Error != nil {
				event = log.Error().Err(v.Error)
			}
			event.
				Str("request_id", v.RequestID).
				Str("method", v.Method).
				Str("uri", v.URI).
				Int("status", v.Status).
				Dur("latency", v.Latency).
				Msg("request")
			return nil
		},
	})
}
