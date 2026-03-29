package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/butaqueando/api/internal/http/middleware"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

type stubChecker struct {
	result CheckResult
}

func (s stubChecker) Check(ctx context.Context) CheckResult {
	return s.result
}

func TestHandlerGetHealthy(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	h := &Handler{service: stubChecker{result: CheckResult{Ready: true, Database: true}}}
	router.GET("/v1/health", h.Get)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	request.Header.Set(httpx.RequestIDHeader, "health-healthy-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	assertHealthSuccessEnvelope(t, recorder.Body.Bytes(), "health-healthy-test", true, true)
}

func TestHandlerGetUnhealthy(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	h := &Handler{service: stubChecker{result: CheckResult{Ready: false, Database: false}}}
	router.GET("/v1/health", h.Get)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	request.Header.Set(httpx.RequestIDHeader, "health-unhealthy-test")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var body struct {
		Data  any `json:"data"`
		Error *struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
		RequestID string `json:"requestId"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Data != nil {
		t.Fatalf("expected data to be nil")
	}

	if body.Error == nil {
		t.Fatalf("expected error payload")
	}

	if body.Error.Code != "SERVICE_UNAVAILABLE" {
		t.Fatalf("expected code %q, got %q", "SERVICE_UNAVAILABLE", body.Error.Code)
	}

	if body.Error.Message != "service unavailable" {
		t.Fatalf("expected message %q, got %q", "service unavailable", body.Error.Message)
	}

	if body.RequestID != "health-unhealthy-test" {
		t.Fatalf("expected request id %q, got %q", "health-unhealthy-test", body.RequestID)
	}

	database, ok := body.Error.Details["database"].(bool)
	if !ok {
		t.Fatalf("expected details.database to be boolean")
	}

	if database {
		t.Fatalf("expected details.database false, got true")
	}
}

func assertHealthSuccessEnvelope(t *testing.T, payload []byte, expectedRequestID string, expectedReady bool, expectedDB bool) {
	t.Helper()

	var body struct {
		Data struct {
			OK        bool   `json:"ok"`
			Database  bool   `json:"database"`
			Timestamp string `json:"timestamp"`
		} `json:"data"`
		Error     any    `json:"error"`
		RequestID string `json:"requestId"`
	}

	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error != nil {
		t.Fatalf("expected error nil")
	}

	if body.RequestID != expectedRequestID {
		t.Fatalf("expected request id %q, got %q", expectedRequestID, body.RequestID)
	}

	if body.Data.OK != expectedReady {
		t.Fatalf("expected ok %t, got %t", expectedReady, body.Data.OK)
	}

	if body.Data.Database != expectedDB {
		t.Fatalf("expected database %t, got %t", expectedDB, body.Data.Database)
	}

	if _, err := time.Parse(time.RFC3339Nano, body.Data.Timestamp); err != nil {
		t.Fatalf("timestamp is not RFC3339Nano: %v", err)
	}
}
