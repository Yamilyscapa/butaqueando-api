package config

import "testing"

func TestLoadRejectsDisabledEmailVerificationInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_ISSUER", "butaqueando-api")
	t.Setenv("JWT_ACCESS_SECRET", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("JWT_REFRESH_SECRET", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	t.Setenv("JWT_ACCESS_TTL", "15m")
	t.Setenv("JWT_REFRESH_TTL", "720h")
	t.Setenv("EMAIL_VERIFICATION_REQUIRED", "false")
	t.Setenv("EMAIL_VERIFICATION_REDIRECT_BASE", "https://app.butaqueando.com/verify-email")
	t.Setenv("PASSWORD_RESET_REDIRECT_BASE", "https://app.butaqueando.com/reset-password")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected production guard error")
	}
}

func TestLoadAllowsEnabledEmailVerificationInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_ISSUER", "butaqueando-api")
	t.Setenv("JWT_ACCESS_SECRET", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("JWT_REFRESH_SECRET", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	t.Setenv("JWT_ACCESS_TTL", "15m")
	t.Setenv("JWT_REFRESH_TTL", "720h")
	t.Setenv("EMAIL_VERIFICATION_REQUIRED", "true")
	t.Setenv("EMAIL_VERIFICATION_REDIRECT_BASE", "https://app.butaqueando.com/verify-email")
	t.Setenv("PASSWORD_RESET_REDIRECT_BASE", "https://app.butaqueando.com/reset-password")
	t.Setenv("RESEND_API_KEY", "re_test_123")
	t.Setenv("RESEND_FROM_EMAIL", "Butaqueando <noreply@butaqueando.com>")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config load success, got error: %v", err)
	}

	if !cfg.EmailVerificationRequired {
		t.Fatalf("expected email verification required true")
	}
}

func TestLoadRejectsMissingPasswordResetRedirectInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_ISSUER", "butaqueando-api")
	t.Setenv("JWT_ACCESS_SECRET", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("JWT_REFRESH_SECRET", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	t.Setenv("JWT_ACCESS_TTL", "15m")
	t.Setenv("JWT_REFRESH_TTL", "720h")
	t.Setenv("EMAIL_VERIFICATION_REQUIRED", "true")
	t.Setenv("EMAIL_VERIFICATION_REDIRECT_BASE", "https://app.butaqueando.com/verify-email")
	t.Setenv("PASSWORD_RESET_REDIRECT_BASE", "")
	t.Setenv("RESEND_API_KEY", "re_test_123")
	t.Setenv("RESEND_FROM_EMAIL", "Butaqueando <noreply@butaqueando.com>")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected password reset redirect validation error")
	}
}

func TestLoadRejectsMissingVerificationRedirectWhenVerificationRequired(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_ISSUER", "butaqueando-api")
	t.Setenv("JWT_ACCESS_SECRET", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("JWT_REFRESH_SECRET", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	t.Setenv("JWT_ACCESS_TTL", "15m")
	t.Setenv("JWT_REFRESH_TTL", "720h")
	t.Setenv("EMAIL_VERIFICATION_REQUIRED", "true")
	t.Setenv("EMAIL_VERIFICATION_REDIRECT_BASE", "")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected redirect base validation error")
	}
}
