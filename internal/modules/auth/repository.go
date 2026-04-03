package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRecord struct {
	ID              string
	DisplayName     string
	Email           string
	PasswordHash    string
	Role            string
	EmailVerifiedAt *time.Time
}

type RefreshTokenRecord struct {
	TokenID   string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type CreatePendingUserInput struct {
	DisplayName                string
	Email                      string
	PasswordHash               string
	Role                       string
	EmailVerificationTokenHash string
	EmailVerificationExpiresAt time.Time
	CreatedAt                  time.Time
}

type Repository struct {
	db *gorm.DB
}

var (
	ErrEmailVerificationTokenInvalid  = errors.New("email verification token invalid")
	ErrEmailVerificationTokenExpired  = errors.New("email verification token expired")
	ErrEmailVerificationTokenConsumed = errors.New("email verification token consumed")
	ErrPasswordResetTokenInvalid      = errors.New("password reset token invalid")
	ErrPasswordResetTokenExpired      = errors.New("password reset token expired")
	ErrPasswordResetTokenConsumed     = errors.New("password reset token consumed")
)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (UserRecord, error) {
	if err := r.ensureDB(); err != nil {
		return UserRecord{}, err
	}

	var entity userEntity
	err := r.db.WithContext(ctx).
		Where("lower(email) = ?", email).
		Take(&entity).Error
	if err != nil {
		return UserRecord{}, err
	}

	return mapUserEntityToRecord(entity), nil
}

func (r *Repository) FindUserByID(ctx context.Context, userID string) (UserRecord, error) {
	if err := r.ensureDB(); err != nil {
		return UserRecord{}, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return UserRecord{}, err
	}

	var entity userEntity
	err = r.db.WithContext(ctx).
		Where("id = ?", userUUID).
		Take(&entity).Error
	if err != nil {
		return UserRecord{}, err
	}

	return mapUserEntityToRecord(entity), nil
}

func (r *Repository) InsertRefreshToken(ctx context.Context, tokenID string, userID string, expiresAt time.Time, createdAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	tokenUUID, err := parseUUID(tokenID)
	if err != nil {
		return err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return err
	}

	entity := userRefreshTokenEntity{
		TokenID:   tokenUUID,
		UserID:    userUUID,
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}

	return r.db.WithContext(ctx).Create(&entity).Error
}

func (r *Repository) FindActiveRefreshToken(ctx context.Context, tokenID string, userID string, now time.Time) (RefreshTokenRecord, error) {
	if err := r.ensureDB(); err != nil {
		return RefreshTokenRecord{}, err
	}

	tokenUUID, err := parseUUID(tokenID)
	if err != nil {
		return RefreshTokenRecord{}, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return RefreshTokenRecord{}, err
	}

	var entity userRefreshTokenEntity
	err = r.db.WithContext(ctx).
		Where("token_id = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?", tokenUUID, userUUID, now).
		Take(&entity).Error
	if err != nil {
		return RefreshTokenRecord{}, err
	}

	return mapRefreshTokenEntityToRecord(entity), nil
}

func (r *Repository) RotateRefreshToken(ctx context.Context, oldTokenID string, userID string, newTokenID string, newExpiresAt time.Time, now time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	oldTokenUUID, err := parseUUID(oldTokenID)
	if err != nil {
		return err
	}

	newTokenUUID, err := parseUUID(newTokenID)
	if err != nil {
		return err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&userRefreshTokenEntity{}).
			Where("token_id = ? AND user_id = ? AND revoked_at IS NULL", oldTokenUUID, userUUID).
			Updates(map[string]any{"revoked_at": now, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		newEntity := userRefreshTokenEntity{
			TokenID:   newTokenUUID,
			UserID:    userUUID,
			ExpiresAt: newExpiresAt,
			CreatedAt: now,
			UpdatedAt: now,
		}

		return tx.Create(&newEntity).Error
	})
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenID string, userID string, now time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	tokenUUID, err := parseUUID(tokenID)
	if err != nil {
		return err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Model(&userRefreshTokenEntity{}).
		Where("token_id = ? AND user_id = ?", tokenUUID, userUUID).
		Updates(map[string]any{
			"revoked_at": gorm.Expr("COALESCE(revoked_at, ?)", now),
			"updated_at": now,
		}).Error
}

func (r *Repository) CreatePendingUser(ctx context.Context, input CreatePendingUserInput) (UserRecord, error) {
	if err := r.ensureDB(); err != nil {
		return UserRecord{}, err
	}

	var createdUser userEntity
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		createdUser = userEntity{
			DisplayName:  input.DisplayName,
			Email:        input.Email,
			PasswordHash: input.PasswordHash,
			Role:         input.Role,
			CreatedAt:    input.CreatedAt,
			UpdatedAt:    input.CreatedAt,
		}

		if err := tx.Create(&createdUser).Error; err != nil {
			return err
		}

		profile := userProfileEntity{
			UserID:    createdUser.ID,
			Bio:       nil,
			CreatedAt: input.CreatedAt,
			UpdatedAt: input.CreatedAt,
		}
		if err := tx.Create(&profile).Error; err != nil {
			return err
		}

		verificationToken := emailVerificationTokenEntity{
			UserID:    createdUser.ID,
			TokenHash: input.EmailVerificationTokenHash,
			ExpiresAt: input.EmailVerificationExpiresAt,
			CreatedAt: input.CreatedAt,
		}
		return tx.Create(&verificationToken).Error
	})
	if err != nil {
		return UserRecord{}, err
	}

	return mapUserEntityToRecord(createdUser), nil
}

func (r *Repository) VerifyEmailByTokenHash(ctx context.Context, tokenHash string, now time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var token emailVerificationTokenEntity
		err := tx.Where("token_hash = ?", tokenHash).Take(&token).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEmailVerificationTokenInvalid
			}

			return err
		}

		if token.ConsumedAt != nil {
			return ErrEmailVerificationTokenConsumed
		}

		if !token.ExpiresAt.After(now) {
			return ErrEmailVerificationTokenExpired
		}

		consumeResult := tx.Model(&emailVerificationTokenEntity{}).
			Where("id = ? AND consumed_at IS NULL", token.ID).
			Updates(map[string]any{"consumed_at": now})
		if consumeResult.Error != nil {
			return consumeResult.Error
		}

		if consumeResult.RowsAffected == 0 {
			return ErrEmailVerificationTokenConsumed
		}

		return tx.Model(&userEntity{}).
			Where("id = ?", token.UserID).
			Updates(map[string]any{"email_verified_at": now, "updated_at": now}).Error
	})
}

func (r *Repository) CreateEmailVerificationToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time, createdAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return err
	}

	entity := emailVerificationTokenEntity{
		UserID:    userUUID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
	}

	return r.db.WithContext(ctx).Create(&entity).Error
}

func (r *Repository) InvalidateEmailVerificationTokensForUser(ctx context.Context, userID string, now time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Model(&emailVerificationTokenEntity{}).
		Where("user_id = ? AND consumed_at IS NULL", userUUID).
		Updates(map[string]any{"consumed_at": now}).Error
}

func (r *Repository) CreatePasswordResetToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time, createdAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return err
	}

	entity := passwordResetTokenEntity{
		UserID:    userUUID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
	}

	return r.db.WithContext(ctx).Create(&entity).Error
}

func (r *Repository) InvalidatePasswordResetTokensForUser(ctx context.Context, userID string, now time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Model(&passwordResetTokenEntity{}).
		Where("user_id = ? AND consumed_at IS NULL", userUUID).
		Updates(map[string]any{"consumed_at": now}).Error
}

func (r *Repository) ResetPasswordByTokenHash(ctx context.Context, tokenHash string, passwordHash string, now time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var token passwordResetTokenEntity
		err := tx.Where("token_hash = ?", tokenHash).Take(&token).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPasswordResetTokenInvalid
			}

			return err
		}

		if token.ConsumedAt != nil {
			return ErrPasswordResetTokenConsumed
		}

		if !token.ExpiresAt.After(now) {
			return ErrPasswordResetTokenExpired
		}

		consumeResult := tx.Model(&passwordResetTokenEntity{}).
			Where("id = ? AND consumed_at IS NULL", token.ID).
			Updates(map[string]any{"consumed_at": now})
		if consumeResult.Error != nil {
			return consumeResult.Error
		}

		if consumeResult.RowsAffected == 0 {
			return ErrPasswordResetTokenConsumed
		}

		if err := tx.Model(&userEntity{}).
			Where("id = ?", token.UserID).
			Updates(map[string]any{"password_hash": passwordHash, "updated_at": now}).Error; err != nil {
			return err
		}

		return tx.Model(&userRefreshTokenEntity{}).
			Where("user_id = ? AND revoked_at IS NULL", token.UserID).
			Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error
	})
}

func (r *Repository) ensureDB() error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}

	return nil
}

type userEntity struct {
	ID              uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	DisplayName     string     `gorm:"column:display_name"`
	Email           string     `gorm:"column:email"`
	PasswordHash    string     `gorm:"column:password_hash"`
	Role            string     `gorm:"column:role"`
	EmailVerifiedAt *time.Time `gorm:"column:email_verified_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (userEntity) TableName() string {
	return "app.users"
}

type userProfileEntity struct {
	UserID    uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	Bio       *string   `gorm:"column:bio"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (userProfileEntity) TableName() string {
	return "app.user_profiles"
}

type userRefreshTokenEntity struct {
	ID        uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	TokenID   uuid.UUID  `gorm:"column:token_id;type:uuid;uniqueIndex"`
	UserID    uuid.UUID  `gorm:"column:user_id;type:uuid"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	RevokedAt *time.Time `gorm:"column:revoked_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

func (userRefreshTokenEntity) TableName() string {
	return "app.user_refresh_tokens"
}

type emailVerificationTokenEntity struct {
	ID         uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     uuid.UUID  `gorm:"column:user_id;type:uuid"`
	TokenHash  string     `gorm:"column:token_hash"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	ConsumedAt *time.Time `gorm:"column:consumed_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
}

func (emailVerificationTokenEntity) TableName() string {
	return "app.email_verification_tokens"
}

type passwordResetTokenEntity struct {
	ID         uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     uuid.UUID  `gorm:"column:user_id;type:uuid"`
	TokenHash  string     `gorm:"column:token_hash"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	ConsumedAt *time.Time `gorm:"column:consumed_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
}

func (passwordResetTokenEntity) TableName() string {
	return "app.password_reset_tokens"
}

func mapUserEntityToRecord(entity userEntity) UserRecord {
	return UserRecord{
		ID:              entity.ID.String(),
		DisplayName:     entity.DisplayName,
		Email:           entity.Email,
		PasswordHash:    entity.PasswordHash,
		Role:            entity.Role,
		EmailVerifiedAt: entity.EmailVerifiedAt,
	}
}

func mapRefreshTokenEntityToRecord(entity userRefreshTokenEntity) RefreshTokenRecord {
	return RefreshTokenRecord{
		TokenID:   entity.TokenID.String(),
		UserID:    entity.UserID.String(),
		ExpiresAt: entity.ExpiresAt,
		RevokedAt: entity.RevokedAt,
	}
}

func parseUUID(raw string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse uuid: %w", err)
	}

	return parsed, nil
}
