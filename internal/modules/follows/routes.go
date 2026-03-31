package follows

import (
	"github.com/butaqueando/api/internal/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/follows"

type Dependencies struct {
	DB                *gorm.DB
	AccessTokenParser middleware.AccessTokenParser
}

func RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	repo := NewRepository(deps.DB)
	service := NewService(repo)
	handler := NewHandler(service)

	users := v1.Group("/users")
	protected := users.Group("")
	protected.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	protected.POST("/:userId/follow", handler.Follow)
	protected.DELETE("/:userId/follow", handler.Unfollow)
	protected.GET("/me/followings", handler.MyFollowings)

	users.GET("/:userId/followers", handler.UserFollowers)
	users.GET("/:userId/followings", handler.UserFollowings)
}
