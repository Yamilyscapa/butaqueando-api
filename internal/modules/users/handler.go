package users

import (
	"context"
	"net/http"

	"github.com/butaqueando/api/internal/http/middleware"
	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

type servicePort interface {
	GetPublicProfile(ctx context.Context, userID string) (PublicProfileData, error)
	GetMeProfile(ctx context.Context, userID string) (MeProfileData, error)
	UpdateMeProfile(ctx context.Context, userID string, req UpdateMeProfileRequest) (MeProfileData, error)
}

type Handler struct {
	service servicePort
}

func NewHandler(service servicePort) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID := c.Param("userId")
	data, err := h.service.GetPublicProfile(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
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

	httpx.WriteData(c, http.StatusOK, data)
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

func (h *Handler) notImplemented(c *gin.Context) {
	_ = c.Error(sharederrors.NotImplemented("users endpoint not implemented yet", nil))
}
