package ratelimiter

import (
	"context"
	"rate-limiter/cache"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiterService interface {
	GetKey(gc *gin.Context) string
	Allow(key string) (bool, error)
}

type RateLimiter struct {
	cs      cache.CacheService
	options *RateLimiterOptions
}

func NewRateLimiter(cs cache.CacheService, options ...Options) *RateLimiter {
	rlopts := NewRateLimiterOptions()
	for _, option := range options {
		option(rlopts)
	}

	return &RateLimiter{
		cs:      cs,
		options: rlopts,
	}
}

func (rl *RateLimiter) GetKey(gc *gin.Context) (string, string) {
	key := gc.GetHeader("API_KEY")
	keyType := "api_key"
	if key == "" {
		key = gc.ClientIP()
		keyType = "ip"
	}

	return key, keyType
}

func (rl *RateLimiter) GetKeyConfg(key string, keyType string) (int, time.Duration) {
	if keyType == "api_key" {
		config, ok := rl.options.TokenLimits[key]
		if !ok {
			return rl.options.TokenRateLimit, rl.options.TokenDurationTime
		}

		return config.Limit, config.BlockDuration
	}

	return rl.options.IpRateLimit, rl.options.IpDurationTime
}

func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int, blockDuration time.Duration) (bool, error) {
	blocked, err := rl.cs.IsBlocked(ctx, key)
	if err != nil {
		return false, err
	}
	if blocked {
		return false, nil
	}

	count, err := rl.cs.Increment(ctx, key, blockDuration)
	if err != nil {
		return false, err
	}

	if count > limit {
		err = rl.cs.Block(ctx, key, blockDuration)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	return true, nil
}
