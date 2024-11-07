package ratelimiter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"rate-limiter/cache"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestMiddleware(t *testing.T) {
	ctx := context.Background()
	redisContainer, err := redis.Run(ctx, "redis:6")
	require.NoError(t, err, "Failed to start redis container")

	host, err := redisContainer.Host(ctx)
	require.NoError(t, err, "Failed to get redis container host")

	port, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err, "Failed to get redis container port")

	cs, err := cache.NewCacheService(ctx, fmt.Sprintf("%s:%s", host, port.Port()), "")
	if err != nil {
		logrus.Fatalf("Error creating cache service: %v", err)
	}

	rls := NewRateLimiter(cs)

	r := gin.Default()
	r.Use(rls.Middleware())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	go func() {
		r.Run()
	}()

	time.Sleep(2 * time.Second)

	req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
	require.NoError(t, err, "Failed to create request")

	client := http.DefaultClient
	resp, err := client.Do(req)
	require.NoError(t, err, "Failed to make request")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	require.Equal(t, `{"message":"pong"}`, string(body))
}
