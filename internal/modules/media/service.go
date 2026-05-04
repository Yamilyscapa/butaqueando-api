package media

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/butaqueando/api/internal/shared/cache"
	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultDownloadURLTTL = 15 * time.Minute
const mediaRedirectCacheKeyVersion = "v1"
const mediaRedirectCacheSafetyWindow = time.Minute

type repositoryPort interface {
	GetPublishedPlayMediaObjectKey(ctx context.Context, playID string, mediaID string) (string, error)
	GetUserAvatarObjectKey(ctx context.Context, userID string) (string, error)
}

type Service struct {
	repo           repositoryPort
	playsStorage   storage.Client
	usersStorage   storage.Client
	downloadURLTTL time.Duration
	cache          cache.Client
}

func NewService(repo repositoryPort, playsStorage storage.Client, usersStorage storage.Client, downloadURLTTL time.Duration, cacheClient cache.Client) *Service {
	if playsStorage == nil {
		playsStorage = storage.NoopClient{}
	}

	if usersStorage == nil {
		usersStorage = storage.NoopClient{}
	}

	if downloadURLTTL <= 0 {
		downloadURLTTL = defaultDownloadURLTTL
	}

	if cacheClient == nil {
		cacheClient = cache.NoopClient{}
	}

	return &Service{repo: repo, playsStorage: playsStorage, usersStorage: usersStorage, downloadURLTTL: downloadURLTTL, cache: cacheClient}
}

func (s *Service) GetPlayMediaRedirectURL(ctx context.Context, playID string, mediaID string) (string, error) {
	if !isValidUUID(playID) {
		return "", sharederrors.Validation("invalid playId", nil)
	}

	if !isValidUUID(mediaID) {
		return "", sharederrors.Validation("invalid mediaId", nil)
	}

	cacheKey := buildPlayMediaRedirectCacheKey(playID, mediaID)
	if cachedURL, err := s.cache.Get(ctx, cacheKey); err == nil && cachedURL != "" {
		cache.RecordHit("media:redirect")
		return cachedURL, nil
	} else if err != nil {
		if errors.Is(err, cache.ErrCacheMiss) || errors.Is(err, cache.ErrClientNotConfigured) {
			cache.RecordMiss("media:redirect")
		} else {
			cache.RecordError("media:redirect")
			log.Printf("media redirect cache get failed: key=%s err=%v", cacheKey, err)
		}
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

	if ttl := cacheTTLForRedirect(s.downloadURLTTL); ttl > 0 {
		if err := s.cache.Set(ctx, cacheKey, url, ttl); err != nil {
			if !errors.Is(err, cache.ErrClientNotConfigured) {
				cache.RecordError("media:redirect")
			}
			log.Printf("media redirect cache set failed: key=%s err=%v", cacheKey, err)
		} else {
			cache.RecordSet("media:redirect")
		}
	} else {
		log.Printf("media redirect cache skipped due to ttl: key=%s download_ttl=%s", cacheKey, s.downloadURLTTL)
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

func buildPlayMediaRedirectCacheKey(playID string, mediaID string) string {
	return fmt.Sprintf("media:plays:redirect:%s:%s:%s", mediaRedirectCacheKeyVersion, playID, mediaID)
}

func cacheTTLForRedirect(downloadURLTTL time.Duration) time.Duration {
	cacheTTL := downloadURLTTL - mediaRedirectCacheSafetyWindow
	if cacheTTL <= 0 {
		return 0
	}

	return cacheTTL
}
