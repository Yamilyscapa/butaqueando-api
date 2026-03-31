package follows

import (
	"context"
	"errors"
	"testing"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
)

type fakeRepository struct {
	userExistsFn     func(ctx context.Context, userID string) (bool, error)
	createFollowFn   func(ctx context.Context, followerUserID string, followingUserID string, createdAt time.Time) (bool, error)
	deleteFollowFn   func(ctx context.Context, followerUserID string, followingUserID string) (bool, error)
	listFollowersFn  func(ctx context.Context, userID string, after *followCursor, limit int) ([]FollowRecord, error)
	listFollowingsFn func(ctx context.Context, userID string, after *followCursor, limit int) ([]FollowRecord, error)
}

func (f *fakeRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	if f.userExistsFn != nil {
		return f.userExistsFn(ctx, userID)
	}

	return false, nil
}

func (f *fakeRepository) CreateFollow(ctx context.Context, followerUserID string, followingUserID string, createdAt time.Time) (bool, error) {
	if f.createFollowFn != nil {
		return f.createFollowFn(ctx, followerUserID, followingUserID, createdAt)
	}

	return true, nil
}

func (f *fakeRepository) DeleteFollow(ctx context.Context, followerUserID string, followingUserID string) (bool, error) {
	if f.deleteFollowFn != nil {
		return f.deleteFollowFn(ctx, followerUserID, followingUserID)
	}

	return true, nil
}

func (f *fakeRepository) ListFollowers(ctx context.Context, userID string, after *followCursor, limit int) ([]FollowRecord, error) {
	if f.listFollowersFn != nil {
		return f.listFollowersFn(ctx, userID, after, limit)
	}

	return nil, nil
}

func (f *fakeRepository) ListFollowings(ctx context.Context, userID string, after *followCursor, limit int) ([]FollowRecord, error) {
	if f.listFollowingsFn != nil {
		return f.listFollowingsFn(ctx, userID, after, limit)
	}

	return nil, nil
}

func TestServiceFollowRejectsSelfFollow(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.Follow(context.Background(), "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000001")
	if err == nil {
		t.Fatalf("expected validation error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestServiceFollowReturnsNotFoundWhenTargetMissing(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{userExistsFn: func(ctx context.Context, userID string) (bool, error) {
		return false, nil
	}})

	_, err := service.Follow(context.Background(), "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002")
	if err == nil {
		t.Fatalf("expected not found error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %q", appErr.Code)
	}
}

func TestServiceListUserFollowingsBuildsNextCursor(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	service := NewService(&fakeRepository{
		userExistsFn: func(ctx context.Context, userID string) (bool, error) {
			return true, nil
		},
		listFollowingsFn: func(ctx context.Context, userID string, after *followCursor, limit int) ([]FollowRecord, error) {
			if limit != 2 {
				t.Fatalf("expected limit %d, got %d", 2, limit)
			}

			return []FollowRecord{
				{UserID: "00000000-0000-0000-0000-000000000010", DisplayName: "A", FollowedAt: now},
				{UserID: "00000000-0000-0000-0000-000000000009", DisplayName: "B", FollowedAt: now.Add(-time.Minute)},
			}, nil
		},
	})

	data, err := service.ListUserFollowings(context.Background(), "00000000-0000-0000-0000-000000000001", ListFollowsQuery{Limit: 1})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(data.Items))
	}

	if data.NextCursor == nil || *data.NextCursor == "" {
		t.Fatalf("expected next cursor")
	}
}
