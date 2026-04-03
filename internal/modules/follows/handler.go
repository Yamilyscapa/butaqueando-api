package follows

import (
	"context"
	"net/http"

	"github.com/butaqueando/api/internal/http/middleware"
	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

type servicePort interface {
	Follow(ctx context.Context, actorUserID string, targetUserID string) (FollowActionData, error)
	Unfollow(ctx context.Context, actorUserID string, targetUserID string) (FollowActionData, error)
	ListMyFollowings(ctx context.Context, actorUserID string, query ListFollowsQuery) (FollowListData, error)
	ListUserFollowers(ctx context.Context, userID string, query ListFollowsQuery) (FollowListData, error)
	ListUserFollowings(ctx context.Context, userID string, query ListFollowsQuery) (FollowListData, error)
}

type Handler struct {
	service servicePort
}

func NewHandler(service servicePort) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Follow(c *gin.Context) {
	actorUserID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	targetUserID := c.Param("userId")
	data, err := h.service.Follow(c.Request.Context(), actorUserID, targetUserID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) Unfollow(c *gin.Context) {
	actorUserID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	targetUserID := c.Param("userId")
	data, err := h.service.Unfollow(c.Request.Context(), actorUserID, targetUserID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteData(c, http.StatusOK, data)
}

func (h *Handler) MyFollowings(c *gin.Context) {
	actorUserID, ok := middleware.GetAuthenticatedUserID(c)
	if !ok {
		_ = c.Error(sharederrors.Unauthorized("invalid access token", nil))
		return
	}

	var query ListFollowsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListMyFollowings(c.Request.Context(), actorUserID, query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) UserFollowers(c *gin.Context) {
	var query ListFollowsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListUserFollowers(c.Request.Context(), c.Param("userId"), query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) UserFollowings(c *gin.Context) {
	var query ListFollowsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(sharederrors.Validation("invalid query params", gin.H{"cause": err.Error()}))
		return
	}

	data, err := h.service.ListUserFollowings(c.Request.Context(), c.Param("userId"), query)
	if err != nil {
		_ = c.Error(err)
		return
	}

	httpx.WriteDataWithETag(c, http.StatusOK, data)
}

func (h *Handler) notImplemented(c *gin.Context) {
	_ = c.Error(sharederrors.NotImplemented("follows endpoint not implemented yet", nil))
}
