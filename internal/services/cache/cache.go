// Package cache provides caching interfaces and implementations for the fitness application.
// It supports both in-memory and distributed caching (Redis) with a unified interface
// for nutrition data caching and idempotency key management.
package cache

import (
	"context"
	"time"
)

// Cache defines the unified interface for all cache implementations.
// Both in-memory and Redis implementations must satisfy this interface,
// allowing for drop-in replacement without code changes.
type Cache interface {
	// Nutrition cache operations for AI response caching
	// GetNutrition retrieves cached nutrition data by key.
	// Returns ErrCacheMiss if the key is not found or has expired.
	GetNutrition(ctx context.Context, key string) (*NutritionResponse, error)

	// SetNutrition stores nutrition data with an expiration time.
	// ttl of 0 means no expiration (cache until manually deleted or evicted).
	SetNutrition(ctx context.Context, key string, value *NutritionResponse, ttl time.Duration) error

	// Idempotency operations for duplicate request prevention
	// GetIdempotency retrieves idempotency record by key.
	// Returns ErrCacheMiss if the key is not found or has expired.
	GetIdempotency(ctx context.Context, key string) (*IdempotencyRecord, error)

	// SetIdempotency stores idempotency record with an expiration time.
	// ttl should typically be 24 hours for idempotency keys.
	SetIdempotency(ctx context.Context, key string, value *IdempotencyRecord, ttl time.Duration) error

	// Generic cache operations
	// Delete removes a key from the cache.
	// Returns nil even if the key doesn't exist (idempotent operation).
	Delete(ctx context.Context, key string) error

	// Clear removes all keys from the cache.
	// Use with caution in production environments.
	Clear(ctx context.Context) error

	// Stats returns cache statistics (hits, misses, size, etc.)
	Stats() CacheStats

	// Close gracefully shuts down the cache and releases resources.
	Close() error
}

// NutritionResponse represents cached nutrition analysis from AI.
type NutritionResponse struct {
	// FoodName is the name of the analyzed food item
	FoodName string `json:"food_name"`

	// Calories is the total caloric content
	Calories float64 `json:"calories"`

	// Protein in grams
	Protein float64 `json:"protein"`

	// Carbohydrates in grams
	Carbohydrates float64 `json:"carbohydrates"`

	// Fat in grams
	Fat float64 `json:"fat"`

	// Fiber in grams
	Fiber float64 `json:"fiber"`

	// ServingSize describes the portion analyzed
	ServingSize string `json:"serving_size"`

	// Confidence is the AI's confidence score (0.0 to 1.0)
	Confidence float64 `json:"confidence"`

	// Source identifies which AI provider generated this data
	Source string `json:"source"`

	// CachedAt is when this response was cached
	CachedAt time.Time `json:"cached_at"`
}

// IdempotencyRecord tracks request processing to prevent duplicates.
type IdempotencyRecord struct {
	// RequestID is the unique identifier for the original request
	RequestID string `json:"request_id"`

	// StatusCode is the HTTP status code of the completed request
	StatusCode int `json:"status_code"`

	// ResponseBody is the serialized response body
	ResponseBody []byte `json:"response_body"`

	// CreatedAt is when the request was first processed
	CreatedAt time.Time `json:"created_at"`

	// CompletedAt is when the request completed processing
	CompletedAt time.Time `json:"completed_at"`

	// InProgress indicates if the request is still being processed
	InProgress bool `json:"in_progress"`
}

// CacheStats provides metrics about cache performance.
type CacheStats struct {
	// Hits is the number of successful cache retrievals
	Hits uint64 `json:"hits"`

	// Misses is the number of cache key not found
	Misses uint64 `json:"misses"`

	// Evictions is the number of entries removed due to memory limits
	Evictions uint64 `json:"evictions"`

	// Size is the current number of entries in the cache
	Size int `json:"size"`

	// MemoryUsage is approximate memory used in bytes (if available)
	MemoryUsage uint64 `json:"memory_usage,omitempty"`

	// HitRate is the cache hit ratio (0.0 to 1.0)
	HitRate float64 `json:"hit_rate"`
}

// CacheError represents cache-specific errors.
type CacheError struct {
	Op  string // Operation that failed (e.g., "Get", "Set")
	Err error  // Underlying error
}

func (e *CacheError) Error() string {
	return "cache " + e.Op + ": " + e.Err.Error()
}

func (e *CacheError) Unwrap() error {
	return e.Err
}

// Common cache errors
var (
	// ErrCacheMiss indicates the requested key was not found in the cache
	ErrCacheMiss = &CacheError{Op: "Get", Err: errMiss}

	// ErrCacheFull indicates the cache has reached its memory limit
	ErrCacheFull = &CacheError{Op: "Set", Err: errFull}
)

// Internal error types (not exported)
type cacheErr string

func (e cacheErr) Error() string { return string(e) }

const (
	errMiss cacheErr = "cache miss"
	errFull cacheErr = "cache full"
)

// IsCacheMiss checks if an error is a cache miss error.
func IsCacheMiss(err error) bool {
	if err == nil {
		return false
	}
	var cErr *CacheError
	if ce, ok := err.(*CacheError); ok {
		cErr = ce
	}
	return cErr != nil && cErr.Err == errMiss
}

// Config holds configuration for cache implementations.
type Config struct {
	// MaxMemoryMB is the maximum memory in megabytes (for in-memory cache)
	MaxMemoryMB int

	// DefaultTTL is the default expiration time for cached items
	DefaultTTL time.Duration

	// CleanupInterval is how often to run expired entry cleanup
	CleanupInterval time.Duration

	// RedisAddr is the Redis server address (for Redis cache)
	RedisAddr string

	// RedisPassword is the Redis password (for Redis cache)
	RedisPassword string

	// RedisDB is the Redis database number (for Redis cache)
	RedisDB int

	// EnableStats enables cache statistics tracking
	EnableStats bool
}

// DefaultConfig returns sensible defaults for cache configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxMemoryMB:     100,                // 100 MB max memory
		DefaultTTL:      1 * time.Hour,      // 1 hour default TTL
		CleanupInterval: 5 * time.Minute,    // Cleanup every 5 minutes
		EnableStats:     true,               // Stats enabled by default
	}
}
