package follows

import (
	"context"
	"strings"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/google/uuid"
)

const (
	defaultListLimit = 20
	maxListLimit     = 50
)

type repositoryPort interface {
	UserExists(ctx context.Context, userID string) (bool, error)
	CreateFollow(ctx context.Context, followerUserID string, followingUserID string, createdAt time.Time) (bool, error)
	DeleteFollow(ctx context.Context, followerUserID string, followingUserID string) (bool, error)
	ListFollowers(ctx context.Context, userID string, after *followCursor, limit int) ([]FollowRecord, error)
	ListFollowings(ctx context.Context, userID string, after *followCursor, limit int) ([]FollowRecord, error)
}

type Service struct {
	repo repositoryPort
	now  func() time.Time
}

func NewService(repo repositoryPort) *Service {
	return &Service{
		repo: repo,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) Follow(ctx context.Context, actorUserID string, targetUserID string) (FollowActionData, error) {
	if !isValidUUID(actorUserID) {
		return FollowActionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(targetUserID) {
		return FollowActionData{}, sharederrors.Validation("invalid userId", nil)
	}

	if actorUserID == targetUserID {
		return FollowActionData{}, sharederrors.Validation("users cannot follow themselves", nil)
	}

	exists, err := s.repo.UserExists(ctx, targetUserID)
	if err != nil {
		return FollowActionData{}, sharederrors.Internal("failed to follow user", nil)
	}

	if !exists {
		return FollowActionData{}, sharederrors.NotFound("user not found", nil)
	}

	if _, err := s.repo.CreateFollow(ctx, actorUserID, targetUserID, s.now()); err != nil {
		return FollowActionData{}, sharederrors.Internal("failed to follow user", nil)
	}

	return FollowActionData{UserID: targetUserID, Following: true}, nil
}

func (s *Service) Unfollow(ctx context.Context, actorUserID string, targetUserID string) (FollowActionData, error) {
	if !isValidUUID(actorUserID) {
		return FollowActionData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(targetUserID) {
		return FollowActionData{}, sharederrors.Validation("invalid userId", nil)
	}

	if actorUserID == targetUserID {
		return FollowActionData{}, sharederrors.Validation("users cannot unfollow themselves", nil)
	}

	exists, err := s.repo.UserExists(ctx, targetUserID)
	if err != nil {
		return FollowActionData{}, sharederrors.Internal("failed to unfollow user", nil)
	}

	if !exists {
		return FollowActionData{}, sharederrors.NotFound("user not found", nil)
	}

	if _, err := s.repo.DeleteFollow(ctx, actorUserID, targetUserID); err != nil {
		return FollowActionData{}, sharederrors.Internal("failed to unfollow user", nil)
	}

	return FollowActionData{UserID: targetUserID, Following: false}, nil
}

func (s *Service) ListUserFollowers(ctx context.Context, userID string, query ListFollowsQuery) (FollowListData, error) {
	if !isValidUUID(userID) {
		return FollowListData{}, sharederrors.Validation("invalid userId", nil)
	}

	return s.listFollowers(ctx, userID, query)
}

func (s *Service) ListUserFollowings(ctx context.Context, userID string, query ListFollowsQuery) (FollowListData, error) {
	if !isValidUUID(userID) {
		return FollowListData{}, sharederrors.Validation("invalid userId", nil)
	}

	return s.listFollowings(ctx, userID, query)
}

func (s *Service) ListMyFollowings(ctx context.Context, actorUserID string, query ListFollowsQuery) (FollowListData, error) {
	if !isValidUUID(actorUserID) {
		return FollowListData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	return s.listFollowings(ctx, actorUserID, query)
}

func (s *Service) listFollowers(ctx context.Context, userID string, query ListFollowsQuery) (FollowListData, error) {
	if err := s.ensureUserExists(ctx, userID); err != nil {
		return FollowListData{}, err
	}

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return FollowListData{}, err
	}

	cursor, err := decodeFollowCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return FollowListData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.ListFollowers(ctx, userID, cursor, limit+1)
	if err != nil {
		return FollowListData{}, sharederrors.Internal("failed to list followers", nil)
	}

	return buildFollowListData(records, limit)
}

func (s *Service) listFollowings(ctx context.Context, userID string, query ListFollowsQuery) (FollowListData, error) {
	if err := s.ensureUserExists(ctx, userID); err != nil {
		return FollowListData{}, err
	}

	limit, err := normalizeListLimit(query.Limit)
	if err != nil {
		return FollowListData{}, err
	}

	cursor, err := decodeFollowCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return FollowListData{}, sharederrors.Validation("invalid cursor", nil)
	}

	records, err := s.repo.ListFollowings(ctx, userID, cursor, limit+1)
	if err != nil {
		return FollowListData{}, sharederrors.Internal("failed to list followings", nil)
	}

	return buildFollowListData(records, limit)
}

func (s *Service) ensureUserExists(ctx context.Context, userID string) error {
	exists, err := s.repo.UserExists(ctx, userID)
	if err != nil {
		return sharederrors.Internal("failed to load user", nil)
	}

	if !exists {
		return sharederrors.NotFound("user not found", nil)
	}

	return nil
}

func buildFollowListData(records []FollowRecord, limit int) (FollowListData, error) {
	hasNext := len(records) > limit
	if hasNext {
		records = records[:limit]
	}

	items := make([]FollowListItemData, 0, len(records))
	for _, record := range records {
		items = append(items, FollowListItemData{
			ID:          record.UserID,
			DisplayName: record.DisplayName,
			Bio:         record.Bio,
			FollowedAt:  record.FollowedAt.UTC().Format(time.RFC3339Nano),
		})
	}

	response := FollowListData{Items: items}
	if hasNext && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor, err := encodeFollowCursor(followCursor{CreatedAt: last.FollowedAt.UTC(), UserID: last.UserID})
		if err != nil {
			return FollowListData{}, sharederrors.Internal("failed to build pagination cursor", nil)
		}

		response.NextCursor = &nextCursor
	}

	return response, nil
}

func normalizeListLimit(rawLimit int) (int, error) {
	if rawLimit == 0 {
		return defaultListLimit, nil
	}

	if rawLimit < 1 || rawLimit > maxListLimit {
		return 0, sharederrors.Validation("limit must be between 1 and 50", nil)
	}

	return rawLimit, nil
}

func isValidUUID(raw string) bool {
	_, err := uuid.Parse(strings.TrimSpace(raw))
	return err == nil
}
