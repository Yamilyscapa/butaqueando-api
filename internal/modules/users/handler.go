package users

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

func (h *Handler) List(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) GetProfile(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) GetMe(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) UpdateMe(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) notImplemented(c *gin.Context) {
	_ = c.Error(sharederrors.NotImplemented("users endpoint not implemented yet", nil))
}
