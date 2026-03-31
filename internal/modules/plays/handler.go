package plays

import (
	"context"
	"net/http"

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
	CreateSubmission(ctx context.Context, userID string, req CreateSubmissionRequest) (SubmissionData, error)
	ListMySubmissions(ctx context.Context, userID string, query ListSubmissionsQuery) (SubmissionListData, error)
	UpdateMySubmission(ctx context.Context, userID string, playID string, req UpdateSubmissionRequest) (SubmissionData, error)
	ListAdminSubmissions(ctx context.Context, userID string, role string, query ListSubmissionsQuery) (SubmissionListData, error)
	ApproveSubmission(ctx context.Context, userID string, role string, playID string) (SubmissionData, error)
	RejectSubmission(ctx context.Context, userID string, role string, playID string, req RejectSubmissionRequest) (SubmissionData, error)
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

	httpx.WriteData(c, http.StatusOK, data)
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

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) GetByID(c *gin.Context) {
	data, err := h.service.GetByID(c.Request.Context(), c.Param("playId"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
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

	httpx.WriteData(c, http.StatusOK, data)
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

	httpx.WriteData(c, http.StatusOK, data)
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
