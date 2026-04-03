package follows

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
	followFn           func(ctx context.Context, actorUserID string, targetUserID string) (FollowActionData, error)
	unfollowFn         func(ctx context.Context, actorUserID string, targetUserID string) (FollowActionData, error)
	listMyFollowingsFn func(ctx context.Context, actorUserID string, query ListFollowsQuery) (FollowListData, error)
	listFollowersFn    func(ctx context.Context, userID string, query ListFollowsQuery) (FollowListData, error)
	listFollowingsFn   func(ctx context.Context, userID string, query ListFollowsQuery) (FollowListData, error)
}

func (f *fakeService) Follow(ctx context.Context, actorUserID string, targetUserID string) (FollowActionData, error) {
	if f.followFn != nil {
		return f.followFn(ctx, actorUserID, targetUserID)
	}

	return FollowActionData{UserID: targetUserID, Following: true}, nil
}

func (f *fakeService) Unfollow(ctx context.Context, actorUserID string, targetUserID string) (FollowActionData, error) {
	if f.unfollowFn != nil {
		return f.unfollowFn(ctx, actorUserID, targetUserID)
	}

	return FollowActionData{UserID: targetUserID, Following: false}, nil
}

func (f *fakeService) ListMyFollowings(ctx context.Context, actorUserID string, query ListFollowsQuery) (FollowListData, error) {
	if f.listMyFollowingsFn != nil {
		return f.listMyFollowingsFn(ctx, actorUserID, query)
	}

	return FollowListData{}, nil
}

func (f *fakeService) ListUserFollowers(ctx context.Context, userID string, query ListFollowsQuery) (FollowListData, error) {
	if f.listFollowersFn != nil {
		return f.listFollowersFn(ctx, userID, query)
	}

	return FollowListData{}, nil
}

func (f *fakeService) ListUserFollowings(ctx context.Context, userID string, query ListFollowsQuery) (FollowListData, error) {
	if f.listFollowingsFn != nil {
		return f.listFollowingsFn(ctx, userID, query)
	}

	return FollowListData{}, nil
}

func TestHandlerFollowRequiresAuth(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope(), middleware.RequireAccessToken(func(token string) (middleware.AccessTokenClaims, error) {
		return middleware.AccessTokenClaims{}, nil
	}))
	handler := NewHandler(&fakeService{})
	router.POST("/v1/users/:userId/follow", handler.Follow)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/users/00000000-0000-0000-0000-000000000002/follow", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestHandlerFollowSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope(), middleware.RequireAccessToken(func(token string) (middleware.AccessTokenClaims, error) {
		return middleware.AccessTokenClaims{UserID: "00000000-0000-0000-0000-000000000001", Role: "user"}, nil
	}))
	handler := NewHandler(&fakeService{})
	router.POST("/v1/users/:userId/follow", handler.Follow)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/users/00000000-0000-0000-0000-000000000002/follow", nil)
	request.Header.Set("Authorization", "Bearer token")
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

func TestHandlerUserFollowersSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{listFollowersFn: func(ctx context.Context, userID string, query ListFollowsQuery) (FollowListData, error) {
		return FollowListData{Items: []FollowListItemData{{ID: "00000000-0000-0000-0000-000000000001", DisplayName: "Ana", FollowedAt: "2026-03-31T12:00:00Z"}}}, nil
	}})
	router.GET("/v1/users/:userId/followers", handler.UserFollowers)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/users/00000000-0000-0000-0000-000000000002/followers?limit=10", nil)
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

func TestHandlerUserFollowersReturnsNotModifiedWhenETagMatches(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{listFollowersFn: func(ctx context.Context, userID string, query ListFollowsQuery) (FollowListData, error) {
		return FollowListData{Items: []FollowListItemData{{ID: "00000000-0000-0000-0000-000000000001", DisplayName: "Ana", FollowedAt: "2026-03-31T12:00:00Z"}}}, nil
	}})
	router.GET("/v1/users/:userId/followers", handler.UserFollowers)

	firstRecorder := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodGet, "/v1/users/00000000-0000-0000-0000-000000000002/followers?limit=10", nil)
	router.ServeHTTP(firstRecorder, firstRequest)

	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, firstRecorder.Code)
	}

	etag := firstRecorder.Header().Get(httpx.ETagHeader)
	if etag == "" {
		t.Fatalf("expected etag header")
	}

	secondRecorder := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodGet, "/v1/users/00000000-0000-0000-0000-000000000002/followers?limit=10", nil)
	secondRequest.Header.Set(httpx.IfNoneMatchHeader, etag)
	router.ServeHTTP(secondRecorder, secondRequest)

	if secondRecorder.Code != http.StatusNotModified {
		t.Fatalf("expected status %d, got %d", http.StatusNotModified, secondRecorder.Code)
	}
}
