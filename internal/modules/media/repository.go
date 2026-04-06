package media

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetPublishedPlayMediaObjectKey(ctx context.Context, playID string, mediaID string) (string, error) {
	if err := r.ensureDB(); err != nil {
		return "", err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return "", err
	}

	mediaUUID, err := parseUUID(mediaID)
	if err != nil {
		return "", err
	}

	var row struct {
		ObjectKey string `gorm:"column:object_key"`
	}

	err = r.db.WithContext(ctx).
		Table("app.play_media AS pm").
		Select("pm.object_key").
		Joins("JOIN app.plays AS p ON p.id = pm.play_id").
		Where("pm.id = ? AND pm.play_id = ? AND p.curation_status = ?", mediaUUID, playUUID, "published").
		Take(&row).Error
	if err != nil {
		return "", err
	}

	return row.ObjectKey, nil
}

func (r *Repository) GetUserAvatarObjectKey(ctx context.Context, userID string) (string, error) {
	if err := r.ensureDB(); err != nil {
		return "", err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return "", err
	}

	var row struct {
		AvatarObjectKey string `gorm:"column:avatar_object_key"`
	}

	err = r.db.WithContext(ctx).
		Table("app.user_profiles AS up").
		Select("up.avatar_object_key").
		Where("up.user_id = ? AND up.avatar_object_key IS NOT NULL", userUUID).
		Take(&row).Error
	if err != nil {
		return "", err
	}

	return row.AvatarObjectKey, nil
}

func (r *Repository) ensureDB() error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}

	return nil
}

func parseUUID(raw string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse uuid: %w", err)
	}

	return parsed, nil
}
