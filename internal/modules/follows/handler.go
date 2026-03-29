package follows

import (
	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	deps Dependencies
}

func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) Follow(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) Unfollow(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) MyFollowings(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) UserFollowers(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) UserFollowings(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) notImplemented(c *gin.Context) {
	_ = c.Error(sharederrors.NotImplemented("follows endpoint not implemented yet", nil))
}
