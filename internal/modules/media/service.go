package media

import (
	"context"
	"errors"
	"time"

	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultDownloadURLTTL = 15 * time.Minute

type repositoryPort interface {
	GetPublishedPlayMediaObjectKey(ctx context.Context, playID string, mediaID string) (string, error)
	GetUserAvatarObjectKey(ctx context.Context, userID string) (string, error)
}

type Service struct {
	repo           repositoryPort
	playsStorage   storage.Client
	usersStorage   storage.Client
	downloadURLTTL time.Duration
}

func NewService(repo repositoryPort, playsStorage storage.Client, usersStorage storage.Client, downloadURLTTL time.Duration) *Service {
	if playsStorage == nil {
		playsStorage = storage.NoopClient{}
	}

	if usersStorage == nil {
		usersStorage = storage.NoopClient{}
	}

	if downloadURLTTL <= 0 {
		downloadURLTTL = defaultDownloadURLTTL
	}

	return &Service{repo: repo, playsStorage: playsStorage, usersStorage: usersStorage, downloadURLTTL: downloadURLTTL}
}

func (s *Service) GetPlayMediaRedirectURL(ctx context.Context, playID string, mediaID string) (string, error) {
	if !isValidUUID(playID) {
		return "", sharederrors.Validation("invalid playId", nil)
	}

	if !isValidUUID(mediaID) {
		return "", sharederrors.Validation("invalid mediaId", nil)
	}

	objectKey, err := s.repo.GetPublishedPlayMediaObjectKey(ctx, playID, mediaID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", sharederrors.NotFound("play media not found", nil)
		}

		return "", sharederrors.Internal("failed to load media", nil)
	}

	url, err := s.playsStorage.PresignGetObject(ctx, storage.PresignGetObjectInput{ObjectKey: objectKey, ExpiresIn: s.downloadURLTTL})
	if err != nil {
		return "", sharederrors.Internal("failed to load media", nil)
	}

	return url, nil
}

func (s *Service) GetUserAvatarRedirectURL(ctx context.Context, userID string) (string, error) {
	if !isValidUUID(userID) {
		return "", sharederrors.Validation("invalid userId", nil)
	}

	objectKey, err := s.repo.GetUserAvatarObjectKey(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", sharederrors.NotFound("avatar not found", nil)
		}

		return "", sharederrors.Internal("failed to load media", nil)
	}

	url, err := s.usersStorage.PresignGetObject(ctx, storage.PresignGetObjectInput{ObjectKey: objectKey, ExpiresIn: s.downloadURLTTL})
	if err != nil {
		return "", sharederrors.Internal("failed to load media", nil)
	}

	return url, nil
}

func isValidUUID(raw string) bool {
	_, err := uuid.Parse(raw)
	return err == nil
}
