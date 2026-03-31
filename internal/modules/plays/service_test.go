package plays

import (
	"context"
	"errors"
	"testing"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"gorm.io/gorm"
)

type fakeRepository struct {
	listFeedFn           func(ctx context.Context, params FeedListParams) ([]PlayListRecord, error)
	searchPublishedPlays func(ctx context.Context, params SearchListParams) ([]PlayListRecord, error)
	getPublishedPlayByID func(ctx context.Context, playID string) (PlayDetailsRecord, error)
	listPlayGenresFn     func(ctx context.Context, playID string) ([]PlayGenreRecord, error)
	listPlayCastFn       func(ctx context.Context, playID string) ([]PlayCastRecord, error)
	listPlayMediaFn      func(ctx context.Context, playID string) ([]PlayMediaRecord, error)
}

func (f *fakeRepository) ListFeed(ctx context.Context, params FeedListParams) ([]PlayListRecord, error) {
	if f.listFeedFn != nil {
		return f.listFeedFn(ctx, params)
	}

	return nil, nil
}

func (f *fakeRepository) SearchPublishedPlays(ctx context.Context, params SearchListParams) ([]PlayListRecord, error) {
	if f.searchPublishedPlays != nil {
		return f.searchPublishedPlays(ctx, params)
	}

	return nil, nil
}

func (f *fakeRepository) GetPublishedPlayByID(ctx context.Context, playID string) (PlayDetailsRecord, error) {
	if f.getPublishedPlayByID != nil {
		return f.getPublishedPlayByID(ctx, playID)
	}

	return PlayDetailsRecord{}, gorm.ErrRecordNotFound
}

func (f *fakeRepository) ListPlayGenres(ctx context.Context, playID string) ([]PlayGenreRecord, error) {
	if f.listPlayGenresFn != nil {
		return f.listPlayGenresFn(ctx, playID)
	}

	return []PlayGenreRecord{}, nil
}

func (f *fakeRepository) ListPlayCast(ctx context.Context, playID string) ([]PlayCastRecord, error) {
	if f.listPlayCastFn != nil {
		return f.listPlayCastFn(ctx, playID)
	}

	return []PlayCastRecord{}, nil
}

func (f *fakeRepository) ListPlayMedia(ctx context.Context, playID string) ([]PlayMediaRecord, error) {
	if f.listPlayMediaFn != nil {
		return f.listPlayMediaFn(ctx, playID)
	}

	return []PlayMediaRecord{}, nil
}

func TestServiceFeedRejectsInvalidSection(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.Feed(context.Background(), FeedQuery{Section: "unknown"})
	if err == nil {
		t.Fatalf("expected validation error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestServiceFeedGenreRequiresGenreID(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.Feed(context.Background(), FeedQuery{Section: "genre"})
	if err == nil {
		t.Fatalf("expected validation error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestServiceFeedBuildsNextCursor(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	service := NewService(&fakeRepository{listFeedFn: func(ctx context.Context, params FeedListParams) ([]PlayListRecord, error) {
		if params.Limit != 2 {
			t.Fatalf("expected limit 2, got %d", params.Limit)
		}

		return []PlayListRecord{
			{ID: "00000000-0000-0000-0000-000000000201", Title: "A", TheaterName: "T1", AvailabilityStatus: "in_theaters", PublishedAt: now},
			{ID: "00000000-0000-0000-0000-000000000202", Title: "B", TheaterName: "T2", AvailabilityStatus: "archive", PublishedAt: now.Add(-time.Minute)},
		}, nil
	}})

	data, err := service.Feed(context.Background(), FeedQuery{Section: "highlighted", Limit: 1})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(data.Items))
	}

	if data.NextCursor == nil || *data.NextCursor == "" {
		t.Fatalf("expected next cursor")
	}
}

func TestServiceSearchRequiresAtLeastOneFilter(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.Search(context.Background(), SearchQuery{})
	if err == nil {
		t.Fatalf("expected validation error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestServiceSearchRejectsInvalidAvailabilityStatus(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.Search(context.Background(), SearchQuery{Q: "hamlet", AvailabilityStatus: "coming_soon"})
	if err == nil {
		t.Fatalf("expected validation error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestServiceGetByIDMapsNotFound(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{getPublishedPlayByID: func(ctx context.Context, playID string) (PlayDetailsRecord, error) {
		return PlayDetailsRecord{}, gorm.ErrRecordNotFound
	}})

	_, err := service.GetByID(context.Background(), "00000000-0000-0000-0000-000000000201")
	if err == nil {
		t.Fatalf("expected not found error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %q", appErr.Code)
	}
}

func TestServiceGetByIDSuccess(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	service := NewService(&fakeRepository{
		getPublishedPlayByID: func(ctx context.Context, playID string) (PlayDetailsRecord, error) {
			return PlayDetailsRecord{
				ID:                 playID,
				Title:              "Hamlet en la Habana",
				Synopsis:           "Version contemporanea.",
				Director:           "Carlos",
				DurationMinutes:    120,
				TheaterName:        "Gran Teatro",
				AvailabilityStatus: "in_theaters",
				PublishedAt:        now,
				ReviewCount:        2,
			}, nil
		},
		listPlayGenresFn: func(ctx context.Context, playID string) ([]PlayGenreRecord, error) {
			return []PlayGenreRecord{{ID: "00000000-0000-0000-0000-000000000101", Name: "Drama"}}, nil
		},
		listPlayCastFn: func(ctx context.Context, playID string) ([]PlayCastRecord, error) {
			return []PlayCastRecord{{PersonName: "Luis", RoleName: "Hamlet", BillingOrder: 1}}, nil
		},
		listPlayMediaFn: func(ctx context.Context, playID string) ([]PlayMediaRecord, error) {
			return []PlayMediaRecord{{Kind: "poster", URL: "https://cdn/poster.jpg", SortOrder: 0}}, nil
		},
	})

	data, err := service.GetByID(context.Background(), "00000000-0000-0000-0000-000000000201")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if data.ID == "" || data.Title == "" {
		t.Fatalf("expected play details data")
	}

	if len(data.Genres) != 1 || len(data.Cast) != 1 || len(data.Media) != 1 {
		t.Fatalf("expected related collections to be mapped")
	}
}
