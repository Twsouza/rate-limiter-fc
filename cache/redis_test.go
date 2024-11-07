package cache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

var (
	redisContainer *redis.RedisContainer
	cacheService   CacheService
	ctx            = context.Background()
)

func TestMain(m *testing.M) {
	var err error
	// Start Redis container
	redisContainer, err := redis.Run(ctx, "redis:6")
	if err != nil {
		fmt.Printf("Failed to start Redis container: %v\n", err)
		os.Exit(1)
	}

	// Retrieve host and port
	host, err := redisContainer.Host(ctx)
	if err != nil {
		fmt.Printf("Failed to get Redis host and port: %v\n", err)
		os.Exit(1)
	}

	port, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		fmt.Printf("Failed to get Redis host and port: %v\n", err)
		os.Exit(1)
	}

	address := fmt.Sprintf("%s:%s", host, port.Port())

	// Initialize cache service
	cacheService, err = NewCacheService(ctx, address, "")
	if err != nil {
		fmt.Printf("Failed to create cache service: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Teardown Redis container
	if err := redisContainer.Terminate(ctx); err != nil {
		fmt.Printf("Failed to terminate Redis container: %v\n", err)
	}

	os.Exit(code)
}

func TestNewCacheService(t *testing.T) {
	// Already tested in TestMain
	assert.NotNil(t, cacheService, "CacheService should be initialized")
}

func TestIncrement(t *testing.T) {
	key := "testIncrement"
	expiry := time.Minute

	// Increment a key and verify the count
	count, err := cacheService.Increment(ctx, key, expiry)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Count should be 1 after first increment")

	// Test expiration is set on first increment
	ttl, err := cacheService.(*RedisCache).client.TTL(ctx, key).Result()
	assert.NoError(t, err)
	assert.True(t, ttl > 0, "TTL should be set on first increment")

	// Increment again and verify count
	count, err = cacheService.Increment(ctx, key, expiry)
	assert.NoError(t, err)
	assert.Equal(t, 2, count, "Count should be 2 after second increment")
}

func TestGet(t *testing.T) {
	key := "testGet"

	// Set a key
	err := cacheService.(*RedisCache).client.Set(ctx, key, "5", 0).Err()
	assert.NoError(t, err, "Setting key should not produce an error")

	// Retrieve it using Get
	count, err := cacheService.Get(ctx, key)
	assert.NoError(t, err, "Get should not produce an error")
	assert.Equal(t, 5, count, "Retrieved count should be 5")

	// Test retrieval of non-existent key returns 0
	count, err = cacheService.Get(ctx, "nonexistent")
	assert.NoError(t, err, "Get on nonexistent key should not produce an error")
	assert.Equal(t, 0, count, "Nonexistent key should return 0")
}

func TestSetExpiration(t *testing.T) {
	key := "testSetExpiration"
	expiry := 2 * time.Minute

	// Set a key
	err := cacheService.(*RedisCache).client.Set(ctx, key, "10", 0).Err()
	assert.NoError(t, err, "Setting key should not produce an error")

	// Set expiration on the key
	err = cacheService.SetExpiration(ctx, key, expiry)
	assert.NoError(t, err, "SetExpiration should not produce an error")

	// Verify expiration is set
	ttl, err := cacheService.(*RedisCache).client.TTL(ctx, key).Result()
	assert.NoError(t, err, "TTL retrieval should not produce an error")
	assert.True(t, ttl > 0, "TTL should be set")
}

func TestIsBlocked(t *testing.T) {
	key := "testIsBlocked"

	// Block a key
	err := cacheService.Block(ctx, key, 5*time.Minute)
	assert.NoError(t, err, "Blocking key should not produce an error")

	// Verify IsBlocked returns true
	blocked, err := cacheService.IsBlocked(ctx, key)
	assert.NoError(t, err, "IsBlocked should not produce an error")
	assert.True(t, blocked, "Key should be blocked")

	// Test unblocked key returns false
	blocked, err = cacheService.IsBlocked(ctx, "unblockedKey")
	assert.NoError(t, err, "IsBlocked should not produce an error")
	assert.False(t, blocked, "Key should not be blocked")
}

func TestBlock(t *testing.T) {
	key := "testBlock"
	blockDuration := 5 * time.Minute

	// Block a key
	err := cacheService.Block(ctx, key, blockDuration)
	assert.NoError(t, err, "Block should not produce an error")

	// Verify it's blocked
	blocked, err := cacheService.IsBlocked(ctx, key)
	assert.NoError(t, err, "IsBlocked should not produce an error")
	assert.True(t, blocked, "Key should be blocked")

	// Ensure the original key is deleted
	exists, err := cacheService.(*RedisCache).client.Exists(ctx, key).Result()
	assert.NoError(t, err, "Exists should not produce an error")
	assert.Equal(t, int64(0), exists, "Original key should be deleted")
}

func TestClose(t *testing.T) {
	// Close the cache service
	err := cacheService.Close()
	assert.NoError(t, err, "Close should not produce an error")
}
