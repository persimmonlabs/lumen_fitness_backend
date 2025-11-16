# Cache Usage Examples

This document provides practical examples of using the cache system in the Lumen fitness application.

## Quick Start

### Initialize Cache

```go
package main

import (
    "github.com/pradord/lumen_final/backend/internal/services/cache"
)

func main() {
    // Use default configuration
    c := cache.NewMemoryCache(nil)
    defer c.Close()

    // Or custom configuration
    config := &cache.Config{
        MaxMemoryMB:     200,
        DefaultTTL:      2 * time.Hour,
        CleanupInterval: 10 * time.Minute,
        EnableStats:     true,
    }
    c = cache.NewMemoryCache(config)
    defer c.Close()
}
```

## Example 1: Caching Nutrition Data from AI

```go
package nutrition

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/pradord/lumen_final/backend/internal/services/cache"
)

type NutritionService struct {
    cache    cache.Cache
    aiClient AIClient
}

func NewNutritionService(c cache.Cache, ai AIClient) *NutritionService {
    return &NutritionService{
        cache:    c,
        aiClient: ai,
    }
}

// GetNutrition retrieves nutrition data, using cache when possible
func (s *NutritionService) GetNutrition(ctx context.Context, foodName, servingSize string) (*cache.NutritionResponse, error) {
    // Build cache key (normalized)
    cacheKey := s.buildCacheKey(foodName, servingSize)

    // Try cache first
    cached, err := s.cache.GetNutrition(ctx, cacheKey)
    if err == nil {
        // Cache hit - return immediately
        return cached, nil
    }

    // Cache miss - call AI API
    nutrition, err := s.aiClient.AnalyzeFood(ctx, foodName, servingSize)
    if err != nil {
        return nil, fmt.Errorf("AI analysis failed: %w", err)
    }

    // Convert AI response to cache format
    cacheData := &cache.NutritionResponse{
        FoodName:      nutrition.Name,
        Calories:      nutrition.Calories,
        Protein:       nutrition.Protein,
        Carbohydrates: nutrition.Carbs,
        Fat:           nutrition.Fat,
        Fiber:         nutrition.Fiber,
        ServingSize:   servingSize,
        Confidence:    nutrition.Confidence,
        Source:        "OpenAI",
        CachedAt:      time.Now(),
    }

    // Determine TTL based on confidence
    ttl := s.determineTTL(nutrition.Confidence)

    // Store in cache for future requests
    if err := s.cache.SetNutrition(ctx, cacheKey, cacheData, ttl); err != nil {
        // Log error but don't fail the request
        log.Warn("failed to cache nutrition data", "error", err)
    }

    return cacheData, nil
}

func (s *NutritionService) buildCacheKey(foodName, servingSize string) string {
    // Normalize to ensure consistent cache hits
    food := strings.ToLower(strings.TrimSpace(foodName))
    serving := strings.ToLower(strings.TrimSpace(servingSize))
    return fmt.Sprintf("nutrition:%s:%s", food, serving)
}

func (s *NutritionService) determineTTL(confidence float64) time.Duration {
    switch {
    case confidence >= 0.9:
        return 7 * 24 * time.Hour // 7 days for high confidence
    case confidence >= 0.7:
        return 24 * time.Hour // 1 day for medium confidence
    default:
        return 1 * time.Hour // 1 hour for low confidence
    }
}
```

## Example 2: Idempotency for API Requests

```go
package handlers

import (
    "context"
    "encoding/json"
    "net/http"
    "time"

    "github.com/pradord/lumen_final/backend/internal/services/cache"
)

type PaymentHandler struct {
    cache          cache.Cache
    paymentService PaymentService
}

// ProcessPayment handles payment with idempotency protection
func (h *PaymentHandler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Extract idempotency key from header
    idempotencyKey := r.Header.Get("Idempotency-Key")
    if idempotencyKey == "" {
        http.Error(w, "Idempotency-Key header required", http.StatusBadRequest)
        return
    }

    // Check if request already processed
    record, err := h.cache.GetIdempotency(ctx, idempotencyKey)
    if err == nil {
        // Request already processed
        if record.InProgress {
            // Still processing - tell client to retry
            w.Header().Set("Retry-After", "5")
            http.Error(w, "Request in progress", http.StatusConflict)
            return
        }

        // Return cached response
        w.WriteHeader(record.StatusCode)
        w.Write(record.ResponseBody)
        return
    }

    // Mark request as in progress
    inProgressRecord := &cache.IdempotencyRecord{
        RequestID:  idempotencyKey,
        InProgress: true,
        CreatedAt:  time.Now(),
    }
    if err := h.cache.SetIdempotency(ctx, idempotencyKey, inProgressRecord, 72*time.Hour); err != nil {
        http.Error(w, "Failed to process request", http.StatusInternalServerError)
        return
    }

    // Parse request
    var req PaymentRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Process payment
    result, err := h.paymentService.ProcessPayment(ctx, &req)
    if err != nil {
        // Store failure result
        h.storeResult(ctx, idempotencyKey, http.StatusInternalServerError,
            []byte(`{"error":"payment failed"}`), inProgressRecord.CreatedAt)
        http.Error(w, "Payment failed", http.StatusInternalServerError)
        return
    }

    // Store success result
    responseBody, _ := json.Marshal(result)
    h.storeResult(ctx, idempotencyKey, http.StatusOK, responseBody, inProgressRecord.CreatedAt)

    // Return response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(responseBody)
}

func (h *PaymentHandler) storeResult(ctx context.Context, key string, status int, body []byte, createdAt time.Time) {
    record := &cache.IdempotencyRecord{
        RequestID:    key,
        StatusCode:   status,
        ResponseBody: body,
        CreatedAt:    createdAt,
        CompletedAt:  time.Now(),
        InProgress:   false,
    }
    _ = h.cache.SetIdempotency(ctx, key, record, 72*time.Hour)
}
```

## Example 3: Cache Warming on Startup

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/pradord/lumen_final/backend/internal/services/cache"
)

// WarmCache pre-populates cache with common foods
func WarmCache(ctx context.Context, c cache.Cache) error {
    commonFoods := []struct {
        name    string
        serving string
        data    *cache.NutritionResponse
    }{
        {
            name:    "chicken-breast",
            serving: "100g",
            data: &cache.NutritionResponse{
                FoodName:      "Chicken Breast",
                Calories:      165,
                Protein:       31,
                Carbohydrates: 0,
                Fat:           3.6,
                Fiber:         0,
                ServingSize:   "100g",
                Confidence:    0.95,
                Source:        "USDA",
                CachedAt:      time.Now(),
            },
        },
        {
            name:    "white-rice",
            serving: "1 cup cooked",
            data: &cache.NutritionResponse{
                FoodName:      "White Rice",
                Calories:      205,
                Protein:       4.2,
                Carbohydrates: 45,
                Fat:           0.4,
                Fiber:         0.6,
                ServingSize:   "1 cup cooked",
                Confidence:    0.95,
                Source:        "USDA",
                CachedAt:      time.Now(),
            },
        },
        // Add more common foods...
    }

    for _, food := range commonFoods {
        key := buildCacheKey(food.name, food.serving)
        if err := c.SetNutrition(ctx, key, food.data, 30*24*time.Hour); err != nil {
            log.Printf("Failed to warm cache for %s: %v", food.name, err)
            continue
        }
    }

    log.Printf("Cache warmed with %d common foods", len(commonFoods))
    return nil
}

func buildCacheKey(name, serving string) string {
    return fmt.Sprintf("nutrition:%s:%s", name, serving)
}
```

## Example 4: Monitoring Cache Performance

```go
package monitoring

import (
    "context"
    "log"
    "time"

    "github.com/pradord/lumen_final/backend/internal/services/cache"
)

type CacheMonitor struct {
    cache cache.Cache
}

// MonitorCache periodically logs cache statistics
func (m *CacheMonitor) MonitorCache(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            m.logStats()
        }
    }
}

func (m *CacheMonitor) logStats() {
    stats := m.cache.Stats()

    log.Printf("Cache Stats: hits=%d misses=%d evictions=%d size=%d memory=%dMB hit_rate=%.2f%%",
        stats.Hits,
        stats.Misses,
        stats.Evictions,
        stats.Size,
        stats.MemoryUsage/1024/1024,
        stats.HitRate*100,
    )

    // Alert if performance degraded
    if stats.HitRate < 0.6 {
        log.Printf("WARNING: Cache hit rate below 60%%: %.2f%%", stats.HitRate*100)
    }

    if stats.Evictions > 1000 {
        log.Printf("WARNING: High eviction count: %d (consider increasing MaxMemoryMB)", stats.Evictions)
    }
}
```

## Example 5: Integration with HTTP Handler

```go
package handlers

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/pradord/lumen_final/backend/internal/services/cache"
)

type NutritionHandler struct {
    cache cache.Cache
    ai    AIService
}

func (h *NutritionHandler) AnalyzeFood(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Parse request
    var req struct {
        FoodName    string `json:"food_name"`
        ServingSize string `json:"serving_size"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Build cache key
    cacheKey := buildNutritionKey(req.FoodName, req.ServingSize)

    // Try cache first
    cached, err := h.cache.GetNutrition(ctx, cacheKey)
    if err == nil {
        // Cache hit
        w.Header().Set("X-Cache", "HIT")
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(cached)
        return
    }

    // Cache miss - call AI
    w.Header().Set("X-Cache", "MISS")

    nutrition, err := h.ai.Analyze(ctx, req.FoodName, req.ServingSize)
    if err != nil {
        http.Error(w, "Analysis failed", http.StatusInternalServerError)
        return
    }

    // Store in cache
    _ = h.cache.SetNutrition(ctx, cacheKey, nutrition, 24*time.Hour)

    // Return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(nutrition)
}

func buildNutritionKey(food, serving string) string {
    return "nutrition:" + strings.ToLower(food) + ":" + strings.ToLower(serving)
}
```

## Example 6: Testing with Cache

```go
package services_test

import (
    "context"
    "testing"
    "time"

    "github.com/pradord/lumen_final/backend/internal/services/cache"
)

func TestNutritionService_WithCache(t *testing.T) {
    // Create test cache
    testCache := cache.NewMemoryCache(nil)
    defer testCache.Close()

    // Create service with test cache
    service := NewNutritionService(testCache, mockAIClient)

    ctx := context.Background()

    // First call - should miss cache and call AI
    result1, err := service.GetNutrition(ctx, "apple", "1 medium")
    if err != nil {
        t.Fatalf("GetNutrition failed: %v", err)
    }

    // Verify AI was called
    if mockAIClient.CallCount != 1 {
        t.Errorf("Expected 1 AI call, got %d", mockAIClient.CallCount)
    }

    // Second call - should hit cache
    result2, err := service.GetNutrition(ctx, "apple", "1 medium")
    if err != nil {
        t.Fatalf("GetNutrition failed: %v", err)
    }

    // Verify AI was not called again
    if mockAIClient.CallCount != 1 {
        t.Errorf("Expected still 1 AI call, got %d", mockAIClient.CallCount)
    }

    // Verify results match
    if result1.Calories != result2.Calories {
        t.Errorf("Cached result differs from original")
    }

    // Check cache stats
    stats := testCache.Stats()
    if stats.Hits != 1 {
        t.Errorf("Expected 1 cache hit, got %d", stats.Hits)
    }
    if stats.Misses != 1 {
        t.Errorf("Expected 1 cache miss, got %d", stats.Misses)
    }
}
```

## Best Practices Summary

1. **Always handle cache errors gracefully** - Don't let cache failures break functionality
2. **Normalize cache keys** - Use lowercase, consistent formatting
3. **Choose appropriate TTLs** - Balance freshness vs. performance
4. **Monitor cache stats** - Track hit rate, evictions, memory usage
5. **Use idempotency for critical operations** - Payments, account changes, etc.
6. **Warm cache on startup** - Pre-populate with common data
7. **Test with cache** - Verify caching behavior in tests
8. **Log cache operations** - Debug issues with proper logging

## Common Pitfalls to Avoid

1. **Don't cache user-specific data with generic keys**
   ```go
   // BAD - all users share same cache
   key := "user-profile"

   // GOOD - user-specific cache
   key := fmt.Sprintf("user-profile:%s", userID)
   ```

2. **Don't ignore cache errors in critical paths**
   ```go
   // BAD
   cached, _ := cache.Get(ctx, key)
   return cached // Will panic if nil

   // GOOD
   cached, err := cache.Get(ctx, key)
   if err != nil {
       return fetchFromDB(ctx, key)
   }
   return cached
   ```

3. **Don't use same TTL for all data**
   ```go
   // BAD - same TTL for everything
   cache.Set(ctx, key, data, 1*time.Hour)

   // GOOD - TTL based on data characteristics
   ttl := determineTTL(data.Type, data.UpdateFrequency)
   cache.Set(ctx, key, data, ttl)
   ```

4. **Don't forget to close cache**
   ```go
   // GOOD - always defer Close()
   cache := cache.NewMemoryCache(config)
   defer cache.Close()
   ```
