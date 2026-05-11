package media

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/butaqueando/api/internal/shared/cache"
	sharederrors "github.com/butaqueando/api/internal/shared/errors"
	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultDownloadURLTTL = 15 * time.Minute
const mediaRedirectCacheKeyVersion = "v2"
const mediaRedirectCacheSafetyWindow = time.Minute

type repositoryPort interface {
	GetPublishedPlayMediaObjectKey(ctx context.Context, playID string, mediaID string) (string, error)
	GetUserAvatarObjectKey(ctx context.Context, userID string) (string, error)
	GetPublishedPlayMedia(ctx context.Context, playID string, mediaID string) (MediaRecord, error)
	GetUserAvatar(ctx context.Context, userID string) (MediaRecord, error)
}

type Service struct {
	repo            repositoryPort
	playsStorage    storage.Client
	usersStorage    storage.Client
	downloadURLTTL  time.Duration
	cache           cache.Client
	playsVariants   []int
	avatarsVariants []int
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

	return &Service{
		repo:            repo,
		playsStorage:    playsStorage,
		usersStorage:    usersStorage,
		downloadURLTTL:  downloadURLTTL,
		cache:           cacheClient,
		playsVariants:   []int{320, 720, 1080},
		avatarsVariants: []int{96, 240, 512},
	}
}

// SetVariantWidths overrides the default variant widths used to clamp the ?w= query.
func (s *Service) SetVariantWidths(plays []int, avatars []int) {
	if len(plays) > 0 {
		copied := make([]int, 0, len(plays))
		for _, w := range plays {
			if w > 0 {
				copied = append(copied, w)
			}
		}
		if len(copied) > 0 {
			sort.Ints(copied)
			s.playsVariants = copied
		}
	}
	if len(avatars) > 0 {
		copied := make([]int, 0, len(avatars))
		for _, w := range avatars {
			if w > 0 {
				copied = append(copied, w)
			}
		}
		if len(copied) > 0 {
			sort.Ints(copied)
			s.avatarsVariants = copied
		}
	}
}

func (s *Service) GetPlayMediaRedirectURL(ctx context.Context, playID string, mediaID string, requestedWidth int) (string, error) {
	if !isValidUUID(playID) {
		return "", sharederrors.Validation("invalid playId", nil)
	}

	if !isValidUUID(mediaID) {
		return "", sharederrors.Validation("invalid mediaId", nil)
	}

	cacheKey := buildPlayMediaRedirectCacheKey(playID, mediaID, requestedWidth)
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

	record, err := s.repo.GetPublishedPlayMedia(ctx, playID, mediaID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", sharederrors.NotFound("play media not found", nil)
		}

		return "", sharederrors.Internal("failed to load media", nil)
	}

	objectKey := pickVariantObjectKey(record, requestedWidth)

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

func (s *Service) GetUserAvatarRedirectURL(ctx context.Context, userID string, requestedWidth int) (string, error) {
	if !isValidUUID(userID) {
		return "", sharederrors.Validation("invalid userId", nil)
	}

	record, err := s.repo.GetUserAvatar(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", sharederrors.NotFound("avatar not found", nil)
		}

		return "", sharederrors.Internal("failed to load media", nil)
	}

	objectKey := pickVariantObjectKey(record, requestedWidth)

	url, err := s.usersStorage.PresignGetObject(ctx, storage.PresignGetObjectInput{ObjectKey: objectKey, ExpiresIn: s.downloadURLTTL})
	if err != nil {
		return "", sharederrors.Internal("failed to load media", nil)
	}

	return url, nil
}

// pickVariantObjectKey selects the smallest variant whose width >= requestedWidth.
// requestedWidth <= 0 falls back to the legacy/primary object_key (typically the largest variant).
// If variants is empty, the legacy object_key is returned as-is.
func pickVariantObjectKey(record MediaRecord, requestedWidth int) string {
	if len(record.Variants) == 0 || requestedWidth <= 0 {
		return record.ObjectKey
	}

	sorted := make([]MediaVariant, len(record.Variants))
	copy(sorted, record.Variants)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Width < sorted[j].Width })

	for _, variant := range sorted {
		if variant.Width >= requestedWidth {
			return variant.ObjectKey
		}
	}
	// All variants smaller than requested: serve the largest (= legacy).
	return sorted[len(sorted)-1].ObjectKey
}

func isValidUUID(raw string) bool {
	_, err := uuid.Parse(raw)
	return err == nil
}

func buildPlayMediaRedirectCacheKey(playID string, mediaID string, width int) string {
	return fmt.Sprintf("media:plays:redirect:%s:%s:%s:%d", mediaRedirectCacheKeyVersion, playID, mediaID, width)
}

func cacheTTLForRedirect(downloadURLTTL time.Duration) time.Duration {
	cacheTTL := downloadURLTTL - mediaRedirectCacheSafetyWindow
	if cacheTTL <= 0 {
		return 0
	}

	return cacheTTL
}
