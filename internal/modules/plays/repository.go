package plays

import (
	"context"
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
				media.poster_url
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
		City:               row.City,
		AvailabilityStatus: row.AvailabilityStatus,
		PublishedAt:        row.PublishedAt,
		AverageRating:      row.AverageRating,
		ReviewCount:        row.ReviewCount,
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
		Select("kind, url, alt_text, sort_order").
		Where("play_id = ?", playUUID).
		Order("sort_order ASC").
		Order("created_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	media := make([]PlayMediaRecord, 0, len(rows))
	for _, row := range rows {
		media = append(media, PlayMediaRecord{Kind: row.Kind, URL: row.URL, AltText: row.AltText, SortOrder: row.SortOrder})
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
			Rating:           int(row.Rating),
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
			media.poster_url,
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
				SELECT pm.url AS poster_url
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
			PosterURL:          row.PosterURL,
			Rating:             int(row.Rating),
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
		Rating:           int16(params.Rating),
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
		updates["rating"] = int16(*params.Rating)
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

	return EngagementStateRecord{Wishlist: wishlistCount > 0, Attended: attendedCount > 0}, nil
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
			media.poster_url,
			e.created_at AS engaged_at
		`).
		Joins("JOIN app.plays AS p ON p.id = e.play_id").
		Joins("LEFT JOIN app.play_rating_stats AS prs ON prs.play_id = p.id").
		Joins(`
			LEFT JOIN LATERAL (
				SELECT pm.url AS poster_url
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
			PosterURL:          row.PosterURL,
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
		City:               params.City,
		AvailabilityStatus: params.AvailabilityStatus,
		CurationStatus:     "pending",
		CreatedByUserID:    creatorUUID,
		CreatedAt:          params.CreatedAt,
		UpdatedAt:          params.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&entity).Error; err != nil {
		return SubmissionRecord{}, err
	}

	return r.getSubmissionByUUID(ctx, entity.ID)
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
			media.poster_url
		`).
		Joins("LEFT JOIN app.play_rating_stats AS prs ON prs.play_id = p.id").
		Joins(`
			LEFT JOIN LATERAL (
				SELECT pm.url AS poster_url
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
			PosterURL:          row.PosterURL,
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
			City:               row.City,
			AvailabilityStatus: row.AvailabilityStatus,
			CurationStatus:     row.CurationStatus,
			CreatedByUserID:    row.CreatedByUserID.String(),
			ModeratedByUserID:  nullableUUIDToString(row.ModeratedByUserID),
			ModeratedAt:        row.ModeratedAt,
			PublishedAt:        row.PublishedAt,
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
		Rating:           int(row.Rating),
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
	ID                 uuid.UUID `gorm:"column:id"`
	Title              string    `gorm:"column:title"`
	TheaterName        string    `gorm:"column:theater_name"`
	City               *string   `gorm:"column:city"`
	AvailabilityStatus string    `gorm:"column:availability_status"`
	PublishedAt        time.Time `gorm:"column:published_at"`
	AverageRating      *float64  `gorm:"column:avg_rating"`
	ReviewCount        int64     `gorm:"column:review_count"`
	TrendScore         int64     `gorm:"column:trend_score"`
	PosterURL          *string   `gorm:"column:poster_url"`
}

type playDetailsRow struct {
	ID                 uuid.UUID `gorm:"column:id"`
	Title              string    `gorm:"column:title"`
	Synopsis           string    `gorm:"column:synopsis"`
	Director           string    `gorm:"column:director"`
	DurationMinutes    int       `gorm:"column:duration_minutes"`
	TheaterName        string    `gorm:"column:theater_name"`
	City               *string   `gorm:"column:city"`
	AvailabilityStatus string    `gorm:"column:availability_status"`
	PublishedAt        time.Time `gorm:"column:published_at"`
	AverageRating      *float64  `gorm:"column:avg_rating"`
	ReviewCount        int64     `gorm:"column:review_count"`
}

type engagementPlayRow struct {
	ID                 uuid.UUID `gorm:"column:id"`
	Title              string    `gorm:"column:title"`
	TheaterName        string    `gorm:"column:theater_name"`
	City               *string   `gorm:"column:city"`
	AvailabilityStatus string    `gorm:"column:availability_status"`
	PublishedAt        time.Time `gorm:"column:published_at"`
	AverageRating      *float64  `gorm:"column:avg_rating"`
	ReviewCount        int64     `gorm:"column:review_count"`
	PosterURL          *string   `gorm:"column:poster_url"`
	EngagedAt          time.Time `gorm:"column:engaged_at"`
}

type submissionRow struct {
	ID                 uuid.UUID  `gorm:"column:id"`
	Title              string     `gorm:"column:title"`
	Synopsis           string     `gorm:"column:synopsis"`
	Director           string     `gorm:"column:director"`
	DurationMinutes    int        `gorm:"column:duration_minutes"`
	TheaterName        string     `gorm:"column:theater_name"`
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
	Kind      string  `gorm:"column:kind"`
	URL       string  `gorm:"column:url"`
	AltText   *string `gorm:"column:alt_text"`
	SortOrder int     `gorm:"column:sort_order"`
}

type reviewRow struct {
	ID               uuid.UUID `gorm:"column:id"`
	UserID           uuid.UUID `gorm:"column:user_id"`
	DisplayName      string    `gorm:"column:display_name"`
	Rating           int16     `gorm:"column:rating"`
	Title            *string   `gorm:"column:title"`
	Body             string    `gorm:"column:body"`
	ContainsSpoilers bool      `gorm:"column:contains_spoilers"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

type userReviewRow struct {
	ID                 uuid.UUID `gorm:"column:id"`
	PlayID             uuid.UUID `gorm:"column:play_id"`
	PlayTitle          string    `gorm:"column:play_title"`
	TheaterName        string    `gorm:"column:theater_name"`
	City               *string   `gorm:"column:city"`
	AvailabilityStatus string    `gorm:"column:availability_status"`
	PublishedAt        time.Time `gorm:"column:published_at"`
	PosterURL          *string   `gorm:"column:poster_url"`
	Rating             int16     `gorm:"column:rating"`
	Title              *string   `gorm:"column:title"`
	Body               string    `gorm:"column:body"`
	ContainsSpoilers   bool      `gorm:"column:contains_spoilers"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
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
	Rating           int16     `gorm:"column:rating"`
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

func (playEntity) TableName() string {
	return "app.plays"
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
