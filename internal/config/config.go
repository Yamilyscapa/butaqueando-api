package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv            string
	Port              string
	DatabaseURL       string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
}

func Load() (Config, error) {
	appEnv := normalizeEnv(os.Getenv("APP_ENV"))
	if err := loadDotEnv(appEnv); err != nil {
		return Config{}, err
	}

	resolvedAppEnv := normalizeEnv(envOrDefault("APP_ENV", appEnv))

	cfg := Config{
		AppEnv:            resolvedAppEnv,
		Port:              envOrDefault("PORT", "3000"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		DBMaxOpenConns:    intFromEnv("DB_MAX_OPEN_CONNS", 20),
		DBMaxIdleConns:    intFromEnv("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime: durationFromEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func intFromEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func normalizeEnv(value string) string {
	env := strings.ToLower(strings.TrimSpace(value))
	if env == "" {
		return "development"
	}

	return env
}

func loadDotEnv(appEnv string) error {
	if appEnv == "production" {
		return nil
	}

	if err := godotenv.Overload(".env"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("load .env: %w", err)
	}

	return nil
}
