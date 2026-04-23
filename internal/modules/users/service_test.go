package users

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"strings"
	"testing"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/storage"
	"gorm.io/gorm"
)

type fakeRepository struct {
	getPublicProfileFn func(ctx context.Context, userID string) (PublicProfileRecord, error)
	getMeProfileFn     func(ctx context.Context, userID string) (MeProfileRecord, error)
	updateMeProfileFn  func(ctx context.Context, userID string, patch UpdateMeProfilePatch) (MeProfileRecord, error)
	createDeletionFn   func(ctx context.Context, userID string) (AccountDeletionRequestRecord, error)
}

type fakeStorage struct {
	presignPutObjectFn func(ctx context.Context, input storage.PresignPutObjectInput) (string, error)
	headObjectFn       func(ctx context.Context, input storage.HeadObjectInput) (storage.HeadObjectOutput, error)
	getObjectFn        func(ctx context.Context, input storage.GetObjectInput) ([]byte, error)
	putObjectFn        func(ctx context.Context, input storage.PutObjectInput) error
}

func (f fakeStorage) Bucket() string {
	return ""
}

func (f fakeStorage) PresignPutObject(ctx context.Context, input storage.PresignPutObjectInput) (string, error) {
	if f.presignPutObjectFn != nil {
		return f.presignPutObjectFn(ctx, input)
	}

	return "https://upload.example.com", nil
}

func (f fakeStorage) PresignGetObject(_ context.Context, _ storage.PresignGetObjectInput) (string, error) {
	return "", nil
}

func (f fakeStorage) HeadObject(ctx context.Context, input storage.HeadObjectInput) (storage.HeadObjectOutput, error) {
	if f.headObjectFn != nil {
		return f.headObjectFn(ctx, input)
	}

	return storage.HeadObjectOutput{}, nil
}

func (f fakeStorage) GetObject(ctx context.Context, input storage.GetObjectInput) ([]byte, error) {
	if f.getObjectFn != nil {
		return f.getObjectFn(ctx, input)
	}

	return nil, nil
}

func (f fakeStorage) PutObject(ctx context.Context, input storage.PutObjectInput) error {
	if f.putObjectFn != nil {
		return f.putObjectFn(ctx, input)
	}

	return nil
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

func (f *fakeRepository) CreateAccountDeletionRequest(ctx context.Context, userID string) (AccountDeletionRequestRecord, error) {
	if f.createDeletionFn != nil {
		return f.createDeletionFn(ctx, userID)
	}

	return AccountDeletionRequestRecord{}, gorm.ErrRecordNotFound
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

func TestServiceCreateAvatarUploadUsesWebPObjectKey(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{}, WithMediaStorage(fakeStorage{}))

	data, err := service.CreateAvatarUpload(context.Background(), "00000000-0000-0000-0000-000000000002", CreateAvatarUploadRequest{ContentType: "image/jpeg", ContentLength: 1024})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if !strings.HasSuffix(data.ObjectKey, ".webp") {
		t.Fatalf("expected webp object key, got %q", data.ObjectKey)
	}
}

func TestServiceUpdateMeProfileOptimizesAvatarAndNormalizesKey(t *testing.T) {
	t.Parallel()

	avatar := image.NewRGBA(image.Rect(0, 0, 640, 640))
	for y := 0; y < avatar.Bounds().Dy(); y++ {
		for x := 0; x < avatar.Bounds().Dx(); x++ {
			avatar.SetRGBA(x, y, color.RGBA{R: 10, G: 40, B: 140, A: 255})
		}
	}

	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, avatar, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode test avatar: %v", err)
	}

	avatarKey := "users/00000000-0000-0000-0000-000000000002/avatar/input.jpg"
	service := NewService(&fakeRepository{updateMeProfileFn: func(ctx context.Context, userID string, patch UpdateMeProfilePatch) (MeProfileRecord, error) {
		if patch.AvatarObjectKey == nil || !strings.HasSuffix(*patch.AvatarObjectKey, ".webp") {
			t.Fatalf("expected normalized webp avatar key")
		}

		return MeProfileRecord{ID: userID, DisplayName: "Ana", Email: "ana@example.com", Role: "user", AvatarObjectKey: patch.AvatarObjectKey, AvatarVersion: "2025-01-01T00:00:00Z"}, nil
	}}, WithMediaStorage(fakeStorage{
		headObjectFn: func(ctx context.Context, input storage.HeadObjectInput) (storage.HeadObjectOutput, error) {
			return storage.HeadObjectOutput{ContentType: "image/jpeg", ContentLength: int64(len(buf.Bytes()))}, nil
		},
		getObjectFn: func(ctx context.Context, input storage.GetObjectInput) ([]byte, error) {
			return buf.Bytes(), nil
		},
		putObjectFn: func(ctx context.Context, input storage.PutObjectInput) error {
			if !strings.HasSuffix(input.ObjectKey, ".webp") {
				t.Fatalf("expected put object key to be webp")
			}

			if input.ContentType != "image/webp" {
				t.Fatalf("expected webp content type, got %q", input.ContentType)
			}

			if input.CacheControl == "" {
				t.Fatalf("expected cache control to be set")
			}

			return nil
		},
	}))

	profile, err := service.UpdateMeProfile(context.Background(), "00000000-0000-0000-0000-000000000002", UpdateMeProfileRequest{AvatarObjectKey: &avatarKey})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if profile.AvatarURL == nil || !strings.Contains(*profile.AvatarURL, "/avatar?") {
		t.Fatalf("expected avatar URL to contain /avatar, got %v", profile.AvatarURL)
	}

	if profile.AvatarVersion == nil || *profile.AvatarVersion == "" {
		t.Fatalf("expected avatar version to be present")
	}
}

func TestServiceCreateAccountDeletionRequestSuccess(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{createDeletionFn: func(ctx context.Context, userID string) (AccountDeletionRequestRecord, error) {
		return AccountDeletionRequestRecord{
			ID:          "11111111-1111-1111-1111-111111111111",
			Status:      "pending",
			RequestedAt: "2026-01-01T00:00:00Z",
		}, nil
	}})

	data, err := service.CreateAccountDeletionRequest(context.Background(), "00000000-0000-0000-0000-000000000002")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if data.Status != "pending" {
		t.Fatalf("expected pending status, got %q", data.Status)
	}
}

func TestServiceCreateAccountDeletionRequestUnauthorizedWithInvalidUserID(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{})
	_, err := service.CreateAccountDeletionRequest(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatalf("expected error")
	}

	var appErr *sharederrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error")
	}

	if appErr.Code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED, got %q", appErr.Code)
	}
}
