package cache

import (
	"context"
	"time"
)

type Client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Close() error
}

type NoopClient struct{}

func (NoopClient) Get(_ context.Context, _ string) (string, error) {
	return "", ErrClientNotConfigured
}

func (NoopClient) Set(_ context.Context, _ string, _ string, _ time.Duration) error {
	return ErrClientNotConfigured
}

func (NoopClient) Close() error {
	return nil
}
