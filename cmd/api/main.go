package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/apolinario0x21/students-api/internal/api"
	"github.com/apolinario0x21/students-api/internal/cache"
	"github.com/apolinario0x21/students-api/internal/config"
	"github.com/apolinario0x21/students-api/internal/db"
	"github.com/apolinario0x21/students-api/internal/repository"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg := config.Load()
	configureLogging(cfg.LogLevel)

	database, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	log.Info().Str("driver", cfg.DB.Driver).Msg("Connected to database")

	students := buildRepository(database, cfg.Redis)
	server := api.NewServer(students)

	go func() {
		log.Info().Str("port", cfg.Port).Msg("Starting server")
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

// buildRepository monta o repositório de estudantes, decorando-o com o cache
// Redis quando este estiver habilitado. Sem Redis, o banco é acessado direto.
func buildRepository(database *gorm.DB, redisCfg config.RedisConfig) api.StudentRepository {
	base := repository.NewStudentRepository(database)

	if !redisCfg.Enabled() {
		log.Info().Msg("Redis cache disabled; using database directly")
		return base
	}

	redisClient, err := cache.Connect(context.Background(), redisCfg)
	if err != nil {
		// Cache é um otimizador, não uma dependência dura: seguimos sem ele.
		log.Warn().Err(err).Msg("Redis unavailable; using database directly")
		return base
	}

	log.Info().Str("addr", redisCfg.Addr).Msg("Redis cache enabled")
	return repository.NewCachedStudentRepository(base, redisClient)
}

func configureLogging(level string) {
	zerolog.TimeFieldFormat = time.RFC3339
	parsed, err := zerolog.ParseLevel(level)
	if err != nil {
		parsed = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(parsed)
}
