package plays

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FeedListParams struct {
	Section       string
	GenreID       *string
	After         *playListCursor
	TrendingAfter *trendingFeedCursor
	Limit         int
}

type SearchListParams struct {
	Q                  string
	GenreID            *string
	City               *string
	Theater            *string
	AvailabilityStatus *string
	After              *playListCursor
	Limit              int
}

type ListReviewsParams struct {
	PlayID string
	After  *reviewListCursor
	Limit  int
}

type ListUserEngagementPlaysParams struct {
	Kind  string
	After *engagementPlayListCursor
	Limit int
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListFeed(ctx context.Context, params FeedListParams) ([]PlayListRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	query := r.basePlayListQuery(ctx)
	trendScoreSQL := `(
		COALESCE(trending_attended.recent_attended_count, 0) * 4 +
		COALESCE(trending_reviews.recent_review_count, 0) * 3 +
		COALESCE(trending_wishlist.recent_wishlist_count, 0) * 2
	)`

	switch params.Section {
	case "highlighted":
		query = query.Order("COALESCE(prs.review_count, 0) DESC")
	case "trending":
		now := time.Now().UTC()
		trendingSince := now.Add(-14 * 24 * time.Hour)

		query = query.
			Where("p.published_at >= ?", now.Add(-30*24*time.Hour)).
			Select(`
				p.id,
				p.title,
				p.theater_name,
				p.city,
				p.availability_status,
				p.published_at,
				prs.avg_rating,
				COALESCE(prs.review_count, 0) AS review_count,
				`+trendScoreSQL+` AS trend_score,
				media.poster_media_id
			`).
			Joins(`
				LEFT JOIN (
					SELECT r.play_id, COUNT(*)::bigint AS recent_review_count
					FROM app.reviews AS r
					WHERE r.status = 'published' AND r.created_at >= ?
					GROUP BY r.play_id
				) AS trending_reviews ON trending_reviews.play_id = p.id
			`, trendingSince).
			Joins(`
				LEFT JOIN (
					SELECT e.play_id, COUNT(*)::bigint AS recent_wishlist_count
					FROM app.user_play_engagements AS e
					WHERE e.kind = 'wishlist' AND e.created_at >= ?
					GROUP BY e.play_id
				) AS trending_wishlist ON trending_wishlist.play_id = p.id
			`, trendingSince).
			Joins(`
				LEFT JOIN (
					SELECT e.play_id, COUNT(*)::bigint AS recent_attended_count
					FROM app.user_play_engagements AS e
					WHERE e.kind = 'attended' AND e.created_at >= ?
					GROUP BY e.play_id
				) AS trending_attended ON trending_attended.play_id = p.id
			`, trendingSince)

		if params.GenreID != nil {
			genreUUID, err := parseUUID(*params.GenreID)
			if err != nil {
				return nil, err
			}

			query = query.
				Joins("JOIN app.play_genres AS pg_filter ON pg_filter.play_id = p.id").
				Where("pg_filter.genre_id = ?", genreUUID)
		}
	case "genre":
		if params.GenreID != nil {
			genreUUID, err := parseUUID(*params.GenreID)
			if err != nil {
				return nil, err
			}

			query = query.
				Joins("JOIN app.play_genres AS pg_filter ON pg_filter.play_id = p.id").
				Where("pg_filter.genre_id = ?", genreUUID)
		}
	}

	var err error
	if params.Section == "trending" {
		query, err = applyTrendingFeedCursor(query, params.TrendingAfter, trendScoreSQL)
		if err != nil {
			return nil, err
		}
	} else {
		query, err = applyPlayListCursor(query, params.After)
		if err != nil {
			return nil, err
		}
	}

	var rows []playListRow
	if params.Section == "trending" {
		err = query.
			Order("trend_score DESC").
			Order("p.published_at DESC").
			Order("p.id DESC").
			Limit(params.Limit).
			Scan(&rows).Error
	} else {
		err = query.
			Order("p.published_at DESC").
			Order("p.id DESC").
			Limit(params.Limit).
			Scan(&rows).Error
	}
	if err != nil {
		return nil, err
	}

	return mapPlayListRows(rows), nil
}

func (r *Repository) SearchPublishedPlays(ctx context.Context, params SearchListParams) ([]PlayListRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	query := r.basePlayListQuery(ctx)

	if params.Q != "" {
		q := "%" + escapeLike(params.Q) + "%"
		query = query.Where("(p.title ILIKE ? OR p.synopsis ILIKE ?)", q, q)
	}

	if params.GenreID != nil {
		genreUUID, err := parseUUID(*params.GenreID)
		if err != nil {
			return nil, err
		}

		query = query.
			Joins("JOIN app.play_genres AS pg_filter ON pg_filter.play_id = p.id").
			Where("pg_filter.genre_id = ?", genreUUID)
	}

	if params.City != nil {
		query = query.Where("p.city ILIKE ?", "%"+escapeLike(*params.City)+"%")
	}

	if params.Theater != nil {
		query = query.Where("p.theater_name ILIKE ?", "%"+escapeLike(*params.Theater)+"%")
	}

	if params.AvailabilityStatus != nil {
		query = query.Where("p.availability_status = ?", *params.AvailabilityStatus)
	}

	query, err := applyPlayListCursor(query, params.After)
	if err != nil {
		return nil, err
	}

	var rows []playListRow
	err = query.
		Order("p.published_at DESC").
		Order("p.id DESC").
		Limit(params.Limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	return mapPlayListRows(rows), nil
}

func (r *Repository) GetPublishedPlayByID(ctx context.Context, playID string) (PlayDetailsRecord, error) {
	if err := r.ensureDB(); err != nil {
		return PlayDetailsRecord{}, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return PlayDetailsRecord{}, err
	}

	var row playDetailsRow
	err = r.db.WithContext(ctx).
		Table("app.plays AS p").
		Select(`
			p.id,
			p.title,
			p.synopsis,
			p.director,
			p.duration_minutes,
			p.theater_name,
			p.is_custom_theater,
			p.custom_genre_name,
			p.production,
			p.city,
			p.availability_status,
			p.published_at,
			prs.avg_rating,
			COALESCE(prs.review_count, 0) AS review_count
		`).
		Joins("LEFT JOIN app.play_rating_stats AS prs ON prs.play_id = p.id").
		Where("p.id = ? AND p.curation_status = ?", playUUID, "published").
		Take(&row).Error
	if err != nil {
		return PlayDetailsRecord{}, err
	}

	return PlayDetailsRecord{
		ID:                 row.ID.String(),
		Title:              row.Title,
		Synopsis:           row.Synopsis,
		Director:           row.Director,
		DurationMinutes:    row.DurationMinutes,
		TheaterName:        row.TheaterName,
		IsCustomTheater:    row.IsCustomTheater,
		City:               row.City,
		AvailabilityStatus: row.AvailabilityStatus,
		PublishedAt:        row.PublishedAt,
		AverageRating:      row.AverageRating,
		ReviewCount:        row.ReviewCount,
		CustomGenreName:    row.CustomGenreName,
		Production:         row.Production,
	}, nil
}

func (r *Repository) ListPlayGenres(ctx context.Context, playID string) ([]PlayGenreRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return nil, err
	}

	var rows []playGenreRow
	err = r.db.WithContext(ctx).
		Table("app.play_genres AS pg").
		Select("g.id, g.name").
		Joins("JOIN app.genres AS g ON g.id = pg.genre_id").
		Where("pg.play_id = ?", playUUID).
		Order("g.name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	genres := make([]PlayGenreRecord, 0, len(rows))
	for _, row := range rows {
		genres = append(genres, PlayGenreRecord{ID: row.ID.String(), Name: row.Name})
	}

	return genres, nil
}

func (r *Repository) ListPlayCast(ctx context.Context, playID string) ([]PlayCastRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return nil, err
	}

	var rows []playCastRow
	err = r.db.WithContext(ctx).
		Table("app.play_cast_members").
		Select("person_name, role_name, billing_order").
		Where("play_id = ?", playUUID).
		Order("billing_order ASC").
		Order("person_name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	cast := make([]PlayCastRecord, 0, len(rows))
	for _, row := range rows {
		cast = append(cast, PlayCastRecord{PersonName: row.PersonName, RoleName: row.RoleName, BillingOrder: row.BillingOrder})
	}

	return cast, nil
}

func (r *Repository) ListPlayMedia(ctx context.Context, playID string) ([]PlayMediaRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return nil, err
	}

	var rows []playMediaRow
	err = r.db.WithContext(ctx).
		Table("app.play_media").
		Select("id, kind, object_key, alt_text, sort_order, variants, blurhash").
		Where("play_id = ?", playUUID).
		Order("sort_order ASC").
		Order("created_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	media := make([]PlayMediaRecord, 0, len(rows))
	for _, row := range rows {
		media = append(media, PlayMediaRecord{
			ID:        row.ID.String(),
			Kind:      row.Kind,
			ObjectKey: row.ObjectKey,
			AltText:   row.AltText,
			SortOrder: row.SortOrder,
			PlayID:    playID,
			Variants:  decodeMediaVariants(row.Variants),
			Blurhash:  row.Blurhash,
		})
	}

	return media, nil
}

func (r *Repository) IsPlayPublished(ctx context.Context, playID string) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return false, err
	}

	var count int64
	err = r.db.WithContext(ctx).
		Table("app.plays").
		Where("id = ? AND curation_status = ?", playUUID, "published").
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
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
	err = r.db.WithContext(ctx).
		Table("app.users").
		Where("id = ?", userUUID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *Repository) ListPublishedReviews(ctx context.Context, params ListReviewsParams) ([]ReviewRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	playUUID, err := parseUUID(params.PlayID)
	if err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).
		Table("app.reviews AS r").
		Select(`
			r.id,
			r.user_id,
			u.display_name,
			r.rating,
			r.title,
			r.body,
			r.contains_spoilers,
			r.created_at,
			r.updated_at
		`).
		Joins("JOIN app.users AS u ON u.id = r.user_id").
		Where("r.play_id = ? AND r.status = ?", playUUID, "published")

	query, err = applyReviewListCursor(query, params.After, "r")
	if err != nil {
		return nil, err
	}

	var rows []reviewRow
	err = query.
		Order("r.created_at DESC").
		Order("r.id DESC").
		Limit(params.Limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	records := make([]ReviewRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, ReviewRecord{
			ID:               row.ID.String(),
			UserID:           row.UserID.String(),
			DisplayName:      row.DisplayName,
			Rating:           row.Rating,
			Title:            row.Title,
			Body:             row.Body,
			ContainsSpoilers: row.ContainsSpoilers,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		})
	}

	return records, nil
}

func (r *Repository) ListUserPublishedReviews(ctx context.Context, userID string, params ListUserReviewsParams) ([]UserReviewRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).
		Table("app.reviews AS r").
		Select(`
			r.id,
			r.play_id,
			p.title AS play_title,
			p.theater_name,
			p.city,
			p.availability_status,
			p.published_at,
			media.poster_media_id,
			r.rating,
			r.title,
			r.body,
			r.contains_spoilers,
			r.created_at,
			r.updated_at
		`).
		Joins("JOIN app.plays AS p ON p.id = r.play_id").
		Joins(`
			LEFT JOIN LATERAL (
				SELECT pm.id AS poster_media_id
				FROM app.play_media AS pm
				WHERE pm.play_id = p.id
				ORDER BY CASE WHEN pm.kind = 'poster' THEN 0 ELSE 1 END, pm.sort_order ASC, pm.created_at ASC
				LIMIT 1
			) AS media ON true
		`).
		Where("r.user_id = ? AND r.status = ? AND p.curation_status = ?", userUUID, "published", "published")

	query, err = applyReviewListCursor(query, params.After, "r")
	if err != nil {
		return nil, err
	}

	var rows []userReviewRow
	err = query.
		Order("r.created_at DESC").
		Order("r.id DESC").
		Limit(params.Limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	records := make([]UserReviewRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, UserReviewRecord{
			ID:                 row.ID.String(),
			PlayID:             row.PlayID.String(),
			PlayTitle:          row.PlayTitle,
			TheaterName:        row.TheaterName,
			City:               row.City,
			AvailabilityStatus: row.AvailabilityStatus,
			PublishedAt:        row.PublishedAt,
			PosterMediaID:      nullableUUIDToString(row.PosterMediaID),
			Rating:             row.Rating,
			Title:              row.Title,
			Body:               row.Body,
			ContainsSpoilers:   row.ContainsSpoilers,
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
		})
	}

	return records, nil
}

func (r *Repository) CreateReview(ctx context.Context, userID string, playID string, params CreateReviewParams) (ReviewRecord, error) {
	if err := r.ensureDB(); err != nil {
		return ReviewRecord{}, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return ReviewRecord{}, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return ReviewRecord{}, err
	}

	entity := reviewEntity{
		PlayID:           playUUID,
		UserID:           userUUID,
		Rating:           params.Rating,
		Title:            params.Title,
		Body:             params.Body,
		ContainsSpoilers: params.ContainsSpoilers,
		Status:           "published",
		CreatedAt:        params.CreatedAt,
		UpdatedAt:        params.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&entity).Error; err != nil {
		return ReviewRecord{}, err
	}

	return r.getReviewByID(ctx, entity.ID)
}

func (r *Repository) GetReviewMetadata(ctx context.Context, reviewID string) (ReviewMetadataRecord, error) {
	if err := r.ensureDB(); err != nil {
		return ReviewMetadataRecord{}, err
	}

	reviewUUID, err := parseUUID(reviewID)
	if err != nil {
		return ReviewMetadataRecord{}, err
	}

	var row reviewMetadataRow
	err = r.db.WithContext(ctx).
		Table("app.reviews AS r").
		Select("r.id, r.play_id, r.user_id, r.status, p.curation_status").
		Joins("JOIN app.plays AS p ON p.id = r.play_id").
		Where("r.id = ?", reviewUUID).
		Take(&row).Error
	if err != nil {
		return ReviewMetadataRecord{}, err
	}

	return ReviewMetadataRecord{
		ReviewID:           row.ReviewID.String(),
		PlayID:             row.PlayID.String(),
		UserID:             row.UserID.String(),
		ReviewStatus:       row.ReviewStatus,
		PlayCurationStatus: row.PlayCurationStatus,
	}, nil
}

func (r *Repository) UpdateReview(ctx context.Context, reviewID string, params UpdateReviewParams) (ReviewRecord, error) {
	if err := r.ensureDB(); err != nil {
		return ReviewRecord{}, err
	}

	reviewUUID, err := parseUUID(reviewID)
	if err != nil {
		return ReviewRecord{}, err
	}

	updates := map[string]any{
		"updated_at": params.UpdatedAt,
	}

	if params.Rating != nil {
		updates["rating"] = *params.Rating
	}

	if params.TitleProvided {
		updates["title"] = params.Title
	}

	if params.Body != nil {
		updates["body"] = *params.Body
	}

	if params.ContainsSpoilers != nil {
		updates["contains_spoilers"] = *params.ContainsSpoilers
	}

	err = r.db.WithContext(ctx).
		Model(&reviewEntity{}).
		Where("id = ?", reviewUUID).
		Updates(updates).Error
	if err != nil {
		return ReviewRecord{}, err
	}

	return r.getReviewByID(ctx, reviewUUID)
}

func (r *Repository) CreateReviewComment(ctx context.Context, userID string, reviewID string, params CreateReviewCommentParams) (ReviewCommentRecord, error) {
	if err := r.ensureDB(); err != nil {
		return ReviewCommentRecord{}, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return ReviewCommentRecord{}, err
	}

	reviewUUID, err := parseUUID(reviewID)
	if err != nil {
		return ReviewCommentRecord{}, err
	}

	entity := reviewCommentEntity{
		ReviewID:  reviewUUID,
		UserID:    userUUID,
		Body:      params.Body,
		Status:    "published",
		CreatedAt: params.CreatedAt,
		UpdatedAt: params.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&entity).Error; err != nil {
		return ReviewCommentRecord{}, err
	}

	var row reviewCommentRow
	err = r.db.WithContext(ctx).
		Table("app.review_comments AS rc").
		Select(`
			rc.id,
			rc.review_id,
			rc.user_id,
			u.display_name,
			rc.body,
			rc.created_at,
			rc.updated_at
		`).
		Joins("JOIN app.users AS u ON u.id = rc.user_id").
		Where("rc.id = ?", entity.ID).
		Take(&row).Error
	if err != nil {
		return ReviewCommentRecord{}, err
	}

	return ReviewCommentRecord{
		ID:          row.ID.String(),
		ReviewID:    row.ReviewID.String(),
		UserID:      row.UserID.String(),
		DisplayName: row.DisplayName,
		Body:        row.Body,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *Repository) UpdateReviewCommentStatus(ctx context.Context, commentID string, status string, updatedAt time.Time) (ReviewCommentStatusRecord, error) {
	if err := r.ensureDB(); err != nil {
		return ReviewCommentStatusRecord{}, err
	}

	commentUUID, err := parseUUID(commentID)
	if err != nil {
		return ReviewCommentStatusRecord{}, err
	}

	result := r.db.WithContext(ctx).
		Model(&reviewCommentEntity{}).
		Where("id = ?", commentUUID).
		Updates(map[string]any{"status": status, "updated_at": updatedAt})
	if result.Error != nil {
		return ReviewCommentStatusRecord{}, result.Error
	}

	if result.RowsAffected == 0 {
		return ReviewCommentStatusRecord{}, gorm.ErrRecordNotFound
	}

	var row reviewCommentStatusRow
	err = r.db.WithContext(ctx).
		Table("app.review_comments").
		Select("id, review_id, status, updated_at").
		Where("id = ?", commentUUID).
		Take(&row).Error
	if err != nil {
		return ReviewCommentStatusRecord{}, err
	}

	return ReviewCommentStatusRecord{
		ID:        row.ID.String(),
		ReviewID:  row.ReviewID.String(),
		Status:    row.Status,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *Repository) SetEngagement(ctx context.Context, userID string, playID string, kind string, createdAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		switch kind {
		case "attended":
			engagement := userPlayEngagementEntity{
				UserID:    userUUID,
				PlayID:    playUUID,
				Kind:      kind,
				CreatedAt: createdAt,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&engagement).Error; err != nil {
				return err
			}

			if err := tx.Where("user_id = ? AND play_id = ? AND kind = ?", userUUID, playUUID, "wishlist").Delete(&userPlayEngagementEntity{}).Error; err != nil {
				return err
			}
		case "wishlist":
			var attendedCount int64
			if err := tx.Model(&userPlayEngagementEntity{}).Where("user_id = ? AND play_id = ? AND kind = ?", userUUID, playUUID, "attended").Count(&attendedCount).Error; err != nil {
				return err
			}

			if attendedCount > 0 {
				return nil
			}

			engagement := userPlayEngagementEntity{
				UserID:    userUUID,
				PlayID:    playUUID,
				Kind:      kind,
				CreatedAt: createdAt,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&engagement).Error; err != nil {
				return err
			}
		case "favorited":
			engagement := userPlayEngagementEntity{
				UserID:    userUUID,
				PlayID:    playUUID,
				Kind:      kind,
				CreatedAt: createdAt,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&engagement).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *Repository) DeleteEngagement(ctx context.Context, userID string, playID string, kind string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Where("user_id = ? AND play_id = ? AND kind = ?", userUUID, playUUID, kind).
		Delete(&userPlayEngagementEntity{}).Error
}

func (r *Repository) GetEngagementState(ctx context.Context, userID string, playID string) (EngagementStateRecord, error) {
	if err := r.ensureDB(); err != nil {
		return EngagementStateRecord{}, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return EngagementStateRecord{}, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return EngagementStateRecord{}, err
	}

	var wishlistCount int64
	err = r.db.WithContext(ctx).
		Model(&userPlayEngagementEntity{}).
		Where("user_id = ? AND play_id = ? AND kind = ?", userUUID, playUUID, "wishlist").
		Count(&wishlistCount).Error
	if err != nil {
		return EngagementStateRecord{}, err
	}

	var attendedCount int64
	err = r.db.WithContext(ctx).
		Model(&userPlayEngagementEntity{}).
		Where("user_id = ? AND play_id = ? AND kind = ?", userUUID, playUUID, "attended").
		Count(&attendedCount).Error
	if err != nil {
		return EngagementStateRecord{}, err
	}

	var favoritedCount int64
	err = r.db.WithContext(ctx).
		Model(&userPlayEngagementEntity{}).
		Where("user_id = ? AND play_id = ? AND kind = ?", userUUID, playUUID, "favorited").
		Count(&favoritedCount).Error
	if err != nil {
		return EngagementStateRecord{}, err
	}

	return EngagementStateRecord{Wishlist: wishlistCount > 0, Attended: attendedCount > 0, Favorited: favoritedCount > 0}, nil
}

func (r *Repository) ListUserEngagementPlays(ctx context.Context, userID string, params ListUserEngagementPlaysParams) ([]EngagementPlayRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).
		Table("app.user_play_engagements AS e").
		Select(`
			p.id,
			p.title,
			p.theater_name,
			p.city,
			p.availability_status,
			p.published_at,
			prs.avg_rating,
			COALESCE(prs.review_count, 0) AS review_count,
			media.poster_media_id,
			e.created_at AS engaged_at
		`).
		Joins("JOIN app.plays AS p ON p.id = e.play_id").
		Joins("LEFT JOIN app.play_rating_stats AS prs ON prs.play_id = p.id").
		Joins(`
			LEFT JOIN LATERAL (
				SELECT pm.id AS poster_media_id
				FROM app.play_media AS pm
				WHERE pm.play_id = p.id
				ORDER BY CASE WHEN pm.kind = 'poster' THEN 0 ELSE 1 END, pm.sort_order ASC, pm.created_at ASC
				LIMIT 1
			) AS media ON true
		`).
		Where("e.user_id = ? AND e.kind = ? AND p.curation_status = ?", userUUID, params.Kind, "published")

	query, err = applyEngagementPlayListCursor(query, params.After)
	if err != nil {
		return nil, err
	}

	var rows []engagementPlayRow
	err = query.
		Order("e.created_at DESC").
		Order("p.id DESC").
		Limit(params.Limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	records := make([]EngagementPlayRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, EngagementPlayRecord{
			ID:                 row.ID.String(),
			Title:              row.Title,
			TheaterName:        row.TheaterName,
			City:               row.City,
			AvailabilityStatus: row.AvailabilityStatus,
			PublishedAt:        row.PublishedAt,
			PosterMediaID:      nullableUUIDToString(row.PosterMediaID),
			AverageRating:      row.AverageRating,
			ReviewCount:        row.ReviewCount,
			EngagedAt:          row.EngagedAt,
		})
	}

	return records, nil
}

func (r *Repository) CreateSubmission(ctx context.Context, userID string, params CreateSubmissionParams) (SubmissionRecord, error) {
	if err := r.ensureDB(); err != nil {
		return SubmissionRecord{}, err
	}

	creatorUUID, err := parseUUID(userID)
	if err != nil {
		return SubmissionRecord{}, err
	}

	entity := playEntity{
		Title:              params.Title,
		Synopsis:           params.Synopsis,
		Director:           params.Director,
		DurationMinutes:    params.DurationMinutes,
		TheaterName:        params.TheaterName,
		IsCustomTheater:    params.IsCustomTheater,
		CustomGenreName:    params.CustomGenreName,
		Production:         params.Production,
		City:               params.City,
		AvailabilityStatus: params.AvailabilityStatus,
		CurationStatus:     "pending",
		CreatedByUserID:    creatorUUID,
		CreatedAt:          params.CreatedAt,
		UpdatedAt:          params.UpdatedAt,
	}

	genreEntities := make([]playGenreEntity, 0, len(params.GenreIDs))
	for _, genreID := range params.GenreIDs {
		genreUUID, parseErr := parseUUID(genreID)
		if parseErr != nil {
			return SubmissionRecord{}, parseErr
		}

		genreEntities = append(genreEntities, playGenreEntity{GenreID: genreUUID})
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&entity).Error; err != nil {
			return err
		}

		for idx := range genreEntities {
			genreEntities[idx].PlayID = entity.ID
		}

		if len(genreEntities) > 0 {
			if err := tx.Create(&genreEntities).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return SubmissionRecord{}, err
	}

	return r.getSubmissionByUUID(ctx, entity.ID)
}

func (r *Repository) CountGenresByIDs(ctx context.Context, genreIDs []string) (int64, error) {
	if err := r.ensureDB(); err != nil {
		return 0, err
	}

	if len(genreIDs) == 0 {
		return 0, nil
	}

	parsedIDs := make([]uuid.UUID, 0, len(genreIDs))
	for _, genreID := range genreIDs {
		genreUUID, err := parseUUID(genreID)
		if err != nil {
			return 0, err
		}

		parsedIDs = append(parsedIDs, genreUUID)
	}

	var count int64
	err := r.db.WithContext(ctx).
		Table("app.genres").
		Where("id IN ?", parsedIDs).
		Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repository) GetSubmissionByID(ctx context.Context, playID string) (SubmissionRecord, error) {
	if err := r.ensureDB(); err != nil {
		return SubmissionRecord{}, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return SubmissionRecord{}, err
	}

	return r.getSubmissionByUUID(ctx, playUUID)
}

func (r *Repository) GetOwnedPublishedPlayByID(ctx context.Context, playID string, userID string) (OwnedPlayRecord, error) {
	if err := r.ensureDB(); err != nil {
		return OwnedPlayRecord{}, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return OwnedPlayRecord{}, err
	}

	ownerUUID, err := parseUUID(userID)
	if err != nil {
		return OwnedPlayRecord{}, err
	}

	var row ownedPlayRow
	err = r.db.WithContext(ctx).
		Table("app.plays AS p").
		Select(`
			p.id,
			p.title,
			p.theater_name,
			p.city,
			p.availability_status,
			p.published_at,
			p.created_at
		`).
		Where("p.id = ? AND p.created_by_user_id = ? AND p.curation_status = ?", playUUID, ownerUUID, "published").
		Take(&row).Error
	if err != nil {
		return OwnedPlayRecord{}, err
	}

	return OwnedPlayRecord{
		ID:                 row.ID.String(),
		Title:              row.Title,
		TheaterName:        row.TheaterName,
		City:               row.City,
		AvailabilityStatus: row.AvailabilityStatus,
		PublishedAt:        row.PublishedAt,
		CreatedAt:          row.CreatedAt,
	}, nil
}

func (r *Repository) ListGenres(ctx context.Context, params ListGenresParams) ([]GenreRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).
		Table("app.genres AS g").
		Select(`
			g.id,
			g.name
		`)

	query, err := applyNameIDListCursor(query, params.After, "g.name", "g.id")
	if err != nil {
		return nil, err
	}

	var rows []genreRow
	err = query.
		Order("g.name ASC").
		Order("g.id ASC").
		Limit(params.Limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	return mapGenreRows(rows), nil
}

func (r *Repository) CreateGenre(ctx context.Context, name string) (GenreRecord, error) {
	if err := r.ensureDB(); err != nil {
		return GenreRecord{}, err
	}

	entity := genreEntity{Name: name, CreatedAt: time.Now().UTC()}
	if err := r.db.WithContext(ctx).Create(&entity).Error; err != nil {
		return GenreRecord{}, err
	}

	return GenreRecord{ID: entity.ID.String(), Name: entity.Name}, nil
}

func (r *Repository) DeleteGenre(ctx context.Context, genreID string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	genreUUID, err := parseUUID(genreID)
	if err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Where("id = ?", genreUUID).Delete(&genreEntity{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *Repository) ListCities(ctx context.Context, params ListCitiesParams) ([]CityRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).
		Table("app.cities AS c").
		Select("c.id, c.name").
		Where("c.is_active = true")

	query, err := applyNameIDListCursor(query, params.After, "c.name", "c.id")
	if err != nil {
		return nil, err
	}

	var rows []genreRow
	if err := query.Order("c.name ASC").Order("c.id ASC").Limit(params.Limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	records := make([]CityRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, CityRecord{ID: row.ID.String(), Name: row.Name})
	}

	return records, nil
}

func (r *Repository) ListTheaters(ctx context.Context, params ListTheatersParams) ([]TheaterRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).
		Table("app.theaters AS t").
		Select("t.id, t.city_id, t.name").
		Where("t.is_active = true")

	if params.CityID != nil {
		cityUUID, err := parseUUID(*params.CityID)
		if err != nil {
			return nil, err
		}
		query = query.Where("t.city_id = ?", cityUUID)
	}

	query, err := applyNameIDListCursor(query, params.After, "t.name", "t.id")
	if err != nil {
		return nil, err
	}

	var rows []theaterRow
	if err := query.Order("t.name ASC").Order("t.id ASC").Limit(params.Limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	records := make([]TheaterRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, TheaterRecord{ID: row.ID.String(), CityID: row.CityID.String(), Name: row.Name})
	}

	return records, nil
}

func (r *Repository) CityExists(ctx context.Context, cityName string) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}

	var count int64
	err := r.db.WithContext(ctx).Table("app.cities").Where("LOWER(name) = LOWER(?) AND is_active = true", cityName).Count(&count).Error
	return count > 0, err
}

func (r *Repository) TheaterExistsInCity(ctx context.Context, cityName string, theaterName string) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}

	var count int64
	err := r.db.WithContext(ctx).
		Table("app.theaters AS t").
		Joins("JOIN app.cities AS c ON c.id = t.city_id").
		Where("LOWER(c.name) = LOWER(?) AND LOWER(t.name) = LOWER(?) AND c.is_active = true AND t.is_active = true", cityName, theaterName).
		Count(&count).Error
	return count > 0, err
}

func (r *Repository) CreateCity(ctx context.Context, name string) (CityRecord, error) {
	if err := r.ensureDB(); err != nil {
		return CityRecord{}, err
	}

	entity := cityEntity{Name: name, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), IsActive: true}
	if err := r.db.WithContext(ctx).Create(&entity).Error; err != nil {
		return CityRecord{}, err
	}

	return CityRecord{ID: entity.ID.String(), Name: entity.Name}, nil
}

func (r *Repository) DeleteCity(ctx context.Context, cityID string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	cityUUID, err := parseUUID(cityID)
	if err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Model(&cityEntity{}).Where("id = ?", cityUUID).Updates(map[string]any{"is_active": false, "updated_at": time.Now().UTC()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *Repository) CreateTheater(ctx context.Context, cityID string, name string) (TheaterRecord, error) {
	if err := r.ensureDB(); err != nil {
		return TheaterRecord{}, err
	}

	cityUUID, err := parseUUID(cityID)
	if err != nil {
		return TheaterRecord{}, err
	}

	entity := theaterEntity{CityID: cityUUID, Name: name, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), IsActive: true}
	if err := r.db.WithContext(ctx).Create(&entity).Error; err != nil {
		return TheaterRecord{}, err
	}

	return TheaterRecord{ID: entity.ID.String(), CityID: entity.CityID.String(), Name: entity.Name}, nil
}

func (r *Repository) DeleteTheater(ctx context.Context, theaterID string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	theaterUUID, err := parseUUID(theaterID)
	if err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Model(&theaterEntity{}).Where("id = ?", theaterUUID).Updates(map[string]any{"is_active": false, "updated_at": time.Now().UTC()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *Repository) DeletePlay(ctx context.Context, playID string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Where("id = ?", playUUID).Delete(&playEntity{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *Repository) ListUserSubmissions(ctx context.Context, userID string, params ListSubmissionsParams) ([]SubmissionRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	creatorUUID, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).
		Table("app.plays AS p").
		Select(`
			p.id,
			p.title,
			p.synopsis,
			p.director,
			p.duration_minutes,
			p.theater_name,
			p.is_custom_theater,
			p.custom_genre_name,
			p.production,
			p.city,
			p.availability_status,
			p.curation_status,
			p.created_by_user_id,
			p.moderated_by_user_id,
			p.moderated_at,
			p.published_at,
			p.rejected_reason,
			p.created_at,
			p.updated_at
		`).
		Where("p.created_by_user_id = ?", creatorUUID)

	if params.Status != nil {
		query = query.Where("p.curation_status = ?", *params.Status)
	}

	query, err = applySubmissionListCursor(query, params.After)
	if err != nil {
		return nil, err
	}

	var rows []submissionRow
	err = query.
		Order("p.created_at DESC").
		Order("p.id DESC").
		Limit(params.Limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	return mapSubmissionRows(rows), nil
}

func (r *Repository) ListOwnedPublishedPlays(ctx context.Context, userID string, params ListSubmissionsParams) ([]OwnedPlayRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	ownerUUID, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).
		Table("app.plays AS p").
		Select(`
			p.id,
			p.title,
			p.theater_name,
			p.city,
			p.availability_status,
			p.published_at,
			p.created_at,
			media.poster_media_id
		`).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT pm.id AS poster_media_id
				FROM app.play_media AS pm
				WHERE pm.play_id = p.id
				ORDER BY CASE WHEN pm.kind = 'poster' THEN 0 ELSE 1 END, pm.sort_order ASC, pm.created_at ASC
				LIMIT 1
			) AS media ON true
		`).
		Where("p.created_by_user_id = ? AND p.curation_status = ?", ownerUUID, "published")

	query, err = applySubmissionListCursor(query, params.After)
	if err != nil {
		return nil, err
	}

	var rows []ownedPlayRow
	if err := query.
		Order("p.created_at DESC").
		Order("p.id DESC").
		Limit(params.Limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	records := make([]OwnedPlayRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, OwnedPlayRecord{
			ID:                 row.ID.String(),
			Title:              row.Title,
			TheaterName:        row.TheaterName,
			City:               row.City,
			AvailabilityStatus: row.AvailabilityStatus,
			PublishedAt:        row.PublishedAt,
			PosterMediaID:      nullableUUIDToString(row.PosterMediaID),
			CreatedAt:          row.CreatedAt,
		})
	}

	return records, nil
}

func (r *Repository) ListAdminSubmissions(ctx context.Context, params ListSubmissionsParams) ([]SubmissionRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).
		Table("app.plays AS p").
		Select(`
			p.id,
			p.title,
			p.synopsis,
			p.director,
			p.duration_minutes,
			p.theater_name,
			p.is_custom_theater,
			p.custom_genre_name,
			p.production,
			p.city,
			p.availability_status,
			p.curation_status,
			p.created_by_user_id,
			p.moderated_by_user_id,
			p.moderated_at,
			p.published_at,
			p.rejected_reason,
			p.created_at,
			p.updated_at
		`)

	if params.Status != nil {
		query = query.Where("p.curation_status = ?", *params.Status)
	}

	query, err := applySubmissionListCursor(query, params.After)
	if err != nil {
		return nil, err
	}

	var rows []submissionRow
	err = query.
		Order("p.created_at DESC").
		Order("p.id DESC").
		Limit(params.Limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	return mapSubmissionRows(rows), nil
}

func (r *Repository) ListPlayEditSuggestions(ctx context.Context, params ListPlayEditSuggestionsParams) ([]PlayEditSuggestionRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).
		Table("app.play_edit_suggestions AS s").
		Select(`
			s.id,
			s.play_id,
			s.title,
			s.synopsis,
			s.director,
			s.duration_minutes,
			s.theater_name,
			s.city,
			s.availability_status,
			s.status,
			s.created_by_user_id,
			s.moderated_by_user_id,
			s.moderated_at,
			s.rejected_reason,
			s.created_at,
			s.updated_at
		`)

	if params.CreatedByUserID != nil {
		userUUID, err := parseUUID(*params.CreatedByUserID)
		if err != nil {
			return nil, err
		}
		query = query.Where("s.created_by_user_id = ?", userUUID)
	}

	if params.PlayID != nil {
		playUUID, err := parseUUID(*params.PlayID)
		if err != nil {
			return nil, err
		}
		query = query.Where("s.play_id = ?", playUUID)
	}

	if params.Status != nil {
		query = query.Where("s.status = ?", *params.Status)
	}

	query, err := applyPlayEditSuggestionListCursor(query, params.After)
	if err != nil {
		return nil, err
	}

	var rows []playEditSuggestionRow
	if err := query.
		Order("s.created_at DESC").
		Order("s.id DESC").
		Limit(params.Limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	return mapPlayEditSuggestionRows(rows), nil
}

func (r *Repository) GetPlayEditSuggestionByID(ctx context.Context, suggestionID string) (PlayEditSuggestionRecord, error) {
	if err := r.ensureDB(); err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	suggestionUUID, err := parseUUID(suggestionID)
	if err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	var row playEditSuggestionRow
	if err := r.db.WithContext(ctx).
		Table("app.play_edit_suggestions AS s").
		Select(`
			s.id,
			s.play_id,
			s.title,
			s.synopsis,
			s.director,
			s.duration_minutes,
			s.theater_name,
			s.city,
			s.availability_status,
			s.status,
			s.created_by_user_id,
			s.moderated_by_user_id,
			s.moderated_at,
			s.rejected_reason,
			s.created_at,
			s.updated_at
		`).
		Where("s.id = ?", suggestionUUID).
		Take(&row).Error; err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	records := mapPlayEditSuggestionRows([]playEditSuggestionRow{row})
	if len(records) == 0 {
		return PlayEditSuggestionRecord{}, gorm.ErrRecordNotFound
	}

	return records[0], nil
}

func (r *Repository) CreatePlayEditSuggestion(ctx context.Context, params CreatePlayEditSuggestionParams) (PlayEditSuggestionRecord, error) {
	if err := r.ensureDB(); err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	playUUID, err := parseUUID(params.PlayID)
	if err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	ownerUUID, err := parseUUID(params.CreatedByUserID)
	if err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	entity := playEditSuggestionEntity{
		PlayID:             playUUID,
		CreatedByUserID:    ownerUUID,
		Status:             "pending",
		Title:              params.Title,
		Synopsis:           params.Synopsis,
		Director:           params.Director,
		DurationMinutes:    params.DurationMinutes,
		TheaterName:        params.TheaterName,
		City:               params.City,
		AvailabilityStatus: params.AvailabilityStatus,
		CreatedAt:          params.CreatedAt,
		UpdatedAt:          params.UpdatedAt,
	}

	genreEntities := make([]playEditSuggestionGenreEntity, 0, len(params.GenreIDs))
	for _, genreID := range params.GenreIDs {
		genreUUID, parseErr := parseUUID(genreID)
		if parseErr != nil {
			return PlayEditSuggestionRecord{}, parseErr
		}
		genreEntities = append(genreEntities, playEditSuggestionGenreEntity{GenreID: genreUUID})
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&entity).Error; err != nil {
			return err
		}
		if len(genreEntities) > 0 {
			for idx := range genreEntities {
				genreEntities[idx].SuggestionID = entity.ID
			}
			if err := tx.Create(&genreEntities).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	return r.GetPlayEditSuggestionByID(ctx, entity.ID.String())
}

func (r *Repository) UpdatePlayEditSuggestion(ctx context.Context, suggestionID string, params UpdatePlayEditSuggestionParams) (PlayEditSuggestionRecord, error) {
	if err := r.ensureDB(); err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	suggestionUUID, err := parseUUID(suggestionID)
	if err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	updates := map[string]any{
		"updated_at": params.UpdatedAt,
	}

	if params.Title != nil {
		updates["title"] = *params.Title
	}
	if params.Synopsis != nil {
		updates["synopsis"] = *params.Synopsis
	}
	if params.Director != nil {
		updates["director"] = *params.Director
	}
	if params.DurationMinutes != nil {
		updates["duration_minutes"] = *params.DurationMinutes
	}
	if params.TheaterName != nil {
		updates["theater_name"] = *params.TheaterName
	}
	if params.CityProvided {
		updates["city"] = params.City
	}
	if params.AvailabilityStatus != nil {
		updates["availability_status"] = *params.AvailabilityStatus
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&playEditSuggestionEntity{}).
			Where("id = ? AND status = ?", suggestionUUID, "pending").
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		if params.GenreIDsProvided {
			if err := tx.Where("suggestion_id = ?", suggestionUUID).Delete(&playEditSuggestionGenreEntity{}).Error; err != nil {
				return err
			}
			if len(params.GenreIDs) > 0 {
				genreEntities := make([]playEditSuggestionGenreEntity, 0, len(params.GenreIDs))
				for _, genreID := range params.GenreIDs {
					genreUUID, parseErr := parseUUID(genreID)
					if parseErr != nil {
						return parseErr
					}
					genreEntities = append(genreEntities, playEditSuggestionGenreEntity{
						SuggestionID: suggestionUUID,
						GenreID:      genreUUID,
					})
				}
				if err := tx.Create(&genreEntities).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	return r.GetPlayEditSuggestionByID(ctx, suggestionID)
}

func (r *Repository) ModeratePlayEditSuggestion(ctx context.Context, suggestionID string, params ModeratePlayEditSuggestionParams) (PlayEditSuggestionRecord, error) {
	if err := r.ensureDB(); err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	suggestionUUID, err := parseUUID(suggestionID)
	if err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	adminUUID, err := parseUUID(params.AdminUserID)
	if err != nil {
		return PlayEditSuggestionRecord{}, err
	}

	updates := map[string]any{
		"status":               params.Status,
		"moderated_by_user_id": adminUUID,
		"moderated_at":         params.ModeratedAt,
		"updated_at":           params.ModeratedAt,
		"rejected_reason":      params.RejectedReason,
	}

	result := r.db.WithContext(ctx).
		Model(&playEditSuggestionEntity{}).
		Where("id = ? AND status = ?", suggestionUUID, "pending").
		Updates(updates)
	if result.Error != nil {
		return PlayEditSuggestionRecord{}, result.Error
	}
	if result.RowsAffected == 0 {
		return PlayEditSuggestionRecord{}, gorm.ErrRecordNotFound
	}

	return r.GetPlayEditSuggestionByID(ctx, suggestionID)
}

func (r *Repository) ReplacePlayEditSuggestionGenres(ctx context.Context, suggestionID string, genreIDs []string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	suggestionUUID, err := parseUUID(suggestionID)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("suggestion_id = ?", suggestionUUID).Delete(&playEditSuggestionGenreEntity{}).Error; err != nil {
			return err
		}
		if len(genreIDs) == 0 {
			return nil
		}
		entities := make([]playEditSuggestionGenreEntity, 0, len(genreIDs))
		for _, genreID := range genreIDs {
			genreUUID, parseErr := parseUUID(genreID)
			if parseErr != nil {
				return parseErr
			}
			entities = append(entities, playEditSuggestionGenreEntity{SuggestionID: suggestionUUID, GenreID: genreUUID})
		}
		return tx.Create(&entities).Error
	})
}

func (r *Repository) ListPlayEditSuggestionGenres(ctx context.Context, suggestionID string) ([]PlayGenreRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	suggestionUUID, err := parseUUID(suggestionID)
	if err != nil {
		return nil, err
	}

	var rows []playGenreRow
	err = r.db.WithContext(ctx).
		Table("app.play_edit_suggestion_genres AS sg").
		Select("g.id, g.name").
		Joins("JOIN app.genres AS g ON g.id = sg.genre_id").
		Where("sg.suggestion_id = ?", suggestionUUID).
		Order("g.name ASC").
		Order("g.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	records := make([]PlayGenreRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, PlayGenreRecord{ID: row.ID.String(), Name: row.Name})
	}
	return records, nil
}

func (r *Repository) ListPlayEditSuggestionMedia(ctx context.Context, suggestionID string) ([]PlayMediaRecord, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	suggestionUUID, err := parseUUID(suggestionID)
	if err != nil {
		return nil, err
	}

	var rows []playEditSuggestionMediaRow
	err = r.db.WithContext(ctx).
		Table("app.play_edit_suggestion_media AS sm").
		Select("sm.id, sm.kind, sm.object_key, sm.alt_text, sm.sort_order, sm.variants, sm.blurhash, sm.suggestion_id").
		Where("sm.suggestion_id = ?", suggestionUUID).
		Order("sm.sort_order ASC").
		Order("sm.created_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	records := make([]PlayMediaRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, PlayMediaRecord{
			ID:        row.ID.String(),
			Kind:      row.Kind,
			ObjectKey: row.ObjectKey,
			AltText:   row.AltText,
			SortOrder: row.SortOrder,
			Variants:  decodeMediaVariants(row.Variants),
			Blurhash:  row.Blurhash,
			PlayID:    row.SuggestionID.String(),
		})
	}
	return records, nil
}

func (r *Repository) CreatePlayEditSuggestionMedia(ctx context.Context, suggestionID string, params CreatePlayMediaParams) (PlayMediaRecord, error) {
	if err := r.ensureDB(); err != nil {
		return PlayMediaRecord{}, err
	}

	suggestionUUID, err := parseUUID(suggestionID)
	if err != nil {
		return PlayMediaRecord{}, err
	}

	variantsJSON, err := encodeMediaVariants(params.Variants)
	if err != nil {
		return PlayMediaRecord{}, err
	}

	entity := playEditSuggestionMediaEntity{
		SuggestionID: suggestionUUID,
		Kind:         params.Kind,
		ObjectKey:    params.ObjectKey,
		AltText:      params.AltText,
		SortOrder:    params.SortOrder,
		Variants:     variantsJSON,
		Blurhash:     params.Blurhash,
		CreatedAt:    params.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&entity).Error; err != nil {
		return PlayMediaRecord{}, err
	}

	return PlayMediaRecord{
		ID:        entity.ID.String(),
		Kind:      entity.Kind,
		ObjectKey: entity.ObjectKey,
		AltText:   entity.AltText,
		SortOrder: entity.SortOrder,
		Variants:  params.Variants,
		Blurhash:  params.Blurhash,
		PlayID:    suggestionID,
	}, nil
}

func (r *Repository) GetPlayEditSuggestionMediaByID(ctx context.Context, mediaID string) (PlayMediaRecord, error) {
	if err := r.ensureDB(); err != nil {
		return PlayMediaRecord{}, err
	}

	mediaUUID, err := parseUUID(mediaID)
	if err != nil {
		return PlayMediaRecord{}, err
	}

	var row playEditSuggestionMediaRow
	err = r.db.WithContext(ctx).
		Table("app.play_edit_suggestion_media AS sm").
		Select("sm.id, sm.kind, sm.object_key, sm.alt_text, sm.sort_order, sm.variants, sm.blurhash, sm.suggestion_id").
		Where("sm.id = ?", mediaUUID).
		Take(&row).Error
	if err != nil {
		return PlayMediaRecord{}, err
	}

	return PlayMediaRecord{
		ID:        row.ID.String(),
		Kind:      row.Kind,
		ObjectKey: row.ObjectKey,
		AltText:   row.AltText,
		SortOrder: row.SortOrder,
		Variants:  decodeMediaVariants(row.Variants),
		Blurhash:  row.Blurhash,
		PlayID:    row.SuggestionID.String(),
	}, nil
}

func (r *Repository) DeletePlayEditSuggestionMedia(ctx context.Context, mediaID string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	mediaUUID, err := parseUUID(mediaID)
	if err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Where("id = ?", mediaUUID).Delete(&playEditSuggestionMediaEntity{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *Repository) ReplacePlayEditSuggestionMedia(ctx context.Context, suggestionID string, media []CreatePlayMediaParams) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	suggestionUUID, err := parseUUID(suggestionID)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("suggestion_id = ?", suggestionUUID).Delete(&playEditSuggestionMediaEntity{}).Error; err != nil {
			return err
		}
		if len(media) == 0 {
			return nil
		}
		entities := make([]playEditSuggestionMediaEntity, 0, len(media))
		for _, item := range media {
			variantsJSON, err := encodeMediaVariants(item.Variants)
			if err != nil {
				return err
			}
			entities = append(entities, playEditSuggestionMediaEntity{
				SuggestionID: suggestionUUID,
				Kind:         item.Kind,
				ObjectKey:    item.ObjectKey,
				AltText:      item.AltText,
				SortOrder:    item.SortOrder,
				Variants:     variantsJSON,
				Blurhash:     item.Blurhash,
				CreatedAt:    item.CreatedAt,
			})
		}
		return tx.Create(&entities).Error
	})
}

func (r *Repository) ApplyApprovedPlayEditSuggestion(ctx context.Context, suggestionID string, updatedAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	suggestionUUID, err := parseUUID(suggestionID)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var suggestion playEditSuggestionEntity
		if err := tx.Where("id = ? AND status = ?", suggestionUUID, "approved").Take(&suggestion).Error; err != nil {
			return err
		}

		if err := tx.Model(&playEntity{}).Where("id = ?", suggestion.PlayID).Updates(map[string]any{
			"title":               suggestion.Title,
			"synopsis":            suggestion.Synopsis,
			"director":            suggestion.Director,
			"duration_minutes":    suggestion.DurationMinutes,
			"theater_name":        suggestion.TheaterName,
			"city":                suggestion.City,
			"availability_status": suggestion.AvailabilityStatus,
			"updated_at":          updatedAt,
		}).Error; err != nil {
			return err
		}

		if err := tx.Where("play_id = ?", suggestion.PlayID).Delete(&playGenreEntity{}).Error; err != nil {
			return err
		}

		var suggestionGenres []playEditSuggestionGenreEntity
		if err := tx.Where("suggestion_id = ?", suggestionUUID).Find(&suggestionGenres).Error; err != nil {
			return err
		}
		if len(suggestionGenres) > 0 {
			genres := make([]playGenreEntity, 0, len(suggestionGenres))
			for _, g := range suggestionGenres {
				genres = append(genres, playGenreEntity{PlayID: suggestion.PlayID, GenreID: g.GenreID})
			}
			if err := tx.Create(&genres).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("play_id = ?", suggestion.PlayID).Delete(&playMediaEntity{}).Error; err != nil {
			return err
		}

		var suggestionMedia []playEditSuggestionMediaEntity
		if err := tx.Where("suggestion_id = ?", suggestionUUID).Order("sort_order ASC").Order("created_at ASC").Find(&suggestionMedia).Error; err != nil {
			return err
		}
		if len(suggestionMedia) > 0 {
			mediaEntities := make([]playMediaEntity, 0, len(suggestionMedia))
			for _, m := range suggestionMedia {
				mediaEntities = append(mediaEntities, playMediaEntity{
					PlayID:    suggestion.PlayID,
					Kind:      m.Kind,
					ObjectKey: m.ObjectKey,
					AltText:   m.AltText,
					SortOrder: m.SortOrder,
					Variants:  m.Variants,
					Blurhash:  m.Blurhash,
					CreatedAt: updatedAt,
				})
			}
			if err := tx.Create(&mediaEntities).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *Repository) UpdateSubmission(ctx context.Context, playID string, params UpdateSubmissionParams) (SubmissionRecord, error) {
	if err := r.ensureDB(); err != nil {
		return SubmissionRecord{}, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return SubmissionRecord{}, err
	}

	updates := map[string]any{
		"updated_at": params.UpdatedAt,
	}

	if params.Title != nil {
		updates["title"] = *params.Title
	}

	if params.Synopsis != nil {
		updates["synopsis"] = *params.Synopsis
	}

	if params.Director != nil {
		updates["director"] = *params.Director
	}

	if params.DurationMinutes != nil {
		updates["duration_minutes"] = *params.DurationMinutes
	}

	if params.TheaterName != nil {
		updates["theater_name"] = *params.TheaterName
	}

	if params.CityProvided {
		updates["city"] = params.City
	}

	if params.AvailabilityStatus != nil {
		updates["availability_status"] = *params.AvailabilityStatus
	}

	if params.SetPendingResubmit {
		updates["curation_status"] = "pending"
	}

	if params.ClearModerationAudit {
		updates["moderated_by_user_id"] = nil
		updates["moderated_at"] = nil
		updates["published_at"] = nil
		updates["rejected_reason"] = nil
	}

	result := r.db.WithContext(ctx).
		Model(&playEntity{}).
		Where("id = ?", playUUID).
		Updates(updates)
	if result.Error != nil {
		return SubmissionRecord{}, result.Error
	}

	if result.RowsAffected == 0 {
		return SubmissionRecord{}, gorm.ErrRecordNotFound
	}

	return r.getSubmissionByUUID(ctx, playUUID)
}

func (r *Repository) ApproveSubmission(ctx context.Context, playID string, adminUserID string, now time.Time) (SubmissionRecord, error) {
	if err := r.ensureDB(); err != nil {
		return SubmissionRecord{}, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return SubmissionRecord{}, err
	}

	adminUUID, err := parseUUID(adminUserID)
	if err != nil {
		return SubmissionRecord{}, err
	}

	result := r.db.WithContext(ctx).
		Model(&playEntity{}).
		Where("id = ?", playUUID).
		Updates(map[string]any{
			"curation_status":      "published",
			"moderated_by_user_id": adminUUID,
			"moderated_at":         now,
			"published_at":         now,
			"rejected_reason":      nil,
			"updated_at":           now,
		})
	if result.Error != nil {
		return SubmissionRecord{}, result.Error
	}

	if result.RowsAffected == 0 {
		return SubmissionRecord{}, gorm.ErrRecordNotFound
	}

	return r.getSubmissionByUUID(ctx, playUUID)
}

func (r *Repository) RejectSubmission(ctx context.Context, playID string, adminUserID string, reason string, now time.Time) (SubmissionRecord, error) {
	if err := r.ensureDB(); err != nil {
		return SubmissionRecord{}, err
	}

	playUUID, err := parseUUID(playID)
	if err != nil {
		return SubmissionRecord{}, err
	}

	adminUUID, err := parseUUID(adminUserID)
	if err != nil {
		return SubmissionRecord{}, err
	}

	result := r.db.WithContext(ctx).
		Model(&playEntity{}).
		Where("id = ?", playUUID).
		Updates(map[string]any{
			"curation_status":      "rejected",
			"moderated_by_user_id": adminUUID,
			"moderated_at":         now,
			"published_at":         nil,
			"rejected_reason":      reason,
			"updated_at":           now,
		})
	if result.Error != nil {
		return SubmissionRecord{}, result.Error
	}

	if result.RowsAffected == 0 {
		return SubmissionRecord{}, gorm.ErrRecordNotFound
	}

	return r.getSubmissionByUUID(ctx, playUUID)
}

func (r *Repository) CreatePlayMedia(ctx context.Context, params CreatePlayMediaParams) (PlayMediaRecord, error) {
	if err := r.ensureDB(); err != nil {
		return PlayMediaRecord{}, err
	}

	playUUID, err := parseUUID(params.PlayID)
	if err != nil {
		return PlayMediaRecord{}, err
	}

	variantsJSON, err := encodeMediaVariants(params.Variants)
	if err != nil {
		return PlayMediaRecord{}, err
	}

	entity := playMediaEntity{
		PlayID:    playUUID,
		Kind:      params.Kind,
		ObjectKey: params.ObjectKey,
		AltText:   params.AltText,
		SortOrder: params.SortOrder,
		Variants:  variantsJSON,
		Blurhash:  params.Blurhash,
		CreatedAt: params.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&entity).Error; err != nil {
		return PlayMediaRecord{}, err
	}

	return PlayMediaRecord{
		ID:        entity.ID.String(),
		Kind:      entity.Kind,
		ObjectKey: entity.ObjectKey,
		AltText:   entity.AltText,
		SortOrder: entity.SortOrder,
		Variants:  params.Variants,
		Blurhash:  params.Blurhash,
		PlayID:    params.PlayID,
	}, nil
}

func (r *Repository) UpdatePlayMediaVariants(ctx context.Context, mediaID string, variants []MediaVariantRecord, blurhash *string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	mediaUUID, err := parseUUID(mediaID)
	if err != nil {
		return err
	}
	variantsJSON, err := encodeMediaVariants(variants)
	if err != nil {
		return err
	}
	updates := map[string]any{
		"variants": variantsJSON,
		"blurhash": blurhash,
	}
	return r.db.WithContext(ctx).Table("app.play_media").Where("id = ?", mediaUUID).Updates(updates).Error
}

func (r *Repository) basePlayListQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("app.plays AS p").
		Select(`
			p.id,
			p.title,
			p.theater_name,
			p.city,
			p.availability_status,
			p.published_at,
			prs.avg_rating,
			COALESCE(prs.review_count, 0) AS review_count,
			0::bigint AS trend_score,
			media.poster_media_id
		`).
		Joins("LEFT JOIN app.play_rating_stats AS prs ON prs.play_id = p.id").
		Joins(`
			LEFT JOIN LATERAL (
				SELECT pm.id AS poster_media_id
				FROM app.play_media AS pm
				WHERE pm.play_id = p.id
				ORDER BY CASE WHEN pm.kind = 'poster' THEN 0 ELSE 1 END, pm.sort_order ASC, pm.created_at ASC
				LIMIT 1
			) AS media ON true
		`).
		Where("p.curation_status = ?", "published")
}

func applyPlayListCursor(query *gorm.DB, after *playListCursor) (*gorm.DB, error) {
	if after == nil {
		return query, nil
	}

	playUUID, err := parseUUID(after.PlayID)
	if err != nil {
		return nil, err
	}

	return query.Where(
		"(p.published_at < ?) OR (p.published_at = ? AND p.id < ?)",
		after.PublishedAt,
		after.PublishedAt,
		playUUID,
	), nil
}

func applyTrendingFeedCursor(query *gorm.DB, after *trendingFeedCursor, trendScoreSQL string) (*gorm.DB, error) {
	if after == nil {
		return query, nil
	}

	playUUID, err := parseUUID(after.PlayID)
	if err != nil {
		return nil, err
	}

	return query.Where(
		fmt.Sprintf("(%s < ?) OR (%s = ? AND p.published_at < ?) OR (%s = ? AND p.published_at = ? AND p.id < ?)", trendScoreSQL, trendScoreSQL, trendScoreSQL),
		after.TrendScore,
		after.TrendScore,
		after.PublishedAt,
		after.TrendScore,
		after.PublishedAt,
		playUUID,
	), nil
}

func applyReviewListCursor(query *gorm.DB, after *reviewListCursor, tableAlias string) (*gorm.DB, error) {
	if after == nil {
		return query, nil
	}

	reviewUUID, err := parseUUID(after.ReviewID)
	if err != nil {
		return nil, err
	}

	return query.Where(
		fmt.Sprintf("(%s.created_at < ?) OR (%s.created_at = ? AND %s.id < ?)", tableAlias, tableAlias, tableAlias),
		after.CreatedAt,
		after.CreatedAt,
		reviewUUID,
	), nil
}

func mapPlayListRows(rows []playListRow) []PlayListRecord {
	records := make([]PlayListRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, PlayListRecord{
			ID:                 row.ID.String(),
			Title:              row.Title,
			TheaterName:        row.TheaterName,
			City:               row.City,
			AvailabilityStatus: row.AvailabilityStatus,
			PublishedAt:        row.PublishedAt,
			PosterMediaID:      nullableUUIDToString(row.PosterMediaID),
			AverageRating:      row.AverageRating,
			ReviewCount:        row.ReviewCount,
			TrendScore:         row.TrendScore,
		})
	}

	return records
}

func applySubmissionListCursor(query *gorm.DB, after *submissionListCursor) (*gorm.DB, error) {
	if after == nil {
		return query, nil
	}

	playUUID, err := parseUUID(after.PlayID)
	if err != nil {
		return nil, err
	}

	return query.Where(
		"(p.created_at < ?) OR (p.created_at = ? AND p.id < ?)",
		after.CreatedAt,
		after.CreatedAt,
		playUUID,
	), nil
}

func applyPlayEditSuggestionListCursor(query *gorm.DB, after *playEditSuggestionListCursor) (*gorm.DB, error) {
	if after == nil {
		return query, nil
	}

	suggestionUUID, err := parseUUID(after.SuggestionID)
	if err != nil {
		return nil, err
	}

	return query.Where(
		"(s.created_at < ?) OR (s.created_at = ? AND s.id < ?)",
		after.CreatedAt,
		after.CreatedAt,
		suggestionUUID,
	), nil
}

func applyNameIDListCursor(query *gorm.DB, after *genreListCursor, nameCol, idCol string) (*gorm.DB, error) {
	if after == nil {
		return query, nil
	}

	rowUUID, err := parseUUID(after.GenreID)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(after.Name)
	if name == "" {
		return nil, fmt.Errorf("invalid list cursor")
	}

	return query.Where(
		fmt.Sprintf("(%s > ?) OR (%s = ? AND %s > ?)", nameCol, nameCol, idCol),
		name,
		name,
		rowUUID,
	), nil
}

func applyEngagementPlayListCursor(query *gorm.DB, after *engagementPlayListCursor) (*gorm.DB, error) {
	if after == nil {
		return query, nil
	}

	playUUID, err := parseUUID(after.PlayID)
	if err != nil {
		return nil, err
	}

	return query.Where(
		"(e.created_at < ?) OR (e.created_at = ? AND p.id < ?)",
		after.EngagedAt,
		after.EngagedAt,
		playUUID,
	), nil
}

func mapSubmissionRows(rows []submissionRow) []SubmissionRecord {
	records := make([]SubmissionRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, SubmissionRecord{
			ID:                 row.ID.String(),
			Title:              row.Title,
			Synopsis:           row.Synopsis,
			Director:           row.Director,
			DurationMinutes:    row.DurationMinutes,
			TheaterName:        row.TheaterName,
			IsCustomTheater:    row.IsCustomTheater,
			City:               row.City,
			AvailabilityStatus: row.AvailabilityStatus,
			CurationStatus:     row.CurationStatus,
			CreatedByUserID:    row.CreatedByUserID.String(),
			ModeratedByUserID:  nullableUUIDToString(row.ModeratedByUserID),
			ModeratedAt:        row.ModeratedAt,
			PublishedAt:        row.PublishedAt,
			RejectedReason:     row.RejectedReason,
			CustomGenreName:    row.CustomGenreName,
			Production:         row.Production,
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
		})
	}

	return records
}

func mapGenreRows(rows []genreRow) []GenreRecord {
	records := make([]GenreRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, GenreRecord{ID: row.ID.String(), Name: row.Name})
	}

	return records
}

func mapPlayEditSuggestionRows(rows []playEditSuggestionRow) []PlayEditSuggestionRecord {
	records := make([]PlayEditSuggestionRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, PlayEditSuggestionRecord{
			ID:                 row.ID.String(),
			PlayID:             row.PlayID.String(),
			Title:              row.Title,
			Synopsis:           row.Synopsis,
			Director:           row.Director,
			DurationMinutes:    row.DurationMinutes,
			TheaterName:        row.TheaterName,
			City:               row.City,
			AvailabilityStatus: row.AvailabilityStatus,
			Status:             row.Status,
			CreatedByUserID:    row.CreatedByUserID.String(),
			ModeratedByUserID:  nullableUUIDToString(row.ModeratedByUserID),
			ModeratedAt:        row.ModeratedAt,
			RejectedReason:     row.RejectedReason,
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
		})
	}
	return records
}

func nullableUUIDToString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}

	raw := id.String()
	return &raw
}

func (r *Repository) getSubmissionByUUID(ctx context.Context, playUUID uuid.UUID) (SubmissionRecord, error) {
	var row submissionRow
	err := r.db.WithContext(ctx).
		Table("app.plays AS p").
		Select(`
			p.id,
			p.title,
			p.synopsis,
			p.director,
			p.duration_minutes,
			p.theater_name,
			p.is_custom_theater,
			p.custom_genre_name,
			p.production,
			p.city,
			p.availability_status,
			p.curation_status,
			p.created_by_user_id,
			p.moderated_by_user_id,
			p.moderated_at,
			p.published_at,
			p.rejected_reason,
			p.created_at,
			p.updated_at
		`).
		Where("p.id = ?", playUUID).
		Take(&row).Error
	if err != nil {
		return SubmissionRecord{}, err
	}

	records := mapSubmissionRows([]submissionRow{row})
	if len(records) == 0 {
		return SubmissionRecord{}, gorm.ErrRecordNotFound
	}

	return records[0], nil
}

func (r *Repository) getReviewByID(ctx context.Context, reviewID uuid.UUID) (ReviewRecord, error) {
	var row reviewRow
	err := r.db.WithContext(ctx).
		Table("app.reviews AS r").
		Select(`
			r.id,
			r.user_id,
			u.display_name,
			r.rating,
			r.title,
			r.body,
			r.contains_spoilers,
			r.created_at,
			r.updated_at
		`).
		Joins("JOIN app.users AS u ON u.id = r.user_id").
		Where("r.id = ?", reviewID).
		Take(&row).Error
	if err != nil {
		return ReviewRecord{}, err
	}

	return ReviewRecord{
		ID:               row.ID.String(),
		UserID:           row.UserID.String(),
		DisplayName:      row.DisplayName,
		Rating:           row.Rating,
		Title:            row.Title,
		Body:             row.Body,
		ContainsSpoilers: row.ContainsSpoilers,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func (r *Repository) ensureDB() error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}

	return nil
}

type playListRow struct {
	ID                 uuid.UUID  `gorm:"column:id"`
	Title              string     `gorm:"column:title"`
	TheaterName        string     `gorm:"column:theater_name"`
	City               *string    `gorm:"column:city"`
	AvailabilityStatus string     `gorm:"column:availability_status"`
	PublishedAt        time.Time  `gorm:"column:published_at"`
	AverageRating      *float64   `gorm:"column:avg_rating"`
	ReviewCount        int64      `gorm:"column:review_count"`
	TrendScore         int64      `gorm:"column:trend_score"`
	PosterMediaID      *uuid.UUID `gorm:"column:poster_media_id"`
}

type playDetailsRow struct {
	ID                 uuid.UUID `gorm:"column:id"`
	Title              string    `gorm:"column:title"`
	Synopsis           string    `gorm:"column:synopsis"`
	Director           string    `gorm:"column:director"`
	DurationMinutes    int       `gorm:"column:duration_minutes"`
	TheaterName        string    `gorm:"column:theater_name"`
	IsCustomTheater    bool      `gorm:"column:is_custom_theater"`
	CustomGenreName    *string   `gorm:"column:custom_genre_name"`
	Production         *string   `gorm:"column:production"`
	City               *string   `gorm:"column:city"`
	AvailabilityStatus string    `gorm:"column:availability_status"`
	PublishedAt        time.Time `gorm:"column:published_at"`
	AverageRating      *float64  `gorm:"column:avg_rating"`
	ReviewCount        int64     `gorm:"column:review_count"`
}

type engagementPlayRow struct {
	ID                 uuid.UUID  `gorm:"column:id"`
	Title              string     `gorm:"column:title"`
	TheaterName        string     `gorm:"column:theater_name"`
	City               *string    `gorm:"column:city"`
	AvailabilityStatus string     `gorm:"column:availability_status"`
	PublishedAt        time.Time  `gorm:"column:published_at"`
	AverageRating      *float64   `gorm:"column:avg_rating"`
	ReviewCount        int64      `gorm:"column:review_count"`
	PosterMediaID      *uuid.UUID `gorm:"column:poster_media_id"`
	EngagedAt          time.Time  `gorm:"column:engaged_at"`
}

type submissionRow struct {
	ID                 uuid.UUID  `gorm:"column:id"`
	Title              string     `gorm:"column:title"`
	Synopsis           string     `gorm:"column:synopsis"`
	Director           string     `gorm:"column:director"`
	DurationMinutes    int        `gorm:"column:duration_minutes"`
	TheaterName        string     `gorm:"column:theater_name"`
	IsCustomTheater    bool       `gorm:"column:is_custom_theater"`
	CustomGenreName    *string    `gorm:"column:custom_genre_name"`
	Production         *string    `gorm:"column:production"`
	City               *string    `gorm:"column:city"`
	AvailabilityStatus string     `gorm:"column:availability_status"`
	CurationStatus     string     `gorm:"column:curation_status"`
	CreatedByUserID    uuid.UUID  `gorm:"column:created_by_user_id"`
	ModeratedByUserID  *uuid.UUID `gorm:"column:moderated_by_user_id"`
	ModeratedAt        *time.Time `gorm:"column:moderated_at"`
	PublishedAt        *time.Time `gorm:"column:published_at"`
	RejectedReason     *string    `gorm:"column:rejected_reason"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

type ownedPlayRow struct {
	ID                 uuid.UUID  `gorm:"column:id"`
	Title              string     `gorm:"column:title"`
	TheaterName        string     `gorm:"column:theater_name"`
	City               *string    `gorm:"column:city"`
	AvailabilityStatus string     `gorm:"column:availability_status"`
	PublishedAt        time.Time  `gorm:"column:published_at"`
	PosterMediaID      *uuid.UUID `gorm:"column:poster_media_id"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
}

type playEditSuggestionRow struct {
	ID                 uuid.UUID  `gorm:"column:id"`
	PlayID             uuid.UUID  `gorm:"column:play_id"`
	Title              string     `gorm:"column:title"`
	Synopsis           string     `gorm:"column:synopsis"`
	Director           string     `gorm:"column:director"`
	DurationMinutes    int        `gorm:"column:duration_minutes"`
	TheaterName        string     `gorm:"column:theater_name"`
	City               *string    `gorm:"column:city"`
	AvailabilityStatus string     `gorm:"column:availability_status"`
	Status             string     `gorm:"column:status"`
	CreatedByUserID    uuid.UUID  `gorm:"column:created_by_user_id"`
	ModeratedByUserID  *uuid.UUID `gorm:"column:moderated_by_user_id"`
	ModeratedAt        *time.Time `gorm:"column:moderated_at"`
	RejectedReason     *string    `gorm:"column:rejected_reason"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

type genreRow struct {
	ID   uuid.UUID `gorm:"column:id"`
	Name string    `gorm:"column:name"`
}

type theaterRow struct {
	ID     uuid.UUID `gorm:"column:id"`
	CityID uuid.UUID `gorm:"column:city_id"`
	Name   string    `gorm:"column:name"`
}

type playGenreRow struct {
	ID   uuid.UUID `gorm:"column:id"`
	Name string    `gorm:"column:name"`
}

type playCastRow struct {
	PersonName   string `gorm:"column:person_name"`
	RoleName     string `gorm:"column:role_name"`
	BillingOrder int    `gorm:"column:billing_order"`
}

type playMediaRow struct {
	ID        uuid.UUID `gorm:"column:id"`
	Kind      string    `gorm:"column:kind"`
	ObjectKey string    `gorm:"column:object_key"`
	AltText   *string   `gorm:"column:alt_text"`
	SortOrder int       `gorm:"column:sort_order"`
	Variants  []byte    `gorm:"column:variants"`
	Blurhash  *string   `gorm:"column:blurhash"`
}

type playEditSuggestionMediaRow struct {
	ID           uuid.UUID `gorm:"column:id"`
	Kind         string    `gorm:"column:kind"`
	ObjectKey    string    `gorm:"column:object_key"`
	AltText      *string   `gorm:"column:alt_text"`
	SortOrder    int       `gorm:"column:sort_order"`
	Variants     []byte    `gorm:"column:variants"`
	Blurhash     *string   `gorm:"column:blurhash"`
	SuggestionID uuid.UUID `gorm:"column:suggestion_id"`
}

func decodeMediaVariants(raw []byte) []MediaVariantRecord {
	if len(raw) == 0 {
		return nil
	}
	var out []MediaVariantRecord
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func encodeMediaVariants(variants []MediaVariantRecord) ([]byte, error) {
	if len(variants) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(variants)
}

type reviewRow struct {
	ID               uuid.UUID `gorm:"column:id"`
	UserID           uuid.UUID `gorm:"column:user_id"`
	DisplayName      string    `gorm:"column:display_name"`
	Rating           float64   `gorm:"column:rating"`
	Title            *string   `gorm:"column:title"`
	Body             string    `gorm:"column:body"`
	ContainsSpoilers bool      `gorm:"column:contains_spoilers"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

type userReviewRow struct {
	ID                 uuid.UUID  `gorm:"column:id"`
	PlayID             uuid.UUID  `gorm:"column:play_id"`
	PlayTitle          string     `gorm:"column:play_title"`
	TheaterName        string     `gorm:"column:theater_name"`
	City               *string    `gorm:"column:city"`
	AvailabilityStatus string     `gorm:"column:availability_status"`
	PublishedAt        time.Time  `gorm:"column:published_at"`
	PosterMediaID      *uuid.UUID `gorm:"column:poster_media_id"`
	Rating             float64    `gorm:"column:rating"`
	Title              *string    `gorm:"column:title"`
	Body               string     `gorm:"column:body"`
	ContainsSpoilers   bool       `gorm:"column:contains_spoilers"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

type reviewMetadataRow struct {
	ReviewID           uuid.UUID `gorm:"column:id"`
	PlayID             uuid.UUID `gorm:"column:play_id"`
	UserID             uuid.UUID `gorm:"column:user_id"`
	ReviewStatus       string    `gorm:"column:status"`
	PlayCurationStatus string    `gorm:"column:curation_status"`
}

type reviewCommentRow struct {
	ID          uuid.UUID `gorm:"column:id"`
	ReviewID    uuid.UUID `gorm:"column:review_id"`
	UserID      uuid.UUID `gorm:"column:user_id"`
	DisplayName string    `gorm:"column:display_name"`
	Body        string    `gorm:"column:body"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

type reviewCommentStatusRow struct {
	ID        uuid.UUID `gorm:"column:id"`
	ReviewID  uuid.UUID `gorm:"column:review_id"`
	Status    string    `gorm:"column:status"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

type reviewEntity struct {
	ID               uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	PlayID           uuid.UUID `gorm:"column:play_id;type:uuid"`
	UserID           uuid.UUID `gorm:"column:user_id;type:uuid"`
	Rating           float64   `gorm:"column:rating"`
	Title            *string   `gorm:"column:title"`
	Body             string    `gorm:"column:body"`
	ContainsSpoilers bool      `gorm:"column:contains_spoilers"`
	Status           string    `gorm:"column:status"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

type playEntity struct {
	ID                 uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	Title              string     `gorm:"column:title"`
	Synopsis           string     `gorm:"column:synopsis"`
	Director           string     `gorm:"column:director"`
	DurationMinutes    int        `gorm:"column:duration_minutes"`
	TheaterName        string     `gorm:"column:theater_name"`
	IsCustomTheater    bool       `gorm:"column:is_custom_theater"`
	CustomGenreName    *string    `gorm:"column:custom_genre_name"`
	Production         *string    `gorm:"column:production"`
	City               *string    `gorm:"column:city"`
	AvailabilityStatus string     `gorm:"column:availability_status"`
	CurationStatus     string     `gorm:"column:curation_status"`
	CreatedByUserID    uuid.UUID  `gorm:"column:created_by_user_id;type:uuid"`
	ModeratedByUserID  *uuid.UUID `gorm:"column:moderated_by_user_id;type:uuid"`
	ModeratedAt        *time.Time `gorm:"column:moderated_at"`
	PublishedAt        *time.Time `gorm:"column:published_at"`
	RejectedReason     *string    `gorm:"column:rejected_reason"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

type genreEntity struct {
	ID        uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

type cityEntity struct {
	ID        uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string    `gorm:"column:name"`
	IsActive  bool      `gorm:"column:is_active"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

type theaterEntity struct {
	ID        uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	CityID    uuid.UUID `gorm:"column:city_id;type:uuid"`
	Name      string    `gorm:"column:name"`
	IsActive  bool      `gorm:"column:is_active"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

type playGenreEntity struct {
	PlayID  uuid.UUID `gorm:"column:play_id;type:uuid;primaryKey"`
	GenreID uuid.UUID `gorm:"column:genre_id;type:uuid;primaryKey"`
}

type playMediaEntity struct {
	ID        uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	PlayID    uuid.UUID `gorm:"column:play_id;type:uuid"`
	Kind      string    `gorm:"column:kind"`
	ObjectKey string    `gorm:"column:object_key"`
	AltText   *string   `gorm:"column:alt_text"`
	SortOrder int       `gorm:"column:sort_order"`
	Variants  []byte    `gorm:"column:variants;type:jsonb"`
	Blurhash  *string   `gorm:"column:blurhash"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

type playEditSuggestionEntity struct {
	ID                 uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	PlayID             uuid.UUID  `gorm:"column:play_id;type:uuid"`
	CreatedByUserID    uuid.UUID  `gorm:"column:created_by_user_id;type:uuid"`
	Status             string     `gorm:"column:status"`
	Title              string     `gorm:"column:title"`
	Synopsis           string     `gorm:"column:synopsis"`
	Director           string     `gorm:"column:director"`
	DurationMinutes    int        `gorm:"column:duration_minutes"`
	TheaterName        string     `gorm:"column:theater_name"`
	City               *string    `gorm:"column:city"`
	AvailabilityStatus string     `gorm:"column:availability_status"`
	ModeratedByUserID  *uuid.UUID `gorm:"column:moderated_by_user_id;type:uuid"`
	ModeratedAt        *time.Time `gorm:"column:moderated_at"`
	RejectedReason     *string    `gorm:"column:rejected_reason"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

type playEditSuggestionGenreEntity struct {
	SuggestionID uuid.UUID `gorm:"column:suggestion_id;type:uuid;primaryKey"`
	GenreID      uuid.UUID `gorm:"column:genre_id;type:uuid;primaryKey"`
}

type playEditSuggestionMediaEntity struct {
	ID           uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	SuggestionID uuid.UUID `gorm:"column:suggestion_id;type:uuid"`
	Kind         string    `gorm:"column:kind"`
	ObjectKey    string    `gorm:"column:object_key"`
	AltText      *string   `gorm:"column:alt_text"`
	SortOrder    int       `gorm:"column:sort_order"`
	Variants     []byte    `gorm:"column:variants;type:jsonb"`
	Blurhash     *string   `gorm:"column:blurhash"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (playEntity) TableName() string {
	return "app.plays"
}

func (genreEntity) TableName() string {
	return "app.genres"
}

func (cityEntity) TableName() string {
	return "app.cities"
}

func (theaterEntity) TableName() string {
	return "app.theaters"
}

func (playGenreEntity) TableName() string {
	return "app.play_genres"
}

func (playMediaEntity) TableName() string {
	return "app.play_media"
}

func (playEditSuggestionEntity) TableName() string {
	return "app.play_edit_suggestions"
}

func (playEditSuggestionGenreEntity) TableName() string {
	return "app.play_edit_suggestion_genres"
}

func (playEditSuggestionMediaEntity) TableName() string {
	return "app.play_edit_suggestion_media"
}

func (reviewEntity) TableName() string {
	return "app.reviews"
}

type reviewCommentEntity struct {
	ID              uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	ReviewID        uuid.UUID  `gorm:"column:review_id;type:uuid"`
	UserID          uuid.UUID  `gorm:"column:user_id;type:uuid"`
	ParentCommentID *uuid.UUID `gorm:"column:parent_comment_id;type:uuid"`
	Body            string     `gorm:"column:body"`
	Status          string     `gorm:"column:status"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (reviewCommentEntity) TableName() string {
	return "app.review_comments"
}

type userPlayEngagementEntity struct {
	ID        uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID `gorm:"column:user_id;type:uuid"`
	PlayID    uuid.UUID `gorm:"column:play_id;type:uuid"`
	Kind      string    `gorm:"column:kind"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (userPlayEngagementEntity) TableName() string {
	return "app.user_play_engagements"
}

func (r *Repository) GetPlayMediaByID(ctx context.Context, mediaID string) (PlayMediaRecord, error) {
	if err := r.ensureDB(); err != nil {
		return PlayMediaRecord{}, err
	}

	mediaUUID, err := parseUUID(mediaID)
	if err != nil {
		return PlayMediaRecord{}, err
	}

	type mediaRowFull struct {
		ID        uuid.UUID `gorm:"column:id"`
		Kind      string    `gorm:"column:kind"`
		ObjectKey string    `gorm:"column:object_key"`
		AltText   *string   `gorm:"column:alt_text"`
		SortOrder int       `gorm:"column:sort_order"`
		PlayID    uuid.UUID `gorm:"column:play_id"`
	}

	var row mediaRowFull
	err = r.db.WithContext(ctx).
		Table("app.play_media").
		Select("id, kind, object_key, alt_text, sort_order, play_id").
		Where("id = ?", mediaUUID).
		Take(&row).Error
	if err != nil {
		return PlayMediaRecord{}, err
	}

	return PlayMediaRecord{
		ID:        row.ID.String(),
		Kind:      row.Kind,
		ObjectKey: row.ObjectKey,
		AltText:   row.AltText,
		SortOrder: row.SortOrder,
		PlayID:    row.PlayID.String(),
	}, nil
}

func (r *Repository) DeletePlayMedia(ctx context.Context, mediaID string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	mediaUUID, err := parseUUID(mediaID)
	if err != nil {
		return err
	}

	result := r.db.WithContext(ctx).
		Where("id = ?", mediaUUID).
		Delete(&playMediaEntity{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
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

func escapeLike(raw string) string {
	replacer := strings.NewReplacer(`\\`, `\\\\`, `%`, `\\%`, `_`, `\\_`)
	return replacer.Replace(raw)
}
