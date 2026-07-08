package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/apolinario0x21/effective-octo-bassoon/internal/api"
	"github.com/apolinario0x21/effective-octo-bassoon/internal/config"
	"github.com/apolinario0x21/effective-octo-bassoon/internal/db"
	"github.com/apolinario0x21/effective-octo-bassoon/internal/repository"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg.DBPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	students := repository.NewStudentRepository(database)
	server := api.NewServer(students)

	go func() {
		if err := server.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to shut down server gracefully")
	}
}
