package app

import (
	"fmt"

	"github.com/butaqueando/api/internal/config"
	"github.com/butaqueando/api/internal/database"
	apihttp "github.com/butaqueando/api/internal/http"
	authmodule "github.com/butaqueando/api/internal/modules/auth"
	sharedemail "github.com/butaqueando/api/internal/shared/email"
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
	})

	return &Application{
		Config: cfg,
		Router: router,
		DB:     db,
		SQLDB:  sqlDB,
	}, nil
}
