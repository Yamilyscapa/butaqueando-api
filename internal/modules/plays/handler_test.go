package plays

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/butaqueando/api/internal/http/middleware"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

type fakeService struct {
	feedFn   func(ctx context.Context, query FeedQuery) (FeedData, error)
	searchFn func(ctx context.Context, query SearchQuery) (SearchData, error)
	getByID  func(ctx context.Context, playID string) (PlayDetailsData, error)
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
