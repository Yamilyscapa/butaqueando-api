package auth

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/auth"

type Dependencies struct {
	DB *gorm.DB
}

func RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	handler := NewHandler(deps)
	group := v1.Group(BasePath)

	group.POST("/sign-in", handler.SignIn)
	group.POST("/refresh", handler.Refresh)
	group.POST("/sign-out", handler.SignOut)
}
