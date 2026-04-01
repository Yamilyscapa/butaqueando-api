package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/butaqueando/api/internal/http/middleware"
	authmodule "github.com/butaqueando/api/internal/modules/auth"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

func TestRequireAccessTokenMissingHeader(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope(), middleware.RequireAccessToken(func(token string) (middleware.AccessTokenClaims, error) {
		return middleware.AccessTokenClaims{}, authmodule.ErrInvalidToken
	}))
	router.GET("/v1/me/profile", func(c *gin.Context) {
		httpx.WriteData(c, http.StatusOK, gin.H{"ok": true})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/me/profile", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var body httpx.ResponseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error == nil || body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED error")
	}
}

func TestRequireAccessTokenValidBearerToken(t *testing.T) {
	t.Parallel()

	manager := authmodule.NewTokenManager(authmodule.TokenConfig{
		Issuer:        "butaqueando-api",
		AccessSecret:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RefreshSecret: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    24 * time.Hour,
	})

	accessToken, _, err := manager.GenerateAccessToken("00000000-0000-0000-0000-000000000002", "user")
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	parse := func(token string) (middleware.AccessTokenClaims, error) {
		claims, err := manager.ParseAccessToken(token)
		if err != nil {
			return middleware.AccessTokenClaims{}, err
		}

		return middleware.AccessTokenClaims{UserID: claims.UserID, Role: claims.Role}, nil
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.ErrorEnvelope(), middleware.RequireAccessToken(parse))
	router.GET("/v1/me/profile", func(c *gin.Context) {
		userID, ok := middleware.GetAuthenticatedUserID(c)
		if !ok {
			httpx.WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user in context", nil)
			return
		}

		httpx.WriteData(c, http.StatusOK, gin.H{"userId": userID})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/me/profile", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var body struct {
		Data struct {
			UserID string `json:"userId"`
		} `json:"data"`
		Error any `json:"error"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error != nil {
		t.Fatalf("expected no error payload")
	}

	if body.Data.UserID != "00000000-0000-0000-0000-000000000002" {
		t.Fatalf("expected user id in response data")
	}
}
