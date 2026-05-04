package cache

import (
	"context"
	"time"
)

type Client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	DelByPattern(ctx context.Context, pattern string) (int, error)
	Close() error
}

type NoopClient struct{}

func (NoopClient) Get(_ context.Context, _ string) (string, error) {
	return "", ErrClientNotConfigured
}

func (NoopClient) Set(_ context.Context, _ string, _ string, _ time.Duration) error {
	return ErrClientNotConfigured
}

func (NoopClient) Del(_ context.Context, _ ...string) error {
	return nil
}

func (NoopClient) DelByPattern(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (NoopClient) Close() error {
	return nil
}
