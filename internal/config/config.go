package config

import "os"

// Config concentra as configurações da aplicação, lidas do ambiente.
type Config struct {
	Port   string
	DBPath string
}

// Load lê as configurações das variáveis de ambiente, com valores padrão.
func Load() Config {
	return Config{
		Port:   getEnv("PORT", "8080"),
		DBPath: getEnv("DB_PATH", "student.db"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
