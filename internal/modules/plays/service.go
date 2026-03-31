package plays

import (
	"context"
	"errors"
	"strings"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultListLimit = 20
	maxListLimit     = 50
)

type repositoryPort interface {
	ListFeed(ctx context.Context, params FeedListParams) ([]PlayListRecord, error)
	SearchPublishedPlays(ctx context.Context, params SearchListParams) ([]PlayListRecord, error)
	GetPublishedPlayByID(ctx context.Context, playID string) (PlayDetailsRecord, error)
	ListPlayGenres(ctx context.Context, playID string) ([]PlayGenreRecord, error)
	ListPlayCast(ctx context.Context, playID string) ([]PlayCastRecord, error)
	ListPlayMedia(ctx context.Context, playID string) ([]PlayMediaRecord, error)
}

type Service struct {
	repo repositoryPort
}

func NewService(repo repositoryPort) *Service {
	return &Service{repo: repo}
}

func (s *Service) Feed(ctx context.Context, query FeedQuery) (FeedData, error) {
	section := strings.TrimSpace(strings.ToLower(query.Section))
	if section != "highlighted" && section != "trending" && section != "genre" {
		return FeedData{}, sharederrors.Validation("section must be one of: highlighted, trending, genre", nil)
	}

	var genreID *string
	rawGenreID := strings.TrimSpace(query.GenreID)
	if section == "genre" {
		if rawGenreID == "" {
			return FeedData{}, sharederrors.Validation("genreId is required when section=genre", nil)
		}

		if !isValidUUID(rawGenreID) {
			return FeedData{}, sharederrors.Validation("invalid genreId", nil)
		}

		genreID = &rawGenreID
	} else if rawGenreID != "" {
		if !isValidUUID(rawGenreID) {
			return FeedData{}, sharederrors.Validation("invalid genreId", nil)
		}
		genreID = &rawGenreID
	}

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return FeedData{}, err
	}

	cursor, err := decodePlayListCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return FeedData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.ListFeed(ctx, FeedListParams{
		Section: section,
		GenreID: genreID,
		After:   cursor,
		Limit:   limit + 1,
	})
	if err != nil {
		return FeedData{}, sharederrors.Internal("failed to load feed", nil)
	}

	return buildPlayListData(records, limit)
}

func (s *Service) Search(ctx context.Context, query SearchQuery) (SearchData, error) {
	q := strings.TrimSpace(query.Q)
	genreIDRaw := strings.TrimSpace(query.GenreID)
	city := strings.TrimSpace(query.City)
	theater := strings.TrimSpace(query.Theater)
	availabilityStatus := strings.TrimSpace(strings.ToLower(query.AvailabilityStatus))

	if q == "" && genreIDRaw == "" && city == "" && theater == "" && availabilityStatus == "" {
		return SearchData{}, sharederrors.Validation("at least one search filter must be provided", nil)
	}

	var genreID *string
	if genreIDRaw != "" {
		if !isValidUUID(genreIDRaw) {
			return SearchData{}, sharederrors.Validation("invalid genreId", nil)
		}
		genreID = &genreIDRaw
	}

	var cityFilter *string
	if city != "" {
		cityFilter = &city
	}

	var theaterFilter *string
	if theater != "" {
		theaterFilter = &theater
	}

	var availabilityFilter *string
	if availabilityStatus != "" {
		if availabilityStatus != "in_theaters" && availabilityStatus != "archive" {
			return SearchData{}, sharederrors.Validation("availabilityStatus must be one of: in_theaters, archive", nil)
		}
		availabilityFilter = &availabilityStatus
	}

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return SearchData{}, err
	}

	cursor, err := decodePlayListCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return SearchData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.SearchPublishedPlays(ctx, SearchListParams{
		Q:                  q,
		GenreID:            genreID,
		City:               cityFilter,
		Theater:            theaterFilter,
		AvailabilityStatus: availabilityFilter,
		After:              cursor,
		Limit:              limit + 1,
	})
	if err != nil {
		return SearchData{}, sharederrors.Internal("failed to search plays", nil)
	}

	data, err := buildPlayListData(records, limit)
	if err != nil {
		return SearchData{}, err
	}

	return SearchData{Items: data.Items, NextCursor: data.NextCursor}, nil
}

func (s *Service) GetByID(ctx context.Context, playID string) (PlayDetailsData, error) {
	if !isValidUUID(playID) {
		return PlayDetailsData{}, sharederrors.Validation("invalid playId", nil)
	}

	play, err := s.repo.GetPublishedPlayByID(ctx, playID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlayDetailsData{}, sharederrors.NotFound("play not found", nil)
		}

		return PlayDetailsData{}, sharederrors.Internal("failed to load play", nil)
	}

	genres, err := s.repo.ListPlayGenres(ctx, playID)
	if err != nil {
		return PlayDetailsData{}, sharederrors.Internal("failed to load play", nil)
	}

	cast, err := s.repo.ListPlayCast(ctx, playID)
	if err != nil {
		return PlayDetailsData{}, sharederrors.Internal("failed to load play", nil)
	}

	media, err := s.repo.ListPlayMedia(ctx, playID)
	if err != nil {
		return PlayDetailsData{}, sharederrors.Internal("failed to load play", nil)
	}

	return mapPlayDetails(play, genres, cast, media), nil
}

func buildPlayListData(records []PlayListRecord, limit int) (FeedData, error) {
	hasNext := len(records) > limit
	if hasNext {
		records = records[:limit]
	}

	items := make([]PlayCardData, 0, len(records))
	for _, record := range records {
		items = append(items, PlayCardData{
			ID:                 record.ID,
			Title:              record.Title,
			TheaterName:        record.TheaterName,
			City:               record.City,
			AvailabilityStatus: record.AvailabilityStatus,
			PublishedAt:        record.PublishedAt.UTC().Format(time.RFC3339Nano),
			PosterURL:          record.PosterURL,
			AverageRating:      record.AverageRating,
			ReviewCount:        record.ReviewCount,
		})
	}

	response := FeedData{Items: items}
	if hasNext && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor, err := encodePlayListCursor(playListCursor{PublishedAt: last.PublishedAt.UTC(), PlayID: last.ID})
		if err != nil {
			return FeedData{}, sharederrors.Internal("failed to build pagination cursor", nil)
		}

		response.NextCursor = &nextCursor
	}

	return response, nil
}

func mapPlayDetails(play PlayDetailsRecord, genres []PlayGenreRecord, cast []PlayCastRecord, media []PlayMediaRecord) PlayDetailsData {
	genreItems := make([]PlayGenreData, 0, len(genres))
	for _, genre := range genres {
		genreItems = append(genreItems, PlayGenreData{ID: genre.ID, Name: genre.Name})
	}

	castItems := make([]PlayCastMemberData, 0, len(cast))
	for _, member := range cast {
		castItems = append(castItems, PlayCastMemberData{
			PersonName:   member.PersonName,
			RoleName:     member.RoleName,
			BillingOrder: member.BillingOrder,
		})
	}

	mediaItems := make([]PlayMediaData, 0, len(media))
	for _, item := range media {
		mediaItems = append(mediaItems, PlayMediaData{
			Kind:      item.Kind,
			URL:       item.URL,
			AltText:   item.AltText,
			SortOrder: item.SortOrder,
		})
	}

	return PlayDetailsData{
		ID:                 play.ID,
		Title:              play.Title,
		Synopsis:           play.Synopsis,
		Director:           play.Director,
		DurationMinutes:    play.DurationMinutes,
		TheaterName:        play.TheaterName,
		City:               play.City,
		AvailabilityStatus: play.AvailabilityStatus,
		PublishedAt:        play.PublishedAt.UTC().Format(time.RFC3339Nano),
		Stats: PlayStatsData{
			AverageRating: play.AverageRating,
			ReviewCount:   play.ReviewCount,
		},
		Genres: genreItems,
		Cast:   castItems,
		Media:  mediaItems,
	}
}

func normalizeListLimit(rawLimit int) (int, error) {
	if rawLimit == 0 {
		return defaultListLimit, nil
	}

	if rawLimit < 1 || rawLimit > maxListLimit {
		return 0, sharederrors.Validation("limit must be between 1 and 50", nil)
	}

	return rawLimit, nil
}

func isValidUUID(raw string) bool {
	_, err := uuid.Parse(strings.TrimSpace(raw))
	return err == nil
}
