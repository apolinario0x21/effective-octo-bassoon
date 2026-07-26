package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config concentra as configurações da aplicação, lidas do ambiente.
type Config struct {
	Port     string
	LogLevel string
	AppEnv   string
	DB       DBConfig
	Redis    RedisConfig
	Auth     AuthConfig
	Admin    AdminSeed
}

// IsProduction indica se a aplicação roda em ambiente de produção.
func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// AuthConfig descreve os parâmetros de autenticação (JWT + refresh token).
//
// Secret assina os JWTs e nunca deve ser hardcoded. Em produção é obrigatório
// (a aplicação falha no boot se estiver vazio); em desenvolvimento, quando vazio,
// um segredo efêmero aleatório é gerado.
type AuthConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// AdminSeed permite provisionar um usuário admin inicial no boot. Quando ambos os
// campos estão preenchidos e o usuário ainda não existe, ele é criado.
type AdminSeed struct {
	Username string
	Password string
}

// Enabled indica se o seed de admin deve ser executado.
func (a AdminSeed) Enabled() bool {
	return a.Username != "" && a.Password != ""
}

// DBConfig descreve a conexão com o banco de dados.
//
// Driver aceita "postgres" (produção) ou "sqlite" (desenvolvimento/testes).
// Para "sqlite", apenas DSN é usado (caminho do arquivo, ex.: "student.db").
type DBConfig struct {
	Driver   string
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	DSN      string
}

// DSNString devolve a DSN efetiva do banco. Para Postgres, monta a string a
// partir dos campos individuais quando DSN não foi informada diretamente.
func (c DBConfig) DSNString() string {
	if c.DSN != "" {
		return c.DSN
	}
	if c.Driver == "sqlite" {
		return "student.db"
	}
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// RedisConfig descreve a conexão com o Redis usado como cache.
//
// Quando Addr está vazio, o cache é desabilitado e a aplicação acessa o banco
// diretamente — útil em testes e ambientes sem Redis.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	TTL      time.Duration
}

// Enabled indica se o cache Redis deve ser usado.
func (c RedisConfig) Enabled() bool {
	return c.Addr != ""
}

// Load lê as configurações das variáveis de ambiente, com valores padrão.
func Load() Config {
	return Config{
		Port:     getEnv("PORT", "8080"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		AppEnv:   getEnv("APP_ENV", "development"),
		Auth: AuthConfig{
			Secret:     getEnv("JWT_SECRET", ""),
			AccessTTL:  getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL: getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Admin: AdminSeed{
			Username: getEnv("ADMIN_USERNAME", ""),
			Password: getEnv("ADMIN_PASSWORD", ""),
		},
		DB: DBConfig{
			Driver:   getEnv("DB_DRIVER", "postgres"),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "students"),
			Password: getEnv("DB_PASSWORD", "students"),
			Name:     getEnv("DB_NAME", "students"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			DSN:      getEnv("DB_DSN", ""),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", ""),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			TTL:      getEnvDuration("REDIS_TTL", 5*time.Minute),
		},
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}
