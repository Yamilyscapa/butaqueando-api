package users

import (
	"time"

	"github.com/butaqueando/api/internal/http/middleware"
	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/users"

type Dependencies struct {
	DB                *gorm.DB
	AccessTokenParser middleware.AccessTokenParser
	MediaStorage      storage.Client
	UploadURLTTL      time.Duration
	MaxImageBytes     int64
	ImageQueue        ImageQueue
	OptimizeImages    bool
	WebPQuality       int
	VariantWidths     []int
	BlurhashEnabled   bool
}

func RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	repo := NewRepository(deps.DB)
	service := NewService(
		repo,
		WithMediaStorage(deps.MediaStorage),
		WithMediaUploadTTL(deps.UploadURLTTL),
		WithMaxImageBytes(deps.MaxImageBytes),
		WithImageQueue(deps.ImageQueue),
		WithImageOptimization(deps.OptimizeImages, deps.WebPQuality),
		WithImageVariantWidths(deps.VariantWidths),
		WithBlurhashEnabled(deps.BlurhashEnabled),
	)
	handler := NewHandler(service)
	group := v1.Group(BasePath)

	group.GET("", handler.List)
	group.GET("/:userId/profile", handler.GetProfile)

	me := v1.Group("/me")
	me.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	me.GET("/profile", handler.GetMe)
	me.PATCH("/profile", handler.UpdateMe)
	me.POST("/profile/avatar/uploads", handler.CreateAvatarUpload)
	me.POST("/account-deletion-requests", handler.CreateAccountDeletionRequest)
}
