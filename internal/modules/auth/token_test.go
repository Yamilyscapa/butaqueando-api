package auth

import (
	"testing"
	"time"
)

func TestTokenManagerGenerateAndParseRefreshToken(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager(TokenConfig{
		Issuer:        "butaqueando-api",
		AccessSecret:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RefreshSecret: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    30 * 24 * time.Hour,
	})

	refreshToken, tokenID, _, err := manager.GenerateRefreshToken("user-1", "user")
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	claims, err := manager.ParseRefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}

	if claims.UserID != "user-1" {
		t.Fatalf("expected user id %q, got %q", "user-1", claims.UserID)
	}

	if claims.Role != "user" {
		t.Fatalf("expected role %q, got %q", "user", claims.Role)
	}

	if claims.TokenID != tokenID {
		t.Fatalf("expected token id %q, got %q", tokenID, claims.TokenID)
	}
}

func TestTokenManagerRejectsAccessTokenAsRefreshToken(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager(TokenConfig{
		Issuer:        "butaqueando-api",
		AccessSecret:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RefreshSecret: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    30 * 24 * time.Hour,
	})

	accessToken, _, err := manager.GenerateAccessToken("user-1", "user")
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	if _, err := manager.ParseRefreshToken(accessToken); err == nil {
		t.Fatalf("expected parse refresh token to fail for access token")
	}
}
