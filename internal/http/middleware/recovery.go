package middleware

import (
	"log"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestID := httpx.GetRequestID(c)
				log.Printf("panic recovered request_id=%s err=%v", requestID, recovered)

				appErr := sharederrors.Internal("internal server error", nil)
				httpx.AbortError(c, appErr.StatusCode(), appErr.Code, appErr.Message, appErr.Details)
			}
		}()

		c.Next()
	}
}
