package auth

import (
	"context"
	"net/http"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

type servicePort interface {
	SignIn(ctx context.Context, req SignInRequest) (AuthTokensData, error)
	Refresh(ctx context.Context, req RefreshRequest) (AuthTokensData, error)
	SignOut(ctx context.Context, req SignOutRequest) (SignOutData, error)
}

type Handler struct {
	service servicePort
}

func NewHandler(service servicePort) *Handler {
	return &Handler{service: service}
}

func (h *Handler) SignIn(c *gin.Context) {
	var req SignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.SignIn(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.Refresh(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) SignOut(c *gin.Context) {
	var req SignOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.SignOut(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}
