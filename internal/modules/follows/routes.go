package follows

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/follows"

type Dependencies struct {
	DB *gorm.DB
}

func RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	handler := NewHandler(deps)
	group := v1.Group(BasePath)

	group.POST("/:userId", handler.Follow)
	group.DELETE("/:userId", handler.Unfollow)

	me := v1.Group("/me")
	me.GET("/followings", handler.MyFollowings)

	users := v1.Group("/users")
	users.GET("/:userId/followers", handler.UserFollowers)
	users.GET("/:userId/followings", handler.UserFollowings)
}
