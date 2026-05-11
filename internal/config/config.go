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
	AppEnv                      string
	Port                        string
	DatabaseURL                 string
	DBMaxOpenConns              int
	DBMaxIdleConns              int
	DBConnMaxLifetime           time.Duration
	S3UploadURLTTL              time.Duration
	S3DownloadURLTTL            time.Duration
	S3MaxImageBytes             int64
	ImageOptimizationEnabled    bool
	ImageWebPQuality            int
	ImageWorkerPoolSize         int
	ImageBlurhashEnabled        bool
	ImageVariantWidthsPlays     []int
	ImageVariantWidthsAvatars   []int
	PlaysS3                     S3BucketConfig
	UsersS3                     S3BucketConfig
	JWTIssuer                   string
	JWTAccessSecret             string
	JWTRefreshSecret            string
	JWTAccessTTL                time.Duration
	JWTRefreshTTL               time.Duration
	EmailVerificationRequired   bool
	PasswordResetTokenTTL       time.Duration
	RedisURL                    string
	ResendAPIKey                string
	ResendFromEmail             string
	ResendTemplateLoginCode     string
	ResendTemplatePasswordReset string
	EmailVerificationRedirect   string
	PasswordResetRedirect       string
}

type S3BucketConfig struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
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

	passwordResetTokenTTL, err := durationFromEnvStrict("PASSWORD_RESET_TOKEN_TTL", time.Hour)
	if err != nil {
		return Config{}, err
	}

	s3UploadURLTTL, err := durationFromEnvStrict("S3_UPLOAD_URL_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}

	s3DownloadURLTTL, err := durationFromEnvStrict("S3_DOWNLOAD_URL_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}

	s3MaxImageBytes, err := int64FromEnvStrict("S3_MAX_IMAGE_BYTES", 10*1024*1024)
	if err != nil {
		return Config{}, err
	}

	imageOptimizationEnabled, err := boolFromEnvStrict("IMAGE_OPTIMIZATION_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	imageWebPQuality := intFromEnv("IMAGE_WEBP_QUALITY", 75)
	imageWorkerPoolSize := intFromEnv("IMAGE_WORKER_POOL_SIZE", 2)
	imageBlurhashEnabled, err := boolFromEnvStrict("IMAGE_BLURHASH_ENABLED", true)
	if err != nil {
		return Config{}, err
	}
	imageVariantWidthsPlays, err := intListFromEnv("IMAGE_VARIANT_WIDTHS_PLAYS", []int{320, 720, 1080})
	if err != nil {
		return Config{}, err
	}
	imageVariantWidthsAvatars, err := intListFromEnv("IMAGE_VARIANT_WIDTHS_AVATARS", []int{96, 240, 512})
	if err != nil {
		return Config{}, err
	}

	playsS3 := S3BucketConfig{
		Endpoint:        strings.TrimSpace(os.Getenv("PLAYS_S3_ENDPOINT")),
		Region:          strings.TrimSpace(os.Getenv("PLAYS_S3_REGION")),
		AccessKeyID:     strings.TrimSpace(os.Getenv("PLAYS_S3_ACCESS_KEY_ID")),
		SecretAccessKey: strings.TrimSpace(os.Getenv("PLAYS_S3_SECRET_ACCESS_KEY")),
		Bucket:          strings.TrimSpace(os.Getenv("PLAYS_S3_BUCKET")),
	}

	usersS3 := S3BucketConfig{
		Endpoint:        strings.TrimSpace(os.Getenv("USERS_S3_ENDPOINT")),
		Region:          strings.TrimSpace(os.Getenv("USERS_S3_REGION")),
		AccessKeyID:     strings.TrimSpace(os.Getenv("USERS_S3_ACCESS_KEY_ID")),
		SecretAccessKey: strings.TrimSpace(os.Getenv("USERS_S3_SECRET_ACCESS_KEY")),
		Bucket:          strings.TrimSpace(os.Getenv("USERS_S3_BUCKET")),
	}

	cfg := Config{
		AppEnv:                      resolvedAppEnv,
		Port:                        envOrDefault("PORT", "3000"),
		DatabaseURL:                 os.Getenv("DATABASE_URL"),
		DBMaxOpenConns:              intFromEnv("DB_MAX_OPEN_CONNS", 20),
		DBMaxIdleConns:              intFromEnv("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime:           durationFromEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		S3UploadURLTTL:              s3UploadURLTTL,
		S3DownloadURLTTL:            s3DownloadURLTTL,
		S3MaxImageBytes:             s3MaxImageBytes,
		ImageOptimizationEnabled:    imageOptimizationEnabled,
		ImageWebPQuality:            imageWebPQuality,
		ImageWorkerPoolSize:         imageWorkerPoolSize,
		ImageBlurhashEnabled:        imageBlurhashEnabled,
		ImageVariantWidthsPlays:     imageVariantWidthsPlays,
		ImageVariantWidthsAvatars:   imageVariantWidthsAvatars,
		PlaysS3:                     playsS3,
		UsersS3:                     usersS3,
		JWTIssuer:                   envOrDefault("JWT_ISSUER", "butaqueando-api"),
		JWTAccessSecret:             os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:            os.Getenv("JWT_REFRESH_SECRET"),
		JWTAccessTTL:                jwtAccessTTL,
		JWTRefreshTTL:               jwtRefreshTTL,
		EmailVerificationRequired:   emailVerificationRequired,
		PasswordResetTokenTTL:       passwordResetTokenTTL,
		RedisURL:                    strings.TrimSpace(os.Getenv("REDIS_URL")),
		ResendAPIKey:                strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
		ResendFromEmail:             strings.TrimSpace(os.Getenv("RESEND_FROM_EMAIL")),
		ResendTemplateLoginCode:     envOrDefault("RESEND_TEMPLATE_LOGIN_CODE", "login-code"),
		ResendTemplatePasswordReset: envOrDefault("RESEND_TEMPLATE_PASSWORD_RESET", "reset-password"),
		EmailVerificationRedirect:   strings.TrimSpace(os.Getenv("EMAIL_VERIFICATION_REDIRECT_BASE")),
		PasswordResetRedirect:       strings.TrimSpace(os.Getenv("PASSWORD_RESET_REDIRECT_BASE")),
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

	if cfg.PasswordResetTokenTTL <= 0 {
		return Config{}, fmt.Errorf("PASSWORD_RESET_TOKEN_TTL must be greater than 0")
	}

	if cfg.S3UploadURLTTL <= 0 {
		return Config{}, fmt.Errorf("S3_UPLOAD_URL_TTL must be greater than 0")
	}

	if cfg.S3DownloadURLTTL <= 0 {
		return Config{}, fmt.Errorf("S3_DOWNLOAD_URL_TTL must be greater than 0")
	}

	if cfg.S3MaxImageBytes <= 0 {
		return Config{}, fmt.Errorf("S3_MAX_IMAGE_BYTES must be greater than 0")
	}

	if cfg.ImageWebPQuality < 1 || cfg.ImageWebPQuality > 100 {
		return Config{}, fmt.Errorf("IMAGE_WEBP_QUALITY must be between 1 and 100")
	}

	if cfg.ImageWorkerPoolSize <= 0 {
		return Config{}, fmt.Errorf("IMAGE_WORKER_POOL_SIZE must be greater than 0")
	}

	if len(cfg.ImageVariantWidthsPlays) == 0 {
		return Config{}, fmt.Errorf("IMAGE_VARIANT_WIDTHS_PLAYS must contain at least one width")
	}

	if len(cfg.ImageVariantWidthsAvatars) == 0 {
		return Config{}, fmt.Errorf("IMAGE_VARIANT_WIDTHS_AVATARS must contain at least one width")
	}

	if err := validateS3BucketConfig(cfg.PlaysS3, "PLAYS"); err != nil {
		return Config{}, err
	}

	if err := validateS3BucketConfig(cfg.UsersS3, "USERS"); err != nil {
		return Config{}, err
	}

	if cfg.AppEnv == "production" && cfg.EmailVerificationRequired {
		if cfg.ResendAPIKey == "" {
			return Config{}, fmt.Errorf("RESEND_API_KEY is required when EMAIL_VERIFICATION_REQUIRED is true in production")
		}

		if cfg.ResendFromEmail == "" {
			return Config{}, fmt.Errorf("RESEND_FROM_EMAIL is required when EMAIL_VERIFICATION_REQUIRED is true in production")
		}

		if cfg.PasswordResetRedirect == "" {
			return Config{}, fmt.Errorf("PASSWORD_RESET_REDIRECT_BASE is required in production")
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

func intListFromEnv(key string, fallback []int) ([]int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		out := make([]int, len(fallback))
		copy(out, fallback)
		return out, nil
	}

	parts := strings.Split(value, ",")
	result := make([]int, 0, len(parts))
	for _, raw := range parts {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, fmt.Errorf("%s must be a comma-separated list of integers: %w", key, err)
		}
		if parsed <= 0 {
			return nil, fmt.Errorf("%s values must be positive", key)
		}
		result = append(result, parsed)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("%s must contain at least one value", key)
	}
	return result, nil
}

func int64FromEnvStrict(key string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer: %w", key, err)
	}

	return parsed, nil
}

func validateS3BucketConfig(cfg S3BucketConfig, prefix string) error {
	if cfg.Endpoint == "" {
		return fmt.Errorf("%s_S3_ENDPOINT is required", prefix)
	}

	if cfg.Region == "" {
		return fmt.Errorf("%s_S3_REGION is required", prefix)
	}

	if cfg.AccessKeyID == "" {
		return fmt.Errorf("%s_S3_ACCESS_KEY_ID is required", prefix)
	}

	if cfg.SecretAccessKey == "" {
		return fmt.Errorf("%s_S3_SECRET_ACCESS_KEY is required", prefix)
	}

	if cfg.Bucket == "" {
		return fmt.Errorf("%s_S3_BUCKET is required", prefix)
	}

	return nil
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
