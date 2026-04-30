package cache

import "errors"

var ErrClientNotConfigured = errors.New("cache client not configured")
var ErrCacheMiss = errors.New("cache miss")
