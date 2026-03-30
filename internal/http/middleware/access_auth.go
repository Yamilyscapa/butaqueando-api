package middleware

import (
	"strings"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/gin-gonic/gin"
)

const (
	authenticatedUserIDKey = "auth.userId"
	authenticatedRoleKey   = "auth.role"
)

type AccessTokenClaims struct {
	UserID string
	Role   string
}

type AccessTokenParser func(token string) (AccessTokenClaims, error)

func RequireAccessToken(parse AccessTokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		if parse == nil {
			_ = c.Error(sharederrors.Internal("access token parser is not configured", nil))
			c.Abort()
			return
		}

		rawHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if rawHeader == "" {
			_ = c.Error(sharederrors.Unauthorized("missing bearer token", nil))
			c.Abort()
			return
		}

		parts := strings.SplitN(rawHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			_ = c.Error(sharederrors.Unauthorized("invalid authorization header", nil))
			c.Abort()
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			_ = c.Error(sharederrors.Unauthorized("missing bearer token", nil))
			c.Abort()
			return
		}

		claims, err := parse(token)
		if err != nil {
			_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
			c.Abort()
			return
		}

		if strings.TrimSpace(claims.UserID) == "" {
			_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
			c.Abort()
			return
		}

		c.Set(authenticatedUserIDKey, claims.UserID)
		c.Set(authenticatedRoleKey, claims.Role)
		c.Next()
	}
}

func GetAuthenticatedUserID(c *gin.Context) (string, bool) {
	value, ok := c.Get(authenticatedUserIDKey)
	if !ok {
		return "", false
	}

	userID, ok := value.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return "", false
	}

	return userID, true
}

func GetAuthenticatedRole(c *gin.Context) (string, bool) {
	value, ok := c.Get(authenticatedRoleKey)
	if !ok {
		return "", false
	}

	role, ok := value.(string)
	if !ok || strings.TrimSpace(role) == "" {
		return "", false
	}

	return role, true
}
