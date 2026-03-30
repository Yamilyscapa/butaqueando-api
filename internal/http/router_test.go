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
