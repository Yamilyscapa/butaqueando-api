package plays

import (
	"context"
	"net/http"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

type servicePort interface {
	Feed(ctx context.Context, query FeedQuery) (FeedData, error)
	Search(ctx context.Context, query SearchQuery) (SearchData, error)
	GetByID(ctx context.Context, playID string) (PlayDetailsData, error)
}

type Handler struct {
	service servicePort
}

func NewHandler(service servicePort) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Feed(c *gin.Context) {
	var query FeedQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.Feed(c.Request.Context(), query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) Search(c *gin.Context) {
	var query SearchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.Search(c.Request.Context(), query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) GetByID(c *gin.Context) {
	data, err := h.service.GetByID(c.Request.Context(), c.Param("playId"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
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
