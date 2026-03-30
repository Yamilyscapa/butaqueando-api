package auth

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
	signInFn  func(ctx context.Context, req SignInRequest) (AuthTokensData, error)
	refreshFn func(ctx context.Context, req RefreshRequest) (AuthTokensData, error)
	signOutFn func(ctx context.Context, req SignOutRequest) (SignOutData, error)
}

func (f *fakeService) SignIn(ctx context.Context, req SignInRequest) (AuthTokensData, error) {
	return f.signInFn(ctx, req)
}

func (f *fakeService) Refresh(ctx context.Context, req RefreshRequest) (AuthTokensData, error) {
	return f.refreshFn(ctx, req)
}

func (f *fakeService) SignOut(ctx context.Context, req SignOutRequest) (SignOutData, error) {
	return f.signOutFn(ctx, req)
}

func TestHandlerSignInSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{signInFn: func(ctx context.Context, req SignInRequest) (AuthTokensData, error) {
		return AuthTokensData{TokenType: "Bearer", AccessToken: "access", RefreshToken: "refresh", AccessTokenExpiresIn: 900, RefreshTokenExpiresIn: 3600}, nil
	}})
	router.POST("/v1/auth/sign-in", handler.SignIn)

	body := bytes.NewBufferString(`{"email":"ana@butaqueando.local","password":"password"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/sign-in", body)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(httpx.RequestIDHeader, "auth-handler-signin")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error != nil {
		t.Fatalf("expected error nil")
	}

	if response.RequestID != "auth-handler-signin" {
		t.Fatalf("expected request id %q, got %q", "auth-handler-signin", response.RequestID)
	}
}

func TestHandlerSignInValidationError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{signInFn: func(ctx context.Context, req SignInRequest) (AuthTokensData, error) {
		return AuthTokensData{}, nil
	}})
	router.POST("/v1/auth/sign-in", handler.SignIn)

	body := bytes.NewBufferString(`{"email":"not-an-email"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/sign-in", body)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var response httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error == nil {
		t.Fatalf("expected error payload")
	}

	if response.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected error code %q, got %q", "VALIDATION_ERROR", response.Error.Code)
	}
}
