package ratelimiter

import "time"

type RateLimiterOptions struct {
	IpRateLimit       int
	IpDurationTime    time.Duration
	TokenRateLimit    int
	TokenDurationTime time.Duration
	TokenLimits       map[string]TokenLimitConfig
}

type TokenLimitConfig struct {
	Limit         int
	BlockDuration time.Duration
}

type Options func(*RateLimiterOptions)

func NewRateLimiterOptions() *RateLimiterOptions {
	return &RateLimiterOptions{
		IpRateLimit:       10,
		IpDurationTime:    time.Minute,
		TokenRateLimit:    10,
		TokenDurationTime: time.Minute,
	}
}

func WithIpRateLimit(rateLimit int) Options {
	return func(o *RateLimiterOptions) {
		o.IpRateLimit = rateLimit
	}
}

func WithIpDurationTime(duration time.Duration) Options {
	return func(o *RateLimiterOptions) {
		o.IpDurationTime = duration
	}
}

func WithTokenRateLimit(rateLimit int) Options {
	return func(o *RateLimiterOptions) {
		o.TokenRateLimit = rateLimit
	}
}

func WithTokenDurationTime(duration time.Duration) Options {
	return func(o *RateLimiterOptions) {
		o.TokenDurationTime = duration
	}
}

func WithTokenLimits(tokenLimits map[string]TokenLimitConfig) Options {
	return func(o *RateLimiterOptions) {
		o.TokenLimits = tokenLimits
	}
}
