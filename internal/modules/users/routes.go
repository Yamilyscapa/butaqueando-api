package users

import (
	"github.com/butaqueando/api/internal/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/users"

type Dependencies struct {
	DB                *gorm.DB
	AccessTokenParser middleware.AccessTokenParser
}

func RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	repo := NewRepository(deps.DB)
	service := NewService(repo)
	handler := NewHandler(service)
	group := v1.Group(BasePath)

	group.GET("", handler.List)
	group.GET("/:userId/profile", handler.GetProfile)

	me := v1.Group("/me")
	me.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	me.GET("/profile", handler.GetMe)
	me.PATCH("/profile", handler.UpdateMe)
}
