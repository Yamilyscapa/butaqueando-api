package follows

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UserExists(ctx context.Context, userID string) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return false, err
	}

	var count int64
	err = r.db.WithContext(ctx).Table("app.users").Where("id = ?", userUUID).Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *Repository) CreateFollow(ctx context.Context, followerUserID string, followingUserID string, createdAt time.Time) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}

	followerUUID, err := parseUUID(followerUserID)
	if err != nil {
		return false, err
	}

	followingUUID, err := parseUUID(followingUserID)
	if err != nil {
		return false, err
	}

	entity := userFollowEntity{
		FollowerUserID:  followerUUID,
		FollowingUserID: followingUUID,
		CreatedAt:       createdAt,
	}

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&entity)
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

func (r *Repository) DeleteFollow(ctx context.Context, followerUserID string, followingUserID string) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}

	followerUUID, err := parseUUID(followerUserID)
	if err != nil {
		return false, err
	}

	followingUUID, err := parseUUID(followingUserID)
	if err != nil {
		return false, err
	}

	result := r.db.WithContext(ctx).
		Where("follower_user_id = ? AND following_user_id = ?", followerUUID, followingUUID).
		Delete(&userFollowEntity{})
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

func (r *Repository) ListFollowers(ctx context.Context, userID string, after *followCursor, limit int) ([]FollowRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).
		Table("app.user_follows AS uf").
		Select("uf.follower_user_id AS user_id, uf.created_at AS followed_at, u.display_name, up.bio").
		Joins("JOIN app.users AS u ON u.id = uf.follower_user_id").
		Joins("LEFT JOIN app.user_profiles AS up ON up.user_id = u.id").
		Where("uf.following_user_id = ?", userUUID)

	if after != nil {
		afterUUID, err := parseUUID(after.UserID)
		if err != nil {
			return nil, err
		}

		query = query.Where("(uf.created_at < ?) OR (uf.created_at = ? AND uf.follower_user_id < ?)", after.CreatedAt, after.CreatedAt, afterUUID)
	}

	var rows []followListRow
	err = query.
		Order("uf.created_at DESC").
		Order("uf.follower_user_id DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	return mapFollowRows(rows), nil
}

func (r *Repository) ListFollowings(ctx context.Context, userID string, after *followCursor, limit int) ([]FollowRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).
		Table("app.user_follows AS uf").
		Select("uf.following_user_id AS user_id, uf.created_at AS followed_at, u.display_name, up.bio").
		Joins("JOIN app.users AS u ON u.id = uf.following_user_id").
		Joins("LEFT JOIN app.user_profiles AS up ON up.user_id = u.id").
		Where("uf.follower_user_id = ?", userUUID)

	if after != nil {
		afterUUID, err := parseUUID(after.UserID)
		if err != nil {
			return nil, err
		}

		query = query.Where("(uf.created_at < ?) OR (uf.created_at = ? AND uf.following_user_id < ?)", after.CreatedAt, after.CreatedAt, afterUUID)
	}

	var rows []followListRow
	err = query.
		Order("uf.created_at DESC").
		Order("uf.following_user_id DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	return mapFollowRows(rows), nil
}

func (r *Repository) ensureDB() error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}

	return nil
}

type userFollowEntity struct {
	FollowerUserID  uuid.UUID `gorm:"column:follower_user_id;type:uuid;primaryKey"`
	FollowingUserID uuid.UUID `gorm:"column:following_user_id;type:uuid;primaryKey"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (userFollowEntity) TableName() string {
	return "app.user_follows"
}

type followListRow struct {
	UserID      uuid.UUID `gorm:"column:user_id"`
	DisplayName string    `gorm:"column:display_name"`
	Bio         *string   `gorm:"column:bio"`
	FollowedAt  time.Time `gorm:"column:followed_at"`
}

func mapFollowRows(rows []followListRow) []FollowRecord {
	records := make([]FollowRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, FollowRecord{
			UserID:      row.UserID.String(),
			DisplayName: row.DisplayName,
			Bio:         row.Bio,
			FollowedAt:  row.FollowedAt,
		})
	}

	return records
}

func parseUUID(raw string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse uuid: %w", err)
	}

	return parsed, nil
}
