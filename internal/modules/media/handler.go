package media

import (
	"context"
	"net/http"
	"strconv"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/gin-gonic/gin"
)

type servicePort interface {
	GetPlayMediaRedirectURL(ctx context.Context, playID string, mediaID string, requestedWidth int) (string, error)
	GetUserAvatarRedirectURL(ctx context.Context, userID string, requestedWidth int) (string, error)
}

type Handler struct {
	service servicePort
}

func NewHandler(service servicePort) *Handler {
	return &Handler{service: service}
}

func parseWidthQuery(raw string) int {
	if raw == "" {
		return 0
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return 0
	}
	if parsed > 8192 {
		return 8192
	}
	return parsed
}

func (h *Handler) GetPlayMedia(c *gin.Context) {
	width := parseWidthQuery(c.Query("w"))
	redirectURL, err := h.service.GetPlayMediaRedirectURL(c.Request.Context(), c.Param("playId"), c.Param("mediaId"), width)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (h *Handler) GetUserAvatar(c *gin.Context) {
	width := parseWidthQuery(c.Query("w"))
	redirectURL, err := h.service.GetUserAvatarRedirectURL(c.Request.Context(), c.Param("userId"), width)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Header("Cache-Control", "no-store, private")
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (h *Handler) notConfigured(c *gin.Context) {
	_ = c.Error(sharederrors.Internal("media service not configured", nil))
}
