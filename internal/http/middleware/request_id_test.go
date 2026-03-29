package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

func TestRequestIDGeneratedWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())
	router.GET("/id", func(c *gin.Context) {
		httpx.WriteData(c, http.StatusOK, gin.H{"ok": true})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/id", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	requestIDHeader := recorder.Header().Get(httpx.RequestIDHeader)
	if requestIDHeader == "" {
		t.Fatalf("expected %s response header", httpx.RequestIDHeader)
	}

	var payload httpx.ResponseEnvelope

	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.RequestID != requestIDHeader {
		t.Fatalf("expected response body requestId %q, got %q", requestIDHeader, payload.RequestID)
	}

	if payload.Error != nil {
		t.Fatalf("expected error to be nil")
	}
}

func TestRequestIDUsesIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())
	router.GET("/id", func(c *gin.Context) {
		httpx.WriteData(c, http.StatusOK, gin.H{"ok": true})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/id", nil)
	request.Header.Set(httpx.RequestIDHeader, "external-request-id")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if got := recorder.Header().Get(httpx.RequestIDHeader); got != "external-request-id" {
		t.Fatalf("expected response header request id %q, got %q", "external-request-id", got)
	}

	var payload httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.RequestID != "external-request-id" {
		t.Fatalf("expected response body request id %q, got %q", "external-request-id", payload.RequestID)
	}
}
