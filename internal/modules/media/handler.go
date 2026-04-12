package media

import (
	"context"
	"net/http"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/gin-gonic/gin"
)

type servicePort interface {
	GetPlayMediaRedirectURL(ctx context.Context, playID string, mediaID string) (string, error)
	GetUserAvatarRedirectURL(ctx context.Context, userID string) (string, error)
}

type Handler struct {
	service servicePort
}

func NewHandler(service servicePort) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetPlayMedia(c *gin.Context) {
	redirectURL, err := h.service.GetPlayMediaRedirectURL(c.Request.Context(), c.Param("playId"), c.Param("mediaId"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (h *Handler) GetUserAvatar(c *gin.Context) {
	redirectURL, err := h.service.GetUserAvatarRedirectURL(c.Request.Context(), c.Param("userId"))
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
