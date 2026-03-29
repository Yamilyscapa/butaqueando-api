package plays

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

func (h *Handler) GetByID(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) ListReviews(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) CreateReview(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) SetEngagement(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) DeleteEngagement(c *gin.Context) {
	h.notImplemented(c)
}

func (h *Handler) notImplemented(c *gin.Context) {
	_ = c.Error(sharederrors.NotImplemented("plays endpoint not implemented yet", nil))
}
