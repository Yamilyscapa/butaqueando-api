package health

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/health"

type Dependencies struct {
	DB *gorm.DB
}

func RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	handler := NewHandler(deps)
	group := v1.Group(BasePath)

	group.GET("", handler.Get)
	group.GET("/", handler.Get)
}
