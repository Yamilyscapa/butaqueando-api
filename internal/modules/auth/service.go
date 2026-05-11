package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	sharedemail "github.com/butaqueando/api/internal/shared/email"
	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/modules/users"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	defaultVerificationTokenTTL  = 10 * time.Minute
	defaultPasswordResetTokenTTL = 10 * time.Minute
)

var otpPattern = regexp.MustCompile(`^\d{6}$`)

type repositoryPort interface {
	FindUserByEmail(ctx context.Context, email string) (UserRecord, error)
	FindUserByID(ctx context.Context, userID string) (UserRecord, error)
	InsertRefreshToken(ctx context.Context, tokenID string, userID string, expiresAt time.Time, createdAt time.Time) error
	FindActiveRefreshToken(ctx context.Context, tokenID string, userID string, now time.Time) (RefreshTokenRecord, error)
	RotateRefreshToken(ctx context.Context, oldTokenID string, userID string, newTokenID string, newExpiresAt time.Time, now time.Time) error
	RevokeRefreshToken(ctx context.Context, tokenID string, userID string, now time.Time) error
	CreatePendingUser(ctx context.Context, input CreatePendingUserInput) (UserRecord, error)
	ConsumeEmailVerificationOTP(ctx context.Context, userID string, tokenHash string, now time.Time) error
	CreateEmailVerificationToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time, createdAt time.Time) error
	InvalidateEmailVerificationTokensForUser(ctx context.Context, userID string, now time.Time) error
	CreatePasswordResetToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time, createdAt time.Time) error
	InvalidatePasswordResetTokensForUser(ctx context.Context, userID string, now time.Time) error
	ConsumePasswordResetOTP(ctx context.Context, userID string, tokenHash string, passwordHash string, now time.Time) error
}

type emailSender interface {
	SendVerificationEmail(ctx context.Context, input sharedemail.VerificationEmailInput) error
	SendPasswordResetEmail(ctx context.Context, input sharedemail.PasswordResetEmailInput) error
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
	PasswordResetTokenTTL     time.Duration
	VerificationEmailSender   emailSender
}

type Service struct {
	repo                      repositoryPort
	tokens                    tokenManagerPort
	emailSender               emailSender
	now                       func() time.Time
	emailVerificationRequired bool
	exposeVerificationToken   bool
	verificationTokenTTL      time.Duration
	passwordResetTokenTTL     time.Duration
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

	passwordResetTokenTTL := options.PasswordResetTokenTTL
	if passwordResetTokenTTL <= 0 {
		passwordResetTokenTTL = defaultPasswordResetTokenTTL
	}

	return &Service{
		repo:                      repo,
		tokens:                    tokens,
		emailSender:               options.VerificationEmailSender,
		now:                       func() time.Time { return time.Now().UTC() },
		emailVerificationRequired: emailVerificationRequired,
		exposeVerificationToken:   options.ExposeVerificationToken,
		verificationTokenTTL:      verificationTokenTTL,
		passwordResetTokenTTL:     passwordResetTokenTTL,
	}
}

func (s *Service) SignUp(ctx context.Context, req SignUpRequest) (SignUpData, error) {
	displayName := strings.TrimSpace(req.DisplayName)
	email := normalizeEmail(req.Email)
	password := strings.TrimSpace(req.Password)

	if displayName == "" || email == "" || password == "" {
		return SignUpData{}, sharederrors.Validation("displayName, email and password are required", nil)
	}

	username, err := users.ValidateUsername(req.Username)
	if err != nil {
		return SignUpData{}, err
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
	verificationTokenHash := hashVerificationToken(verificationToken)

	now := s.now()
	createdUser, err := s.repo.CreatePendingUser(ctx, CreatePendingUserInput{
		DisplayName:                displayName,
		Username:                   username,
		Email:                      email,
		PasswordHash:               string(passwordHash),
		Role:                       "user",
		EmailVerificationTokenHash: verificationTokenHash,
		EmailVerificationExpiresAt: now.Add(s.verificationTokenTTL),
		CreatedAt:                  now,
	})
	if err != nil {
		if isUsernameAlreadyInUseError(err) {
			return SignUpData{}, sharederrors.New(http.StatusConflict, "USERNAME_TAKEN", "username already taken", nil)
		}
		if isEmailAlreadyInUseError(err) {
			return SignUpData{}, sharederrors.New(http.StatusConflict, "EMAIL_ALREADY_IN_USE", "email already in use", nil)
		}

		return SignUpData{}, sharederrors.Internal("failed to sign up", nil)
	}

	response := SignUpData{
		UserID:                    createdUser.ID,
		Username:                  createdUser.Username,
		Email:                     createdUser.Email,
		EmailVerificationRequired: s.emailVerificationRequired,
	}

	if s.exposeVerificationToken {
		response.VerificationToken = &verificationToken
	}

	if s.emailVerificationRequired {
		if err := s.sendVerificationEmail(ctx, createdUser.ID, createdUser.Email, verificationToken, verificationTokenHash); err != nil {
			return SignUpData{}, err
		}
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

func (s *Service) VerifyEmail(ctx context.Context, req VerifyEmailRequest) (VerifyEmailData, error) {
	email := normalizeEmail(req.Email)
	rawToken := strings.TrimSpace(req.Token)
	if email == "" || rawToken == "" {
		return VerifyEmailData{}, sharederrors.Validation("email and token are required", nil)
	}

	if !otpPattern.MatchString(rawToken) {
		return VerifyEmailData{}, sharederrors.New(http.StatusBadRequest, "EMAIL_VERIFICATION_TOKEN_INVALID", "invalid email verification token", nil)
	}

	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return VerifyEmailData{}, sharederrors.New(http.StatusBadRequest, "EMAIL_VERIFICATION_TOKEN_INVALID", "invalid email verification token", nil)
		}

		return VerifyEmailData{}, sharederrors.Internal("failed to verify email", nil)
	}

	err = s.repo.ConsumeEmailVerificationOTP(ctx, user.ID, hashVerificationToken(rawToken), s.now())
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailVerificationTokenExpired):
			return VerifyEmailData{}, sharederrors.New(http.StatusBadRequest, "EMAIL_VERIFICATION_TOKEN_EXPIRED", "email verification token has expired", nil)
		case errors.Is(err, ErrEmailVerificationTokenConsumed):
			return VerifyEmailData{}, sharederrors.New(http.StatusBadRequest, "EMAIL_VERIFICATION_TOKEN_ALREADY_USED", "email verification token was already used", nil)
		case errors.Is(err, ErrEmailVerificationTokenInvalid):
			return VerifyEmailData{}, sharederrors.New(http.StatusBadRequest, "EMAIL_VERIFICATION_TOKEN_INVALID", "invalid email verification token", nil)
		default:
			return VerifyEmailData{}, sharederrors.Internal("failed to verify email", nil)
		}
	}

	return VerifyEmailData{OK: true}, nil
}

func (s *Service) ResendVerification(ctx context.Context, req ResendVerificationRequest) (ResendVerificationData, error) {
	email := normalizeEmail(req.Email)
	if email == "" {
		return ResendVerificationData{}, sharederrors.Validation("email is required", nil)
	}

	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ResendVerificationData{OK: true}, nil
		}

		return ResendVerificationData{}, sharederrors.Internal("failed to resend verification email", nil)
	}

	if user.EmailVerifiedAt != nil {
		return ResendVerificationData{OK: true}, nil
	}

	verificationToken, err := generateVerificationToken()
	if err != nil {
		return ResendVerificationData{}, sharederrors.Internal("failed to resend verification email", nil)
	}

	now := s.now()
	if err := s.repo.InvalidateEmailVerificationTokensForUser(ctx, user.ID, now); err != nil {
		return ResendVerificationData{}, sharederrors.Internal("failed to resend verification email", nil)
	}

	verificationTokenHash := hashVerificationToken(verificationToken)
	if err := s.repo.CreateEmailVerificationToken(ctx, user.ID, verificationTokenHash, now.Add(s.verificationTokenTTL), now); err != nil {
		return ResendVerificationData{}, sharederrors.Internal("failed to resend verification email", nil)
	}

	if err := s.sendVerificationEmail(ctx, user.ID, user.Email, verificationToken, verificationTokenHash); err != nil {
		return ResendVerificationData{}, err
	}

	return ResendVerificationData{OK: true}, nil
}

func (s *Service) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) (ForgotPasswordData, error) {
	email := normalizeEmail(req.Email)
	if email == "" {
		return ForgotPasswordData{}, sharederrors.Validation("email is required", nil)
	}

	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ForgotPasswordData{OK: true}, nil
		}

		return ForgotPasswordData{}, sharederrors.Internal("failed to request password reset", nil)
	}

	resetToken, err := generateVerificationToken()
	if err != nil {
		return ForgotPasswordData{}, sharederrors.Internal("failed to request password reset", nil)
	}

	now := s.now()
	if err := s.repo.InvalidatePasswordResetTokensForUser(ctx, user.ID, now); err != nil {
		return ForgotPasswordData{}, sharederrors.Internal("failed to request password reset", nil)
	}

	resetTokenHash := hashVerificationToken(resetToken)
	if err := s.repo.CreatePasswordResetToken(ctx, user.ID, resetTokenHash, now.Add(s.passwordResetTokenTTL), now); err != nil {
		return ForgotPasswordData{}, sharederrors.Internal("failed to request password reset", nil)
	}

	if err := s.sendPasswordResetEmail(ctx, user.ID, user.Email, resetToken, resetTokenHash); err != nil {
		return ForgotPasswordData{}, err
	}

	return ForgotPasswordData{OK: true}, nil
}

func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordRequest) (ResetPasswordData, error) {
	email := normalizeEmail(req.Email)
	rawToken := strings.TrimSpace(req.Token)
	newPassword := strings.TrimSpace(req.NewPassword)

	if email == "" || rawToken == "" {
		return ResetPasswordData{}, sharederrors.Validation("email and token are required", nil)
	}

	if newPassword == "" {
		return ResetPasswordData{}, sharederrors.Validation("newPassword is required", nil)
	}

	if !otpPattern.MatchString(rawToken) {
		return ResetPasswordData{}, sharederrors.New(http.StatusBadRequest, "PASSWORD_RESET_TOKEN_INVALID", "invalid password reset token", nil)
	}

	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ResetPasswordData{}, sharederrors.New(http.StatusBadRequest, "PASSWORD_RESET_TOKEN_INVALID", "invalid password reset token", nil)
		}

		return ResetPasswordData{}, sharederrors.Internal("failed to reset password", nil)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return ResetPasswordData{}, sharederrors.Internal("failed to reset password", nil)
	}

	err = s.repo.ConsumePasswordResetOTP(ctx, user.ID, hashVerificationToken(rawToken), string(passwordHash), s.now())
	if err != nil {
		switch {
		case errors.Is(err, ErrPasswordResetTokenExpired):
			return ResetPasswordData{}, sharederrors.New(http.StatusBadRequest, "PASSWORD_RESET_TOKEN_EXPIRED", "password reset token has expired", nil)
		case errors.Is(err, ErrPasswordResetTokenConsumed):
			return ResetPasswordData{}, sharederrors.New(http.StatusBadRequest, "PASSWORD_RESET_TOKEN_ALREADY_USED", "password reset token was already used", nil)
		case errors.Is(err, ErrPasswordResetTokenInvalid):
			return ResetPasswordData{}, sharederrors.New(http.StatusBadRequest, "PASSWORD_RESET_TOKEN_INVALID", "invalid password reset token", nil)
		default:
			return ResetPasswordData{}, sharederrors.Internal("failed to reset password", nil)
		}
	}

	return ResetPasswordData{OK: true}, nil
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
			Username:    user.Username,
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

func (s *Service) sendVerificationEmail(ctx context.Context, userID string, userEmail string, verificationToken string, verificationTokenHash string) error {
	if s.emailSender == nil {
		return sharederrors.ServiceUnavailable("failed to send verification email", nil)
	}

	if err := s.emailSender.SendVerificationEmail(ctx, sharedemail.VerificationEmailInput{
		ToEmail:        userEmail,
		Code:           verificationToken,
		IdempotencyKey: buildVerificationIdempotencyKey(userID, verificationTokenHash),
	}); err != nil {
		return sharederrors.ServiceUnavailable("failed to send verification email", nil)
	}

	return nil
}

func (s *Service) sendPasswordResetEmail(ctx context.Context, userID string, userEmail string, resetToken string, resetTokenHash string) error {
	if s.emailSender == nil {
		return sharederrors.ServiceUnavailable("failed to send password reset email", nil)
	}

	if err := s.emailSender.SendPasswordResetEmail(ctx, sharedemail.PasswordResetEmailInput{
		ToEmail:        userEmail,
		Code:           resetToken,
		IdempotencyKey: buildPasswordResetIdempotencyKey(userID, resetTokenHash),
	}); err != nil {
		return sharederrors.ServiceUnavailable("failed to send password reset email", nil)
	}

	return nil
}

func buildVerificationIdempotencyKey(userID string, tokenHash string) string {
	hashPrefix := tokenHash
	if len(hashPrefix) > 16 {
		hashPrefix = hashPrefix[:16]
	}

	return fmt.Sprintf("email-verification/%s/%s", strings.TrimSpace(userID), hashPrefix)
}

func buildPasswordResetIdempotencyKey(userID string, tokenHash string) string {
	hashPrefix := tokenHash
	if len(hashPrefix) > 16 {
		hashPrefix = hashPrefix[:16]
	}

	return fmt.Sprintf("password-reset/%s/%s", strings.TrimSpace(userID), hashPrefix)
}

func normalizeEmail(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func generateVerificationToken() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashVerificationToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func isEmailAlreadyInUseError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code != "23505" {
			return false
		}
		name := strings.ToLower(pgErr.ConstraintName)
		if name == "" {
			// Older drivers may not expose the constraint name; fall back to
			// treating any unique-violation as email (existing behavior).
			return true
		}
		return strings.Contains(name, "email")
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	return false
}

func isUsernameAlreadyInUseError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code != "23505" {
			return false
		}
		return strings.Contains(strings.ToLower(pgErr.ConstraintName), "username")
	}

	return false
}
