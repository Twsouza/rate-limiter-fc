package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"rate-limiter/ratelimiter"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestMain(t *testing.T) {
	ctx := context.Background()
	redisContainer, err := redis.Run(ctx, "redis:6")
	require.NoError(t, err, "Failed to start redis container")

	host, err := redisContainer.Host(ctx)
	require.NoError(t, err, "Failed to get redis container host")

	port, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err, "Failed to get redis container port")

	redisAddr := fmt.Sprintf("%s:%s", host, port.Port())
	os.Setenv("REDIS_ADDR", redisAddr)
	os.Setenv("REDIS_PASSWORD", "")

	os.Setenv("IP_RATE_LIMIT", "10")
	os.Setenv("IP_BLOCK_DURATION", "5")

	os.Setenv("TOKEN_RATE_LIMIT", "12")
	os.Setenv("TOKEN_BLOCK_DURATION", "3")

	configurableLimit := 10000
	os.Setenv("TOKEN_LIMITS", fmt.Sprintf(`{"token10":{"limit":%d,"block_duration":1}}`, configurableLimit))

	go func() {
		main()
	}()

	time.Sleep(2 * time.Second)

	t.Run("with ip rate limit", func(t *testing.T) {
		client := http.DefaultClient
		wg := &sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()

			for i := 0; i < 5; i++ {
				req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
				require.NoError(t, err, "Failed to create request")

				resp, err := client.Do(req)
				require.NoError(t, err, "Failed to make request")
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "Failed to read response body")
				assert.Equal(t, `{"message":"pong"}`, string(body))
			}
		}()

		go func() {
			defer wg.Done()

			for i := 0; i < 5; i++ {
				req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
				require.NoError(t, err, "Failed to create request")

				resp, err := client.Do(req)
				require.NoError(t, err, "Failed to make request")
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		}()

		wg.Wait()

		req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
		require.NoError(t, err, "Failed to create request")

		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to make request")

		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

		bytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		assert.Equal(t, "{\"error\":\"you have reached the maximum number of requests or actions allowed within a certain time frame\"}", string(bytes))

		// wait 5 seconds and try again
		time.Sleep(5 * time.Second)

		req, err = http.NewRequest("GET", "http://localhost:8080/", nil)
		require.NoError(t, err)

		resp, err = client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		bytes, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, `{"message":"pong"}`, string(bytes))
	})

	t.Run("with token rate limit", func(t *testing.T) {
		client := http.DefaultClient
		wg := &sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()

			for i := 0; i < 5; i++ {
				req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
				require.NoError(t, err, "Failed to create request")
				req.Header.Set("API_KEY", "token1")

				resp, err := client.Do(req)
				require.NoError(t, err, "Failed to make request")
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "Failed to read response body")
				assert.Equal(t, `{"message":"pong"}`, string(body))
			}
		}()

		go func() {
			defer wg.Done()

			for i := 0; i < 5; i++ {
				req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
				require.NoError(t, err, "Failed to create request")
				req.Header.Set("API_KEY", "token1")

				resp, err := client.Do(req)
				require.NoError(t, err, "Failed to make request")
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		}()

		wg.Wait()

		req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
		require.NoError(t, err, "Failed to create request")
		req.Header.Set("API_KEY", "token1")

		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to make request")

		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

		bytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		assert.Equal(t, `{\"error\":\"you have reached the maximum number of requests or actions allowed within a certain time frame\"}`, string(bytes))

		// other token should still be able to access
		req, err = http.NewRequest("GET", "http://localhost:8080/", nil)
		require.NoError(t, err, "Failed to create request")
		req.Header.Set("API_KEY", "token2")

		resp, err = client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		bytes, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, `{"message":"pong"}`, string(bytes))

		// wait 3 seconds and try again
		time.Sleep(3 * time.Second)

		req, err = http.NewRequest("GET", "http://localhost:8080/", nil)
		require.NoError(t, err, "Failed to create request")
		req.Header.Set("API_KEY", "token1")

		resp, err = client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		bytes, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, `{"message":"pong"}`, string(bytes))
	})

	t.Run("token with a different rate limit", func(t *testing.T) {
		client := http.DefaultClient
		wg := &sync.WaitGroup{}
		wg.Add(configurableLimit)

		for i := 0; i < configurableLimit; i++ {
			go func() {
				defer wg.Done()

				req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
				require.NoError(t, err, "Failed to create request")
				req.Header.Set("API_KEY", "token10")

				resp, err := client.Do(req)
				require.NoError(t, err, "Failed to make request")
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "Failed to read response body")
				assert.Equal(t, `{"message":"pong"}`, string(body))
			}()
		}

		wg.Wait()

		req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
		require.NoError(t, err, "Failed to create request")
		req.Header.Set("API_KEY", "token10")

		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to make request")

		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

		bytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		assert.Equal(t, "{\"error\":\"you have reached the maximum number of requests or actions allowed within a certain time frame\"}", string(bytes))

		// wait 1 seconds and try again
		time.Sleep(1 * time.Second)

		req, err = http.NewRequest("GET", "http://localhost:8080/", nil)
		require.NoError(t, err, "Failed to create request")
		req.Header.Set("API_KEY", "token10")

		resp, err = client.Do(req)
		require.NoError(t, err, "Failed to make request")
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		bytes, err = io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")
		assert.Equal(t, `{"message":"pong"}`, string(bytes))
	})
}

func TestLoadRateLimiterConfigFromEnv(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		expectedErr    bool
		expectedConfig *ratelimiter.RateLimiterOptions
	}{
		{
			name: "valid IP rate limit",
			envVars: map[string]string{
				"IP_RATE_LIMIT": "10",
			},
			expectedErr: false,
			expectedConfig: &ratelimiter.RateLimiterOptions{
				IpRateLimit: 10,
			},
		},
		{
			name: "invalid IP rate limit",
			envVars: map[string]string{
				"IP_RATE_LIMIT": "invalid",
			},
			expectedErr: true,
		},
		{
			name: "valid token limits",
			envVars: map[string]string{
				"TOKEN_LIMITS": `{"token1":{"limit":5,"block_duration":10}}`,
			},
			expectedErr: false,
			expectedConfig: &ratelimiter.RateLimiterOptions{
				TokenLimits: map[string]ratelimiter.TokenLimitConfig{
					"token1": {
						Limit:         5,
						BlockDuration: 10 * time.Second,
					},
				},
			},
		},
		{
			name: "invalid token limits",
			envVars: map[string]string{
				"TOKEN_LIMITS": `invalid`,
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for the test
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Load configurations from environment variables
			opts, err := LoadRateLimiterConfigFromEnv()
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Apply options to a new RateLimiterOptions instance
				configValues := &ratelimiter.RateLimiterOptions{}
				for _, opt := range opts {
					opt(configValues)
				}

				// Compare the actual config with the expected config
				assert.Equal(t, tt.expectedConfig, configValues)
			}

			// Unset environment variables after the test
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}
