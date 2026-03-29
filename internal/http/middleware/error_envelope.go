package middleware

import (
	"errors"
	"net/http"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

func ErrorEnvelope() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		if c.Writer.Written() {
			return
		}

		writeMappedError(c, c.Errors.Last().Err)
	}
}

func writeMappedError(c *gin.Context, err error) {
	var appErr *sharederrors.AppError
	if errors.As(err, &appErr) {
		httpx.WriteError(c, appErr.StatusCode(), appErr.Code, appErr.Message, appErr.Details)
		return
	}

	httpx.WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
}
