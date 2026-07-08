package api

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Server encapsula o Echo e as dependências dos handlers.
type Server struct {
	echo     *echo.Echo
	students StudentRepository
}

func NewServer(students StudentRepository) *Server {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	server := &Server{
		echo:     e,
		students: students,
	}
	server.registerRoutes()

	return server
}

func (s *Server) registerRoutes() {
	s.echo.GET("/healthz", s.healthCheck)

	s.echo.GET("/students", s.listStudents)
	s.echo.POST("/students", s.createStudent)
	s.echo.GET("/students/:id", s.getStudent)
	s.echo.PUT("/students/:id", s.updateStudent)
	s.echo.DELETE("/students/:id", s.deleteStudent)
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
