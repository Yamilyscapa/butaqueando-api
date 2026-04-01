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
	createSubmissionFn     func(ctx context.Context, userID string, params CreateSubmissionParams) (SubmissionRecord, error)
	listUserSubmissionsFn  func(ctx context.Context, userID string, params ListSubmissionsParams) ([]SubmissionRecord, error)
	getSubmissionByIDFn    func(ctx context.Context, playID string) (SubmissionRecord, error)
	updateSubmissionFn     func(ctx context.Context, playID string, params UpdateSubmissionParams) (SubmissionRecord, error)
	listAdminSubmissionsFn func(ctx context.Context, params ListSubmissionsParams) ([]SubmissionRecord, error)
	approveSubmissionFn    func(ctx context.Context, playID string, adminUserID string, now time.Time) (SubmissionRecord, error)
	rejectSubmissionFn     func(ctx context.Context, playID string, adminUserID string, reason string, now time.Time) (SubmissionRecord, error)
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

func (f *fakeRepository) ListAdminSubmissions(ctx context.Context, params ListSubmissionsParams) ([]SubmissionRecord, error) {
	if f.listAdminSubmissionsFn != nil {
		return f.listAdminSubmissionsFn(ctx, params)
	}

	return nil, nil
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

func TestServiceCreateSubmissionSuccess(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{createSubmissionFn: func(ctx context.Context, userID string, params CreateSubmissionParams) (SubmissionRecord, error) {
		if params.AvailabilityStatus != "in_theaters" {
			t.Fatalf("expected default availability status")
		}

		return SubmissionRecord{ID: "00000000-0000-0000-0000-000000000901", CreatedByUserID: userID, CurationStatus: "pending", Title: params.Title, Synopsis: params.Synopsis, Director: params.Director, DurationMinutes: params.DurationMinutes, TheaterName: params.TheaterName, AvailabilityStatus: params.AvailabilityStatus, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
	}})

	data, err := service.CreateSubmission(context.Background(), "00000000-0000-0000-0000-000000000002", CreateSubmissionRequest{Title: "Submission", Synopsis: "Synopsis", Director: "Director", DurationMinutes: 100, TheaterName: "Theater"})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if data.CurationStatus != "pending" {
		t.Fatalf("expected pending curation status")
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
