package app

import (
	"context"
	"fmt"

	"github.com/butaqueando/api/internal/config"
	"github.com/butaqueando/api/internal/database"
	apihttp "github.com/butaqueando/api/internal/http"
	authmodule "github.com/butaqueando/api/internal/modules/auth"
	sharedemail "github.com/butaqueando/api/internal/shared/email"
	"github.com/butaqueando/api/internal/shared/storage"
)

func Bootstrap() (*Application, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	db, sqlDB, err := database.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	verificationEmailSender := sharedemail.Sender(sharedemail.NoopSender{})
	if cfg.ResendAPIKey != "" && cfg.ResendFromEmail != "" {
		resendSender, senderErr := sharedemail.NewResendSender(
			cfg.ResendAPIKey,
			cfg.ResendFromEmail,
			cfg.ResendTemplateLoginCode,
			cfg.ResendTemplatePasswordReset,
		)
		if senderErr != nil {
			return nil, fmt.Errorf("build resend sender: %w", senderErr)
		}

		verificationEmailSender = resendSender
	}

	playsStorage, err := storage.NewS3Client(context.Background(), storage.BucketConfig{
		Endpoint:        cfg.PlaysS3.Endpoint,
		Region:          cfg.PlaysS3.Region,
		AccessKeyID:     cfg.PlaysS3.AccessKeyID,
		SecretAccessKey: cfg.PlaysS3.SecretAccessKey,
		Bucket:          cfg.PlaysS3.Bucket,
	})
	if err != nil {
		return nil, fmt.Errorf("build plays storage client: %w", err)
	}

	usersStorage, err := storage.NewS3Client(context.Background(), storage.BucketConfig{
		Endpoint:        cfg.UsersS3.Endpoint,
		Region:          cfg.UsersS3.Region,
		AccessKeyID:     cfg.UsersS3.AccessKeyID,
		SecretAccessKey: cfg.UsersS3.SecretAccessKey,
		Bucket:          cfg.UsersS3.Bucket,
	})
	if err != nil {
		return nil, fmt.Errorf("build users storage client: %w", err)
	}

	router := apihttp.NewRouter(apihttp.Dependencies{
		DB: db,
		TokenConfig: authmodule.TokenConfig{
			Issuer:        cfg.JWTIssuer,
			AccessSecret:  cfg.JWTAccessSecret,
			RefreshSecret: cfg.JWTRefreshSecret,
			AccessTTL:     cfg.JWTAccessTTL,
			RefreshTTL:    cfg.JWTRefreshTTL,
		},
		EmailVerificationRequired: cfg.EmailVerificationRequired,
		ExposeVerificationToken:   cfg.AppEnv != "production",
		VerificationEmailSender:   verificationEmailSender,
		EmailVerificationRedirect: cfg.EmailVerificationRedirect,
		PasswordResetRedirect:     cfg.PasswordResetRedirect,
		PasswordResetTokenTTL:     cfg.PasswordResetTokenTTL,
		PlaysStorage:              playsStorage,
		UsersStorage:              usersStorage,
		S3UploadURLTTL:            cfg.S3UploadURLTTL,
		S3DownloadURLTTL:          cfg.S3DownloadURLTTL,
		S3MaxImageBytes:           cfg.S3MaxImageBytes,
	})

	return &Application{
		Config: cfg,
		Router: router,
		DB:     db,
		SQLDB:  sqlDB,
	}, nil
}
