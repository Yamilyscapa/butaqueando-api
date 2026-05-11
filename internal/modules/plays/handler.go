package plays

import (
	"context"
	"net/http"
	"time"

	"github.com/butaqueando/api/internal/http/middleware"
	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

type servicePort interface {
	Feed(ctx context.Context, query FeedQuery) (FeedData, error)
	Search(ctx context.Context, query SearchQuery) (SearchData, error)
	GetByID(ctx context.Context, playID string) (PlayDetailsData, error)
	ListReviews(ctx context.Context, playID string, query ListReviewsQuery) (ReviewListData, error)
	CreateReview(ctx context.Context, userID string, playID string, req CreateReviewRequest) (ReviewData, error)
	UpdateReview(ctx context.Context, userID string, reviewID string, req UpdateReviewRequest) (ReviewData, error)
	CreateReviewComment(ctx context.Context, userID string, reviewID string, req CreateReviewCommentRequest) (ReviewCommentData, error)
	UpdateReviewCommentStatus(ctx context.Context, userID string, role string, commentID string, req UpdateReviewCommentStatusRequest) (ReviewCommentStatusData, error)
	ListUserWatched(ctx context.Context, userID string, query ListMyEngagementsQuery) (MyEngagementPlayListData, error)
	ListUserReviews(ctx context.Context, userID string, query ListUserReviewsQuery) (UserReviewListData, error)
	CreateSubmission(ctx context.Context, userID string, req CreateSubmissionRequest) (SubmissionData, error)
	CreateSubmissionMediaUpload(ctx context.Context, userID string, playID string, req CreateSubmissionMediaUploadRequest) (CreateSubmissionMediaUploadData, error)
	AttachSubmissionMedia(ctx context.Context, userID string, playID string, req AttachSubmissionMediaRequest) (PlayMediaData, error)
	ListMyBookmarks(ctx context.Context, userID string, query ListMyEngagementsQuery) (MyEngagementPlayListData, error)
	ListMyWatched(ctx context.Context, userID string, query ListMyEngagementsQuery) (MyEngagementPlayListData, error)
	ListMyFavorites(ctx context.Context, userID string, query ListMyEngagementsQuery) (MyEngagementPlayListData, error)
	ListMyReviews(ctx context.Context, userID string, query ListUserReviewsQuery) (UserReviewListData, error)
	ListMySubmissions(ctx context.Context, userID string, query ListSubmissionsQuery) (SubmissionListData, error)
	UpdateMySubmission(ctx context.Context, userID string, playID string, req UpdateSubmissionRequest) (SubmissionData, error)
	ListMyOwnedPlays(ctx context.Context, userID string, query ListOwnedPlaysQuery) (OwnedPlayListData, error)
	CreatePlayEditSuggestion(ctx context.Context, userID string, req CreatePlayEditSuggestionRequest) (PlayEditSuggestionData, error)
	ListMyPlayEditSuggestions(ctx context.Context, userID string, query ListPlayEditSuggestionsQuery) (PlayEditSuggestionListData, error)
	GetMyPlayEditSuggestionByID(ctx context.Context, userID string, suggestionID string) (PlayEditSuggestionData, error)
	UpdateMyPlayEditSuggestion(ctx context.Context, userID string, suggestionID string, req UpdatePlayEditSuggestionRequest) (PlayEditSuggestionData, error)
	CreatePlayEditSuggestionMediaUpload(ctx context.Context, userID string, suggestionID string, req CreateSubmissionMediaUploadRequest) (CreateSubmissionMediaUploadData, error)
	AttachPlayEditSuggestionMedia(ctx context.Context, userID string, suggestionID string, req AttachSubmissionMediaRequest) (PlayMediaData, error)
	DeleteMyPlayEditSuggestionMedia(ctx context.Context, userID string, suggestionID string, mediaID string) error
	ListAdminGenres(ctx context.Context, userID string, role string, query ListGenresQuery) (GenreListData, error)
	CreateAdminGenre(ctx context.Context, userID string, role string, req CreateGenreRequest) (GenreData, error)
	DeleteAdminGenre(ctx context.Context, userID string, role string, genreID string) error
	ListGenres(ctx context.Context, query ListGenresQuery) (GenreListData, error)
	ListCities(ctx context.Context, query ListCitiesQuery) (CityListData, error)
	ListTheaters(ctx context.Context, query ListTheatersQuery) (TheaterListData, error)
	CreateAdminCity(ctx context.Context, userID string, role string, req CreateCityRequest) (CityData, error)
	DeleteAdminCity(ctx context.Context, userID string, role string, cityID string) error
	CreateAdminTheater(ctx context.Context, userID string, role string, req CreateTheaterRequest) (TheaterData, error)
	DeleteAdminTheater(ctx context.Context, userID string, role string, theaterID string) error
	DeleteAdminPlay(ctx context.Context, userID string, role string, playID string) error
	ListAdminSubmissions(ctx context.Context, userID string, role string, query ListSubmissionsQuery) (SubmissionListData, error)
	GetAdminSubmissionByID(ctx context.Context, userID string, role string, playID string) (SubmissionData, error)
	UpdateAdminSubmission(ctx context.Context, userID string, role string, playID string, req UpdateSubmissionRequest) (SubmissionData, error)
	ApproveSubmission(ctx context.Context, userID string, role string, playID string) (SubmissionData, error)
	RejectSubmission(ctx context.Context, userID string, role string, playID string, req RejectSubmissionRequest) (SubmissionData, error)
	CreateAdminSubmissionMediaUpload(ctx context.Context, userID string, role string, playID string, req CreateSubmissionMediaUploadRequest) (CreateSubmissionMediaUploadData, error)
	AttachAdminSubmissionMedia(ctx context.Context, userID string, role string, playID string, req AttachSubmissionMediaRequest) (PlayMediaData, error)
	DeleteAdminSubmissionMedia(ctx context.Context, userID string, role string, playID string, mediaID string) error
	ListAdminModerationQueue(ctx context.Context, userID string, role string, query ListModerationQueueQuery) (ModerationQueueData, error)
	GetAdminPlayEditSuggestionByID(ctx context.Context, userID string, role string, suggestionID string) (PlayEditSuggestionData, error)
	ApprovePlayEditSuggestion(ctx context.Context, userID string, role string, suggestionID string) (PlayEditSuggestionData, error)
	RejectPlayEditSuggestion(ctx context.Context, userID string, role string, suggestionID string, req RejectSubmissionRequest) (PlayEditSuggestionData, error)
	SetEngagement(ctx context.Context, userID string, playID string, req SetEngagementRequest) (EngagementStateData, error)
	DeleteEngagement(ctx context.Context, userID string, playID string, kind string) (EngagementStateData, error)
}

type Handler struct {
	service servicePort
}

func NewHandler(service servicePort) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Feed(c *gin.Context) {
	var query FeedQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.Feed(c.Request.Context(), query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.ApplyCachePolicy(c, httpx.PublicMaxAge(60*time.Second))
	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) Search(c *gin.Context) {
	var query SearchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.Search(c.Request.Context(), query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.ApplyCachePolicy(c, httpx.PublicMaxAge(30*time.Second))
	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) GetByID(c *gin.Context) {
	data, err := h.service.GetByID(c.Request.Context(), c.Param("playId"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.ApplyCachePolicy(c, httpx.PublicMaxAge(5*time.Minute))
	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) ListReviews(c *gin.Context) {
	var query ListReviewsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListReviews(c.Request.Context(), c.Param("playId"), query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) CreateReview(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.CreateReview(c.Request.Context(), userID, c.Param("playId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) SetEngagement(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req SetEngagementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.SetEngagement(c.Request.Context(), userID, c.Param("playId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) DeleteEngagement(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	data, err := h.service.DeleteEngagement(c.Request.Context(), userID, c.Param("playId"), c.Param("kind"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) UpdateReview(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.UpdateReview(c.Request.Context(), userID, c.Param("reviewId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) CreateReviewComment(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req CreateReviewCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.CreateReviewComment(c.Request.Context(), userID, c.Param("reviewId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) UpdateReviewCommentStatus(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req UpdateReviewCommentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.UpdateReviewCommentStatus(c.Request.Context(), userID, role, c.Param("commentId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) ListUserWatched(c *gin.Context) {
	var query ListMyEngagementsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListUserWatched(c.Request.Context(), c.Param("userId"), query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) ListUserReviews(c *gin.Context) {
	var query ListUserReviewsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListUserReviews(c.Request.Context(), c.Param("userId"), query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) CreateSubmission(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.CreateSubmission(c.Request.Context(), userID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) ListGenres(c *gin.Context) {
	var query ListGenresQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListGenres(c.Request.Context(), query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.ApplyCachePolicy(c, httpx.PublicMaxAge(15*time.Minute))
	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) ListCities(c *gin.Context) {
	var query ListCitiesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListCities(c.Request.Context(), query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.ApplyCachePolicy(c, httpx.PublicMaxAge(15*time.Minute))
	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) ListTheaters(c *gin.Context) {
	var query ListTheatersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListTheaters(c.Request.Context(), query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.ApplyCachePolicy(c, httpx.PublicMaxAge(15*time.Minute))
	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) ListMyBookmarks(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var query ListMyEngagementsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListMyBookmarks(c.Request.Context(), userID, query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) ListMyWatched(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var query ListMyEngagementsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListMyWatched(c.Request.Context(), userID, query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) ListMyFavorites(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var query ListMyEngagementsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListMyFavorites(c.Request.Context(), userID, query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) ListMyReviews(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var query ListUserReviewsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListMyReviews(c.Request.Context(), userID, query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) ListMySubmissions(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var query ListSubmissionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListMySubmissions(c.Request.Context(), userID, query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) UpdateMySubmission(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req UpdateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.UpdateMySubmission(c.Request.Context(), userID, c.Param("playId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) ListMyOwnedPlays(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var query ListOwnedPlaysQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListMyOwnedPlays(c.Request.Context(), userID, query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) CreatePlayEditSuggestion(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req CreatePlayEditSuggestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.CreatePlayEditSuggestion(c.Request.Context(), userID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) ListMyPlayEditSuggestions(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var query ListPlayEditSuggestionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListMyPlayEditSuggestions(c.Request.Context(), userID, query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) GetMyPlayEditSuggestionByID(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	data, err := h.service.GetMyPlayEditSuggestionByID(c.Request.Context(), userID, c.Param("suggestionId"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) UpdateMyPlayEditSuggestion(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req UpdatePlayEditSuggestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.UpdateMyPlayEditSuggestion(c.Request.Context(), userID, c.Param("suggestionId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) CreatePlayEditSuggestionMediaUpload(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req CreateSubmissionMediaUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.CreatePlayEditSuggestionMediaUpload(c.Request.Context(), userID, c.Param("suggestionId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) AttachPlayEditSuggestionMedia(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req AttachSubmissionMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.AttachPlayEditSuggestionMedia(c.Request.Context(), userID, c.Param("suggestionId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) DeleteMyPlayEditSuggestionMedia(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	if err := h.service.DeleteMyPlayEditSuggestionMedia(c.Request.Context(), userID, c.Param("suggestionId"), c.Param("mediaId")); err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ListAdminGenres(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var query ListGenresQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListAdminGenres(c.Request.Context(), userID, role, query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) CreateAdminGenre(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req CreateGenreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.CreateAdminGenre(c.Request.Context(), userID, role, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) DeleteAdminGenre(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	err := h.service.DeleteAdminGenre(c.Request.Context(), userID, role, c.Param("genreId"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) CreateAdminCity(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req CreateCityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.CreateAdminCity(c.Request.Context(), userID, role, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) DeleteAdminCity(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	if err := h.service.DeleteAdminCity(c.Request.Context(), userID, role, c.Param("cityId")); err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) CreateAdminTheater(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req CreateTheaterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.CreateAdminTheater(c.Request.Context(), userID, role, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) DeleteAdminTheater(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	if err := h.service.DeleteAdminTheater(c.Request.Context(), userID, role, c.Param("theaterId")); err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) DeleteAdminPlay(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	if err := h.service.DeleteAdminPlay(c.Request.Context(), userID, role, c.Param("playId")); err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ListAdminSubmissions(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var query ListSubmissionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListAdminSubmissions(c.Request.Context(), userID, role, query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) GetAdminSubmissionByID(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	data, err := h.service.GetAdminSubmissionByID(c.Request.Context(), userID, role, c.Param("playId"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) UpdateAdminSubmission(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req UpdateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.UpdateAdminSubmission(c.Request.Context(), userID, role, c.Param("playId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) ApproveSubmission(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	data, err := h.service.ApproveSubmission(c.Request.Context(), userID, role, c.Param("playId"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) RejectSubmission(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req RejectSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.RejectSubmission(c.Request.Context(), userID, role, c.Param("playId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) CreateAdminSubmissionMediaUpload(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req CreateSubmissionMediaUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.CreateAdminSubmissionMediaUpload(c.Request.Context(), userID, role, c.Param("playId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) AttachAdminSubmissionMedia(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req AttachSubmissionMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.AttachAdminSubmissionMedia(c.Request.Context(), userID, role, c.Param("playId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) DeleteAdminSubmissionMedia(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	err := h.service.DeleteAdminSubmissionMedia(c.Request.Context(), userID, role, c.Param("playId"), c.Param("mediaId"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ListAdminModerationQueue(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var query ListModerationQueueQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListAdminModerationQueue(c.Request.Context(), userID, role, query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) GetAdminPlayEditSuggestionByID(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	data, err := h.service.GetAdminPlayEditSuggestionByID(c.Request.Context(), userID, role, c.Param("suggestionId"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) ApprovePlayEditSuggestion(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	data, err := h.service.ApprovePlayEditSuggestion(c.Request.Context(), userID, role, c.Param("suggestionId"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) RejectPlayEditSuggestion(c *gin.Context) {
	userID, userOK := middleware.GetAuthenticatedUserID(c)
	role, roleOK := middleware.GetAuthenticatedRole(c)
	if !userOK || !roleOK {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req RejectSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.RejectPlayEditSuggestion(c.Request.Context(), userID, role, c.Param("suggestionId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) CreateSubmissionMediaUpload(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req CreateSubmissionMediaUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.CreateSubmissionMediaUpload(c.Request.Context(), userID, c.Param("playId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}

func (h *Handler) AttachSubmissionMedia(c *gin.Context) {
	userID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var req AttachSubmissionMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(sharederrors.Validation("invalid request body", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.AttachSubmissionMedia(c.Request.Context(), userID, c.Param("playId"), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusCreated, data)
}
