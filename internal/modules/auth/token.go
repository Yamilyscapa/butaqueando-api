package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

var ErrInvalidToken = errors.New("invalid token")

type TokenConfig struct {
	Issuer        string
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type TokenManager struct {
	issuer        string
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

type tokenClaims struct {
	Role string `json:"role"`
	Type string `json:"type"`
	jwt.RegisteredClaims
}

func NewTokenManager(cfg TokenConfig) *TokenManager {
	return &TokenManager{
		issuer:        cfg.Issuer,
		accessSecret:  []byte(cfg.AccessSecret),
		refreshSecret: []byte(cfg.RefreshSecret),
		accessTTL:     cfg.AccessTTL,
		refreshTTL:    cfg.RefreshTTL,
	}
}

func (m *TokenManager) GenerateAccessToken(userID string, role string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.accessTTL)
	claims := tokenClaims{
		Role: role,
		Type: tokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.accessSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return token, expiresAt, nil
}

func (m *TokenManager) GenerateRefreshToken(userID string, role string) (string, string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.refreshTTL)
	tokenID := uuid.NewString()
	claims := tokenClaims{
		Role: role,
		Type: tokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   userID,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.refreshSecret)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("sign refresh token: %w", err)
	}

	return token, tokenID, expiresAt, nil
}

func (m *TokenManager) ParseRefreshToken(rawToken string) (RefreshClaims, error) {
	claims, err := m.parse(rawToken, m.refreshSecret)
	if err != nil {
		return RefreshClaims{}, err
	}

	if claims.Type != tokenTypeRefresh {
		return RefreshClaims{}, ErrInvalidToken
	}

	if claims.ID == "" || claims.Subject == "" {
		return RefreshClaims{}, ErrInvalidToken
	}

	return RefreshClaims{
		UserID:  claims.Subject,
		Role:    claims.Role,
		TokenID: claims.ID,
	}, nil
}

func (m *TokenManager) ParseAccessToken(rawToken string) (AccessClaims, error) {
	claims, err := m.parse(rawToken, m.accessSecret)
	if err != nil {
		return AccessClaims{}, err
	}

	if claims.Type != tokenTypeAccess {
		return AccessClaims{}, ErrInvalidToken
	}

	if claims.Subject == "" {
		return AccessClaims{}, ErrInvalidToken
	}

	return AccessClaims{
		UserID: claims.Subject,
		Role:   claims.Role,
	}, nil
}

func (m *TokenManager) parse(rawToken string, secret []byte) (*tokenClaims, error) {
	claims := &tokenClaims{}
	parsedToken, err := jwt.ParseWithClaims(
		rawToken,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, ErrInvalidToken
			}

			return secret, nil
		},
		jwt.WithIssuer(m.issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if !parsedToken.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
