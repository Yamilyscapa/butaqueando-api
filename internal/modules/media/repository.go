package media

import (
	"context"
	"encoding/json"
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

type MediaVariant struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	ObjectKey string `json:"objectKey"`
	SizeBytes int64  `json:"sizeBytes"`
}

type MediaRecord struct {
	ObjectKey string
	Variants  []MediaVariant
}

func decodeVariants(raw []byte) []MediaVariant {
	if len(raw) == 0 {
		return nil
	}
	var out []MediaVariant
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func (r *Repository) GetPublishedPlayMedia(ctx context.Context, playID string, mediaID string) (MediaRecord, error) {
	if err := r.ensureDB(); err != nil {
		return MediaRecord{}, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return MediaRecord{}, err
	}

	mediaUUID, err := parseUUID(mediaID)
	if err != nil {
		return MediaRecord{}, err
	}

	var row struct {
		ObjectKey string `gorm:"column:object_key"`
		Variants  []byte `gorm:"column:variants"`
	}

	err = r.db.WithContext(ctx).
		Table("app.play_media AS pm").
		Select("pm.object_key, pm.variants").
		Joins("JOIN app.plays AS p ON p.id = pm.play_id").
		Where("pm.id = ? AND pm.play_id = ? AND p.curation_status = ?", mediaUUID, playUUID, "published").
		Take(&row).Error
	if err != nil {
		return MediaRecord{}, err
	}

	return MediaRecord{ObjectKey: row.ObjectKey, Variants: decodeVariants(row.Variants)}, nil
}

func (r *Repository) GetPublishedPlayMediaObjectKey(ctx context.Context, playID string, mediaID string) (string, error) {
	record, err := r.GetPublishedPlayMedia(ctx, playID, mediaID)
	if err != nil {
		return "", err
	}
	return record.ObjectKey, nil
}

func (r *Repository) GetUserAvatar(ctx context.Context, userID string) (MediaRecord, error) {
	if err := r.ensureDB(); err != nil {
		return MediaRecord{}, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return MediaRecord{}, err
	}

	var row struct {
		AvatarObjectKey string `gorm:"column:avatar_object_key"`
		AvatarVariants  []byte `gorm:"column:avatar_variants"`
	}

	err = r.db.WithContext(ctx).
		Table("app.user_profiles AS up").
		Select("up.avatar_object_key, up.avatar_variants").
		Where("up.user_id = ? AND up.avatar_object_key IS NOT NULL", userUUID).
		Take(&row).Error
	if err != nil {
		return MediaRecord{}, err
	}

	return MediaRecord{ObjectKey: row.AvatarObjectKey, Variants: decodeVariants(row.AvatarVariants)}, nil
}

func (r *Repository) GetUserAvatarObjectKey(ctx context.Context, userID string) (string, error) {
	record, err := r.GetUserAvatar(ctx, userID)
	if err != nil {
		return "", err
	}
	return record.ObjectKey, nil
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
