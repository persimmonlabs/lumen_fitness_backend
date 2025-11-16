package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// MemoryCache implements an in-memory cache with TTL support and LRU eviction.
// It is thread-safe and suitable for single-instance deployments or development.
// For production distributed systems, use RedisCache instead.
type MemoryCache struct {
	// mu protects all cache operations
	mu sync.RWMutex

	// entries stores the cached data
	entries map[string]*cacheEntry

	// config holds cache configuration
	config *Config

	// stats tracks cache performance metrics
	stats *cacheStats

	// stopCleanup signals the cleanup goroutine to stop
	stopCleanup chan struct{}

	// cleanupDone signals when cleanup goroutine has finished
	cleanupDone chan struct{}
}

// cacheEntry represents a single cached item with metadata.
type cacheEntry struct {
	// key is the cache key
	key string

	// data is the serialized value
	data []byte

	// expiresAt is when this entry should be evicted (zero means never)
	expiresAt time.Time

	// lastAccessed is used for LRU eviction
	lastAccessed time.Time

	// size is approximate memory usage in bytes
	size int
}

// cacheStats tracks cache performance metrics.
type cacheStats struct {
	mu        sync.RWMutex
	hits      uint64
	misses    uint64
	evictions uint64
}

// NewMemoryCache creates a new in-memory cache with the given configuration.
// If config is nil, DefaultConfig() is used.
func NewMemoryCache(config *Config) *MemoryCache {
	if config == nil {
		config = DefaultConfig()
	}

	mc := &MemoryCache{
		entries:     make(map[string]*cacheEntry),
		config:      config,
		stats:       &cacheStats{},
		stopCleanup: make(chan struct{}),
		cleanupDone: make(chan struct{}),
	}

	// Start background cleanup goroutine
	go mc.cleanupLoop()

	return mc
}

// GetNutrition retrieves cached nutrition data.
func (mc *MemoryCache) GetNutrition(ctx context.Context, key string) (*NutritionResponse, error) {
	data, err := mc.get(ctx, key)
	if err != nil {
		return nil, err
	}

	var nutrition NutritionResponse
	if err := json.Unmarshal(data, &nutrition); err != nil {
		return nil, &CacheError{Op: "GetNutrition", Err: err}
	}

	return &nutrition, nil
}

// SetNutrition stores nutrition data with TTL.
func (mc *MemoryCache) SetNutrition(ctx context.Context, key string, value *NutritionResponse, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return &CacheError{Op: "SetNutrition", Err: err}
	}

	return mc.set(ctx, key, data, ttl)
}

// GetIdempotency retrieves idempotency record.
func (mc *MemoryCache) GetIdempotency(ctx context.Context, key string) (*IdempotencyRecord, error) {
	data, err := mc.get(ctx, key)
	if err != nil {
		return nil, err
	}

	var record IdempotencyRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, &CacheError{Op: "GetIdempotency", Err: err}
	}

	return &record, nil
}

// SetIdempotency stores idempotency record with TTL.
func (mc *MemoryCache) SetIdempotency(ctx context.Context, key string, value *IdempotencyRecord, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return &CacheError{Op: "SetIdempotency", Err: err}
	}

	return mc.set(ctx, key, data, ttl)
}

// Delete removes a key from the cache.
func (mc *MemoryCache) Delete(ctx context.Context, key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.entries, key)
	return nil
}

// Clear removes all entries from the cache.
func (mc *MemoryCache) Clear(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.entries = make(map[string]*cacheEntry)
	return nil
}

// Stats returns current cache statistics.
func (mc *MemoryCache) Stats() CacheStats {
	mc.mu.RLock()
	size := len(mc.entries)
	memoryUsage := mc.calculateMemoryUsage()
	mc.mu.RUnlock()

	mc.stats.mu.RLock()
	defer mc.stats.mu.RUnlock()

	total := mc.stats.hits + mc.stats.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(mc.stats.hits) / float64(total)
	}

	return CacheStats{
		Hits:        mc.stats.hits,
		Misses:      mc.stats.misses,
		Evictions:   mc.stats.evictions,
		Size:        size,
		MemoryUsage: memoryUsage,
		HitRate:     hitRate,
	}
}

// Close stops the cleanup goroutine and releases resources.
func (mc *MemoryCache) Close() error {
	close(mc.stopCleanup)
	<-mc.cleanupDone
	return nil
}

// get retrieves raw data from cache.
func (mc *MemoryCache) get(ctx context.Context, key string) ([]byte, error) {
	mc.mu.RLock()
	entry, exists := mc.entries[key]
	mc.mu.RUnlock()

	if !exists {
		mc.recordMiss()
		return nil, ErrCacheMiss
	}

	// Check if expired
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		// Remove expired entry
		mc.mu.Lock()
		delete(mc.entries, key)
		mc.mu.Unlock()

		mc.recordMiss()
		return nil, ErrCacheMiss
	}

	// Update last accessed time for LRU
	mc.mu.Lock()
	entry.lastAccessed = time.Now()
	mc.mu.Unlock()

	mc.recordHit()
	return entry.data, nil
}

// set stores raw data in cache with TTL.
func (mc *MemoryCache) set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	entry := &cacheEntry{
		key:          key,
		data:         data,
		lastAccessed: time.Now(),
		size:         len(key) + len(data) + 64, // Approximate overhead
	}

	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Check memory limit before adding
	if mc.config.MaxMemoryMB > 0 {
		currentMemory := mc.calculateMemoryUsage()
		maxMemory := uint64(mc.config.MaxMemoryMB) * 1024 * 1024

		if currentMemory+uint64(entry.size) > maxMemory {
			// Try to evict LRU entries to make space
			if err := mc.evictLRU(uint64(entry.size)); err != nil {
				return err
			}
		}
	}

	mc.entries[key] = entry
	return nil
}

// evictLRU removes least recently used entries until we have enough space.
func (mc *MemoryCache) evictLRU(needed uint64) error {
	// Find LRU entries
	type entryWithKey struct {
		key       string
		accessed  time.Time
		size      int
	}

	var entries []entryWithKey
	for k, e := range mc.entries {
		entries = append(entries, entryWithKey{
			key:      k,
			accessed: e.lastAccessed,
			size:     e.size,
		})
	}

	// Sort by last accessed (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].accessed.After(entries[j].accessed) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Evict oldest entries until we have enough space
	var freed uint64
	evicted := 0
	for _, e := range entries {
		if freed >= needed {
			break
		}
		delete(mc.entries, e.key)
		freed += uint64(e.size)
		evicted++
	}

	mc.stats.mu.Lock()
	mc.stats.evictions += uint64(evicted)
	mc.stats.mu.Unlock()

	if freed < needed {
		return ErrCacheFull
	}

	return nil
}

// calculateMemoryUsage returns approximate memory usage in bytes.
func (mc *MemoryCache) calculateMemoryUsage() uint64 {
	var total uint64
	for _, entry := range mc.entries {
		total += uint64(entry.size)
	}
	return total
}

// cleanupLoop periodically removes expired entries.
func (mc *MemoryCache) cleanupLoop() {
	defer close(mc.cleanupDone)

	ticker := time.NewTicker(mc.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mc.stopCleanup:
			return
		case <-ticker.C:
			mc.removeExpired()
		}
	}
}

// removeExpired removes all expired entries.
func (mc *MemoryCache) removeExpired() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()
	for key, entry := range mc.entries {
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
			delete(mc.entries, key)
		}
	}
}

// recordHit increments the hit counter.
func (mc *MemoryCache) recordHit() {
	if mc.config.EnableStats {
		mc.stats.mu.Lock()
		mc.stats.hits++
		mc.stats.mu.Unlock()
	}
}

// recordMiss increments the miss counter.
func (mc *MemoryCache) recordMiss() {
	if mc.config.EnableStats {
		mc.stats.mu.Lock()
		mc.stats.misses++
		mc.stats.mu.Unlock()
	}
}
