package ratelimiter

import (
	"context"
	"net/http"
	"net/http/httptest"
	mocks "rate-limiter/mocks/rate-limiter/cache"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetKey(t *testing.T) {
	rl := &RateLimiter{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("API_KEY", "test_api_key")

	key, keyType := rl.GetKey(c)

	assert.Equal(t, "test_api_key", key)
	assert.Equal(t, "api_key", keyType)

	t.Run("when no api key is defined", func(t *testing.T) {
		rl := &RateLimiter{}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Request.RemoteAddr = "127.0.0.1:8080"

		key, keyType := rl.GetKey(c)

		assert.Equal(t, "127.0.0.1", key)
		assert.Equal(t, "ip", keyType)
	})
}

func TestGetKeyConfg(t *testing.T) {
	rlopts := NewRateLimiterOptions()
	rlopts.TokenRateLimit = 50
	rlopts.TokenDurationTime = time.Minute * 5
	rlopts.TokenLimits = map[string]TokenLimitConfig{
		"existing_api_key": {
			Limit:         100,
			BlockDuration: time.Minute * 10,
		},
	}
	rlopts.IpRateLimit = 20
	rlopts.IpDurationTime = time.Minute * 2
	rl := &RateLimiter{
		options: rlopts,
	}

	t.Run("keyType is 'api_key' and key exists in TokenLimits", func(t *testing.T) {
		limit, duration := rl.GetKeyConfg("existing_api_key", "api_key")
		assert.Equal(t, 100, limit)
		assert.Equal(t, time.Minute*10, duration)
	})

	t.Run("keyType is 'api_key' and key does not exist in TokenLimits", func(t *testing.T) {
		limit, duration := rl.GetKeyConfg("non_existing_api_key", "api_key")
		assert.Equal(t, 50, limit)               // Default TokenRateLimit
		assert.Equal(t, time.Minute*5, duration) // Default TokenDurationTime
	})

	t.Run("keyType is not 'api_key'", func(t *testing.T) {
		limit, duration := rl.GetKeyConfg("127.0.0.1", "ip")
		assert.Equal(t, 20, limit)
		assert.Equal(t, time.Minute*2, duration)
	})
}

func TestAllow(t *testing.T) {
	ctx := context.Background()

	t.Run("when key is blocked", func(t *testing.T) {
		csMock := mocks.NewMockCacheService(t)
		csMock.EXPECT().IsBlocked(ctx, "test_key").Return(true, nil)

		rl := &RateLimiter{
			cs: csMock,
		}
		allow, err := rl.Allow(ctx, "test_key", 1, time.Second)
		require.NoError(t, err)
		assert.False(t, allow)
	})

	t.Run("when key is not blocked and incremente is below limit", func(t *testing.T) {
		keyName := "test_key"
		csMock := mocks.NewMockCacheService(t)
		csMock.EXPECT().IsBlocked(ctx, keyName).Return(false, nil)
		csMock.EXPECT().Increment(ctx, keyName, time.Second).Return(1, nil)

		rl := &RateLimiter{
			cs: csMock,
		}
		allow, err := rl.Allow(ctx, "test_key", 1, time.Second)
		require.NoError(t, err)
		assert.True(t, allow)
	})

	t.Run("when key is not blocked and increment fails", func(t *testing.T) {
		keyName := "test_key"
		csMock := mocks.NewMockCacheService(t)
		csMock.EXPECT().IsBlocked(ctx, keyName).Return(false, nil)
		csMock.EXPECT().Increment(ctx, keyName, time.Second).Return(0, assert.AnError)

		rl := &RateLimiter{
			cs: csMock,
		}
		allow, err := rl.Allow(ctx, keyName, 1, time.Second)
		require.Error(t, err)
		assert.False(t, allow)
	})

	t.Run("when key is not blocked and count is greater than limit", func(t *testing.T) {
		keyName := "test_key"
		csMock := mocks.NewMockCacheService(t)
		csMock.EXPECT().IsBlocked(ctx, keyName).Return(false, nil)
		csMock.EXPECT().Increment(ctx, keyName, time.Second).Return(2, nil)
		csMock.EXPECT().Block(ctx, keyName, time.Second).Return(nil)

		rl := &RateLimiter{
			cs: csMock,
		}
		allow, err := rl.Allow(ctx, keyName, 1, time.Second)
		require.NoError(t, err)
		assert.False(t, allow)
	})
}
