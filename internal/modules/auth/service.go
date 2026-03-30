package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const defaultVerificationTokenTTL = 24 * time.Hour

type repositoryPort interface {
	FindUserByEmail(ctx context.Context, email string) (UserRecord, error)
	FindUserByID(ctx context.Context, userID string) (UserRecord, error)
	InsertRefreshToken(ctx context.Context, tokenID string, userID string, expiresAt time.Time, createdAt time.Time) error
	FindActiveRefreshToken(ctx context.Context, tokenID string, userID string, now time.Time) (RefreshTokenRecord, error)
	RotateRefreshToken(ctx context.Context, oldTokenID string, userID string, newTokenID string, newExpiresAt time.Time, now time.Time) error
	RevokeRefreshToken(ctx context.Context, tokenID string, userID string, now time.Time) error
	CreatePendingUser(ctx context.Context, input CreatePendingUserInput) (UserRecord, error)
}

type tokenManagerPort interface {
	GenerateAccessToken(userID string, role string) (string, time.Time, error)
	GenerateRefreshToken(userID string, role string) (string, string, time.Time, error)
	ParseRefreshToken(rawToken string) (RefreshClaims, error)
}

type ServiceOptions struct {
	EmailVerificationRequired *bool
	ExposeVerificationToken   bool
	VerificationTokenTTL      time.Duration
}

type Service struct {
	repo                      repositoryPort
	tokens                    tokenManagerPort
	now                       func() time.Time
	emailVerificationRequired bool
	exposeVerificationToken   bool
	verificationTokenTTL      time.Duration
}

func NewService(repo repositoryPort, tokens tokenManagerPort, options ServiceOptions) *Service {
	emailVerificationRequired := true
	if options.EmailVerificationRequired != nil {
		emailVerificationRequired = *options.EmailVerificationRequired
	}

	verificationTokenTTL := options.VerificationTokenTTL
	if verificationTokenTTL <= 0 {
		verificationTokenTTL = defaultVerificationTokenTTL
	}

	return &Service{
		repo:                      repo,
		tokens:                    tokens,
		now:                       func() time.Time { return time.Now().UTC() },
		emailVerificationRequired: emailVerificationRequired,
		exposeVerificationToken:   options.ExposeVerificationToken,
		verificationTokenTTL:      verificationTokenTTL,
	}
}

func (s *Service) SignUp(ctx context.Context, req SignUpRequest) (SignUpData, error) {
	displayName := strings.TrimSpace(req.DisplayName)
	email := normalizeEmail(req.Email)
	password := strings.TrimSpace(req.Password)

	if displayName == "" || email == "" || password == "" {
		return SignUpData{}, sharederrors.Validation("displayName, email and password are required", nil)
	}

	if _, err := s.repo.FindUserByEmail(ctx, email); err == nil {
		return SignUpData{}, sharederrors.New(http.StatusConflict, "EMAIL_ALREADY_IN_USE", "email already in use", nil)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return SignUpData{}, sharederrors.Internal("failed to sign up", nil)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return SignUpData{}, sharederrors.Internal("failed to sign up", nil)
	}

	verificationToken, err := generateVerificationToken()
	if err != nil {
		return SignUpData{}, sharederrors.Internal("failed to sign up", nil)
	}

	now := s.now()
	createdUser, err := s.repo.CreatePendingUser(ctx, CreatePendingUserInput{
		DisplayName:                displayName,
		Email:                      email,
		PasswordHash:               string(passwordHash),
		Role:                       "user",
		EmailVerificationTokenHash: hashVerificationToken(verificationToken),
		EmailVerificationExpiresAt: now.Add(s.verificationTokenTTL),
		CreatedAt:                  now,
	})
	if err != nil {
		if isEmailAlreadyInUseError(err) {
			return SignUpData{}, sharederrors.New(http.StatusConflict, "EMAIL_ALREADY_IN_USE", "email already in use", nil)
		}

		return SignUpData{}, sharederrors.Internal("failed to sign up", nil)
	}

	response := SignUpData{
		UserID:                    createdUser.ID,
		Email:                     createdUser.Email,
		EmailVerificationRequired: s.emailVerificationRequired,
	}

	if s.exposeVerificationToken {
		response.VerificationToken = &verificationToken
	}

	return response, nil
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

	if s.emailVerificationRequired && user.EmailVerifiedAt == nil {
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

func generateVerificationToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashVerificationToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func isEmailAlreadyInUseError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
