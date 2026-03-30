package auth

import (
	"context"
	"time"

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

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ensureDB() error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}

	return nil
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (UserRecord, error) {
	if err := r.ensureDB(); err != nil {
		return UserRecord{}, err
	}

	var record UserRecord
	err := r.db.WithContext(ctx).
		Table("app.users").
		Select("id::text AS id, display_name, email, password_hash, role, email_verified_at").
		Where("lower(email) = ?", email).
		Take(&record).Error

	return record, err
}

func (r *Repository) FindUserByID(ctx context.Context, userID string) (UserRecord, error) {
	if err := r.ensureDB(); err != nil {
		return UserRecord{}, err
	}

	var record UserRecord
	err := r.db.WithContext(ctx).
		Table("app.users").
		Select("id::text AS id, display_name, email, password_hash, role, email_verified_at").
		Where("id = ?::uuid", userID).
		Take(&record).Error

	return record, err
}

func (r *Repository) InsertRefreshToken(ctx context.Context, tokenID string, userID string, expiresAt time.Time, createdAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Exec(
		`INSERT INTO app.user_refresh_tokens (token_id, user_id, expires_at, created_at, updated_at)
		 VALUES (?::uuid, ?::uuid, ?, ?, ?)`,
		tokenID,
		userID,
		expiresAt,
		createdAt,
		createdAt,
	).Error
}

func (r *Repository) FindActiveRefreshToken(ctx context.Context, tokenID string, userID string, now time.Time) (RefreshTokenRecord, error) {
	if err := r.ensureDB(); err != nil {
		return RefreshTokenRecord{}, err
	}

	var record RefreshTokenRecord
	err := r.db.WithContext(ctx).
		Table("app.user_refresh_tokens").
		Select("token_id::text AS token_id, user_id::text AS user_id, expires_at, revoked_at").
		Where("token_id = ?::uuid AND user_id = ?::uuid AND revoked_at IS NULL AND expires_at > ?", tokenID, userID, now).
		Take(&record).Error

	return record, err
}

func (r *Repository) RotateRefreshToken(ctx context.Context, oldTokenID string, userID string, newTokenID string, newExpiresAt time.Time, now time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(
			`UPDATE app.user_refresh_tokens
			 SET revoked_at = ?, updated_at = ?
			 WHERE token_id = ?::uuid AND user_id = ?::uuid AND revoked_at IS NULL`,
			now,
			now,
			oldTokenID,
			userID,
		)
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return tx.Exec(
			`INSERT INTO app.user_refresh_tokens (token_id, user_id, expires_at, created_at, updated_at)
			 VALUES (?::uuid, ?::uuid, ?, ?, ?)`,
			newTokenID,
			userID,
			newExpiresAt,
			now,
			now,
		).Error
	})
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenID string, userID string, now time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Exec(
		`UPDATE app.user_refresh_tokens
		 SET revoked_at = COALESCE(revoked_at, ?), updated_at = ?
		 WHERE token_id = ?::uuid AND user_id = ?::uuid`,
		now,
		now,
		tokenID,
		userID,
	).Error
}
