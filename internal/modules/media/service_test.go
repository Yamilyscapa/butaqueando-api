package media

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/butaqueando/api/internal/shared/cache"
	"github.com/butaqueando/api/internal/shared/storage"
)

type fakeRepo struct {
	objectKey      string
	objectKeyErr   error
	objectKeyCalls int
	variants       []MediaVariant
	avatarKey      string
	avatarKeyErr   error
	avatarKeyCalls int
	avatarVariants []MediaVariant
}

func (f *fakeRepo) GetPublishedPlayMediaObjectKey(_ context.Context, _ string, _ string) (string, error) {
	f.objectKeyCalls++
	if f.objectKeyErr != nil {
		return "", f.objectKeyErr
	}
	return f.objectKey, nil
}

func (f *fakeRepo) GetUserAvatarObjectKey(_ context.Context, _ string) (string, error) {
	f.avatarKeyCalls++
	if f.avatarKeyErr != nil {
		return "", f.avatarKeyErr
	}
	return f.avatarKey, nil
}

func (f *fakeRepo) GetPublishedPlayMedia(_ context.Context, _ string, _ string) (MediaRecord, error) {
	f.objectKeyCalls++
	if f.objectKeyErr != nil {
		return MediaRecord{}, f.objectKeyErr
	}
	return MediaRecord{ObjectKey: f.objectKey, Variants: f.variants}, nil
}

func (f *fakeRepo) GetUserAvatar(_ context.Context, _ string) (MediaRecord, error) {
	f.avatarKeyCalls++
	if f.avatarKeyErr != nil {
		return MediaRecord{}, f.avatarKeyErr
	}
	return MediaRecord{ObjectKey: f.avatarKey, Variants: f.avatarVariants}, nil
}

type fakeStorage struct {
	url      string
	err      error
	getCalls int
}

func (f *fakeStorage) Bucket() string { return "test-bucket" }
func (f *fakeStorage) PresignPutObject(context.Context, storage.PresignPutObjectInput) (string, error) {
	return "", nil
}
func (f *fakeStorage) PresignGetObject(_ context.Context, _ storage.PresignGetObjectInput) (string, error) {
	f.getCalls++
	if f.err != nil {
		return "", f.err
	}
	return f.url, nil
}
func (f *fakeStorage) HeadObject(context.Context, storage.HeadObjectInput) (storage.HeadObjectOutput, error) {
	return storage.HeadObjectOutput{}, nil
}
func (f *fakeStorage) GetObject(context.Context, storage.GetObjectInput) ([]byte, error) {
	return nil, nil
}
func (f *fakeStorage) PutObject(context.Context, storage.PutObjectInput) error { return nil }

type fakeCache struct {
	values   map[string]string
	getErr   error
	setErr   error
	setTTL   time.Duration
	setCalls int
}

func (f *fakeCache) Get(_ context.Context, key string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	value, ok := f.values[key]
	if !ok {
		return "", cache.ErrCacheMiss
	}
	return value, nil
}

func (f *fakeCache) Set(_ context.Context, key string, value string, ttl time.Duration) error {
	f.setCalls++
	f.setTTL = ttl
	if f.setErr != nil {
		return f.setErr
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key] = value
	return nil
}

func (f *fakeCache) Del(_ context.Context, keys ...string) error {
	if f.values == nil {
		return nil
	}
	for _, key := range keys {
		delete(f.values, key)
	}
	return nil
}

func (f *fakeCache) DelByPattern(_ context.Context, pattern string) (int, error) {
	if f.values == nil {
		return 0, nil
	}
	prefix := strings.TrimSuffix(pattern, "*")
	deleted := 0
	for key := range f.values {
		if strings.HasPrefix(key, prefix) {
			delete(f.values, key)
			deleted++
		}
	}
	return deleted, nil
}

func (f *fakeCache) Close() error { return nil }

func TestGetPlayMediaRedirectURL_UsesCacheHit(t *testing.T) {
	repo := &fakeRepo{objectKey: "plays/a.jpg"}
	storageClient := &fakeStorage{url: "https://signed.example.com/fresh"}
	cacheClient := &fakeCache{values: map[string]string{buildPlayMediaRedirectCacheKey(validPlayID, validMediaID, 0): "https://signed.example.com/cached"}}

	service := NewService(repo, storageClient, storage.NoopClient{}, 15*time.Minute, cacheClient)

	url, err := service.GetPlayMediaRedirectURL(context.Background(), validPlayID, validMediaID, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if url != "https://signed.example.com/cached" {
		t.Fatalf("expected cached URL, got %q", url)
	}
	if repo.objectKeyCalls != 0 {
		t.Fatalf("expected repository not to be called, got %d", repo.objectKeyCalls)
	}
	if storageClient.getCalls != 0 {
		t.Fatalf("expected storage not to be called, got %d", storageClient.getCalls)
	}
}

func TestGetPlayMediaRedirectURL_CachesOnMiss(t *testing.T) {
	repo := &fakeRepo{objectKey: "plays/a.jpg"}
	storageClient := &fakeStorage{url: "https://signed.example.com/fresh"}
	cacheClient := &fakeCache{values: map[string]string{}}

	service := NewService(repo, storageClient, storage.NoopClient{}, 15*time.Minute, cacheClient)

	url, err := service.GetPlayMediaRedirectURL(context.Background(), validPlayID, validMediaID, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if url != "https://signed.example.com/fresh" {
		t.Fatalf("expected fresh URL, got %q", url)
	}
	if repo.objectKeyCalls != 1 {
		t.Fatalf("expected repository call count 1, got %d", repo.objectKeyCalls)
	}
	if storageClient.getCalls != 1 {
		t.Fatalf("expected storage call count 1, got %d", storageClient.getCalls)
	}
	if cacheClient.setCalls != 1 {
		t.Fatalf("expected cache set call count 1, got %d", cacheClient.setCalls)
	}
	if cacheClient.setTTL != 14*time.Minute {
		t.Fatalf("expected cache ttl 14m, got %s", cacheClient.setTTL)
	}
}

func TestGetPlayMediaRedirectURL_FallbackWhenCacheFails(t *testing.T) {
	repo := &fakeRepo{objectKey: "plays/a.jpg"}
	storageClient := &fakeStorage{url: "https://signed.example.com/fresh"}
	cacheClient := &fakeCache{getErr: errors.New("cache unavailable"), setErr: errors.New("cache unavailable")}

	service := NewService(repo, storageClient, storage.NoopClient{}, 15*time.Minute, cacheClient)

	url, err := service.GetPlayMediaRedirectURL(context.Background(), validPlayID, validMediaID, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if url != "https://signed.example.com/fresh" {
		t.Fatalf("expected fresh URL, got %q", url)
	}
	if repo.objectKeyCalls != 1 {
		t.Fatalf("expected repository call count 1, got %d", repo.objectKeyCalls)
	}
}

func TestGetPlayMediaRedirectURL_SkipsCacheWhenTTLTooShort(t *testing.T) {
	repo := &fakeRepo{objectKey: "plays/a.jpg"}
	storageClient := &fakeStorage{url: "https://signed.example.com/fresh"}
	cacheClient := &fakeCache{values: map[string]string{}}

	service := NewService(repo, storageClient, storage.NoopClient{}, 45*time.Second, cacheClient)

	_, err := service.GetPlayMediaRedirectURL(context.Background(), validPlayID, validMediaID, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cacheClient.setCalls != 0 {
		t.Fatalf("expected cache set to be skipped, got %d calls", cacheClient.setCalls)
	}
}

const (
	validPlayID  = "00000000-0000-0000-0000-000000000201"
	validMediaID = "00000000-0000-0000-0000-000000000401"
)
