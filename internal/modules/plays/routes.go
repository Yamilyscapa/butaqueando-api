package plays

import (
	"time"

	"github.com/butaqueando/api/internal/http/middleware"
	"github.com/butaqueando/api/internal/shared/cache"
	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/plays"

type Dependencies struct {
	DB                *gorm.DB
	AccessTokenParser middleware.AccessTokenParser
	MediaStorage      storage.Client
	UploadURLTTL      time.Duration
	MaxImageBytes     int64
	ImageQueue        ImageQueue
	OptimizeImages    bool
	WebPQuality       int
	Cache             cache.Client
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
		WithCache(deps.Cache),
	)
	handler := NewHandler(service)

	v1.GET("/feed", handler.Feed)
	v1.GET("/search", handler.Search)
	v1.GET("/genres", handler.ListGenres)
	v1.GET("/cities", handler.ListCities)
	v1.GET("/theaters", handler.ListTheaters)

	group := v1.Group(BasePath)

	group.GET("/:playId", handler.GetByID)
	group.GET("/:playId/reviews", handler.ListReviews)
	v1.GET("/users/:userId/watched", handler.ListUserWatched)
	v1.GET("/users/:userId/reviews", handler.ListUserReviews)

	protected := group.Group("")
	protected.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	protected.POST("/:playId/reviews", handler.CreateReview)
	protected.POST("/:playId/engagements", handler.SetEngagement)
	protected.DELETE("/:playId/engagements/:kind", handler.DeleteEngagement)

	reviews := v1.Group("/reviews")
	reviews.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	reviews.PATCH("/:reviewId", handler.UpdateReview)
	reviews.POST("/:reviewId/comments", handler.CreateReviewComment)

	submissions := v1.Group("/submissions")
	submissions.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	submissions.POST("/plays", handler.CreateSubmission)

	myEngagements := v1.Group("/me")
	myEngagements.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	myEngagements.GET("/bookmarks", handler.ListMyBookmarks)
	myEngagements.GET("/watched", handler.ListMyWatched)
	myEngagements.GET("/reviews", handler.ListMyReviews)

	mySubmissions := v1.Group("/me/submissions")
	mySubmissions.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	mySubmissions.GET("/plays", handler.ListMySubmissions)
	mySubmissions.PATCH("/plays/:playId", handler.UpdateMySubmission)
	mySubmissions.POST("/plays/:playId/media/uploads", handler.CreateSubmissionMediaUpload)
	mySubmissions.POST("/plays/:playId/media", handler.AttachSubmissionMedia)

	myOwnedPlays := v1.Group("/me/plays")
	myOwnedPlays.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	myOwnedPlays.GET("/owned", handler.ListMyOwnedPlays)

	myEditSuggestions := v1.Group("/me/play-edit-suggestions")
	myEditSuggestions.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	myEditSuggestions.POST("", handler.CreatePlayEditSuggestion)
	myEditSuggestions.GET("", handler.ListMyPlayEditSuggestions)
	myEditSuggestions.GET("/:suggestionId", handler.GetMyPlayEditSuggestionByID)
	myEditSuggestions.PATCH("/:suggestionId", handler.UpdateMyPlayEditSuggestion)
	myEditSuggestions.POST("/:suggestionId/media/uploads", handler.CreatePlayEditSuggestionMediaUpload)
	myEditSuggestions.POST("/:suggestionId/media", handler.AttachPlayEditSuggestionMedia)
	myEditSuggestions.DELETE("/:suggestionId/media/:mediaId", handler.DeleteMyPlayEditSuggestionMedia)

	adminSubmissions := v1.Group("/admin/submissions")
	adminSubmissions.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	adminSubmissions.GET("/plays", handler.ListAdminSubmissions)
	adminSubmissions.GET("/plays/:playId", handler.GetAdminSubmissionByID)
	adminSubmissions.PATCH("/plays/:playId", handler.UpdateAdminSubmission)
	adminSubmissions.POST("/plays/:playId/approve", handler.ApproveSubmission)
	adminSubmissions.POST("/plays/:playId/reject", handler.RejectSubmission)
	adminSubmissions.POST("/plays/:playId/media/uploads", handler.CreateAdminSubmissionMediaUpload)
	adminSubmissions.POST("/plays/:playId/media", handler.AttachAdminSubmissionMedia)
	adminSubmissions.DELETE("/plays/:playId/media/:mediaId", handler.DeleteAdminSubmissionMedia)

	adminModeration := v1.Group("/admin/moderation")
	adminModeration.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	adminModeration.GET("/queue", handler.ListAdminModerationQueue)

	adminEditSuggestions := v1.Group("/admin/play-edit-suggestions")
	adminEditSuggestions.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	adminEditSuggestions.GET("/:suggestionId", handler.GetAdminPlayEditSuggestionByID)
	adminEditSuggestions.POST("/:suggestionId/approve", handler.ApprovePlayEditSuggestion)
	adminEditSuggestions.POST("/:suggestionId/reject", handler.RejectPlayEditSuggestion)

	adminGenres := v1.Group("/admin/genres")
	adminGenres.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	adminGenres.GET("", handler.ListAdminGenres)
	adminGenres.POST("", handler.CreateAdminGenre)
	adminGenres.DELETE("/:genreId", handler.DeleteAdminGenre)

	adminCities := v1.Group("/admin/cities")
	adminCities.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	adminCities.POST("", handler.CreateAdminCity)
	adminCities.DELETE("/:cityId", handler.DeleteAdminCity)

	adminTheaters := v1.Group("/admin/theaters")
	adminTheaters.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	adminTheaters.POST("", handler.CreateAdminTheater)
	adminTheaters.DELETE("/:theaterId", handler.DeleteAdminTheater)

	adminPlays := v1.Group("/admin/plays")
	adminPlays.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	adminPlays.DELETE("/:playId", handler.DeleteAdminPlay)

	adminReviewComments := v1.Group("/admin/review-comments")
	adminReviewComments.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	adminReviewComments.PATCH("/:commentId/status", handler.UpdateReviewCommentStatus)
}
