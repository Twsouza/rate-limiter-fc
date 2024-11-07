package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewCacheService(ctx context.Context, addr string, password string) (CacheService, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return &RedisCache{
		client: client,
	}, nil
}

func (rs *RedisCache) Increment(ctx context.Context, key string, expiry time.Duration) (int, error) {
	count, err := rs.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		// Define a expiração
		err = rs.client.Expire(ctx, key, expiry).Err()
		if err != nil {
			return 0, err
		}
	}
	return int(count), nil
}

func (rs *RedisCache) Get(ctx context.Context, key string) (int, error) {
	val, err := rs.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	count, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (rs *RedisCache) SetExpiration(ctx context.Context, key string, expiry time.Duration) error {
	return rs.client.Expire(ctx, key, expiry).Err()
}

func (rs *RedisCache) IsBlocked(ctx context.Context, key string) (bool, error) {
	val, err := rs.client.Get(ctx, "block:"+key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	blocked, err := strconv.ParseBool(val)
	if err != nil {
		return false, err
	}
	return blocked, nil
}

func (rs *RedisCache) Block(ctx context.Context, key string, blockDuration time.Duration) error {
	err := rs.client.Set(ctx, "block:"+key, "true", blockDuration).Err()
	if err != nil {
		return err
	}
	rs.client.Del(ctx, key)
	return nil
}

func (rs *RedisCache) Close() error {
	return rs.client.Close()
}
