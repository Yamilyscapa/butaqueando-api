package plays

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/butaqueando/api/internal/shared/cache"
	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type fakePresignStorage struct{}

type fakeCache struct {
	values     map[string]string
	getErr     error
	setErr     error
	setCalls   int
	setTTL     time.Duration
	delCalls   int
	delKeys    []string
	delPatts   []string
	delErr     error
}

func (f *fakeCache) Get(_ context.Context, key string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	value, ok := f.values[key]
	if !ok {
		return "", cache.ErrCacheMiss
	}
	return value, nil
}

func (f *fakeCache) Set(_ context.Context, key string, value string, ttl time.Duration) error {
	f.setCalls++
	f.setTTL = ttl
	if f.setErr != nil {
		return f.setErr
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key] = value
	return nil
}

func (f *fakeCache) Del(_ context.Context, keys ...string) error {
	f.delCalls++
	f.delKeys = append(f.delKeys, keys...)
	if f.delErr != nil {
		return f.delErr
	}
	if f.values == nil {
		return nil
	}
	for _, key := range keys {
		delete(f.values, key)
	}
	return nil
}

func (f *fakeCache) DelByPattern(_ context.Context, pattern string) (int, error) {
	f.delCalls++
	f.delPatts = append(f.delPatts, pattern)
	if f.delErr != nil {
		return 0, f.delErr
	}
	if f.values == nil {
		return 0, nil
	}
	prefix := strings.TrimSuffix(pattern, "*")
	deleted := 0
	for key := range f.values {
		if strings.HasPrefix(key, prefix) {
			delete(f.values, key)
			deleted++
		}
	}
	return deleted, nil
}

func (f *fakeCache) Close() error { return nil }

func (fakePresignStorage) Bucket() string { return "test-bucket" }

func (fakePresignStorage) PresignPutObject(_ context.Context, _ storage.PresignPutObjectInput) (string, error) {
	return "", storage.ErrClientNotConfigured
}

func (fakePresignStorage) PresignGetObject(_ context.Context, input storage.PresignGetObjectInput) (string, error) {
	return "https://presigned.example.com/" + input.ObjectKey, nil
}

func (fakePresignStorage) HeadObject(_ context.Context, _ storage.HeadObjectInput) (storage.HeadObjectOutput, error) {
	return storage.HeadObjectOutput{}, storage.ErrClientNotConfigured
}

func (fakePresignStorage) GetObject(_ context.Context, _ storage.GetObjectInput) ([]byte, error) {
	return nil, storage.ErrClientNotConfigured
}

func (fakePresignStorage) PutObject(_ context.Context, _ storage.PutObjectInput) error {
	return storage.ErrClientNotConfigured
}

type fakeRepository struct {
	listFeedFn             func(ctx context.Context, params FeedListParams) ([]PlayListRecord, error)
	searchPublishedPlays   func(ctx context.Context, params SearchListParams) ([]PlayListRecord, error)
	getPublishedPlayByID   func(ctx context.Context, playID string) (PlayDetailsRecord, error)
	listPlayGenresFn       func(ctx context.Context, playID string) ([]PlayGenreRecord, error)
	listPlayCastFn         func(ctx context.Context, playID string) ([]PlayCastRecord, error)
	listPlayMediaFn        func(ctx context.Context, playID string) ([]PlayMediaRecord, error)
	isPlayPublishedFn      func(ctx context.Context, playID string) (bool, error)
	userExistsFn           func(ctx context.Context, userID string) (bool, error)
	listReviewsFn          func(ctx context.Context, params ListReviewsParams) ([]ReviewRecord, error)
	listUserReviewsFn      func(ctx context.Context, userID string, params ListUserReviewsParams) ([]UserReviewRecord, error)
	createReviewFn         func(ctx context.Context, userID string, playID string, params CreateReviewParams) (ReviewRecord, error)
	getReviewMetadataFn    func(ctx context.Context, reviewID string) (ReviewMetadataRecord, error)
	updateReviewFn         func(ctx context.Context, reviewID string, params UpdateReviewParams) (ReviewRecord, error)
	createCommentFn        func(ctx context.Context, userID string, reviewID string, params CreateReviewCommentParams) (ReviewCommentRecord, error)
	updateCommentStatusFn  func(ctx context.Context, commentID string, status string, updatedAt time.Time) (ReviewCommentStatusRecord, error)
	createSubmissionFn     func(ctx context.Context, userID string, params CreateSubmissionParams) (SubmissionRecord, error)
	listUserSubmissionsFn  func(ctx context.Context, userID string, params ListSubmissionsParams) ([]SubmissionRecord, error)
	getSubmissionByIDFn    func(ctx context.Context, playID string) (SubmissionRecord, error)
	updateSubmissionFn     func(ctx context.Context, playID string, params UpdateSubmissionParams) (SubmissionRecord, error)
	listGenresFn           func(ctx context.Context, params ListGenresParams) ([]GenreRecord, error)
	listCitiesFn           func(ctx context.Context, params ListCitiesParams) ([]CityRecord, error)
	listTheatersFn         func(ctx context.Context, params ListTheatersParams) ([]TheaterRecord, error)
	cityExistsFn           func(ctx context.Context, cityName string) (bool, error)
	theaterExistsInCityFn  func(ctx context.Context, cityName string, theaterName string) (bool, error)
	createCityFn           func(ctx context.Context, name string) (CityRecord, error)
	deleteCityFn           func(ctx context.Context, cityID string) error
	createTheaterFn        func(ctx context.Context, cityID string, name string) (TheaterRecord, error)
	deleteTheaterFn        func(ctx context.Context, theaterID string) error
	deletePlayFn           func(ctx context.Context, playID string) error
	countGenresByIDsFn     func(ctx context.Context, genreIDs []string) (int64, error)
	createGenreFn          func(ctx context.Context, name string) (GenreRecord, error)
	deleteGenreFn          func(ctx context.Context, genreID string) error
	listAdminSubmissionsFn func(ctx context.Context, params ListSubmissionsParams) ([]SubmissionRecord, error)
	listOwnedPlaysFn       func(ctx context.Context, userID string, params ListSubmissionsParams) ([]OwnedPlayRecord, error)
	listEditSuggestionsFn  func(ctx context.Context, params ListPlayEditSuggestionsParams) ([]PlayEditSuggestionRecord, error)
	approveSubmissionFn    func(ctx context.Context, playID string, adminUserID string, now time.Time) (SubmissionRecord, error)
	rejectSubmissionFn     func(ctx context.Context, playID string, adminUserID string, reason string, now time.Time) (SubmissionRecord, error)
	createPlayMediaFn      func(ctx context.Context, params CreatePlayMediaParams) (PlayMediaRecord, error)
	getPlayMediaByIDFn     func(ctx context.Context, mediaID string) (PlayMediaRecord, error)
	deletePlayMediaFn      func(ctx context.Context, mediaID string) error
	setEngagementFn        func(ctx context.Context, userID string, playID string, kind string, createdAt time.Time) error
	deleteEngagementFn     func(ctx context.Context, userID string, playID string, kind string) error
	engagementStateFn      func(ctx context.Context, userID string, playID string) (EngagementStateRecord, error)
	listMyEngagementsFn    func(ctx context.Context, userID string, params ListUserEngagementPlaysParams) ([]EngagementPlayRecord, error)
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

func (f *fakeRepository) IsPlayPublished(ctx context.Context, playID string) (bool, error) {
	if f.isPlayPublishedFn != nil {
		return f.isPlayPublishedFn(ctx, playID)
	}

	return true, nil
}

func (f *fakeRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	if f.userExistsFn != nil {
		return f.userExistsFn(ctx, userID)
	}

	return true, nil
}

func (f *fakeRepository) ListPublishedReviews(ctx context.Context, params ListReviewsParams) ([]ReviewRecord, error) {
	if f.listReviewsFn != nil {
		return f.listReviewsFn(ctx, params)
	}

	return nil, nil
}

func (f *fakeRepository) ListUserPublishedReviews(ctx context.Context, userID string, params ListUserReviewsParams) ([]UserReviewRecord, error) {
	if f.listUserReviewsFn != nil {
		return f.listUserReviewsFn(ctx, userID, params)
	}

	return nil, nil
}

func (f *fakeRepository) CreateReview(ctx context.Context, userID string, playID string, params CreateReviewParams) (ReviewRecord, error) {
	if f.createReviewFn != nil {
		return f.createReviewFn(ctx, userID, playID, params)
	}

	return ReviewRecord{}, nil
}

func (f *fakeRepository) GetReviewMetadata(ctx context.Context, reviewID string) (ReviewMetadataRecord, error) {
	if f.getReviewMetadataFn != nil {
		return f.getReviewMetadataFn(ctx, reviewID)
	}

	return ReviewMetadataRecord{}, gorm.ErrRecordNotFound
}

func (f *fakeRepository) UpdateReview(ctx context.Context, reviewID string, params UpdateReviewParams) (ReviewRecord, error) {
	if f.updateReviewFn != nil {
		return f.updateReviewFn(ctx, reviewID, params)
	}

	return ReviewRecord{}, nil
}

func (f *fakeRepository) CreateReviewComment(ctx context.Context, userID string, reviewID string, params CreateReviewCommentParams) (ReviewCommentRecord, error) {
	if f.createCommentFn != nil {
		return f.createCommentFn(ctx, userID, reviewID, params)
	}

	return ReviewCommentRecord{}, nil
}

func (f *fakeRepository) UpdateReviewCommentStatus(ctx context.Context, commentID string, status string, updatedAt time.Time) (ReviewCommentStatusRecord, error) {
	if f.updateCommentStatusFn != nil {
		return f.updateCommentStatusFn(ctx, commentID, status, updatedAt)
	}

	return ReviewCommentStatusRecord{}, nil
}

func (f *fakeRepository) CreateSubmission(ctx context.Context, userID string, params CreateSubmissionParams) (SubmissionRecord, error) {
	if f.createSubmissionFn != nil {
		return f.createSubmissionFn(ctx, userID, params)
	}

	return SubmissionRecord{}, nil
}

func (f *fakeRepository) ListUserSubmissions(ctx context.Context, userID string, params ListSubmissionsParams) ([]SubmissionRecord, error) {
	if f.listUserSubmissionsFn != nil {
		return f.listUserSubmissionsFn(ctx, userID, params)
	}

	return nil, nil
}

func (f *fakeRepository) GetSubmissionByID(ctx context.Context, playID string) (SubmissionRecord, error) {
	if f.getSubmissionByIDFn != nil {
		return f.getSubmissionByIDFn(ctx, playID)
	}

	return SubmissionRecord{}, gorm.ErrRecordNotFound
}

func (f *fakeRepository) UpdateSubmission(ctx context.Context, playID string, params UpdateSubmissionParams) (SubmissionRecord, error) {
	if f.updateSubmissionFn != nil {
		return f.updateSubmissionFn(ctx, playID, params)
	}

	return SubmissionRecord{}, nil
}

func (f *fakeRepository) ListGenres(ctx context.Context, params ListGenresParams) ([]GenreRecord, error) {
	if f.listGenresFn != nil {
		return f.listGenresFn(ctx, params)
	}

	return nil, nil
}

func (f *fakeRepository) ListCities(ctx context.Context, params ListCitiesParams) ([]CityRecord, error) {
	if f.listCitiesFn != nil {
		return f.listCitiesFn(ctx, params)
	}

	return nil, nil
}

func (f *fakeRepository) ListTheaters(ctx context.Context, params ListTheatersParams) ([]TheaterRecord, error) {
	if f.listTheatersFn != nil {
		return f.listTheatersFn(ctx, params)
	}

	return nil, nil
}

func (f *fakeRepository) CityExists(ctx context.Context, cityName string) (bool, error) {
	if f.cityExistsFn != nil {
		return f.cityExistsFn(ctx, cityName)
	}

	return true, nil
}

func (f *fakeRepository) TheaterExistsInCity(ctx context.Context, cityName string, theaterName string) (bool, error) {
	if f.theaterExistsInCityFn != nil {
		return f.theaterExistsInCityFn(ctx, cityName, theaterName)
	}

	return true, nil
}

func (f *fakeRepository) CreateCity(ctx context.Context, name string) (CityRecord, error) {
	if f.createCityFn != nil {
		return f.createCityFn(ctx, name)
	}

	return CityRecord{}, nil
}

func (f *fakeRepository) DeleteCity(ctx context.Context, cityID string) error {
	if f.deleteCityFn != nil {
		return f.deleteCityFn(ctx, cityID)
	}

	return nil
}

func (f *fakeRepository) CreateTheater(ctx context.Context, cityID string, name string) (TheaterRecord, error) {
	if f.createTheaterFn != nil {
		return f.createTheaterFn(ctx, cityID, name)
	}

	return TheaterRecord{}, nil
}

func (f *fakeRepository) DeleteTheater(ctx context.Context, theaterID string) error {
	if f.deleteTheaterFn != nil {
		return f.deleteTheaterFn(ctx, theaterID)
	}

	return nil
}

func (f *fakeRepository) DeletePlay(ctx context.Context, playID string) error {
	if f.deletePlayFn != nil {
		return f.deletePlayFn(ctx, playID)
	}

	return nil
}

func (f *fakeRepository) CountGenresByIDs(ctx context.Context, genreIDs []string) (int64, error) {
	if f.countGenresByIDsFn != nil {
		return f.countGenresByIDsFn(ctx, genreIDs)
	}

	return 0, nil
}

func (f *fakeRepository) CreateGenre(ctx context.Context, name string) (GenreRecord, error) {
	if f.createGenreFn != nil {
		return f.createGenreFn(ctx, name)
	}

	return GenreRecord{}, nil
}

func (f *fakeRepository) DeleteGenre(ctx context.Context, genreID string) error {
	if f.deleteGenreFn != nil {
		return f.deleteGenreFn(ctx, genreID)
	}

	return nil
}

func (f *fakeRepository) ListAdminSubmissions(ctx context.Context, params ListSubmissionsParams) ([]SubmissionRecord, error) {
	if f.listAdminSubmissionsFn != nil {
		return f.listAdminSubmissionsFn(ctx, params)
	}

	return nil, nil
}

func (f *fakeRepository) ListOwnedPublishedPlays(ctx context.Context, userID string, params ListSubmissionsParams) ([]OwnedPlayRecord, error) {
	if f.listOwnedPlaysFn != nil {
		return f.listOwnedPlaysFn(ctx, userID, params)
	}

	return nil, nil
}

func (f *fakeRepository) GetOwnedPublishedPlayByID(ctx context.Context, playID string, userID string) (OwnedPlayRecord, error) {
	return OwnedPlayRecord{}, gorm.ErrRecordNotFound
}

func (f *fakeRepository) ListPlayEditSuggestions(ctx context.Context, params ListPlayEditSuggestionsParams) ([]PlayEditSuggestionRecord, error) {
	if f.listEditSuggestionsFn != nil {
		return f.listEditSuggestionsFn(ctx, params)
	}

	return nil, nil
}

func (f *fakeRepository) GetPlayEditSuggestionByID(ctx context.Context, suggestionID string) (PlayEditSuggestionRecord, error) {
	return PlayEditSuggestionRecord{}, gorm.ErrRecordNotFound
}

func (f *fakeRepository) CreatePlayEditSuggestion(ctx context.Context, params CreatePlayEditSuggestionParams) (PlayEditSuggestionRecord, error) {
	return PlayEditSuggestionRecord{}, nil
}

func (f *fakeRepository) UpdatePlayEditSuggestion(ctx context.Context, suggestionID string, params UpdatePlayEditSuggestionParams) (PlayEditSuggestionRecord, error) {
	return PlayEditSuggestionRecord{}, nil
}

func (f *fakeRepository) ModeratePlayEditSuggestion(ctx context.Context, suggestionID string, params ModeratePlayEditSuggestionParams) (PlayEditSuggestionRecord, error) {
	return PlayEditSuggestionRecord{}, nil
}

func (f *fakeRepository) ReplacePlayEditSuggestionMedia(ctx context.Context, suggestionID string, media []CreatePlayMediaParams) error {
	return nil
}

func (f *fakeRepository) ListPlayEditSuggestionGenres(ctx context.Context, suggestionID string) ([]PlayGenreRecord, error) {
	return []PlayGenreRecord{}, nil
}

func (f *fakeRepository) ListPlayEditSuggestionMedia(ctx context.Context, suggestionID string) ([]PlayMediaRecord, error) {
	return []PlayMediaRecord{}, nil
}

func (f *fakeRepository) ReplacePlayEditSuggestionGenres(ctx context.Context, suggestionID string, genreIDs []string) error {
	return nil
}

func (f *fakeRepository) CreatePlayEditSuggestionMedia(ctx context.Context, suggestionID string, params CreatePlayMediaParams) (PlayMediaRecord, error) {
	return PlayMediaRecord{}, nil
}

func (f *fakeRepository) GetPlayEditSuggestionMediaByID(ctx context.Context, mediaID string) (PlayMediaRecord, error) {
	return PlayMediaRecord{}, gorm.ErrRecordNotFound
}

func (f *fakeRepository) DeletePlayEditSuggestionMedia(ctx context.Context, mediaID string) error {
	return nil
}

func (f *fakeRepository) ApplyApprovedPlayEditSuggestion(ctx context.Context, suggestionID string, updatedAt time.Time) error {
	return nil
}

func (f *fakeRepository) ApproveSubmission(ctx context.Context, playID string, adminUserID string, now time.Time) (SubmissionRecord, error) {
	if f.approveSubmissionFn != nil {
		return f.approveSubmissionFn(ctx, playID, adminUserID, now)
	}

	return SubmissionRecord{}, nil
}

func (f *fakeRepository) RejectSubmission(ctx context.Context, playID string, adminUserID string, reason string, now time.Time) (SubmissionRecord, error) {
	if f.rejectSubmissionFn != nil {
		return f.rejectSubmissionFn(ctx, playID, adminUserID, reason, now)
	}

	return SubmissionRecord{}, nil
}

func (f *fakeRepository) CreatePlayMedia(ctx context.Context, params CreatePlayMediaParams) (PlayMediaRecord, error) {
	if f.createPlayMediaFn != nil {
		return f.createPlayMediaFn(ctx, params)
	}

	return PlayMediaRecord{}, nil
}

func (f *fakeRepository) GetPlayMediaByID(ctx context.Context, mediaID string) (PlayMediaRecord, error) {
	if f.getPlayMediaByIDFn != nil {
		return f.getPlayMediaByIDFn(ctx, mediaID)
	}

	return PlayMediaRecord{}, gorm.ErrRecordNotFound
}

func (f *fakeRepository) DeletePlayMedia(ctx context.Context, mediaID string) error {
	if f.deletePlayMediaFn != nil {
		return f.deletePlayMediaFn(ctx, mediaID)
	}

	return nil
}

func (f *fakeRepository) SetEngagement(ctx context.Context, userID string, playID string, kind string, createdAt time.Time) error {
	if f.setEngagementFn != nil {
		return f.setEngagementFn(ctx, userID, playID, kind, createdAt)
	}

	return nil
}

func (f *fakeRepository) DeleteEngagement(ctx context.Context, userID string, playID string, kind string) error {
	if f.deleteEngagementFn != nil {
		return f.deleteEngagementFn(ctx, userID, playID, kind)
	}

	return nil
}

func (f *fakeRepository) GetEngagementState(ctx context.Context, userID string, playID string) (EngagementStateRecord, error) {
	if f.engagementStateFn != nil {
		return f.engagementStateFn(ctx, userID, playID)
	}

	return EngagementStateRecord{}, nil
}

func (f *fakeRepository) ListUserEngagementPlays(ctx context.Context, userID string, params ListUserEngagementPlaysParams) ([]EngagementPlayRecord, error) {
	if f.listMyEngagementsFn != nil {
		return f.listMyEngagementsFn(ctx, userID, params)
	}

	return nil, nil
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

func TestServiceFeedTrendingBuildsSectionCursor(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	service := NewService(&fakeRepository{listFeedFn: func(ctx context.Context, params FeedListParams) ([]PlayListRecord, error) {
		if params.Section != "trending" {
			t.Fatalf("expected trending section, got %q", params.Section)
		}

		if params.TrendingAfter != nil {
			t.Fatalf("expected nil trending cursor on first page")
		}

		if params.After != nil {
			t.Fatalf("expected standard cursor to be nil for trending")
		}

		return []PlayListRecord{
			{ID: "00000000-0000-0000-0000-000000000211", Title: "A", TheaterName: "T1", AvailabilityStatus: "in_theaters", PublishedAt: now, TrendScore: 17},
			{ID: "00000000-0000-0000-0000-000000000212", Title: "B", TheaterName: "T2", AvailabilityStatus: "in_theaters", PublishedAt: now.Add(-time.Minute), TrendScore: 8},
		}, nil
	}})

	data, err := service.Feed(context.Background(), FeedQuery{Section: "trending", Limit: 1})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if data.NextCursor == nil || *data.NextCursor == "" {
		t.Fatalf("expected next cursor")
	}

	decoded, err := decodeTrendingFeedCursor(*data.NextCursor)
	if err != nil {
		t.Fatalf("expected valid trending cursor, got error: %v", err)
	}

	if decoded.Section != "trending" {
		t.Fatalf("expected cursor section trending, got %q", decoded.Section)
	}

	if decoded.TrendScore != 17 {
		t.Fatalf("expected trend score 17, got %d", decoded.TrendScore)
	}
}

func TestServiceFeedTrendingRejectsNonTrendingCursor(t *testing.T) {
	t.Parallel()

	cursor, err := encodePlayListCursor(playListCursor{PublishedAt: time.Now().UTC(), PlayID: "00000000-0000-0000-0000-000000000201"})
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	service := NewService(&fakeRepository{})
	_, err = service.Feed(context.Background(), FeedQuery{Section: "trending", Cursor: cursor})
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

func TestServiceFeedTrendingParsesCursorIntoRepositoryParams(t *testing.T) {
	t.Parallel()

	publishedAt := time.Now().UTC().Add(-2 * time.Hour)
	cursor, err := encodeTrendingFeedCursor(trendingFeedCursor{
		Section:     "trending",
		TrendScore:  11,
		PublishedAt: publishedAt,
		PlayID:      "00000000-0000-0000-0000-000000000299",
	})
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	service := NewService(&fakeRepository{listFeedFn: func(ctx context.Context, params FeedListParams) ([]PlayListRecord, error) {
		if params.TrendingAfter == nil {
			t.Fatalf("expected trending cursor to be decoded")
		}

		if params.TrendingAfter.TrendScore != 11 {
			t.Fatalf("expected trend score 11, got %d", params.TrendingAfter.TrendScore)
		}

		if !params.TrendingAfter.PublishedAt.Equal(publishedAt) {
			t.Fatalf("expected cursor publishedAt to match")
		}

		if params.TrendingAfter.PlayID != "00000000-0000-0000-0000-000000000299" {
			t.Fatalf("unexpected cursor playId: %q", params.TrendingAfter.PlayID)
		}

		return []PlayListRecord{}, nil
	}})

	_, err = service.Feed(context.Background(), FeedQuery{Section: "trending", Cursor: cursor, Limit: 10})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestServiceFeedTrendingAcceptsGenreFilter(t *testing.T) {
	t.Parallel()

	genreID := "00000000-0000-0000-0000-000000000101"
	service := NewService(&fakeRepository{listFeedFn: func(ctx context.Context, params FeedListParams) ([]PlayListRecord, error) {
		if params.Section != "trending" {
			t.Fatalf("expected trending section, got %q", params.Section)
		}

		if params.GenreID == nil || *params.GenreID != genreID {
			t.Fatalf("expected trending genreId filter to be forwarded")
		}

		return []PlayListRecord{}, nil
	}})

	_, err := service.Feed(context.Background(), FeedQuery{Section: "trending", GenreID: genreID, Limit: 10})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestServiceFeedUsesCacheHit(t *testing.T) {
	t.Parallel()

	cached := FeedData{Items: []PlayCardData{{ID: "cached-play", Title: "Cached"}}}
	raw, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached feed: %v", err)
	}

	cacheClient := &fakeCache{values: map[string]string{
		buildFeedCacheKey("highlighted", nil, "", 20): string(raw),
	}}

	repoCalls := 0
	service := NewService(
		&fakeRepository{listFeedFn: func(context.Context, FeedListParams) ([]PlayListRecord, error) {
			repoCalls++
			return nil, nil
		}},
		WithCache(cacheClient),
	)

	data, err := service.Feed(context.Background(), FeedQuery{Section: "highlighted"})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(data.Items) != 1 || data.Items[0].ID != "cached-play" {
		t.Fatalf("expected cached feed data, got %+v", data.Items)
	}
	if repoCalls != 0 {
		t.Fatalf("expected repository not called on cache hit, got %d", repoCalls)
	}
}

func TestServiceFeedCachesMissWithSectionTTL(t *testing.T) {
	t.Parallel()

	cacheClient := &fakeCache{values: map[string]string{}}
	now := time.Now().UTC()
	service := NewService(
		&fakeRepository{listFeedFn: func(context.Context, FeedListParams) ([]PlayListRecord, error) {
			return []PlayListRecord{{ID: "00000000-0000-0000-0000-000000000201", Title: "A", TheaterName: "T1", AvailabilityStatus: "in_theaters", PublishedAt: now}}, nil
		}},
		WithCache(cacheClient),
	)

	_, err := service.Feed(context.Background(), FeedQuery{Section: "trending", Limit: 10})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if cacheClient.setCalls != 1 {
		t.Fatalf("expected cache set once, got %d", cacheClient.setCalls)
	}
	if cacheClient.setTTL != feedTrendingTTL {
		t.Fatalf("expected trending ttl %s, got %s", feedTrendingTTL, cacheClient.setTTL)
	}
}

func TestServiceListGenresAlwaysHitsRepo(t *testing.T) {
	t.Parallel()

	repoCalls := 0
	service := NewService(
		&fakeRepository{listGenresFn: func(context.Context, ListGenresParams) ([]GenreRecord, error) {
			repoCalls++
			return []GenreRecord{{ID: "00000000-0000-0000-0000-000000000001", Name: "Drama"}}, nil
		}},
	)

	for i := 0; i < 2; i++ {
		data, err := service.ListGenres(context.Background(), ListGenresQuery{})
		if err != nil {
			t.Fatalf("expected success: %v", err)
		}
		if len(data.Items) != 1 || data.Items[0].Name != "Drama" {
			t.Fatalf("unexpected payload: %+v", data)
		}
	}
	if repoCalls != 2 {
		t.Fatalf("expected repo called twice (no cache), got %d", repoCalls)
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
			return []PlayMediaRecord{{ID: "00000000-0000-0000-0000-000000000401", Kind: "poster", ObjectKey: "plays/00000000-0000-0000-0000-000000000201/poster.jpg", SortOrder: 0}}, nil
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

func TestServiceListReviewsBuildsNextCursor(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	service := NewService(&fakeRepository{listReviewsFn: func(ctx context.Context, params ListReviewsParams) ([]ReviewRecord, error) {
		if params.Limit != 2 {
			t.Fatalf("expected limit 2, got %d", params.Limit)
		}

		return []ReviewRecord{
			{ID: "00000000-0000-0000-0000-000000000501", UserID: "00000000-0000-0000-0000-000000000002", DisplayName: "Ana", Rating: 5, Body: "A", CreatedAt: now, UpdatedAt: now},
			{ID: "00000000-0000-0000-0000-000000000502", UserID: "00000000-0000-0000-0000-000000000003", DisplayName: "Marco", Rating: 4, Body: "B", CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute)},
		}, nil
	}})

	data, err := service.ListReviews(context.Background(), "00000000-0000-0000-0000-000000000201", ListReviewsQuery{Limit: 1})
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

func TestServiceCreateReviewDuplicateReturnsConflict(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{createReviewFn: func(ctx context.Context, userID string, playID string, params CreateReviewParams) (ReviewRecord, error) {
		return ReviewRecord{}, gorm.ErrDuplicatedKey
	}})

	_, err := service.CreateReview(context.Background(), "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000201", CreateReviewRequest{Rating: 5, Body: "Great play"})
	if err == nil {
		t.Fatalf("expected conflict error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "REVIEW_ALREADY_EXISTS" {
		t.Fatalf("expected REVIEW_ALREADY_EXISTS, got %q", appErr.Code)
	}
}

func TestServiceCreateReviewRejectsInvalidRating(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.CreateReview(context.Background(), "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000201", CreateReviewRequest{Rating: 0, Body: "Great play"})
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

func TestServiceSetEngagementAttendedOverridesWishlist(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{
		setEngagementFn: func(ctx context.Context, userID string, playID string, kind string, createdAt time.Time) error {
			if kind != "attended" {
				t.Fatalf("expected attended kind")
			}
			return nil
		},
		engagementStateFn: func(ctx context.Context, userID string, playID string) (EngagementStateRecord, error) {
			return EngagementStateRecord{Wishlist: false, Attended: true}, nil
		},
	})

	data, err := service.SetEngagement(context.Background(), "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000201", SetEngagementRequest{Kind: "attended"})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if data.Wishlist {
		t.Fatalf("expected wishlist false")
	}

	if !data.Attended {
		t.Fatalf("expected attended true")
	}
}

func TestServiceDeleteEngagementIsIdempotent(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{engagementStateFn: func(ctx context.Context, userID string, playID string) (EngagementStateRecord, error) {
		return EngagementStateRecord{Wishlist: false, Attended: false}, nil
	}})

	data, err := service.DeleteEngagement(context.Background(), "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000201", "wishlist")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if data.Wishlist || data.Attended {
		t.Fatalf("expected both engagement flags false")
	}
}

func TestServiceListMyBookmarksBuildsNextCursor(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	service := NewService(&fakeRepository{listMyEngagementsFn: func(ctx context.Context, userID string, params ListUserEngagementPlaysParams) ([]EngagementPlayRecord, error) {
		if userID != "00000000-0000-0000-0000-000000000002" {
			t.Fatalf("unexpected user id")
		}

		if params.Kind != "wishlist" {
			t.Fatalf("expected wishlist kind")
		}

		if params.Limit != 2 {
			t.Fatalf("expected limit 2, got %d", params.Limit)
		}

		return []EngagementPlayRecord{
			{ID: "00000000-0000-0000-0000-000000000201", Title: "A", TheaterName: "T1", AvailabilityStatus: "in_theaters", PublishedAt: now, EngagedAt: now},
			{ID: "00000000-0000-0000-0000-000000000202", Title: "B", TheaterName: "T2", AvailabilityStatus: "archive", PublishedAt: now.Add(-time.Minute), EngagedAt: now.Add(-time.Minute)},
		}, nil
	}})

	data, err := service.ListMyBookmarks(context.Background(), "00000000-0000-0000-0000-000000000002", ListMyEngagementsQuery{Limit: 1})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(data.Items))
	}

	if data.Items[0].EngagedAt == "" {
		t.Fatalf("expected engagedAt in payload")
	}

	if data.NextCursor == nil || *data.NextCursor == "" {
		t.Fatalf("expected next cursor")
	}
}

func TestServiceListMyWatchedRejectsInvalidCursor(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.ListMyWatched(context.Background(), "00000000-0000-0000-0000-000000000002", ListMyEngagementsQuery{Cursor: "%%%"})
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

func TestServiceListUserWatchedReturnsNotFoundWhenUserMissing(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{userExistsFn: func(ctx context.Context, userID string) (bool, error) {
		return false, nil
	}})

	_, err := service.ListUserWatched(context.Background(), "00000000-0000-0000-0000-000000000099", ListMyEngagementsQuery{})
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

func TestServiceListUserReviewsBuildsNextCursor(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	service := NewService(&fakeRepository{
		userExistsFn: func(ctx context.Context, userID string) (bool, error) {
			return true, nil
		},
		listUserReviewsFn: func(ctx context.Context, userID string, params ListUserReviewsParams) ([]UserReviewRecord, error) {
			if params.Limit != 2 {
				t.Fatalf("expected limit 2, got %d", params.Limit)
			}

			return []UserReviewRecord{
				{ID: "00000000-0000-0000-0000-000000000501", PlayID: "00000000-0000-0000-0000-000000000201", PlayTitle: "A", TheaterName: "T1", AvailabilityStatus: "in_theaters", PublishedAt: now, Rating: 5, Body: "Great", CreatedAt: now, UpdatedAt: now},
				{ID: "00000000-0000-0000-0000-000000000502", PlayID: "00000000-0000-0000-0000-000000000202", PlayTitle: "B", TheaterName: "T2", AvailabilityStatus: "archive", PublishedAt: now.Add(-time.Minute), Rating: 4, Body: "Good", CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute)},
			}, nil
		},
	})

	data, err := service.ListUserReviews(context.Background(), "00000000-0000-0000-0000-000000000002", ListUserReviewsQuery{Limit: 1})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(data.Items))
	}

	if data.Items[0].Play.ID == "" {
		t.Fatalf("expected review play context")
	}

	if data.NextCursor == nil || *data.NextCursor == "" {
		t.Fatalf("expected next cursor")
	}
}

func TestServiceListMyReviewsRejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.ListMyReviews(context.Background(), "bad-user-id", ListUserReviewsQuery{})
	if err == nil {
		t.Fatalf("expected unauthorized error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED, got %q", appErr.Code)
	}
}

func TestServiceUpdateReviewRejectsNonOwner(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{getReviewMetadataFn: func(ctx context.Context, reviewID string) (ReviewMetadataRecord, error) {
		return ReviewMetadataRecord{
			ReviewID:           reviewID,
			PlayID:             "00000000-0000-0000-0000-000000000201",
			UserID:             "00000000-0000-0000-0000-000000000099",
			ReviewStatus:       "published",
			PlayCurationStatus: "published",
		}, nil
	}})

	body := "Updated body"
	_, err := service.UpdateReview(context.Background(), "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000501", UpdateReviewRequest{Body: &body})
	if err == nil {
		t.Fatalf("expected forbidden error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestServiceUpdateReviewSuccess(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{
		getReviewMetadataFn: func(ctx context.Context, reviewID string) (ReviewMetadataRecord, error) {
			return ReviewMetadataRecord{
				ReviewID:           reviewID,
				PlayID:             "00000000-0000-0000-0000-000000000201",
				UserID:             "00000000-0000-0000-0000-000000000002",
				ReviewStatus:       "published",
				PlayCurationStatus: "published",
			}, nil
		},
		updateReviewFn: func(ctx context.Context, reviewID string, params UpdateReviewParams) (ReviewRecord, error) {
			if params.Rating == nil || *params.Rating != 4 {
				t.Fatalf("expected rating update")
			}
			return ReviewRecord{ID: reviewID, UserID: "00000000-0000-0000-0000-000000000002", DisplayName: "Ana", Rating: 4, Body: "Edited", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
		},
	})

	rating := 4
	body := "Edited"
	data, err := service.UpdateReview(context.Background(), "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000501", UpdateReviewRequest{Rating: &rating, Body: &body})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if data.Rating != 4 || data.Body != "Edited" {
		t.Fatalf("expected updated review payload")
	}
}

func TestServiceCreateReviewCommentRequiresBody(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.CreateReviewComment(context.Background(), "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000501", CreateReviewCommentRequest{Body: "  "})
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

func TestServiceCreateReviewCommentSuccess(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{
		getReviewMetadataFn: func(ctx context.Context, reviewID string) (ReviewMetadataRecord, error) {
			return ReviewMetadataRecord{
				ReviewID:           reviewID,
				PlayID:             "00000000-0000-0000-0000-000000000201",
				UserID:             "00000000-0000-0000-0000-000000000002",
				ReviewStatus:       "published",
				PlayCurationStatus: "published",
			}, nil
		},
		createCommentFn: func(ctx context.Context, userID string, reviewID string, params CreateReviewCommentParams) (ReviewCommentRecord, error) {
			return ReviewCommentRecord{ID: "00000000-0000-0000-0000-000000000701", ReviewID: reviewID, UserID: userID, DisplayName: "Ana", Body: params.Body, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
		},
	})

	data, err := service.CreateReviewComment(context.Background(), "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000501", CreateReviewCommentRequest{Body: "Great point"})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if data.ID == "" || data.Body != "Great point" {
		t.Fatalf("expected created comment payload")
	}
}

func TestServiceUpdateReviewCommentStatusRequiresAdmin(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.UpdateReviewCommentStatus(context.Background(), "00000000-0000-0000-0000-000000000001", "user", "00000000-0000-0000-0000-000000000701", UpdateReviewCommentStatusRequest{Status: "hidden"})
	if err == nil {
		t.Fatalf("expected forbidden error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestServiceUpdateReviewCommentStatusRejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.UpdateReviewCommentStatus(context.Background(), "00000000-0000-0000-0000-000000000001", "admin", "00000000-0000-0000-0000-000000000701", UpdateReviewCommentStatusRequest{Status: "removed"})
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

func TestServiceUpdateReviewCommentStatusSuccess(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{updateCommentStatusFn: func(ctx context.Context, commentID string, status string, updatedAt time.Time) (ReviewCommentStatusRecord, error) {
		if status != "hidden" {
			t.Fatalf("expected hidden status")
		}

		return ReviewCommentStatusRecord{ID: commentID, ReviewID: "00000000-0000-0000-0000-000000000501", Status: status, UpdatedAt: updatedAt}, nil
	}})

	data, err := service.UpdateReviewCommentStatus(context.Background(), "00000000-0000-0000-0000-000000000001", "admin", "00000000-0000-0000-0000-000000000701", UpdateReviewCommentStatusRequest{Status: "hidden"})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if data.Status != "hidden" {
		t.Fatalf("expected hidden status")
	}
}

func TestServiceCreateSubmissionSuccess(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{
		countGenresByIDsFn: func(ctx context.Context, genreIDs []string) (int64, error) {
			if len(genreIDs) != 1 || genreIDs[0] != "00000000-0000-0000-0000-000000000101" {
				t.Fatalf("expected one normalized genre id")
			}

			return 1, nil
		},
		createSubmissionFn: func(ctx context.Context, userID string, params CreateSubmissionParams) (SubmissionRecord, error) {
			if params.AvailabilityStatus != "in_theaters" {
				t.Fatalf("expected default availability status")
			}

			if len(params.GenreIDs) != 1 || params.GenreIDs[0] != "00000000-0000-0000-0000-000000000101" {
				t.Fatalf("expected one genre id in submission params")
			}

			return SubmissionRecord{ID: "00000000-0000-0000-0000-000000000901", CreatedByUserID: userID, CurationStatus: "pending", Title: params.Title, Synopsis: params.Synopsis, Director: params.Director, DurationMinutes: params.DurationMinutes, TheaterName: params.TheaterName, AvailabilityStatus: params.AvailabilityStatus, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
		},
	})

	data, err := service.CreateSubmission(context.Background(), "00000000-0000-0000-0000-000000000002", CreateSubmissionRequest{Title: "Submission", Synopsis: "Synopsis", Director: "Director", DurationMinutes: 100, TheaterName: "Theater", GenreIDs: []string{"00000000-0000-0000-0000-000000000101"}})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if data.CurationStatus != "pending" {
		t.Fatalf("expected pending curation status")
	}
}

func TestServiceCreateSubmissionRequiresGenres(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.CreateSubmission(context.Background(), "00000000-0000-0000-0000-000000000002", CreateSubmissionRequest{Title: "Submission", Synopsis: "Synopsis", Director: "Director", DurationMinutes: 100, TheaterName: "Theater", GenreIDs: []string{}})
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

func TestServiceUpdateMySubmissionResubmitsRejected(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{
		getSubmissionByIDFn: func(ctx context.Context, playID string) (SubmissionRecord, error) {
			return SubmissionRecord{ID: playID, CreatedByUserID: "00000000-0000-0000-0000-000000000002", CurationStatus: "rejected"}, nil
		},
		updateSubmissionFn: func(ctx context.Context, playID string, params UpdateSubmissionParams) (SubmissionRecord, error) {
			if !params.SetPendingResubmit || !params.ClearModerationAudit {
				t.Fatalf("expected rejected submission resubmission flags")
			}

			return SubmissionRecord{ID: playID, CreatedByUserID: "00000000-0000-0000-0000-000000000002", CurationStatus: "pending", Title: "Updated", Synopsis: "S", Director: "D", DurationMinutes: 100, TheaterName: "T", AvailabilityStatus: "in_theaters", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
		},
	})

	title := "Updated"
	data, err := service.UpdateMySubmission(context.Background(), "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000901", UpdateSubmissionRequest{Title: &title})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if data.CurationStatus != "pending" {
		t.Fatalf("expected pending curation status")
	}
}

func TestServiceListAdminSubmissionsRequiresAdmin(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.ListAdminSubmissions(context.Background(), "00000000-0000-0000-0000-000000000002", "user", ListSubmissionsQuery{})
	if err == nil {
		t.Fatalf("expected forbidden error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestServiceListAdminGenresRequiresAdmin(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.ListAdminGenres(context.Background(), "00000000-0000-0000-0000-000000000001", "user", ListGenresQuery{})
	if err == nil {
		t.Fatalf("expected forbidden error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestServiceCreateAdminGenreRequiresName(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.CreateAdminGenre(context.Background(), "00000000-0000-0000-0000-000000000001", "admin", CreateGenreRequest{Name: "   "})
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

func TestServiceCreateAdminGenreDuplicate(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{createGenreFn: func(ctx context.Context, name string) (GenreRecord, error) {
		return GenreRecord{}, &pgconn.PgError{Code: "23505"}
	}})

	_, err := service.CreateAdminGenre(context.Background(), "00000000-0000-0000-0000-000000000001", "admin", CreateGenreRequest{Name: "Drama"})
	if err == nil {
		t.Fatalf("expected conflict error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "GENRE_ALREADY_EXISTS" {
		t.Fatalf("expected GENRE_ALREADY_EXISTS, got %q", appErr.Code)
	}
}

func TestServiceDeleteAdminGenreInUse(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{deleteGenreFn: func(ctx context.Context, genreID string) error {
		return &pgconn.PgError{Code: "23503"}
	}})

	err := service.DeleteAdminGenre(context.Background(), "00000000-0000-0000-0000-000000000001", "admin", "00000000-0000-0000-0000-000000000101")
	if err == nil {
		t.Fatalf("expected conflict error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "GENRE_IN_USE" {
		t.Fatalf("expected GENRE_IN_USE, got %q", appErr.Code)
	}
}

func TestServiceDeleteAdminGenreNotFound(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{deleteGenreFn: func(ctx context.Context, genreID string) error {
		return gorm.ErrRecordNotFound
	}})

	err := service.DeleteAdminGenre(context.Background(), "00000000-0000-0000-0000-000000000001", "admin", "00000000-0000-0000-0000-000000000101")
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

func TestServiceDeleteAdminPlayRequiresAdmin(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	err := service.DeleteAdminPlay(context.Background(), "00000000-0000-0000-0000-000000000001", "user", "00000000-0000-0000-0000-000000000901")
	if err == nil {
		t.Fatalf("expected forbidden error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestServiceDeleteAdminPlayInvalidID(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	err := service.DeleteAdminPlay(context.Background(), "00000000-0000-0000-0000-000000000001", "admin", "invalid-play-id")
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

func TestServiceDeleteAdminPlayNotFound(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{deletePlayFn: func(ctx context.Context, playID string) error {
		return gorm.ErrRecordNotFound
	}})

	err := service.DeleteAdminPlay(context.Background(), "00000000-0000-0000-0000-000000000001", "admin", "00000000-0000-0000-0000-000000000901")
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

func TestServiceGetAdminSubmissionByIDRequiresAdmin(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.GetAdminSubmissionByID(context.Background(), "00000000-0000-0000-0000-000000000001", "user", "00000000-0000-0000-0000-000000000901")
	if err == nil {
		t.Fatalf("expected forbidden error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestServiceGetAdminSubmissionByIDIncludesMedia(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	repo := &fakeRepository{
		getSubmissionByIDFn: func(ctx context.Context, playID string) (SubmissionRecord, error) {
			return SubmissionRecord{
				ID:              playID,
				Title:           "Hamlet",
				CurationStatus:  "pending",
				CreatedByUserID: "00000000-0000-0000-0000-000000000002",
				CreatedAt:       now,
				UpdatedAt:       now,
			}, nil
		},
		listPlayGenresFn: func(ctx context.Context, playID string) ([]PlayGenreRecord, error) {
			return []PlayGenreRecord{{ID: "00000000-0000-0000-0000-000000000101", Name: "Drama"}}, nil
		},
		listPlayMediaFn: func(ctx context.Context, playID string) ([]PlayMediaRecord, error) {
			return []PlayMediaRecord{
				{ID: "00000000-0000-0000-0000-000000000401", Kind: "poster", ObjectKey: "plays/00000000-0000-0000-0000-000000000901/1.jpg", SortOrder: 0},
				{ID: "00000000-0000-0000-0000-000000000402", Kind: "photo", ObjectKey: "plays/00000000-0000-0000-0000-000000000901/2.jpg", SortOrder: 1},
			}, nil
		},
	}

	service := NewService(repo)
	data, err := service.GetAdminSubmissionByID(context.Background(), "00000000-0000-0000-0000-000000000001", "admin", "00000000-0000-0000-0000-000000000901")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(data.Media) != 2 {
		t.Fatalf("expected 2 media items, got %d", len(data.Media))
	}

	if data.Media[0].Kind != "poster" {
		t.Fatalf("expected first media kind to be poster, got %q", data.Media[0].Kind)
	}

	if data.Media[0].URL == "" {
		t.Fatalf("expected first media item to have a URL")
	}

	if !strings.HasPrefix(data.Media[0].URL, "/v1/media/plays/") {
		t.Fatalf("expected fallback URL path, got %q", data.Media[0].URL)
	}

	if len(data.Genres) != 1 {
		t.Fatalf("expected 1 genre, got %d", len(data.Genres))
	}
}

func TestServiceGetAdminSubmissionByIDPresignedMediaURL(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	repo := &fakeRepository{
		getSubmissionByIDFn: func(ctx context.Context, playID string) (SubmissionRecord, error) {
			return SubmissionRecord{
				ID:              playID,
				Title:           "Hamlet",
				CurationStatus:  "pending",
				CreatedByUserID: "00000000-0000-0000-0000-000000000002",
				CreatedAt:       now,
				UpdatedAt:       now,
			}, nil
		},
		listPlayGenresFn: func(ctx context.Context, playID string) ([]PlayGenreRecord, error) {
			return nil, nil
		},
		listPlayMediaFn: func(ctx context.Context, playID string) ([]PlayMediaRecord, error) {
			return []PlayMediaRecord{
				{ID: "00000000-0000-0000-0000-000000000401", Kind: "poster", ObjectKey: "plays/test-play/1.jpg", SortOrder: 0},
			}, nil
		},
	}

	fakeStorage := &fakePresignStorage{}
	service := NewService(repo, WithMediaStorage(fakeStorage), WithMediaDownloadTTL(15*time.Minute))
	data, err := service.GetAdminSubmissionByID(context.Background(), "00000000-0000-0000-0000-000000000001", "admin", "00000000-0000-0000-0000-000000000901")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(data.Media) != 1 {
		t.Fatalf("expected 1 media item, got %d", len(data.Media))
	}

	if data.Media[0].URL != "https://presigned.example.com/plays/test-play/1.jpg" {
		t.Fatalf("expected presigned URL, got %q", data.Media[0].URL)
	}
}

func TestServiceUpdateAdminSubmissionRequiresPendingStatus(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{getSubmissionByIDFn: func(ctx context.Context, playID string) (SubmissionRecord, error) {
		return SubmissionRecord{ID: playID, CurationStatus: "rejected", CreatedByUserID: "00000000-0000-0000-0000-000000000002"}, nil
	}})

	title := "Updated"
	_, err := service.UpdateAdminSubmission(
		context.Background(),
		"00000000-0000-0000-0000-000000000001",
		"admin",
		"00000000-0000-0000-0000-000000000901",
		UpdateSubmissionRequest{Title: &title},
	)
	if err == nil {
		t.Fatalf("expected conflict error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "INVALID_CURATION_TRANSITION" {
		t.Fatalf("expected INVALID_CURATION_TRANSITION, got %q", appErr.Code)
	}
}

func TestServiceUpdateAdminSubmissionSuccess(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{
		getSubmissionByIDFn: func(ctx context.Context, playID string) (SubmissionRecord, error) {
			return SubmissionRecord{ID: playID, CurationStatus: "pending", CreatedByUserID: "00000000-0000-0000-0000-000000000002"}, nil
		},
		updateSubmissionFn: func(ctx context.Context, playID string, params UpdateSubmissionParams) (SubmissionRecord, error) {
			if params.Title == nil || *params.Title != "Updated" {
				t.Fatalf("expected title patch to be applied")
			}
			if params.UpdatedAt.IsZero() {
				t.Fatalf("expected updated_at to be set")
			}

			return SubmissionRecord{
				ID:                 playID,
				Title:              "Updated",
				Synopsis:           "S",
				Director:           "D",
				DurationMinutes:    100,
				TheaterName:        "T",
				AvailabilityStatus: "in_theaters",
				CurationStatus:     "pending",
				CreatedByUserID:    "00000000-0000-0000-0000-000000000002",
				CreatedAt:          time.Now().UTC(),
				UpdatedAt:          time.Now().UTC(),
			}, nil
		},
	})

	title := "Updated"
	data, err := service.UpdateAdminSubmission(
		context.Background(),
		"00000000-0000-0000-0000-000000000001",
		"admin",
		"00000000-0000-0000-0000-000000000901",
		UpdateSubmissionRequest{Title: &title},
	)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if data.Title != "Updated" {
		t.Fatalf("expected updated title")
	}
}

func TestServiceApproveSubmissionRejectsInvalidTransition(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{getSubmissionByIDFn: func(ctx context.Context, playID string) (SubmissionRecord, error) {
		return SubmissionRecord{ID: playID, CurationStatus: "rejected", CreatedByUserID: "00000000-0000-0000-0000-000000000002"}, nil
	}})

	_, err := service.ApproveSubmission(context.Background(), "00000000-0000-0000-0000-000000000001", "admin", "00000000-0000-0000-0000-000000000901")
	if err == nil {
		t.Fatalf("expected conflict error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "INVALID_CURATION_TRANSITION" {
		t.Fatalf("expected INVALID_CURATION_TRANSITION, got %q", appErr.Code)
	}
}

func TestServiceRejectSubmissionRequiresReason(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.RejectSubmission(context.Background(), "00000000-0000-0000-0000-000000000001", "admin", "00000000-0000-0000-0000-000000000901", RejectSubmissionRequest{Reason: "  "})
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

func TestServiceListMyOwnedPlaysIncludesPosterURLAndPendingSuggestion(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	userID := "00000000-0000-0000-0000-000000000001"
	playID := "00000000-0000-0000-0000-000000000111"
	posterID := "00000000-0000-0000-0000-000000000222"
	suggestionID := "00000000-0000-0000-0000-000000000333"

	svc := NewService(&fakeRepository{
		listOwnedPlaysFn: func(ctx context.Context, uid string, params ListSubmissionsParams) ([]OwnedPlayRecord, error) {
			if uid != userID {
				t.Fatalf("expected userID %q, got %q", userID, uid)
			}

			return []OwnedPlayRecord{
				{
					ID:                 playID,
					Title:              "Play A",
					TheaterName:        "Theater A",
					AvailabilityStatus: "in_theaters",
					PublishedAt:        now,
					PosterMediaID:      &posterID,
					CreatedAt:          now,
				},
			}, nil
		},
		listEditSuggestionsFn: func(ctx context.Context, params ListPlayEditSuggestionsParams) ([]PlayEditSuggestionRecord, error) {
			if params.Status == nil || *params.Status != "pending" {
				t.Fatalf("expected pending status filter for first lookup")
			}

			return []PlayEditSuggestionRecord{
				{
					ID:                 suggestionID,
					PlayID:             playID,
					Title:              "Play A v2",
					Synopsis:           "s",
					Director:           "d",
					DurationMinutes:    90,
					TheaterName:        "Theater A",
					AvailabilityStatus: "in_theaters",
					Status:             "pending",
					CreatedByUserID:    userID,
					CreatedAt:          now,
					UpdatedAt:          now,
				},
			}, nil
		},
	})

	result, err := svc.ListMyOwnedPlays(context.Background(), userID, ListOwnedPlaysQuery{Limit: 10})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.PosterURL == nil {
		t.Fatalf("expected posterUrl to be present")
	}
	if !strings.HasSuffix(*item.PosterURL, "/v1/media/plays/"+playID+"/"+posterID) {
		t.Fatalf("unexpected posterUrl: %q", *item.PosterURL)
	}
	if item.LatestEditSuggestion == nil {
		t.Fatalf("expected latestEditSuggestion to be present")
	}
	if item.LatestEditSuggestion.ID != suggestionID {
		t.Fatalf("expected suggestion %q, got %q", suggestionID, item.LatestEditSuggestion.ID)
	}
	if item.LatestEditSuggestion.Status != "pending" {
		t.Fatalf("expected pending suggestion status, got %q", item.LatestEditSuggestion.Status)
	}
}

func TestServiceListMyOwnedPlaysPosterURLNilWhenNoMedia(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	userID := "00000000-0000-0000-0000-000000000001"
	playID := "00000000-0000-0000-0000-000000000111"

	svc := NewService(&fakeRepository{
		listOwnedPlaysFn: func(ctx context.Context, uid string, params ListSubmissionsParams) ([]OwnedPlayRecord, error) {
			return []OwnedPlayRecord{
				{
					ID:                 playID,
					Title:              "Play A",
					TheaterName:        "Theater A",
					AvailabilityStatus: "in_theaters",
					PublishedAt:        now,
					PosterMediaID:      nil,
					CreatedAt:          now,
				},
			}, nil
		},
		listEditSuggestionsFn: func(ctx context.Context, params ListPlayEditSuggestionsParams) ([]PlayEditSuggestionRecord, error) {
			return nil, nil
		},
	})

	result, err := svc.ListMyOwnedPlays(context.Background(), userID, ListOwnedPlaysQuery{Limit: 10})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].PosterURL != nil {
		t.Fatalf("expected posterUrl to be nil when there is no media")
	}
}

func TestServiceListMyOwnedPlaysFallsBackToLatestSuggestionWhenNoPending(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	userID := "00000000-0000-0000-0000-000000000001"
	playID := "00000000-0000-0000-0000-000000000111"
	approvedSuggestionID := "00000000-0000-0000-0000-000000000444"

	suggestionsCalls := 0
	svc := NewService(&fakeRepository{
		listOwnedPlaysFn: func(ctx context.Context, uid string, params ListSubmissionsParams) ([]OwnedPlayRecord, error) {
			return []OwnedPlayRecord{
				{
					ID:                 playID,
					Title:              "Play A",
					TheaterName:        "Theater A",
					AvailabilityStatus: "in_theaters",
					PublishedAt:        now,
					CreatedAt:          now,
				},
			}, nil
		},
		listEditSuggestionsFn: func(ctx context.Context, params ListPlayEditSuggestionsParams) ([]PlayEditSuggestionRecord, error) {
			suggestionsCalls++

			if params.Status != nil && *params.Status == "pending" {
				return []PlayEditSuggestionRecord{}, nil
			}

			if params.Status != nil {
				t.Fatalf("unexpected status filter: %q", *params.Status)
			}

			return []PlayEditSuggestionRecord{
				{
					ID:                 approvedSuggestionID,
					PlayID:             playID,
					Title:              "Play A approved edit",
					Synopsis:           "s",
					Director:           "d",
					DurationMinutes:    90,
					TheaterName:        "Theater A",
					AvailabilityStatus: "in_theaters",
					Status:             "approved",
					CreatedByUserID:    userID,
					CreatedAt:          now,
					UpdatedAt:          now,
				},
			}, nil
		},
	})

	result, err := svc.ListMyOwnedPlays(context.Background(), userID, ListOwnedPlaysQuery{Limit: 10})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if suggestionsCalls != 2 {
		t.Fatalf("expected 2 suggestion lookups (pending + fallback), got %d", suggestionsCalls)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].LatestEditSuggestion == nil {
		t.Fatalf("expected fallback latest suggestion to be set")
	}
	if result.Items[0].LatestEditSuggestion.ID != approvedSuggestionID {
		t.Fatalf("expected fallback suggestion %q, got %q", approvedSuggestionID, result.Items[0].LatestEditSuggestion.ID)
	}
	if result.Items[0].LatestEditSuggestion.Status != "approved" {
		t.Fatalf("expected approved fallback status, got %q", result.Items[0].LatestEditSuggestion.Status)
	}
}
