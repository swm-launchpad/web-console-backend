package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter stores rate limiters for different clients
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	r        rate.Limit
	b        int
}

// NewRateLimiter creates a new rate limiter
// r is the rate (requests per second)
// b is the burst size
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

// getLimiter returns the rate limiter for the given key (typically IP address or user ID)
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.r, rl.b)
		rl.limiters[key] = limiter
	}

	return limiter
}

// Cleanup removes old limiters to prevent memory leaks
func (rl *RateLimiter) Cleanup() {
	ticker := time.NewTicker(time.Hour)
	go func() {
		for range ticker.C {
			rl.mu.Lock()
			// Reset the map periodically
			rl.limiters = make(map[string]*rate.Limiter)
			rl.mu.Unlock()
		}
	}()
}

// RateLimit returns a middleware that limits requests based on IP address
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	// Start cleanup goroutine
	rl.Cleanup()

	return func(c *gin.Context) {
		// Use IP address as the key
		key := c.ClientIP()

		limiter := rl.getLimiter(key)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByUserID returns a middleware that limits requests based on user ID
// This should be used after authentication middleware
func (rl *RateLimiter) RateLimitByUserID(userIDKey string) gin.HandlerFunc {
	// Start cleanup goroutine
	rl.Cleanup()

	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get(userIDKey)
		if !exists {
			// If no user ID, fall back to IP-based limiting
			key := c.ClientIP()
			limiter := rl.getLimiter(key)
			if !limiter.Allow() {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "Too many requests. Please try again later.",
				})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// Use user ID as the key
		// Convert uint to string for use as map key
		key := fmt.Sprint(userID)
		limiter := rl.getLimiter(key)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// NewEmailRateLimiter creates a rate limiter specifically for email endpoints
// Allows 3 requests per minute with burst of 5
func NewEmailRateLimiter() *RateLimiter {
	return NewRateLimiter(rate.Every(time.Minute/3), 5)
}
