package users

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/users"

type Dependencies struct {
	DB *gorm.DB
}

func RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	handler := NewHandler(deps)
	group := v1.Group(BasePath)

	group.GET("", handler.List)
	group.GET("/:userId/profile", handler.GetProfile)

	me := v1.Group("/me")
	me.GET("/profile", handler.GetMe)
	me.PATCH("/profile", handler.UpdateMe)
}
