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
	AppEnv                    string
	Port                      string
	DatabaseURL               string
	DBMaxOpenConns            int
	DBMaxIdleConns            int
	DBConnMaxLifetime         time.Duration
	JWTIssuer                 string
	JWTAccessSecret           string
	JWTRefreshSecret          string
	JWTAccessTTL              time.Duration
	JWTRefreshTTL             time.Duration
	EmailVerificationRequired bool
	ResendAPIKey              string
	ResendFromEmail           string
	ResendTemplateLoginCode   string
	EmailVerificationRedirect string
}

func Load() (Config, error) {
	appEnv := normalizeEnv(os.Getenv("APP_ENV"))
	if err := loadDotEnv(appEnv); err != nil {
		return Config{}, err
	}

	resolvedAppEnv := normalizeEnv(envOrDefault("APP_ENV", appEnv))
	jwtAccessTTL, err := durationFromEnvStrict("JWT_ACCESS_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}

	jwtRefreshTTL, err := durationFromEnvStrict("JWT_REFRESH_TTL", 30*24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	emailVerificationRequired, err := boolFromEnvStrict("EMAIL_VERIFICATION_REQUIRED", true)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:                    resolvedAppEnv,
		Port:                      envOrDefault("PORT", "3000"),
		DatabaseURL:               os.Getenv("DATABASE_URL"),
		DBMaxOpenConns:            intFromEnv("DB_MAX_OPEN_CONNS", 20),
		DBMaxIdleConns:            intFromEnv("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime:         durationFromEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		JWTIssuer:                 envOrDefault("JWT_ISSUER", "butaqueando-api"),
		JWTAccessSecret:           os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:          os.Getenv("JWT_REFRESH_SECRET"),
		JWTAccessTTL:              jwtAccessTTL,
		JWTRefreshTTL:             jwtRefreshTTL,
		EmailVerificationRequired: emailVerificationRequired,
		ResendAPIKey:              strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
		ResendFromEmail:           strings.TrimSpace(os.Getenv("RESEND_FROM_EMAIL")),
		ResendTemplateLoginCode:   envOrDefault("RESEND_TEMPLATE_LOGIN_CODE", "login-code"),
		EmailVerificationRedirect: strings.TrimSpace(os.Getenv("EMAIL_VERIFICATION_REDIRECT_BASE")),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	if strings.TrimSpace(cfg.JWTIssuer) == "" {
		return Config{}, fmt.Errorf("JWT_ISSUER is required")
	}

	if len(cfg.JWTAccessSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_ACCESS_SECRET must be at least 32 characters")
	}

	if len(cfg.JWTRefreshSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_REFRESH_SECRET must be at least 32 characters")
	}

	if cfg.JWTAccessTTL <= 0 {
		return Config{}, fmt.Errorf("JWT_ACCESS_TTL must be greater than 0")
	}

	if cfg.JWTRefreshTTL <= 0 {
		return Config{}, fmt.Errorf("JWT_REFRESH_TTL must be greater than 0")
	}

	if cfg.AppEnv == "production" && !cfg.EmailVerificationRequired {
		return Config{}, fmt.Errorf("EMAIL_VERIFICATION_REQUIRED must be true in production")
	}

	if cfg.EmailVerificationRequired && cfg.EmailVerificationRedirect == "" {
		return Config{}, fmt.Errorf("EMAIL_VERIFICATION_REDIRECT_BASE is required when EMAIL_VERIFICATION_REQUIRED is true")
	}

	if cfg.AppEnv == "production" && cfg.EmailVerificationRequired {
		if cfg.ResendAPIKey == "" {
			return Config{}, fmt.Errorf("RESEND_API_KEY is required when EMAIL_VERIFICATION_REQUIRED is true in production")
		}

		if cfg.ResendFromEmail == "" {
			return Config{}, fmt.Errorf("RESEND_FROM_EMAIL is required when EMAIL_VERIFICATION_REQUIRED is true in production")
		}
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

func durationFromEnvStrict(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}

	return parsed, nil
}

func boolFromEnvStrict(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a valid boolean: %w", key, err)
	}

	return parsed, nil
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
