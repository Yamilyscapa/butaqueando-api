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

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config load success, got error: %v", err)
	}

	if !cfg.EmailVerificationRequired {
		t.Fatalf("expected email verification required true")
	}
}
