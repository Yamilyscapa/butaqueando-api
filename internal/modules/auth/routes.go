package auth

import (
	"time"

	"github.com/butaqueando/api/internal/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/auth"

type Dependencies struct {
	DB          *gorm.DB
	TokenConfig TokenConfig
}

func RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	repo := NewRepository(deps.DB)
	tokens := NewTokenManager(deps.TokenConfig)
	service := NewService(repo, tokens)
	handler := NewHandler(service)
	group := v1.Group(BasePath)

	group.POST("/sign-in", middleware.AuthRateLimit(15, time.Minute), handler.SignIn)
	group.POST("/refresh", middleware.AuthRateLimit(30, time.Minute), handler.Refresh)
	group.POST("/sign-out", handler.SignOut)
}
