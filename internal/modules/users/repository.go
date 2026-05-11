package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetPublicProfile(ctx context.Context, userID string) (PublicProfileRecord, error) {
	if err := r.ensureDB(); err != nil {
		return PublicProfileRecord{}, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return PublicProfileRecord{}, err
	}

	var user userEntity
	err = r.db.WithContext(ctx).Where("id = ?", userUUID).Take(&user).Error
	if err != nil {
		return PublicProfileRecord{}, err
	}

	var profile userProfileEntity
	err = r.db.WithContext(ctx).Where("user_id = ?", userUUID).Take(&profile).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return PublicProfileRecord{}, err
	}

	followersCount, err := r.countFollowers(ctx, userUUID)
	if err != nil {
		return PublicProfileRecord{}, err
	}

	followingCount, err := r.countFollowing(ctx, userUUID)
	if err != nil {
		return PublicProfileRecord{}, err
	}

	watchedCount, err := r.countWatchedPublished(ctx, userUUID)
	if err != nil {
		return PublicProfileRecord{}, err
	}

	reviewsCount, err := r.countPublishedReviews(ctx, userUUID)
	if err != nil {
		return PublicProfileRecord{}, err
	}

	return PublicProfileRecord{
		ID:              user.ID.String(),
		DisplayName:     user.DisplayName,
		Bio:             profile.Bio,
		AvatarObjectKey: profile.AvatarObjectKey,
		AvatarVersion:   profile.UpdatedAt.Format(time.RFC3339),
		FollowersCount:  followersCount,
		FollowingCount:  followingCount,
		WatchedCount:    watchedCount,
		ReviewsCount:    reviewsCount,
	}, nil
}

func (r *Repository) GetMeProfile(ctx context.Context, userID string) (MeProfileRecord, error) {
	if err := r.ensureDB(); err != nil {
		return MeProfileRecord{}, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return MeProfileRecord{}, err
	}

	var user userEntity
	err = r.db.WithContext(ctx).Where("id = ?", userUUID).Take(&user).Error
	if err != nil {
		return MeProfileRecord{}, err
	}

	var profile userProfileEntity
	err = r.db.WithContext(ctx).Where("user_id = ?", userUUID).Take(&profile).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return MeProfileRecord{}, err
	}

	followersCount, err := r.countFollowers(ctx, userUUID)
	if err != nil {
		return MeProfileRecord{}, err
	}

	followingCount, err := r.countFollowing(ctx, userUUID)
	if err != nil {
		return MeProfileRecord{}, err
	}

	watchedCount, err := r.countWatchedPublished(ctx, userUUID)
	if err != nil {
		return MeProfileRecord{}, err
	}

	reviewsCount, err := r.countPublishedReviews(ctx, userUUID)
	if err != nil {
		return MeProfileRecord{}, err
	}

	return MeProfileRecord{
		ID:              user.ID.String(),
		DisplayName:     user.DisplayName,
		Email:           user.Email,
		Role:            user.Role,
		Bio:             profile.Bio,
		AvatarObjectKey: profile.AvatarObjectKey,
		AvatarVersion:   profile.UpdatedAt.Format(time.RFC3339),
		FollowersCount:  followersCount,
		FollowingCount:  followingCount,
		WatchedCount:    watchedCount,
		ReviewsCount:    reviewsCount,
	}, nil
}

func (r *Repository) UpdateMeProfile(ctx context.Context, userID string, patch UpdateMeProfilePatch) (MeProfileRecord, error) {
	if err := r.ensureDB(); err != nil {
		return MeProfileRecord{}, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return MeProfileRecord{}, err
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if patch.DisplayNameSet {
			if patch.DisplayName == nil {
				return gorm.ErrInvalidData
			}

			result := tx.Model(&userEntity{}).
				Where("id = ?", userUUID).
				Update("display_name", *patch.DisplayName)
			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		}

		if patch.BioSet || patch.AvatarObjectKeySet {
			updates := map[string]any{}
			if patch.BioSet {
				updates["bio"] = patch.Bio
			}

			if patch.AvatarObjectKeySet {
				updates["avatar_object_key"] = patch.AvatarObjectKey
				variantsJSON, err := encodeAvatarVariants(patch.AvatarVariants)
				if err != nil {
					return err
				}
				updates["avatar_variants"] = variantsJSON
				updates["avatar_blurhash"] = patch.AvatarBlurhash
			}

			result := tx.Model(&userProfileEntity{}).
				Where("user_id = ?", userUUID).
				Updates(updates)
			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 0 {
				profile := userProfileEntity{
					UserID: userUUID,
				}

				if patch.BioSet {
					profile.Bio = patch.Bio
				}

				if patch.AvatarObjectKeySet {
					profile.AvatarObjectKey = patch.AvatarObjectKey
					variantsJSON, err := encodeAvatarVariants(patch.AvatarVariants)
					if err != nil {
						return err
					}
					profile.AvatarVariants = variantsJSON
					profile.AvatarBlurhash = patch.AvatarBlurhash
				}

				if err := tx.Create(&profile).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return MeProfileRecord{}, err
	}

	return r.GetMeProfile(ctx, userID)
}

func (r *Repository) CreateAccountDeletionRequest(ctx context.Context, userID string) (AccountDeletionRequestRecord, error) {
	if err := r.ensureDB(); err != nil {
		return AccountDeletionRequestRecord{}, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return AccountDeletionRequestRecord{}, err
	}

	var user userEntity
	if err := r.db.WithContext(ctx).Where("id = ?", userUUID).Take(&user).Error; err != nil {
		return AccountDeletionRequestRecord{}, err
	}

	var existing accountDeletionRequestEntity
	err = r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userUUID, "pending").
		Order("requested_at DESC").
		Take(&existing).Error
	if err == nil {
		return AccountDeletionRequestRecord{
			ID:          existing.ID.String(),
			Status:      existing.Status,
			RequestedAt: existing.RequestedAt.Format(time.RFC3339),
		}, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return AccountDeletionRequestRecord{}, err
	}

	now := time.Now().UTC()
	request := accountDeletionRequestEntity{
		UserID:      userUUID,
		Status:      "pending",
		RequestedAt: now,
	}

	if err := r.db.WithContext(ctx).Create(&request).Error; err != nil {
		return AccountDeletionRequestRecord{}, err
	}

	return AccountDeletionRequestRecord{
		ID:          request.ID.String(),
		Status:      request.Status,
		RequestedAt: request.RequestedAt.Format(time.RFC3339),
	}, nil
}

func (r *Repository) countFollowers(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&userFollowEntity{}).
		Where("following_user_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *Repository) countFollowing(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&userFollowEntity{}).
		Where("follower_user_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *Repository) countWatchedPublished(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("app.user_play_engagements AS e").
		Joins("JOIN app.plays AS p ON p.id = e.play_id").
		Where("e.user_id = ? AND e.kind = ? AND p.curation_status = ?", userID, "attended", "published").
		Count(&count).Error
	return count, err
}

func (r *Repository) countPublishedReviews(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("app.reviews").
		Where("user_id = ? AND status = ?", userID, "published").
		Count(&count).Error
	return count, err
}

func (r *Repository) ensureDB() error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}

	return nil
}

type userEntity struct {
	ID          uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	DisplayName string    `gorm:"column:display_name"`
	Email       string    `gorm:"column:email"`
	Role        string    `gorm:"column:role"`
}

func (userEntity) TableName() string {
	return "app.users"
}

type userProfileEntity struct {
	UserID          uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	Bio             *string   `gorm:"column:bio"`
	AvatarObjectKey *string   `gorm:"column:avatar_object_key"`
	AvatarVariants  []byte    `gorm:"column:avatar_variants;type:jsonb"`
	AvatarBlurhash  *string   `gorm:"column:avatar_blurhash"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func encodeAvatarVariants(variants []AvatarVariantRecord) ([]byte, error) {
	if len(variants) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(variants)
}

func (userProfileEntity) TableName() string {
	return "app.user_profiles"
}

type userFollowEntity struct {
	FollowerUserID  uuid.UUID `gorm:"column:follower_user_id;type:uuid"`
	FollowingUserID uuid.UUID `gorm:"column:following_user_id;type:uuid"`
}

type accountDeletionRequestEntity struct {
	ID          uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	UserID      uuid.UUID  `gorm:"column:user_id;type:uuid"`
	Status      string     `gorm:"column:status"`
	Reason      *string    `gorm:"column:reason"`
	RequestedAt time.Time  `gorm:"column:requested_at"`
	ProcessedAt *time.Time `gorm:"column:processed_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
}

func (accountDeletionRequestEntity) TableName() string {
	return "app.account_deletion_requests"
}

func (userFollowEntity) TableName() string {
	return "app.user_follows"
}

func parseUUID(raw string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse uuid: %w", err)
	}

	return parsed, nil
}
