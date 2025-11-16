package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter manages rate limiting for API requests.
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	logger   *slog.Logger
}

// NewRateLimiter creates a new rate limiter.
//
// Parameters:
//   - requestsPerSecond: Number of requests allowed per second
//   - burst: Maximum burst size
//   - logger: Structured logger
func NewRateLimiter(requestsPerSecond int, burst int, logger *slog.Logger) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
		logger:   logger,
	}
}

// getLimiter returns a rate limiter for the given key (IP address or user ID).
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[key] = limiter
	}

	return limiter
}

// Middleware returns a middleware function that enforces rate limiting.
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use IP address as key
			// TODO: Use user ID if authenticated for better rate limiting
			key := r.RemoteAddr

			limiter := rl.getLimiter(key)

			if !limiter.Allow() {
				rl.logger.Warn("rate limit exceeded",
					slog.String("key", key),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
				)

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "rate limit exceeded", "message": "too many requests"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CleanupRoutine removes old limiters to prevent memory leaks.
// Should be run as a goroutine.
func (rl *RateLimiter) CleanupRoutine(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()

		// Remove limiters that haven't been used recently
		// This is a simple approach - in production, track last used time
		if len(rl.limiters) > 10000 {
			rl.logger.Info("cleaning up rate limiters",
				slog.Int("count", len(rl.limiters)),
			)
			rl.limiters = make(map[string]*rate.Limiter)
		}

		rl.mu.Unlock()
	}
}

// RateLimit creates a simple rate limiting middleware.
// For production, use a distributed rate limiter (Redis, etc.).
func RateLimit(requestsPerSecond, burst int, logger *slog.Logger) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(requestsPerSecond, burst, logger)

	// Start cleanup routine
	go limiter.CleanupRoutine(5 * time.Minute)

	return limiter.Middleware()
}
