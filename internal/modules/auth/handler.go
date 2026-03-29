package auth

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

func (h *Handler) SignIn(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) Refresh(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) SignOut(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) notImplemented(c *gin.Context) {
	_ = c.Error(sharederrors.NotImplemented("auth endpoint not implemented yet", nil))
}
