package plays

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FeedListParams struct {
	Section string
	GenreID *string
	After   *playListCursor
	Limit   int
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

	switch params.Section {
	case "highlighted":
		query = query.Order("COALESCE(prs.review_count, 0) DESC")
	case "trending":
		query = query.Where("p.published_at >= ?", time.Now().UTC().Add(-30*24*time.Hour)).Order("COALESCE(prs.review_count, 0) DESC")
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
		})
	}

	return records
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
