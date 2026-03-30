package auth

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type fakeRepository struct {
	findUserByEmailFn      func(ctx context.Context, email string) (UserRecord, error)
	findUserByIDFn         func(ctx context.Context, userID string) (UserRecord, error)
	insertRefreshTokenFn   func(ctx context.Context, tokenID string, userID string, expiresAt time.Time, createdAt time.Time) error
	findActiveRefreshFn    func(ctx context.Context, tokenID string, userID string, now time.Time) (RefreshTokenRecord, error)
	rotateRefreshTokenFn   func(ctx context.Context, oldTokenID string, userID string, newTokenID string, newExpiresAt time.Time, now time.Time) error
	revokeRefreshTokenFn   func(ctx context.Context, tokenID string, userID string, now time.Time) error
	createPendingUserFn    func(ctx context.Context, input CreatePendingUserInput) (UserRecord, error)
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

func (f *fakeRepository) CreatePendingUser(ctx context.Context, input CreatePendingUserInput) (UserRecord, error) {
	if f.createPendingUserFn != nil {
		return f.createPendingUserFn(ctx, input)
	}

	return UserRecord{}, errors.New("not implemented")
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

	service := NewService(repo, tokens, ServiceOptions{})
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

	service := NewService(&fakeRepository{}, &fakeTokenManager{}, ServiceOptions{})
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

func TestServiceSignInRequiresVerifiedEmailWhenEnabled(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	repo := &fakeRepository{
		findUserByEmailFn: func(ctx context.Context, email string) (UserRecord, error) {
			return UserRecord{
				ID:           "user-1",
				DisplayName:  "User",
				Email:        email,
				PasswordHash: string(hash),
				Role:         "user",
			}, nil
		},
	}

	emailVerificationRequired := true
	service := NewService(repo, &fakeTokenManager{}, ServiceOptions{EmailVerificationRequired: &emailVerificationRequired})
	_, err = service.SignIn(context.Background(), SignInRequest{Email: "user@butaqueando.local", Password: "password123"})
	if err == nil {
		t.Fatalf("expected sign in error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Status != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, appErr.Status)
	}

	if appErr.Code != "EMAIL_NOT_VERIFIED" {
		t.Fatalf("expected code %q, got %q", "EMAIL_NOT_VERIFIED", appErr.Code)
	}
}

func TestServiceSignInAllowsUnverifiedWhenBypassed(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	repo := &fakeRepository{
		findUserByEmailFn: func(ctx context.Context, email string) (UserRecord, error) {
			return UserRecord{
				ID:           "user-1",
				DisplayName:  "User",
				Email:        email,
				PasswordHash: string(hash),
				Role:         "user",
			}, nil
		},
		insertRefreshTokenFn: func(ctx context.Context, tokenID string, userID string, expiresAt time.Time, createdAt time.Time) error {
			return nil
		},
	}

	tokens := &fakeTokenManager{
		generateAccessTokenFn: func(userID string, role string) (string, time.Time, error) {
			return "access-token", time.Now().Add(15 * time.Minute), nil
		},
		generateRefreshTokenFn: func(userID string, role string) (string, string, time.Time, error) {
			return "refresh-token", "token-id", time.Now().Add(24 * time.Hour), nil
		},
	}

	emailVerificationRequired := false
	service := NewService(repo, tokens, ServiceOptions{EmailVerificationRequired: &emailVerificationRequired})
	result, err := service.SignIn(context.Background(), SignInRequest{Email: "user@butaqueando.local", Password: "password123"})
	if err != nil {
		t.Fatalf("expected sign in success, got error: %v", err)
	}

	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatalf("expected tokens in response")
	}
}

func TestServiceSignUpDuplicateEmailReturnsConflict(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		findUserByEmailFn: func(ctx context.Context, email string) (UserRecord, error) {
			return UserRecord{ID: "user-1", Email: email}, nil
		},
	}

	service := NewService(repo, &fakeTokenManager{}, ServiceOptions{})
	_, err := service.SignUp(context.Background(), SignUpRequest{DisplayName: "User", Email: "user@butaqueando.local", Password: "password123"})
	if err == nil {
		t.Fatalf("expected sign up error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Status != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, appErr.Status)
	}

	if appErr.Code != "EMAIL_ALREADY_IN_USE" {
		t.Fatalf("expected code %q, got %q", "EMAIL_ALREADY_IN_USE", appErr.Code)
	}
}

func TestServiceSignUpExposesVerificationTokenInDev(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		findUserByEmailFn: func(ctx context.Context, email string) (UserRecord, error) {
			return UserRecord{}, gorm.ErrRecordNotFound
		},
		createPendingUserFn: func(ctx context.Context, input CreatePendingUserInput) (UserRecord, error) {
			return UserRecord{ID: "user-1", Email: input.Email, DisplayName: input.DisplayName, Role: input.Role}, nil
		},
	}

	emailVerificationRequired := true
	service := NewService(repo, &fakeTokenManager{}, ServiceOptions{ExposeVerificationToken: true, EmailVerificationRequired: &emailVerificationRequired})
	result, err := service.SignUp(context.Background(), SignUpRequest{DisplayName: "User", Email: "user@butaqueando.local", Password: "password123"})
	if err != nil {
		t.Fatalf("expected sign up success, got error: %v", err)
	}

	if result.VerificationToken == nil || *result.VerificationToken == "" {
		t.Fatalf("expected verification token to be exposed")
	}

	if !result.EmailVerificationRequired {
		t.Fatalf("expected email verification required true")
	}
}
