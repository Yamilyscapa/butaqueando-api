package plays

import (
	"github.com/butaqueando/api/internal/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/plays"

type Dependencies struct {
	DB                *gorm.DB
	AccessTokenParser middleware.AccessTokenParser
}

func RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	repo := NewRepository(deps.DB)
	service := NewService(repo)
	handler := NewHandler(service)

	v1.GET("/feed", handler.Feed)
	v1.GET("/search", handler.Search)

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

	adminSubmissions := v1.Group("/admin/submissions")
	adminSubmissions.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	adminSubmissions.GET("/plays", handler.ListAdminSubmissions)
	adminSubmissions.POST("/plays/:playId/approve", handler.ApproveSubmission)
	adminSubmissions.POST("/plays/:playId/reject", handler.RejectSubmission)

	adminReviewComments := v1.Group("/admin/review-comments")
	adminReviewComments.Use(middleware.RequireAccessToken(deps.AccessTokenParser))
	adminReviewComments.PATCH("/:commentId/status", handler.UpdateReviewCommentStatus)
}
