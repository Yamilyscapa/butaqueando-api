package users

import (
	"context"
	"net/http"
	"strconv"

	"github.com/butaqueando/api/internal/http/middleware"
	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

type servicePort interface {
	GetPublicProfile(ctx context.Context, userID string) (PublicProfileData, error)
	GetMeProfile(ctx context.Context, userID string) (MeProfileData, error)
	UpdateMeProfile(ctx context.Context, userID string, req UpdateMeProfileRequest) (MeProfileData, error)
	SearchUsers(ctx context.Context, query string, limit int) ([]PublicProfileData, error)
	CreateAvatarUpload(ctx context.Context, userID string, req CreateAvatarUploadRequest) (CreateAvatarUploadData, error)
	CreateAccountDeletionRequest(ctx context.Context, userID string) (AccountDeletionRequestData, error)
}

type Handler struct {
	service servicePort
}

func NewHandler(service servicePort) *Handler {
	return &Handler{service: service}
}

type UserSearchResponseData struct {
	Items      []PublicProfileData `json:"items"`
	NextCursor *string             `json:"nextCursor"`
}

func (h *Handler) List(c *gin.Context) {
	query := c.Query("q")
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			_ = c.Error(sharederrors.Validation("invalid limit", nil))
			return
		}
		limit = parsed
	}

	items, err := h.service.SearchUsers(c.Request.Context(), query, limit)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, UserSearchResponseData{Items: items, NextCursor: nil})
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID := c.Param("userId")
	data, err := h.service.GetPublicProfile(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) GetMe(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	data, err := h.service.GetMeProfile(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) UpdateMe(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req UpdateMeProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.UpdateMeProfile(c.Request.Context(), userID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) CreateAvatarUpload(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req CreateAvatarUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.CreateAvatarUpload(c.Request.Context(), userID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) CreateAccountDeletionRequest(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	data, err := h.service.CreateAccountDeletionRequest(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

