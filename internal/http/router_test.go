package apihttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

func TestRouterNotImplementedUsesErrorEnvelope(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	request.Header.Set(httpx.RequestIDHeader, "router-not-impl-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("expected status %d, got %d", http.StatusNotImplemented, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "NOT_IMPLEMENTED" {
		t.Fatalf("expected code %q, got %q", "NOT_IMPLEMENTED", body.Error.Code)
	}

	if body.RequestID != "router-not-impl-test" {
		t.Fatalf("expected request id %q, got %q", "router-not-impl-test", body.RequestID)
	}
}

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
	request := httptest.NewRequest(http.MethodGet, "/v1/users/me/profile", nil)
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
	request := httptest.NewRequest(http.MethodGet, "/v1/users/me/followings", nil)
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

func TestRouterMySubmissionsRequiresAuthorization(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{DB: nil})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/users/me/submissions/plays", nil)
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
