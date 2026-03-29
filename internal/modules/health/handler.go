package health

import (
	"context"
	"net/http"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service healthChecker
}

func NewHandler(deps Dependencies) *Handler {
	return &Handler{service: NewService(deps.DB)}
}

func (h *Handler) Get(c *gin.Context) {
	result := h.service.Check(c.Request.Context())
	if !result.Ready {
		_ = c.Error(sharederrors.ServiceUnavailable("service unavailable", gin.H{"database": result.Database}))
		return
	}

	httpx.WriteData(c, http.StatusOK, gin.H{
		"ok":        result.Ready,
		"database":  result.Database,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

type healthChecker interface {
	Check(ctx context.Context) CheckResult
}
