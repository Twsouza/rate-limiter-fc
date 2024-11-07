package cache

import (
	"context"
	"time"
)

type CacheService interface {
	Increment(ctx context.Context, key string, expiry time.Duration) (int, error)
	Get(ctx context.Context, key string) (int, error)
	SetExpiration(ctx context.Context, key string, expiry time.Duration) error
	IsBlocked(ctx context.Context, key string) (bool, error)
	Block(ctx context.Context, key string, blockDuration time.Duration) error
	Close() error
}
