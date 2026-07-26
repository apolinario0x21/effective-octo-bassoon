package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	"github.com/apolinario0x21/students-api/internal/auth"
	"github.com/apolinario0x21/students-api/internal/cache"
	"github.com/apolinario0x21/students-api/internal/config"
	"github.com/apolinario0x21/students-api/internal/crypto"
	"github.com/apolinario0x21/students-api/internal/db"
	"github.com/apolinario0x21/students-api/internal/models"
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
	users := repository.NewUserRepository(database)

	secret := resolveJWTSecret(cfg)
	tokens := auth.NewManager(secret, cfg.Auth.AccessTTL, cfg.Auth.RefreshTTL)

	if cfg.Admin.Enabled() {
		seedAdmin(context.Background(), users, cfg.Admin)
	}

	server := api.NewServer(api.Deps{Students: students, Users: users, Tokens: tokens})

	// Limpeza periódica de refresh tokens expirados/revogados, encerrada no shutdown.
	cleanupCtx, stopCleanup := context.WithCancel(context.Background())
	defer stopCleanup()
	startRefreshTokenCleanup(cleanupCtx, users, cfg.Auth.CleanupInterval)

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
	stopCleanup()

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to shut down server gracefully")
	}
}

// startRefreshTokenCleanup dispara um worker que apaga periodicamente os refresh
// tokens expirados/revogados. Para quando ctx é cancelado (graceful shutdown).
// Um intervalo <= 0 desabilita a limpeza.
func startRefreshTokenCleanup(ctx context.Context, users *repository.GormUserRepository, interval time.Duration) {
	if interval <= 0 {
		log.Info().Msg("Refresh token cleanup disabled")
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				removed, err := users.PurgeExpiredRefreshTokens(ctx, time.Now())
				if err != nil {
					log.Warn().Err(err).Msg("Failed to purge refresh tokens")
					continue
				}
				if removed > 0 {
					log.Info().Int64("removed", removed).Msg("Purged expired/revoked refresh tokens")
				}
			}
		}
	}()
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

// resolveJWTSecret devolve o segredo de assinatura dos JWTs. Em produção ele é
// obrigatório (a aplicação falha no boot se estiver vazio, para nunca rodar com
// segredo hardcoded). Em desenvolvimento, se vazio, gera um segredo efêmero
// aleatório (os tokens não sobrevivem a um restart — aceitável em dev).
func resolveJWTSecret(cfg config.Config) string {
	if cfg.Auth.Secret != "" {
		return cfg.Auth.Secret
	}

	if cfg.IsProduction() {
		log.Fatal().Msg("JWT_SECRET is required in production")
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		log.Fatal().Err(err).Msg("Failed to generate ephemeral JWT secret")
	}
	log.Warn().Msg("JWT_SECRET not set; generated an ephemeral secret (tokens will not survive a restart)")
	return hex.EncodeToString(buf)
}

// seedAdmin cria um usuário admin inicial caso ainda não exista, a partir das
// credenciais em ADMIN_USERNAME/ADMIN_PASSWORD. Falhas são apenas logadas: não
// devem impedir o servidor de subir.
func seedAdmin(ctx context.Context, users *repository.GormUserRepository, seed config.AdminSeed) {
	exists, err := users.ExistsUserWithUsername(ctx, seed.Username)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check admin user; skipping seed")
		return
	}
	if exists {
		return
	}

	hash, err := crypto.HashPassword(seed.Password)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash admin password; skipping seed")
		return
	}

	admin := models.User{Username: seed.Username, PasswordHash: hash, Role: models.RoleAdmin}
	if err := users.CreateUser(ctx, &admin); err != nil {
		log.Error().Err(err).Msg("Failed to create admin user")
		return
	}
	log.Info().Str("username", seed.Username).Msg("Seeded admin user")
}

func configureLogging(level string) {
	zerolog.TimeFieldFormat = time.RFC3339
	parsed, err := zerolog.ParseLevel(level)
	if err != nil {
		parsed = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(parsed)
}
