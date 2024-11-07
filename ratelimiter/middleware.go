package ratelimiter

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key, keyType := rl.GetKey(c)
		limit, blockDuration := rl.GetKeyConfg(key, keyType)

		allow, err := rl.Allow(c.Request.Context(), key, limit, blockDuration)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		if !allow {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "you have reached the maximum number of requests or actions allowed within a certain time frame",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
