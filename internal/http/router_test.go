package apihttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

func TestRouterNoRouteUsesErrorEnvelope(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/unknown/endpoint", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-no-route-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "NOT_FOUND" {
		t.Fatalf("expected code %q, got %q", "NOT_FOUND", body.Error.Code)
	}

	if body.RequestID != "router-no-route-test" {
		t.Fatalf("expected request id %q, got %q", "router-no-route-test", body.RequestID)
	}
}

func TestRouterMeProfileRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/me/profile", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-me-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}

func TestRouterMyFollowingsRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/me/followings", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-me-followings-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}

func TestRouterPlayEngagementRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/plays/00000000-0000-0000-0000-000000000201/engagements", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-play-engagement-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}

func TestRouterReviewPatchRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/v1/reviews/00000000-0000-0000-0000-000000000501", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-review-patch-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}

func TestRouterCreateSubmissionRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/submissions/plays", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-create-submission-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}

func TestRouterAdminSubmissionsRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/submissions/plays", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-admin-submissions-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}

func TestRouterAdminGenresListRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/genres", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-admin-genres-list-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestRouterAdminGenresCreateRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/genres", strings.NewReader(`{"name":"Drama"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(httpx.RequestIDHeader, "router-admin-genres-create-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestRouterAdminGenresDeleteRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/v1/admin/genres/00000000-0000-0000-0000-000000000101", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-admin-genres-delete-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestRouterAdminSubmissionDetailRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/submissions/plays/00000000-0000-0000-0000-000000000901", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-admin-submission-detail-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestRouterAdminSubmissionUpdateRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/v1/admin/submissions/plays/00000000-0000-0000-0000-000000000901", strings.NewReader(`{"title":"Updated"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(httpx.RequestIDHeader, "router-admin-submission-update-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestRouterMySubmissionsRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/me/submissions/plays", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-my-submissions-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}

func TestRouterMyBookmarksRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/me/bookmarks", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-my-bookmarks-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}

func TestRouterMyAvatarUploadRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/me/profile/avatar/uploads", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-my-avatar-upload-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}

func TestRouterMySubmissionMediaUploadRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/me/submissions/plays/00000000-0000-0000-0000-000000000201/media/uploads", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-my-submission-media-upload-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}

func TestRouterMyWatchedRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/me/watched", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-my-watched-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}

func TestRouterMyReviewsRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/me/reviews", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-my-reviews-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}

func TestRouterAdminReviewCommentModerationRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/v1/admin/review-comments/00000000-0000-0000-0000-000000000701/status", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-admin-review-comment-auth-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected code %q, got %q", "UNAUTHORIZED", body.Error.Code)
	}
}
