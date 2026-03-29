package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(httpx.RequestIDHeader))
		if requestID == "" {
			requestID = generateRequestID()
		}

		httpx.SetRequestID(c, requestID)
		c.Header(httpx.RequestIDHeader, requestID)

		c.Next()
	}
}

func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UTC().UnixNano())
	}

	return "req-" + hex.EncodeToString(b)
}
