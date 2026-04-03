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
	signUpFn             func(ctx context.Context, req SignUpRequest) (SignUpData, error)
	signInFn             func(ctx context.Context, req SignInRequest) (AuthTokensData, error)
	refreshFn            func(ctx context.Context, req RefreshRequest) (AuthTokensData, error)
	signOutFn            func(ctx context.Context, req SignOutRequest) (SignOutData, error)
	verifyEmailFn        func(ctx context.Context, req VerifyEmailRequest) (VerifyEmailData, error)
	resendVerificationFn func(ctx context.Context, req ResendVerificationRequest) (ResendVerificationData, error)
	forgotPasswordFn     func(ctx context.Context, req ForgotPasswordRequest) (ForgotPasswordData, error)
	resetPasswordFn      func(ctx context.Context, req ResetPasswordRequest) (ResetPasswordData, error)
}

func (f *fakeService) SignUp(ctx context.Context, req SignUpRequest) (SignUpData, error) {
	if f.signUpFn == nil {
		return SignUpData{}, nil
	}

	return f.signUpFn(ctx, req)
}

func (f *fakeService) SignIn(ctx context.Context, req SignInRequest) (AuthTokensData, error) {
	if f.signInFn == nil {
		return AuthTokensData{}, nil
	}

	return f.signInFn(ctx, req)
}

func (f *fakeService) Refresh(ctx context.Context, req RefreshRequest) (AuthTokensData, error) {
	if f.refreshFn == nil {
		return AuthTokensData{}, nil
	}

	return f.refreshFn(ctx, req)
}

func (f *fakeService) SignOut(ctx context.Context, req SignOutRequest) (SignOutData, error) {
	if f.signOutFn == nil {
		return SignOutData{}, nil
	}

	return f.signOutFn(ctx, req)
}

func (f *fakeService) VerifyEmail(ctx context.Context, req VerifyEmailRequest) (VerifyEmailData, error) {
	if f.verifyEmailFn == nil {
		return VerifyEmailData{}, nil
	}

	return f.verifyEmailFn(ctx, req)
}

func (f *fakeService) ResendVerification(ctx context.Context, req ResendVerificationRequest) (ResendVerificationData, error) {
	if f.resendVerificationFn == nil {
		return ResendVerificationData{}, nil
	}

	return f.resendVerificationFn(ctx, req)
}

func (f *fakeService) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) (ForgotPasswordData, error) {
	if f.forgotPasswordFn == nil {
		return ForgotPasswordData{}, nil
	}

	return f.forgotPasswordFn(ctx, req)
}

func (f *fakeService) ResetPassword(ctx context.Context, req ResetPasswordRequest) (ResetPasswordData, error) {
	if f.resetPasswordFn == nil {
		return ResetPasswordData{}, nil
	}

	return f.resetPasswordFn(ctx, req)
}

func TestHandlerSignUpSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{signUpFn: func(ctx context.Context, req SignUpRequest) (SignUpData, error) {
		return SignUpData{UserID: "user-1", Email: "new@butaqueando.local", EmailVerificationRequired: true}, nil
	}})
	router.POST("/v1/auth/sign-up", handler.SignUp)

	body := bytes.NewBufferString(`{"displayName":"New User","email":"new@butaqueando.local","password":"password123"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/sign-up", body)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(httpx.RequestIDHeader, "auth-handler-signup")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var response httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error != nil {
		t.Fatalf("expected error nil")
	}

	if response.RequestID != "auth-handler-signup" {
		t.Fatalf("expected request id %q, got %q", "auth-handler-signup", response.RequestID)
	}
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

func TestHandlerVerifyEmailSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{verifyEmailFn: func(ctx context.Context, req VerifyEmailRequest) (VerifyEmailData, error) {
		return VerifyEmailData{OK: true}, nil
	}})
	router.POST("/v1/auth/verify-email", handler.VerifyEmail)

	body := bytes.NewBufferString(`{"token":"verification-token"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/verify-email", body)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestHandlerResendVerificationSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{resendVerificationFn: func(ctx context.Context, req ResendVerificationRequest) (ResendVerificationData, error) {
		return ResendVerificationData{OK: true}, nil
	}})
	router.POST("/v1/auth/resend-verification", handler.ResendVerification)

	body := bytes.NewBufferString(`{"email":"ana@butaqueando.local"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/resend-verification", body)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestHandlerForgotPasswordSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{forgotPasswordFn: func(ctx context.Context, req ForgotPasswordRequest) (ForgotPasswordData, error) {
		return ForgotPasswordData{OK: true}, nil
	}})
	router.POST("/v1/auth/forgot-password", handler.ForgotPassword)

	body := bytes.NewBufferString(`{"email":"ana@butaqueando.local"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/forgot-password", body)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestHandlerResetPasswordSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope())
	handler := NewHandler(&fakeService{resetPasswordFn: func(ctx context.Context, req ResetPasswordRequest) (ResetPasswordData, error) {
		return ResetPasswordData{OK: true}, nil
	}})
	router.POST("/v1/auth/reset-password", handler.ResetPassword)

	body := bytes.NewBufferString(`{"token":"reset-token","newPassword":"password123"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/reset-password", body)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}
