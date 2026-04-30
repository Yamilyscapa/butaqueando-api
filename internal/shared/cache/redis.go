package cache

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClientFromURL(rawURL string) (*RedisClient, error) {
	redisURL := strings.TrimSpace(rawURL)
	if redisURL == "" {
		return nil, ErrClientNotConfigured
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	return &RedisClient{client: redis.NewClient(opts)}, nil
}

func (c *RedisClient) Get(ctx context.Context, key string) (string, error) {
	if c == nil || c.client == nil {
		return "", ErrClientNotConfigured
	}

	value, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrCacheMiss
		}
		return "", err
	}

	return value, nil
}

func (c *RedisClient) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return ErrClientNotConfigured
	}

	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *RedisClient) Close() error {
	if c == nil || c.client == nil {
		return nil
	}

	return c.client.Close()
}
