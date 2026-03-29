package plays

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/plays"

type Dependencies struct {
	DB *gorm.DB
}

func RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	handler := NewHandler(deps)
	group := v1.Group(BasePath)

	group.GET("", handler.List)
	group.GET("/:playId", handler.GetByID)
	group.GET("/:playId/reviews", handler.ListReviews)
	group.POST("/:playId/reviews", handler.CreateReview)
	group.POST("/:playId/engagements", handler.SetEngagement)
	group.DELETE("/:playId/engagements/:kind", handler.DeleteEngagement)
}
