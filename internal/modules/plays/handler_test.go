package plays

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/butaqueando/api/internal/http/middleware"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

type fakeService struct {
	feedFn          func(ctx context.Context, query FeedQuery) (FeedData, error)
	searchFn        func(ctx context.Context, query SearchQuery) (SearchData, error)
	getByID         func(ctx context.Context, playID string) (PlayDetailsData, error)
	listReviewsFn   func(ctx context.Context, playID string, query ListReviewsQuery) (ReviewListData, error)
	createReviewFn  func(ctx context.Context, userID string, playID string, req CreateReviewRequest) (ReviewData, error)
	updateReviewFn  func(ctx context.Context, userID string, reviewID string, req UpdateReviewRequest) (ReviewData, error)
	createCommentFn func(ctx context.Context, userID string, reviewID string, req CreateReviewCommentRequest) (ReviewCommentData, error)
	createSubFn     func(ctx context.Context, userID string, req CreateSubmissionRequest) (SubmissionData, error)
	listMySubsFn    func(ctx context.Context, userID string, query ListSubmissionsQuery) (SubmissionListData, error)
	updateMySubFn   func(ctx context.Context, userID string, playID string, req UpdateSubmissionRequest) (SubmissionData, error)
	listAdminSubsFn func(ctx context.Context, userID string, role string, query ListSubmissionsQuery) (SubmissionListData, error)
	approveSubFn    func(ctx context.Context, userID string, role string, playID string) (SubmissionData, error)
	rejectSubFn     func(ctx context.Context, userID string, role string, playID string, req RejectSubmissionRequest) (SubmissionData, error)
	setEngagementFn func(ctx context.Context, userID string, playID string, req SetEngagementRequest) (EngagementStateData, error)
	deleteEngageFn  func(ctx context.Context, userID string, playID string, kind string) (EngagementStateData, error)
}

func (f *fakeService) Feed(ctx context.Context, query FeedQuery) (FeedData, error) {
	if f.feedFn != nil {
		return f.feedFn(ctx, query)
	}

	return FeedData{}, nil
}

func (f *fakeService) Search(ctx context.Context, query SearchQuery) (SearchData, error) {
	if f.searchFn != nil {
		return f.searchFn(ctx, query)
	}

	return SearchData{}, nil
}

func (f *fakeService) GetByID(ctx context.Context, playID string) (PlayDetailsData, error) {
	if f.getByID != nil {
		return f.getByID(ctx, playID)
	}

	return PlayDetailsData{}, nil
}

func (f *fakeService) ListReviews(ctx context.Context, playID string, query ListReviewsQuery) (ReviewListData, error) {
	if f.listReviewsFn != nil {
		return f.listReviewsFn(ctx, playID, query)
	}

	return ReviewListData{}, nil
}

func (f *fakeService) CreateReview(ctx context.Context, userID string, playID string, req CreateReviewRequest) (ReviewData, error) {
	if f.createReviewFn != nil {
		return f.createReviewFn(ctx, userID, playID, req)
	}

	return ReviewData{}, nil
}

func (f *fakeService) UpdateReview(ctx context.Context, userID string, reviewID string, req UpdateReviewRequest) (ReviewData, error) {
	if f.updateReviewFn != nil {
		return f.updateReviewFn(ctx, userID, reviewID, req)
	}

	return ReviewData{}, nil
}

func (f *fakeService) CreateReviewComment(ctx context.Context, userID string, reviewID string, req CreateReviewCommentRequest) (ReviewCommentData, error) {
	if f.createCommentFn != nil {
		return f.createCommentFn(ctx, userID, reviewID, req)
	}

	return ReviewCommentData{}, nil
}

func (f *fakeService) CreateSubmission(ctx context.Context, userID string, req CreateSubmissionRequest) (SubmissionData, error) {
	if f.createSubFn != nil {
		return f.createSubFn(ctx, userID, req)
	}

	return SubmissionData{}, nil
}

func (f *fakeService) ListMySubmissions(ctx context.Context, userID string, query ListSubmissionsQuery) (SubmissionListData, error) {
	if f.listMySubsFn != nil {
		return f.listMySubsFn(ctx, userID, query)
	}

	return SubmissionListData{}, nil
}

func (f *fakeService) UpdateMySubmission(ctx context.Context, userID string, playID string, req UpdateSubmissionRequest) (SubmissionData, error) {
	if f.updateMySubFn != nil {
		return f.updateMySubFn(ctx, userID, playID, req)
	}

	return SubmissionData{}, nil
}

func (f *fakeService) ListAdminSubmissions(ctx context.Context, userID string, role string, query ListSubmissionsQuery) (SubmissionListData, error) {
	if f.listAdminSubsFn != nil {
		return f.listAdminSubsFn(ctx, userID, role, query)
	}

	return SubmissionListData{}, nil
}

func (f *fakeService) ApproveSubmission(ctx context.Context, userID string, role string, playID string) (SubmissionData, error) {
	if f.approveSubFn != nil {
		return f.approveSubFn(ctx, userID, role, playID)
	}

	return SubmissionData{}, nil
}

func (f *fakeService) RejectSubmission(ctx context.Context, userID string, role string, playID string, req RejectSubmissionRequest) (SubmissionData, error) {
	if f.rejectSubFn != nil {
		return f.rejectSubFn(ctx, userID, role, playID, req)
	}

	return SubmissionData{}, nil
}

func (f *fakeService) SetEngagement(ctx context.Context, userID string, playID string, req SetEngagementRequest) (EngagementStateData, error) {
	if f.setEngagementFn != nil {
		return f.setEngagementFn(ctx, userID, playID, req)
	}

	return EngagementStateData{}, nil
}

func (f *fakeService) DeleteEngagement(ctx context.Context, userID string, playID string, kind string) (EngagementStateData, error) {
	if f.deleteEngageFn != nil {
		return f.deleteEngageFn(ctx, userID, playID, kind)
	}

	return EngagementStateData{}, nil
}

func TestHandlerFeedSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{feedFn: func(ctx context.Context, query FeedQuery) (FeedData, error) {
		return FeedData{Items: []PlayCardData{{ID: "00000000-0000-0000-0000-000000000201", Title: "Hamlet"}}}, nil
	}})
	router.GET("/v1/feed", handler.Feed)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/feed?section=highlighted&limit=10", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error != nil {
		t.Fatalf("expected no error payload")
	}
}

func TestHandlerFeedInvalidLimitReturnsValidation(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{})
	router.GET("/v1/feed", handler.Feed)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/feed?section=highlighted&limit=abc", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil || body.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected validation error payload")
	}
}

func TestHandlerSearchSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{searchFn: func(ctx context.Context, query SearchQuery) (SearchData, error) {
		return SearchData{Items: []PlayCardData{{ID: "00000000-0000-0000-0000-000000000202", Title: "Bernarda"}}}, nil
	}})
	router.GET("/v1/search", handler.Search)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/search?q=bernar", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestHandlerGetByIDSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{getByID: func(ctx context.Context, playID string) (PlayDetailsData, error) {
		return PlayDetailsData{ID: playID, Title: "Hamlet"}, nil
	}})
	router.GET("/v1/plays/:playId", handler.GetByID)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/plays/00000000-0000-0000-0000-000000000201", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestHandlerCreateReviewRequiresAuth(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{})
	router.POST("/v1/plays/:playId/reviews", handler.CreateReview)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/plays/00000000-0000-0000-0000-000000000201/reviews", strings.NewReader(`{"rating":5,"body":"great"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestHandlerCreateReviewSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope(), middleware.RequireAccessToken(func(token string) (middleware.AccessTokenClaims, error) {
		return middleware.AccessTokenClaims{UserID: "00000000-0000-0000-0000-000000000002", Role: "user"}, nil
	}))
	handler := NewHandler(&fakeService{createReviewFn: func(ctx context.Context, userID string, playID string, req CreateReviewRequest) (ReviewData, error) {
		return ReviewData{ID: "00000000-0000-0000-0000-000000000501", UserID: userID, Rating: req.Rating, Body: req.Body}, nil
	}})
	router.POST("/v1/plays/:playId/reviews", handler.CreateReview)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/plays/00000000-0000-0000-0000-000000000201/reviews", strings.NewReader(`{"rating":5,"body":"great"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer token")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
}

func TestHandlerSetEngagementRequiresAuth(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{})
	router.POST("/v1/plays/:playId/engagements", handler.SetEngagement)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/plays/00000000-0000-0000-0000-000000000201/engagements", strings.NewReader(`{"kind":"wishlist"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestHandlerUpdateReviewRequiresAuth(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{})
	router.PATCH("/v1/reviews/:reviewId", handler.UpdateReview)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/v1/reviews/00000000-0000-0000-0000-000000000501", strings.NewReader(`{"body":"edited"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestHandlerCreateReviewCommentSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope(), middleware.RequireAccessToken(func(token string) (middleware.AccessTokenClaims, error) {
		return middleware.AccessTokenClaims{UserID: "00000000-0000-0000-0000-000000000002", Role: "user"}, nil
	}))
	handler := NewHandler(&fakeService{createCommentFn: func(ctx context.Context, userID string, reviewID string, req CreateReviewCommentRequest) (ReviewCommentData, error) {
		return ReviewCommentData{ID: "00000000-0000-0000-0000-000000000701", UserID: userID, ReviewID: reviewID, Body: req.Body}, nil
	}})
	router.POST("/v1/reviews/:reviewId/comments", handler.CreateReviewComment)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/reviews/00000000-0000-0000-0000-000000000501/comments", strings.NewReader(`{"body":"nice take"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer token")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
}

func TestHandlerCreateSubmissionRequiresAuth(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{})
	router.POST("/v1/submissions/plays", handler.CreateSubmission)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/submissions/plays", strings.NewReader(`{"title":"New play","synopsis":"syn","director":"dir","durationMinutes":90,"theaterName":"theater"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestHandlerCreateSubmissionSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope(), middleware.RequireAccessToken(func(token string) (middleware.AccessTokenClaims, error) {
		return middleware.AccessTokenClaims{UserID: "00000000-0000-0000-0000-000000000002", Role: "user"}, nil
	}))
	handler := NewHandler(&fakeService{createSubFn: func(ctx context.Context, userID string, req CreateSubmissionRequest) (SubmissionData, error) {
		return SubmissionData{ID: "00000000-0000-0000-0000-000000000901", CreatedByUserID: userID, CurationStatus: "pending", Title: req.Title}, nil
	}})
	router.POST("/v1/submissions/plays", handler.CreateSubmission)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/submissions/plays", strings.NewReader(`{"title":"New play","synopsis":"syn","director":"dir","durationMinutes":90,"theaterName":"theater"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer token")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
}

func TestHandlerApproveSubmissionRequiresAuth(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{})
	router.POST("/v1/admin/submissions/plays/:playId/approve", handler.ApproveSubmission)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/submissions/plays/00000000-0000-0000-0000-000000000901/approve", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}
