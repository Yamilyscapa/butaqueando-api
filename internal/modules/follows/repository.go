package follows

import (
	"context"
	"fmt"
	"strings"
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

func (r *Repository) ListFollowingActivity(ctx context.Context, actorUserID string, after *followingActivityCursor, limit int) ([]FollowingActivityRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	actorUUID, err := parseUUID(actorUserID)
	if err != nil {
		return nil, err
	}

	sql := `
		SELECT * FROM (
			SELECT
				e.id AS activity_id,
				'saved' AS activity_type,
				e.created_at AS activity_at,
				u.id AS actor_id,
				u.display_name AS actor_display_name,
				up.bio AS actor_bio,
				p.id AS play_id,
				p.title AS play_title,
				p.theater_name,
				p.city,
				p.availability_status,
				p.published_at,
				media.poster_media_id,
				prs.avg_rating,
				COALESCE(prs.review_count, 0) AS review_count,
				NULL::uuid AS review_id,
				NULL::numeric AS review_rating,
				NULL::text AS review_title,
				NULL::text AS review_body,
				NULL::boolean AS review_contains_spoilers,
				NULL::timestamptz AS review_created_at,
				NULL::timestamptz AS review_updated_at
			FROM app.user_play_engagements AS e
			JOIN app.user_follows AS uf ON uf.following_user_id = e.user_id
			JOIN app.users AS u ON u.id = e.user_id
			LEFT JOIN app.user_profiles AS up ON up.user_id = u.id
			JOIN app.plays AS p ON p.id = e.play_id
			LEFT JOIN app.play_rating_stats AS prs ON prs.play_id = p.id
			LEFT JOIN LATERAL (
				SELECT pm.id AS poster_media_id
				FROM app.play_media AS pm
				WHERE pm.play_id = p.id
				ORDER BY CASE WHEN pm.kind = 'poster' THEN 0 ELSE 1 END, pm.sort_order ASC, pm.created_at ASC
				LIMIT 1
			) AS media ON true
			WHERE uf.follower_user_id = ?
				AND e.kind = 'wishlist'
				AND p.curation_status = 'published'

			UNION ALL

			SELECT
				r.id AS activity_id,
				'reviewed' AS activity_type,
				r.created_at AS activity_at,
				u.id AS actor_id,
				u.display_name AS actor_display_name,
				up.bio AS actor_bio,
				p.id AS play_id,
				p.title AS play_title,
				p.theater_name,
				p.city,
				p.availability_status,
				p.published_at,
				media.poster_media_id,
				prs.avg_rating,
				COALESCE(prs.review_count, 0) AS review_count,
				r.id AS review_id,
				r.rating AS review_rating,
				r.title AS review_title,
				r.body AS review_body,
				r.contains_spoilers AS review_contains_spoilers,
				r.created_at AS review_created_at,
				r.updated_at AS review_updated_at
			FROM app.reviews AS r
			JOIN app.user_follows AS uf ON uf.following_user_id = r.user_id
			JOIN app.users AS u ON u.id = r.user_id
			LEFT JOIN app.user_profiles AS up ON up.user_id = u.id
			JOIN app.plays AS p ON p.id = r.play_id
			LEFT JOIN app.play_rating_stats AS prs ON prs.play_id = p.id
			LEFT JOIN LATERAL (
				SELECT pm.id AS poster_media_id
				FROM app.play_media AS pm
				WHERE pm.play_id = p.id
				ORDER BY CASE WHEN pm.kind = 'poster' THEN 0 ELSE 1 END, pm.sort_order ASC, pm.created_at ASC
				LIMIT 1
			) AS media ON true
			WHERE uf.follower_user_id = ?
				AND r.status = 'published'
				AND p.curation_status = 'published'

			UNION ALL

			SELECT
				p.id AS activity_id,
				'uploaded' AS activity_type,
				p.published_at AS activity_at,
				u.id AS actor_id,
				u.display_name AS actor_display_name,
				up.bio AS actor_bio,
				p.id AS play_id,
				p.title AS play_title,
				p.theater_name,
				p.city,
				p.availability_status,
				p.published_at,
				media.poster_media_id,
				prs.avg_rating,
				COALESCE(prs.review_count, 0) AS review_count,
				NULL::uuid AS review_id,
				NULL::numeric AS review_rating,
				NULL::text AS review_title,
				NULL::text AS review_body,
				NULL::boolean AS review_contains_spoilers,
				NULL::timestamptz AS review_created_at,
				NULL::timestamptz AS review_updated_at
			FROM app.plays AS p
			JOIN app.user_follows AS uf ON uf.following_user_id = p.created_by_user_id
			JOIN app.users AS u ON u.id = p.created_by_user_id
			LEFT JOIN app.user_profiles AS up ON up.user_id = u.id
			LEFT JOIN app.play_rating_stats AS prs ON prs.play_id = p.id
			LEFT JOIN LATERAL (
				SELECT pm.id AS poster_media_id
				FROM app.play_media AS pm
				WHERE pm.play_id = p.id
				ORDER BY CASE WHEN pm.kind = 'poster' THEN 0 ELSE 1 END, pm.sort_order ASC, pm.created_at ASC
				LIMIT 1
			) AS media ON true
			WHERE uf.follower_user_id = ?
				AND p.curation_status = 'published'
				AND p.published_at IS NOT NULL

			UNION ALL

			SELECT
				uf.follower_user_id AS activity_id,
				'followed_you' AS activity_type,
				uf.created_at AS activity_at,
				u.id AS actor_id,
				u.display_name AS actor_display_name,
				up.bio AS actor_bio,
				NULL::uuid AS play_id,
				NULL::text AS play_title,
				NULL::text AS theater_name,
				NULL::text AS city,
				NULL::text AS availability_status,
				NULL::timestamptz AS published_at,
				NULL::uuid AS poster_media_id,
				NULL::numeric AS avg_rating,
				0::bigint AS review_count,
				NULL::uuid AS review_id,
				NULL::smallint AS review_rating,
				NULL::text AS review_title,
				NULL::text AS review_body,
				NULL::boolean AS review_contains_spoilers,
				NULL::timestamptz AS review_created_at,
				NULL::timestamptz AS review_updated_at
			FROM app.user_follows AS uf
			JOIN app.users AS u ON u.id = uf.follower_user_id
			LEFT JOIN app.user_profiles AS up ON up.user_id = u.id
			WHERE uf.following_user_id = ?
		) AS activity
	`

	args := []any{actorUUID, actorUUID, actorUUID, actorUUID}

	if after != nil {
		afterUUID, err := parseUUID(after.ActivityID)
		if err != nil {
			return nil, err
		}

		sql += " WHERE (activity.activity_at < ?) OR (activity.activity_at = ? AND activity.activity_id < ?)"
		args = append(args, after.ActivityAt, after.ActivityAt, afterUUID)
	}

	sql += " ORDER BY activity.activity_at DESC, activity.activity_id DESC LIMIT ?"
	args = append(args, limit)

	var rows []followingActivityRow
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	return mapFollowingActivityRows(rows), nil
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

type followingActivityRow struct {
	ActivityID             uuid.UUID  `gorm:"column:activity_id"`
	ActivityType           string     `gorm:"column:activity_type"`
	ActivityAt             time.Time  `gorm:"column:activity_at"`
	ActorID                uuid.UUID  `gorm:"column:actor_id"`
	ActorDisplayName       string     `gorm:"column:actor_display_name"`
	ActorBio               *string    `gorm:"column:actor_bio"`
	PlayID                 *uuid.UUID `gorm:"column:play_id"`
	PlayTitle              *string    `gorm:"column:play_title"`
	TheaterName            *string    `gorm:"column:theater_name"`
	City                   *string    `gorm:"column:city"`
	AvailabilityStatus     *string    `gorm:"column:availability_status"`
	PublishedAt            *time.Time `gorm:"column:published_at"`
	PosterMediaID          *uuid.UUID `gorm:"column:poster_media_id"`
	AverageRating          *float64   `gorm:"column:avg_rating"`
	ReviewCount            int64      `gorm:"column:review_count"`
	ReviewID               *uuid.UUID `gorm:"column:review_id"`
	ReviewRating           *float64   `gorm:"column:review_rating"`
	ReviewTitle            *string    `gorm:"column:review_title"`
	ReviewBody             *string    `gorm:"column:review_body"`
	ReviewContainsSpoilers *bool      `gorm:"column:review_contains_spoilers"`
	ReviewCreatedAt        *time.Time `gorm:"column:review_created_at"`
	ReviewUpdatedAt        *time.Time `gorm:"column:review_updated_at"`
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

func mapFollowingActivityRows(rows []followingActivityRow) []FollowingActivityRecord {
	records := make([]FollowingActivityRecord, 0, len(rows))
	for _, row := range rows {
		record := FollowingActivityRecord{
			ActivityType:     strings.TrimSpace(row.ActivityType),
			ActivityAt:       row.ActivityAt,
			ActivityID:       row.ActivityID.String(),
			ActorID:          row.ActorID.String(),
			ActorDisplayName: row.ActorDisplayName,
			ActorBio:         row.ActorBio,
			City:             row.City,
			PosterMediaID:    nullableUUIDToString(row.PosterMediaID),
			AverageRating:    row.AverageRating,
			ReviewCount:      row.ReviewCount,
		}

		if row.PlayID != nil {
			record.PlayID = row.PlayID.String()
		}
		if row.PlayTitle != nil {
			record.PlayTitle = *row.PlayTitle
		}
		if row.TheaterName != nil {
			record.TheaterName = *row.TheaterName
		}
		if row.AvailabilityStatus != nil {
			record.AvailabilityStatus = *row.AvailabilityStatus
		}
		if row.PublishedAt != nil {
			record.PublishedAt = *row.PublishedAt
		}

		if row.ReviewID != nil {
			reviewID := row.ReviewID.String()
			record.ReviewID = &reviewID
		}

		if row.ReviewRating != nil {
			rating := *row.ReviewRating
			record.Rating = &rating
		}

		record.Title = row.ReviewTitle
		record.Body = row.ReviewBody
		record.ContainsSpoilers = row.ReviewContainsSpoilers
		record.ReviewCreatedAt = row.ReviewCreatedAt
		record.ReviewUpdatedAt = row.ReviewUpdatedAt

		records = append(records, record)
	}

	return records
}

func nullableUUIDToString(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}

	encoded := value.String()
	return &encoded
}

func parseUUID(raw string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse uuid: %w", err)
	}

	return parsed, nil
}
