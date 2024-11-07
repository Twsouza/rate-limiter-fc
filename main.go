package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"rate-limiter/cache"
	"rate-limiter/ratelimiter"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		logrus.Debug("Error loading .env file")
	}

	ctx := context.Background()
	cs, err := cache.NewCacheService(ctx, os.Getenv("REDIS_ADDR"), os.Getenv("REDIS_PASSWORD"))
	if err != nil {
		logrus.Fatalf("Error creating cache service on %s: %v", os.Getenv("REDIS_ADDR"), err)
	}

	rlOpts, err := LoadRateLimiterConfigFromEnv()
	if err != nil {
		logrus.Fatalf("Error loading rate limiter config: %v", err)
	}
	rls := ratelimiter.NewRateLimiter(
		cs,
		rlOpts...,
	)

	r := gin.Default()
	r.Use(rls.Middleware())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Errorf("Error starting server: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down server...")

	if err := cs.Close(); err != nil {
		logrus.Errorf("Error closing cache service: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatal("Error while shutting down Server. Initiating force shutdown...", err)
	}

	logrus.Info("Server exiting")
}

func LoadRateLimiterConfigFromEnv() ([]ratelimiter.Options, error) {
	rlOpts := []ratelimiter.Options{}
	ipRateLimitStr := os.Getenv("IP_RATE_LIMIT")
	if ipRateLimitStr != "" {
		ipRateLimit, err := strconv.Atoi(ipRateLimitStr)
		if err != nil {
			return nil, fmt.Errorf("Error parsing IP rate limit: %v", err)
		}
		rlOpts = append(rlOpts, ratelimiter.WithIpRateLimit(ipRateLimit))
	}

	ipDurationTimeStr := os.Getenv("IP_BLOCK_DURATION")
	if ipDurationTimeStr != "" {
		ipDurationTimeInt, err := strconv.Atoi(ipDurationTimeStr)
		if err != nil {
			return nil, fmt.Errorf("Error parsing IP duration time: %v", err)
		}
		ipDurationTime := time.Duration(ipDurationTimeInt) * time.Second
		rlOpts = append(rlOpts, ratelimiter.WithIpDurationTime(ipDurationTime))
	}

	tokenRateLimitStr := os.Getenv("TOKEN_RATE_LIMIT")
	if tokenRateLimitStr != "" {
		tokenRateLimit, err := strconv.Atoi(tokenRateLimitStr)
		if err != nil {
			return nil, fmt.Errorf("Error parsing token rate limit: %v", err)
		}
		rlOpts = append(rlOpts, ratelimiter.WithTokenRateLimit(tokenRateLimit))
	}

	tokenDurationTimeStr := os.Getenv("TOKEN_BLOCK_DURATION")
	if tokenDurationTimeStr != "" {
		tokenDurationTimeInt, err := strconv.Atoi(tokenDurationTimeStr)
		if err != nil {
			return nil, fmt.Errorf("Error parsing token duration time: %v", err)
		}
		tokenDurationTime := time.Duration(tokenDurationTimeInt) * time.Second
		rlOpts = append(rlOpts, ratelimiter.WithTokenDurationTime(tokenDurationTime))
	}

	tokenLimitsStr := os.Getenv("TOKEN_LIMITS")
	tokenLimits := make(map[string]ratelimiter.TokenLimitConfig)
	if tokenLimitsStr != "" {
		var tokenLimitsConfig map[string]struct {
			Limit         int `json:"limit"`
			BlockDuration int `json:"block_duration"`
		}
		err := json.Unmarshal([]byte(tokenLimitsStr), &tokenLimitsConfig)
		if err != nil {
			return nil, fmt.Errorf("Error parsing token limits: %v", err)
		}

		for token, config := range tokenLimitsConfig {
			tokenLimits[token] = ratelimiter.TokenLimitConfig{
				Limit:         config.Limit,
				BlockDuration: time.Duration(config.BlockDuration) * time.Second,
			}
		}

		if len(tokenLimits) > 0 {
			rlOpts = append(rlOpts, ratelimiter.WithTokenLimits(tokenLimits))
		}
	}

	return rlOpts, nil
}
