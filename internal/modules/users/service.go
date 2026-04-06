package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	maxBioLength           = 500
	defaultUploadURLTTL    = 15 * time.Minute
	defaultMaxImageBytes   = 10 * 1024 * 1024
	defaultUserMediaPrefix = "/v1/media/users"
)

type repositoryPort interface {
	GetPublicProfile(ctx context.Context, userID string) (PublicProfileRecord, error)
	GetMeProfile(ctx context.Context, userID string) (MeProfileRecord, error)
	UpdateMeProfile(ctx context.Context, userID string, patch UpdateMeProfilePatch) (MeProfileRecord, error)
}

type Service struct {
	repo             repositoryPort
	mediaStorage     storage.Client
	mediaUploadTTL   time.Duration
	maxImageBytes    int64
	userMediaBaseURL string
}

type ServiceOption func(*Service)

func WithMediaStorage(mediaStorage storage.Client) ServiceOption {
	return func(s *Service) {
		s.mediaStorage = mediaStorage
	}
}

func WithMediaUploadTTL(ttl time.Duration) ServiceOption {
	return func(s *Service) {
		s.mediaUploadTTL = ttl
	}
}

func WithMaxImageBytes(maxBytes int64) ServiceOption {
	return func(s *Service) {
		s.maxImageBytes = maxBytes
	}
}

func WithUserMediaBaseURL(baseURL string) ServiceOption {
	return func(s *Service) {
		s.userMediaBaseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	}
}

func NewService(repo repositoryPort, options ...ServiceOption) *Service {
	service := &Service{
		repo:             repo,
		mediaStorage:     storage.NoopClient{},
		mediaUploadTTL:   defaultUploadURLTTL,
		maxImageBytes:    defaultMaxImageBytes,
		userMediaBaseURL: defaultUserMediaPrefix,
	}

	for _, option := range options {
		if option != nil {
			option(service)
		}
	}

	if service.mediaStorage == nil {
		service.mediaStorage = storage.NoopClient{}
	}

	if service.mediaUploadTTL <= 0 {
		service.mediaUploadTTL = defaultUploadURLTTL
	}

	if service.maxImageBytes <= 0 {
		service.maxImageBytes = defaultMaxImageBytes
	}

	if strings.TrimSpace(service.userMediaBaseURL) == "" {
		service.userMediaBaseURL = defaultUserMediaPrefix
	}

	return service
}

func (s *Service) GetPublicProfile(ctx context.Context, userID string) (PublicProfileData, error) {
	if !isValidUUID(userID) {
		return PublicProfileData{}, sharederrors.Validation("invalid userId", nil)
	}

	record, err := s.repo.GetPublicProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PublicProfileData{}, sharederrors.NotFound("user profile not found", nil)
		}

		return PublicProfileData{}, sharederrors.Internal("failed to load profile", nil)
	}

	return mapPublicProfileRecord(record, s.userMediaBaseURL), nil
}

func (s *Service) GetMeProfile(ctx context.Context, userID string) (MeProfileData, error) {
	if strings.TrimSpace(userID) == "" {
		return MeProfileData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(userID) {
		return MeProfileData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	record, err := s.repo.GetMeProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MeProfileData{}, sharederrors.NotFound("user profile not found", nil)
		}

		return MeProfileData{}, sharederrors.Internal("failed to load profile", nil)
	}

	return mapMeProfileRecord(record, s.userMediaBaseURL), nil
}

func (s *Service) UpdateMeProfile(ctx context.Context, userID string, req UpdateMeProfileRequest) (MeProfileData, error) {
	if strings.TrimSpace(userID) == "" {
		return MeProfileData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	if !isValidUUID(userID) {
		return MeProfileData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	patch, err := validateAndBuildPatch(req)
	if err != nil {
		return MeProfileData{}, err
	}

	if patch.AvatarObjectKeySet {
		if patch.AvatarObjectKey != nil {
			if !strings.HasPrefix(*patch.AvatarObjectKey, "users/"+userID+"/avatar/") {
				return MeProfileData{}, sharederrors.Validation("avatarObjectKey does not belong to this user", nil)
			}

			headObject, headErr := s.mediaStorage.HeadObject(ctx, storage.HeadObjectInput{ObjectKey: *patch.AvatarObjectKey})
			if headErr != nil {
				if storage.IsNotFoundError(headErr) {
					return MeProfileData{}, sharederrors.Validation("uploaded object was not found", nil)
				}

				return MeProfileData{}, sharederrors.Internal("failed to update profile", nil)
			}

			if err := validateImageMetadata(headObject.ContentType, headObject.ContentLength, s.maxImageBytes); err != nil {
				return MeProfileData{}, err
			}
		}
	}

	record, err := s.repo.UpdateMeProfile(ctx, userID, patch)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MeProfileData{}, sharederrors.NotFound("user profile not found", nil)
		}

		return MeProfileData{}, sharederrors.Internal("failed to update profile", nil)
	}

	return mapMeProfileRecord(record, s.userMediaBaseURL), nil
}

func (s *Service) CreateAvatarUpload(ctx context.Context, userID string, req CreateAvatarUploadRequest) (CreateAvatarUploadData, error) {
	if strings.TrimSpace(userID) == "" || !isValidUUID(userID) {
		return CreateAvatarUploadData{}, sharederrors.Unauthorized("invalid access token", nil)
	}

	contentType, extension, err := normalizeImageContentType(req.ContentType)
	if err != nil {
		return CreateAvatarUploadData{}, err
	}

	if req.ContentLength <= 0 {
		return CreateAvatarUploadData{}, sharederrors.Validation("contentLength must be greater than 0", nil)
	}

	if req.ContentLength > s.maxImageBytes {
		return CreateAvatarUploadData{}, sharederrors.Validation("contentLength exceeds max allowed size", nil)
	}

	objectKey := fmt.Sprintf("users/%s/avatar/%s.%s", userID, uuid.NewString(), extension)
	contentLength := req.ContentLength
	uploadURL, err := s.mediaStorage.PresignPutObject(ctx, storage.PresignPutObjectInput{
		ObjectKey:     objectKey,
		ContentType:   contentType,
		ContentLength: &contentLength,
		ExpiresIn:     s.mediaUploadTTL,
	})
	if err != nil {
		return CreateAvatarUploadData{}, sharederrors.Internal("failed to create avatar upload", nil)
	}

	return CreateAvatarUploadData{ObjectKey: objectKey, UploadURL: uploadURL}, nil
}

func validateAndBuildPatch(req UpdateMeProfileRequest) (UpdateMeProfilePatch, error) {
	patch := UpdateMeProfilePatch{}

	if req.DisplayName != nil {
		displayName := strings.TrimSpace(*req.DisplayName)
		if displayName == "" {
			return UpdateMeProfilePatch{}, sharederrors.Validation("displayName must not be empty", nil)
		}

		if len(displayName) < 2 || len(displayName) > 80 {
			return UpdateMeProfilePatch{}, sharederrors.Validation("displayName length must be between 2 and 80", nil)
		}

		patch.DisplayNameSet = true
		patch.DisplayName = &displayName
	}

	if req.Bio != nil {
		bio := strings.TrimSpace(*req.Bio)
		if len(bio) > maxBioLength {
			return UpdateMeProfilePatch{}, sharederrors.Validation("bio length must be at most 500", nil)
		}

		patch.BioSet = true
		if bio == "" {
			patch.Bio = nil
		} else {
			patch.Bio = &bio
		}
	}

	if req.AvatarObjectKey != nil {
		patch.AvatarObjectKeySet = true
		avatarObjectKey := strings.TrimSpace(*req.AvatarObjectKey)
		if avatarObjectKey == "" {
			patch.AvatarObjectKey = nil
		} else {
			patch.AvatarObjectKey = &avatarObjectKey
		}
	}

	if !patch.DisplayNameSet && !patch.BioSet && !patch.AvatarObjectKeySet {
		return UpdateMeProfilePatch{}, sharederrors.Validation("at least one field must be provided", nil)
	}

	return patch, nil
}

func mapMeProfileRecord(record MeProfileRecord, userMediaBaseURL string) MeProfileData {
	return MeProfileData{
		ID:          record.ID,
		DisplayName: record.DisplayName,
		Email:       record.Email,
		Role:        record.Role,
		Bio:         record.Bio,
		AvatarURL:   buildAvatarURL(userMediaBaseURL, record.ID, record.AvatarObjectKey),
		Stats: ProfileStatsData{
			FollowersCount: record.FollowersCount,
			FollowingCount: record.FollowingCount,
			WatchedCount:   record.WatchedCount,
			ReviewsCount:   record.ReviewsCount,
		},
	}
}

func mapPublicProfileRecord(record PublicProfileRecord, userMediaBaseURL string) PublicProfileData {
	return PublicProfileData{
		ID:          record.ID,
		DisplayName: record.DisplayName,
		Bio:         record.Bio,
		AvatarURL:   buildAvatarURL(userMediaBaseURL, record.ID, record.AvatarObjectKey),
		Stats: ProfileStatsData{
			FollowersCount: record.FollowersCount,
			FollowingCount: record.FollowingCount,
			WatchedCount:   record.WatchedCount,
			ReviewsCount:   record.ReviewsCount,
		},
	}
}

func isValidUUID(raw string) bool {
	_, err := uuid.Parse(strings.TrimSpace(raw))
	return err == nil
}

func normalizeImageContentType(raw string) (string, string, error) {
	contentType := strings.ToLower(strings.TrimSpace(raw))
	if contentType == "" {
		return "", "", sharederrors.Validation("contentType must not be empty", nil)
	}

	if strings.Contains(contentType, ";") {
		parts := strings.SplitN(contentType, ";", 2)
		contentType = strings.TrimSpace(parts[0])
	}

	switch contentType {
	case "image/jpeg":
		return contentType, "jpg", nil
	case "image/png":
		return contentType, "png", nil
	case "image/webp":
		return contentType, "webp", nil
	default:
		return "", "", sharederrors.Validation("contentType must be one of: image/jpeg, image/png, image/webp", nil)
	}
}

func validateImageMetadata(contentType string, contentLength int64, maxImageBytes int64) error {
	if _, _, err := normalizeImageContentType(contentType); err != nil {
		return err
	}

	if contentLength <= 0 || contentLength > maxImageBytes {
		return sharederrors.Validation("uploaded object size is invalid", nil)
	}

	return nil
}

func buildAvatarURL(baseURL string, userID string, avatarObjectKey *string) *string {
	if avatarObjectKey == nil || strings.TrimSpace(*avatarObjectKey) == "" {
		return nil
	}

	base := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = defaultUserMediaPrefix
	}

	url := base + "/" + userID + "/avatar"
	return &url
}
