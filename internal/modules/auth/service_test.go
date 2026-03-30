package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
)

type fakeRepository struct {
	findUserByEmailFn      func(ctx context.Context, email string) (UserRecord, error)
	findUserByIDFn         func(ctx context.Context, userID string) (UserRecord, error)
	insertRefreshTokenFn   func(ctx context.Context, tokenID string, userID string, expiresAt time.Time, createdAt time.Time) error
	findActiveRefreshFn    func(ctx context.Context, tokenID string, userID string, now time.Time) (RefreshTokenRecord, error)
	rotateRefreshTokenFn   func(ctx context.Context, oldTokenID string, userID string, newTokenID string, newExpiresAt time.Time, now time.Time) error
	revokeRefreshTokenFn   func(ctx context.Context, tokenID string, userID string, now time.Time) error
	revokedTokenIDCaptured string
	revokedUserIDCaptured  string
}

func (f *fakeRepository) FindUserByEmail(ctx context.Context, email string) (UserRecord, error) {
	return f.findUserByEmailFn(ctx, email)
}

func (f *fakeRepository) FindUserByID(ctx context.Context, userID string) (UserRecord, error) {
	return f.findUserByIDFn(ctx, userID)
}

func (f *fakeRepository) InsertRefreshToken(ctx context.Context, tokenID string, userID string, expiresAt time.Time, createdAt time.Time) error {
	return f.insertRefreshTokenFn(ctx, tokenID, userID, expiresAt, createdAt)
}

func (f *fakeRepository) FindActiveRefreshToken(ctx context.Context, tokenID string, userID string, now time.Time) (RefreshTokenRecord, error) {
	return f.findActiveRefreshFn(ctx, tokenID, userID, now)
}

func (f *fakeRepository) RotateRefreshToken(ctx context.Context, oldTokenID string, userID string, newTokenID string, newExpiresAt time.Time, now time.Time) error {
	return f.rotateRefreshTokenFn(ctx, oldTokenID, userID, newTokenID, newExpiresAt, now)
}

func (f *fakeRepository) RevokeRefreshToken(ctx context.Context, tokenID string, userID string, now time.Time) error {
	f.revokedTokenIDCaptured = tokenID
	f.revokedUserIDCaptured = userID
	if f.revokeRefreshTokenFn == nil {
		return nil
	}

	return f.revokeRefreshTokenFn(ctx, tokenID, userID, now)
}

type fakeTokenManager struct {
	parseRefreshTokenFn    func(rawToken string) (RefreshClaims, error)
	generateAccessTokenFn  func(userID string, role string) (string, time.Time, error)
	generateRefreshTokenFn func(userID string, role string) (string, string, time.Time, error)
}

func (f *fakeTokenManager) GenerateAccessToken(userID string, role string) (string, time.Time, error) {
	if f.generateAccessTokenFn != nil {
		return f.generateAccessTokenFn(userID, role)
	}

	return "", time.Time{}, errors.New("not implemented")
}

func (f *fakeTokenManager) GenerateRefreshToken(userID string, role string) (string, string, time.Time, error) {
	if f.generateRefreshTokenFn != nil {
		return f.generateRefreshTokenFn(userID, role)
	}

	return "", "", time.Time{}, errors.New("not implemented")
}

func (f *fakeTokenManager) ParseRefreshToken(rawToken string) (RefreshClaims, error) {
	if f.parseRefreshTokenFn != nil {
		return f.parseRefreshTokenFn(rawToken)
	}

	return RefreshClaims{}, ErrInvalidToken
}

func TestServiceSignOutRevokesProvidedToken(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	tokens := &fakeTokenManager{
		parseRefreshTokenFn: func(rawToken string) (RefreshClaims, error) {
			return RefreshClaims{UserID: "user-1", TokenID: "token-1", Role: "user"}, nil
		},
	}

	service := NewService(repo, tokens)
	result, err := service.SignOut(context.Background(), SignOutRequest{RefreshToken: "refresh-token"})
	if err != nil {
		t.Fatalf("sign out returned error: %v", err)
	}

	if !result.OK {
		t.Fatalf("expected sign out ok true")
	}

	if repo.revokedTokenIDCaptured != "token-1" {
		t.Fatalf("expected revoked token id %q, got %q", "token-1", repo.revokedTokenIDCaptured)
	}

	if repo.revokedUserIDCaptured != "user-1" {
		t.Fatalf("expected revoked user id %q, got %q", "user-1", repo.revokedUserIDCaptured)
	}
}

func TestServiceRefreshInvalidTokenReturnsUnauthorized(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{}, &fakeTokenManager{})
	_, err := service.Refresh(context.Background(), RefreshRequest{RefreshToken: "invalid"})
	if err == nil {
		t.Fatalf("expected refresh to return error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "UNAUTHORIZED" {
		t.Fatalf("expected error code %q, got %q", "UNAUTHORIZED", appErr.Code)
	}
}
