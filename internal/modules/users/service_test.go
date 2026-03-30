package users

import (
	"context"
	"errors"
	"testing"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"gorm.io/gorm"
)

type fakeRepository struct {
	getPublicProfileFn func(ctx context.Context, userID string) (PublicProfileRecord, error)
	getMeProfileFn     func(ctx context.Context, userID string) (MeProfileRecord, error)
	updateMeProfileFn  func(ctx context.Context, userID string, patch UpdateMeProfilePatch) (MeProfileRecord, error)
}

func (f *fakeRepository) GetPublicProfile(ctx context.Context, userID string) (PublicProfileRecord, error) {
	if f.getPublicProfileFn != nil {
		return f.getPublicProfileFn(ctx, userID)
	}

	return PublicProfileRecord{}, gorm.ErrRecordNotFound
}

func (f *fakeRepository) GetMeProfile(ctx context.Context, userID string) (MeProfileRecord, error) {
	if f.getMeProfileFn != nil {
		return f.getMeProfileFn(ctx, userID)
	}

	return MeProfileRecord{}, gorm.ErrRecordNotFound
}

func (f *fakeRepository) UpdateMeProfile(ctx context.Context, userID string, patch UpdateMeProfilePatch) (MeProfileRecord, error) {
	if f.updateMeProfileFn != nil {
		return f.updateMeProfileFn(ctx, userID, patch)
	}

	return MeProfileRecord{}, gorm.ErrRecordNotFound
}

func TestServiceGetMeProfileSuccess(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{getMeProfileFn: func(ctx context.Context, userID string) (MeProfileRecord, error) {
		return MeProfileRecord{ID: userID, DisplayName: "Ana", Email: "ana@example.com", Role: "user", FollowersCount: 3}, nil
	}})

	profile, err := service.GetMeProfile(context.Background(), "00000000-0000-0000-0000-000000000002")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if profile.DisplayName != "Ana" {
		t.Fatalf("expected display name Ana, got %q", profile.DisplayName)
	}
}

func TestServiceGetPublicProfileSuccess(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{getPublicProfileFn: func(ctx context.Context, userID string) (PublicProfileRecord, error) {
		return PublicProfileRecord{ID: userID, DisplayName: "Ana", FollowersCount: 2}, nil
	}})

	profile, err := service.GetPublicProfile(context.Background(), "00000000-0000-0000-0000-000000000002")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if profile.DisplayName != "Ana" {
		t.Fatalf("expected display name Ana, got %q", profile.DisplayName)
	}
}

func TestServiceGetPublicProfileInvalidID(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.GetPublicProfile(context.Background(), "not-a-uuid")
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

func TestServiceGetMeProfileNotFound(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.GetMeProfile(context.Background(), "00000000-0000-0000-0000-000000000002")
	if err == nil {
		t.Fatalf("expected error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %q", appErr.Code)
	}
}

func TestServiceUpdateMeProfileValidation(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.UpdateMeProfile(context.Background(), "00000000-0000-0000-0000-000000000002", UpdateMeProfileRequest{})
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

func TestServiceUpdateMeProfileSuccess(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{updateMeProfileFn: func(ctx context.Context, userID string, patch UpdateMeProfilePatch) (MeProfileRecord, error) {
		if !patch.DisplayNameSet {
			t.Fatalf("expected displayName patch")
		}

		return MeProfileRecord{ID: userID, DisplayName: *patch.DisplayName, Email: "ana@example.com", Role: "user"}, nil
	}})

	name := "Ana Updated"
	profile, err := service.UpdateMeProfile(context.Background(), "00000000-0000-0000-0000-000000000002", UpdateMeProfileRequest{DisplayName: &name})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if profile.DisplayName != "Ana Updated" {
		t.Fatalf("expected updated display name, got %q", profile.DisplayName)
	}
}
