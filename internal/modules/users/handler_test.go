package users

import (
	"bytes"
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
	getPublicProfileFn func(ctx context.Context, userID string) (PublicProfileData, error)
	getMeProfileFn     func(ctx context.Context, userID string) (MeProfileData, error)
	updateMeProfileFn  func(ctx context.Context, userID string, req UpdateMeProfileRequest) (MeProfileData, error)
}

func (f *fakeService) GetPublicProfile(ctx context.Context, userID string) (PublicProfileData, error) {
	if f.getPublicProfileFn != nil {
		return f.getPublicProfileFn(ctx, userID)
	}

	return PublicProfileData{}, nil
}

func (f *fakeService) GetMeProfile(ctx context.Context, userID string) (MeProfileData, error) {
	if f.getMeProfileFn != nil {
		return f.getMeProfileFn(ctx, userID)
	}

	return MeProfileData{}, nil
}

func (f *fakeService) UpdateMeProfile(ctx context.Context, userID string, req UpdateMeProfileRequest) (MeProfileData, error) {
	if f.updateMeProfileFn != nil {
		return f.updateMeProfileFn(ctx, userID, req)
	}

	return MeProfileData{}, nil
}

func TestHandlerGetMeUnauthorizedWithoutToken(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope(), middleware.RequireAccessToken(func(token string) (middleware.AccessTokenClaims, error) {
		return middleware.AccessTokenClaims{}, nil
	}))
	handler := NewHandler(&fakeService{})
	router.GET("/v1/me/profile", handler.GetMe)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/me/profile", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestHandlerGetProfileSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{getPublicProfileFn: func(ctx context.Context, userID string) (PublicProfileData, error) {
		return PublicProfileData{ID: userID, DisplayName: "Public User"}, nil
	}})
	router.GET("/v1/users/:userId/profile", handler.GetProfile)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/users/00000000-0000-0000-0000-000000000002/profile", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error != nil {
		t.Fatalf("expected no error payload")
	}
}

func TestHandlerGetMeSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope(), middleware.RequireAccessToken(func(token string) (middleware.AccessTokenClaims, error) {
		return middleware.AccessTokenClaims{UserID: "00000000-0000-0000-0000-000000000002", Role: "user"}, nil
	}))
	handler := NewHandler(&fakeService{getMeProfileFn: func(ctx context.Context, userID string) (MeProfileData, error) {
		return MeProfileData{ID: userID, DisplayName: "Ana", Email: "ana@example.com", Role: "user"}, nil
	}})
	router.GET("/v1/me/profile", handler.GetMe)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/me/profile", nil)
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set(httpx.RequestIDHeader, "users-get-me")
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

func TestHandlerUpdateMeSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope(), middleware.RequireAccessToken(func(token string) (middleware.AccessTokenClaims, error) {
		return middleware.AccessTokenClaims{UserID: "00000000-0000-0000-0000-000000000002", Role: "user"}, nil
	}))
	handler := NewHandler(&fakeService{updateMeProfileFn: func(ctx context.Context, userID string, req UpdateMeProfileRequest) (MeProfileData, error) {
		return MeProfileData{ID: userID, DisplayName: "Ana Updated", Email: "ana@example.com", Role: "user", Bio: req.Bio}, nil
	}})
	router.PATCH("/v1/me/profile", handler.UpdateMe)

	body := bytes.NewBufferString(`{"bio":"updated bio"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/v1/me/profile", body)
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error != nil {
		t.Fatalf("expected no error payload")
	}
}

func TestHandlerGetProfileReturnsNotModifiedWhenETagMatches(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{getPublicProfileFn: func(ctx context.Context, userID string) (PublicProfileData, error) {
		return PublicProfileData{ID: userID, DisplayName: "Public User"}, nil
	}})
	router.GET("/v1/users/:userId/profile", handler.GetProfile)

	firstRecorder := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodGet, "/v1/users/00000000-0000-0000-0000-000000000002/profile", nil)
	router.ServeHTTP(firstRecorder, firstRequest)

	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, firstRecorder.Code)
	}

	etag := firstRecorder.Header().Get(httpx.ETagHeader)
	if etag == "" {
		t.Fatalf("expected etag header")
	}

	secondRecorder := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodGet, "/v1/users/00000000-0000-0000-0000-000000000002/profile", nil)
	secondRequest.Header.Set(httpx.IfNoneMatchHeader, etag)
	router.ServeHTTP(secondRecorder, secondRequest)

	if secondRecorder.Code != http.StatusNotModified {
		t.Fatalf("expected status %d, got %d", http.StatusNotModified, secondRecorder.Code)
	}
}
