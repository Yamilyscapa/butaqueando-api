package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type repositoryPort interface {
	FindUserByEmail(ctx context.Context, email string) (UserRecord, error)
	FindUserByID(ctx context.Context, userID string) (UserRecord, error)
	InsertRefreshToken(ctx context.Context, tokenID string, userID string, expiresAt time.Time, createdAt time.Time) error
	FindActiveRefreshToken(ctx context.Context, tokenID string, userID string, now time.Time) (RefreshTokenRecord, error)
	RotateRefreshToken(ctx context.Context, oldTokenID string, userID string, newTokenID string, newExpiresAt time.Time, now time.Time) error
	RevokeRefreshToken(ctx context.Context, tokenID string, userID string, now time.Time) error
}

type tokenManagerPort interface {
	GenerateAccessToken(userID string, role string) (string, time.Time, error)
	GenerateRefreshToken(userID string, role string) (string, string, time.Time, error)
	ParseRefreshToken(rawToken string) (RefreshClaims, error)
}

type Service struct {
	repo   repositoryPort
	tokens tokenManagerPort
	now    func() time.Time
}

func NewService(repo repositoryPort, tokens tokenManagerPort) *Service {
	return &Service{
		repo:   repo,
		tokens: tokens,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) SignIn(ctx context.Context, req SignInRequest) (AuthTokensData, error) {
	email := normalizeEmail(req.Email)
	if email == "" || strings.TrimSpace(req.Password) == "" {
		return AuthTokensData{}, sharederrors.Validation("email and password are required", nil)
	}

	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthTokensData{}, sharederrors.Unauthorized("invalid email or password", nil)
		}

		return AuthTokensData{}, sharederrors.Internal("failed to sign in", nil)
	}

	if user.EmailVerifiedAt == nil {
		return AuthTokensData{}, sharederrors.New(http.StatusForbidden, "EMAIL_NOT_VERIFIED", "email must be verified before signing in", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return AuthTokensData{}, sharederrors.Unauthorized("invalid email or password", nil)
	}

	accessToken, accessExpiresAt, err := s.tokens.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return AuthTokensData{}, sharederrors.Internal("failed to sign in", nil)
	}

	refreshToken, refreshTokenID, refreshExpiresAt, err := s.tokens.GenerateRefreshToken(user.ID, user.Role)
	if err != nil {
		return AuthTokensData{}, sharederrors.Internal("failed to sign in", nil)
	}

	now := s.now()
	if err := s.repo.InsertRefreshToken(ctx, refreshTokenID, user.ID, refreshExpiresAt, now); err != nil {
		return AuthTokensData{}, sharederrors.Internal("failed to sign in", nil)
	}

	return buildTokensData(user, accessToken, accessExpiresAt, refreshToken, refreshExpiresAt), nil
}

func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (AuthTokensData, error) {
	rawToken := strings.TrimSpace(req.RefreshToken)
	if rawToken == "" {
		return AuthTokensData{}, sharederrors.Validation("refreshToken is required", nil)
	}

	claims, err := s.tokens.ParseRefreshToken(rawToken)
	if err != nil {
		return AuthTokensData{}, sharederrors.Unauthorized("invalid refresh token", nil)
	}

	now := s.now()
	if _, err := s.repo.FindActiveRefreshToken(ctx, claims.TokenID, claims.UserID, now); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthTokensData{}, sharederrors.Unauthorized("invalid refresh token", nil)
		}

		return AuthTokensData{}, sharederrors.Internal("failed to refresh token", nil)
	}

	user, err := s.repo.FindUserByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthTokensData{}, sharederrors.Unauthorized("invalid refresh token", nil)
		}

		return AuthTokensData{}, sharederrors.Internal("failed to refresh token", nil)
	}

	accessToken, accessExpiresAt, err := s.tokens.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return AuthTokensData{}, sharederrors.Internal("failed to refresh token", nil)
	}

	refreshToken, refreshTokenID, refreshExpiresAt, err := s.tokens.GenerateRefreshToken(user.ID, user.Role)
	if err != nil {
		return AuthTokensData{}, sharederrors.Internal("failed to refresh token", nil)
	}

	if err := s.repo.RotateRefreshToken(ctx, claims.TokenID, user.ID, refreshTokenID, refreshExpiresAt, now); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthTokensData{}, sharederrors.Unauthorized("invalid refresh token", nil)
		}

		return AuthTokensData{}, sharederrors.Internal("failed to refresh token", nil)
	}

	return buildTokensData(user, accessToken, accessExpiresAt, refreshToken, refreshExpiresAt), nil
}

func (s *Service) SignOut(ctx context.Context, req SignOutRequest) (SignOutData, error) {
	rawToken := strings.TrimSpace(req.RefreshToken)
	if rawToken == "" {
		return SignOutData{}, sharederrors.Validation("refreshToken is required", nil)
	}

	claims, err := s.tokens.ParseRefreshToken(rawToken)
	if err != nil {
		return SignOutData{}, sharederrors.Unauthorized("invalid refresh token", nil)
	}

	if err := s.repo.RevokeRefreshToken(ctx, claims.TokenID, claims.UserID, s.now()); err != nil {
		return SignOutData{}, sharederrors.Internal("failed to sign out", nil)
	}

	return SignOutData{OK: true}, nil
}

func buildTokensData(user UserRecord, accessToken string, accessExpiresAt time.Time, refreshToken string, refreshExpiresAt time.Time) AuthTokensData {
	now := time.Now().UTC()
	return AuthTokensData{
		TokenType:             "Bearer",
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresIn:  durationSeconds(now, accessExpiresAt),
		RefreshTokenExpiresIn: durationSeconds(now, refreshExpiresAt),
		User: &AuthUserData{
			ID:          user.ID,
			DisplayName: user.DisplayName,
			Email:       user.Email,
			Role:        user.Role,
		},
	}
}

func durationSeconds(now time.Time, expiresAt time.Time) int64 {
	seconds := int64(expiresAt.Sub(now).Seconds())
	if seconds < 0 {
		return 0
	}

	return seconds
}

func normalizeEmail(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
