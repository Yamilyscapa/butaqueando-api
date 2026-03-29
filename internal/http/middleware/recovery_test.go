package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

func TestRecoveryReturnsErrorEnvelopeOnPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID(), Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("unexpected failure")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
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

	if body.RequestID == "" {
		t.Fatalf("expected request id")
	}
}
