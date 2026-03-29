package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

func TestErrorEnvelopeMapsAppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID(), ErrorEnvelope())
	router.GET("/app", func(c *gin.Context) {
		_ = c.Error(sharederrors.Validation("invalid payload", gin.H{"field": "email"}))
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/app", nil)
	request.Header.Set(httpx.RequestIDHeader, "request-id-123")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Data != nil {
		t.Fatalf("expected data to be nil")
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected code %q, got %q", "VALIDATION_ERROR", body.Error.Code)
	}

	if body.Error.Message != "invalid payload" {
		t.Fatalf("expected message %q, got %q", "invalid payload", body.Error.Message)
	}

	if body.RequestID != "request-id-123" {
		t.Fatalf("expected request id %q, got %q", "request-id-123", body.RequestID)
	}
}

func TestErrorEnvelopeMapsUnknownError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID(), ErrorEnvelope())
	router.GET("/app", func(c *gin.Context) {
		_ = c.Error(errors.New("boom"))
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/app", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Data != nil {
		t.Fatalf("expected data to be nil")
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "INTERNAL_ERROR" {
		t.Fatalf("expected code %q, got %q", "INTERNAL_ERROR", body.Error.Code)
	}

	if body.Error.Message != "internal server error" {
		t.Fatalf("expected message %q, got %q", "internal server error", body.Error.Message)
	}

	if body.RequestID == "" {
		t.Fatalf("expected request id")
	}
}
