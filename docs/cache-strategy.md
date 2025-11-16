# Cache Strategy for Lumen Fitness Application

## Overview

The cache system provides high-performance in-memory and distributed caching for nutrition data and idempotency key management. This document describes the caching architecture, strategies, and best practices.

## Architecture

### Cache Implementations

1. **MemoryCache** (Current)
   - In-memory cache using Go's sync.Map
   - Thread-safe with RWMutex protection
   - TTL support with background cleanup
   - LRU eviction when memory limits reached
   - **Use Case**: Development, single-instance deployments, testing

2. **RedisCache** (Future)
   - Distributed cache using Redis
   - Supports multiple application instances
   - Persistence and replication
   - **Use Case**: Production multi-instance deployments

### Cache Interface

Both implementations satisfy the same `Cache` interface, allowing drop-in replacement:

```go
type Cache interface {
    GetNutrition(ctx context.Context, key string) (*NutritionResponse, error)
    SetNutrition(ctx context.Context, key string, value *NutritionResponse, ttl time.Duration) error
    GetIdempotency(ctx context.Context, key string) (*IdempotencyRecord, error)
    SetIdempotency(ctx context.Context, key string, value *IdempotencyRecord, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Clear(ctx context.Context) error
    Stats() CacheStats
    Close() error
}
```

## Use Cases

### 1. Nutrition Data Caching

**Purpose**: Cache AI-generated nutrition analysis to reduce API calls and improve response times.

**Key Structure**:
```
nutrition:<food_name>:<serving_size>
Example: nutrition:chicken-breast:100g
```

**TTL Strategy**:
- Default: 24 hours
- High confidence (>0.9): 7 days
- Low confidence (<0.7): 1 hour

**Benefits**:
- Reduces OpenAI API costs
- Improves response time (sub-millisecond vs. seconds)
- Consistent results for same food items
- Offline capability during API outages

**Implementation**:
```go
// Check cache first
cached, err := cache.GetNutrition(ctx, cacheKey)
if err == nil {
    return cached // Cache hit
}

// Cache miss - call AI API
nutrition, err := aiService.AnalyzeNutrition(ctx, foodName)
if err != nil {
    return err
}

// Store in cache
ttl := determineTTL(nutrition.Confidence)
_ = cache.SetNutrition(ctx, cacheKey, nutrition, ttl)

return nutrition
```

### 2. Idempotency Key Management

**Purpose**: Prevent duplicate request processing and return consistent responses for retried requests.

**Key Structure**:
```
idempotency:<request_id>
Example: idempotency:req-550e8400-e29b-41d4-a716-446655440000
```

**TTL Strategy**:
- Standard: 24 hours
- Critical operations (payments): 72 hours

**Benefits**:
- Prevents duplicate charges
- Handles client retries safely
- Network failure resilience
- Consistent user experience

**Implementation**:
```go
// Check if request already processed
record, err := cache.GetIdempotency(ctx, requestID)
if err == nil {
    if record.InProgress {
        return ErrRequestInProgress
    }
    // Return cached response
    return record.ResponseBody, record.StatusCode
}

// Mark as in progress
inProgress := &IdempotencyRecord{
    RequestID:  requestID,
    InProgress: true,
    CreatedAt:  time.Now(),
}
_ = cache.SetIdempotency(ctx, requestID, inProgress, 24*time.Hour)

// Process request
response, status, err := processRequest(ctx, req)

// Store final result
final := &IdempotencyRecord{
    RequestID:    requestID,
    StatusCode:   status,
    ResponseBody: response,
    CreatedAt:    inProgress.CreatedAt,
    CompletedAt:  time.Now(),
    InProgress:   false,
}
_ = cache.SetIdempotency(ctx, requestID, final, 24*time.Hour)
```

## Performance Characteristics

### MemoryCache

| Operation | Complexity | Typical Latency |
|-----------|-----------|-----------------|
| Get       | O(1)      | <1μs            |
| Set       | O(1)      | <1μs            |
| Delete    | O(1)      | <1μs            |
| Clear     | O(n)      | 10-100μs        |
| Eviction  | O(n log n)| 100-500μs       |

### Memory Usage

- **Entry Overhead**: ~64 bytes per entry
- **Data Size**: Actual serialized JSON size
- **Total**: `overhead + len(key) + len(value)`

**Example**:
```go
Nutrition entry:
  Key: "nutrition:chicken-breast:100g" (28 bytes)
  Value: {"food_name":"Chicken Breast",...} (~200 bytes)
  Overhead: 64 bytes
  Total: ~292 bytes per entry
```

### Capacity Planning

| Max Memory | Avg Entry Size | Estimated Capacity |
|------------|----------------|-------------------|
| 100 MB     | 300 bytes      | ~350,000 entries  |
| 500 MB     | 300 bytes      | ~1,750,000 entries|
| 1 GB       | 300 bytes      | ~3,500,000 entries|

## Configuration

### Development
```go
config := &cache.Config{
    MaxMemoryMB:     100,
    DefaultTTL:      1 * time.Hour,
    CleanupInterval: 5 * time.Minute,
    EnableStats:     true,
}
```

### Production (Single Instance)
```go
config := &cache.Config{
    MaxMemoryMB:     500,
    DefaultTTL:      24 * time.Hour,
    CleanupInterval: 10 * time.Minute,
    EnableStats:     true,
}
```

### Production (Multi-Instance - Future Redis)
```go
config := &cache.Config{
    RedisAddr:     "redis:6379",
    RedisPassword: os.Getenv("REDIS_PASSWORD"),
    RedisDB:       0,
    DefaultTTL:    24 * time.Hour,
    EnableStats:   true,
}
```

## Monitoring

### Cache Statistics

```go
stats := cache.Stats()
// CacheStats{
//     Hits:        15234,
//     Misses:      3421,
//     Evictions:   45,
//     Size:        12789,
//     MemoryUsage: 3842560,  // bytes
//     HitRate:     0.817,    // 81.7%
// }
```

### Key Metrics to Monitor

1. **Hit Rate** (Target: >80%)
   - Low hit rate indicates ineffective caching
   - May need to adjust TTL or cache more data

2. **Eviction Rate** (Target: <5%)
   - High evictions mean memory limit too low
   - Consider increasing MaxMemoryMB

3. **Memory Usage** (Target: <90% of max)
   - Monitor for memory leaks
   - Adjust limits based on actual usage

4. **Size Growth**
   - Track number of entries over time
   - Identify potential memory issues early

## Best Practices

### 1. Cache Key Design

**Good**:
```go
// Specific, consistent, normalized
key := fmt.Sprintf("nutrition:%s:%s",
    strings.ToLower(foodName),
    normalizeServingSize(serving))
```

**Bad**:
```go
// Inconsistent, case-sensitive, non-normalized
key := foodName + "-" + serving
```

### 2. TTL Selection

- **Frequently accessed**: Longer TTL (7+ days)
- **Infrequently accessed**: Shorter TTL (1-6 hours)
- **High accuracy needed**: Shorter TTL (1 hour)
- **Static data**: Very long TTL (30+ days)

### 3. Error Handling

```go
// Always handle cache failures gracefully
cached, err := cache.GetNutrition(ctx, key)
if err != nil {
    if cache.IsCacheMiss(err) {
        // Normal miss - fetch from source
        return fetchFromSource(ctx, key)
    }
    // Log error but continue
    log.Warn("cache error", "error", err)
    return fetchFromSource(ctx, key)
}
return cached
```

### 4. Cache Warming

```go
// Pre-populate cache with common items
func WarmCache(ctx context.Context, cache Cache) error {
    commonFoods := []string{
        "chicken-breast", "rice", "broccoli",
        "salmon", "eggs", "oatmeal",
    }

    for _, food := range commonFoods {
        nutrition, err := fetchNutrition(ctx, food)
        if err != nil {
            continue
        }
        _ = cache.SetNutrition(ctx, food, nutrition, 7*24*time.Hour)
    }
    return nil
}
```

### 5. Cache Invalidation

```go
// Invalidate on data updates
func UpdateNutritionData(ctx context.Context, foodName string, data *NutritionData) error {
    // Update database
    if err := db.Update(ctx, data); err != nil {
        return err
    }

    // Invalidate cache
    cacheKey := buildCacheKey(foodName)
    _ = cache.Delete(ctx, cacheKey)

    return nil
}
```

## Testing

### Unit Tests

```bash
# Run cache tests
go test ./internal/services/cache -v

# Run with race detector
go test ./internal/services/cache -race

# Run with coverage
go test ./internal/services/cache -cover -coverprofile=coverage.out
```

### Benchmark Tests

```bash
# Run benchmarks
go test ./internal/services/cache -bench=. -benchmem

# Compare implementations
go test ./internal/services/cache -bench=BenchmarkMemoryCache -benchtime=10s
```

### Load Tests

```go
func TestCacheUnderLoad(t *testing.T) {
    cache := NewMemoryCache(nil)
    defer cache.Close()

    // Simulate 1000 concurrent users
    var wg sync.WaitGroup
    for i := 0; i < 1000; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            // Simulate user operations
            for j := 0; j < 100; j++ {
                performCacheOperation(cache)
            }
        }(i)
    }
    wg.Wait()

    stats := cache.Stats()
    if stats.HitRate < 0.7 {
        t.Errorf("Hit rate too low under load: %.2f", stats.HitRate)
    }
}
```

## Migration Path to Redis

When moving from MemoryCache to Redis:

1. **No Code Changes Required**
   - Both implement same `Cache` interface
   - Change constructor only

2. **Update Configuration**
   ```go
   // From
   cache := cache.NewMemoryCache(config)

   // To
   cache := cache.NewRedisCache(config)
   ```

3. **Test Thoroughly**
   - Run integration tests with Redis
   - Verify serialization/deserialization
   - Check performance under load

4. **Monitor Migration**
   - Compare hit rates
   - Watch for increased latency
   - Monitor Redis memory usage

## Troubleshooting

### High Memory Usage

**Symptoms**: Memory usage approaching MaxMemoryMB, frequent evictions

**Solutions**:
1. Increase `MaxMemoryMB`
2. Reduce `DefaultTTL`
3. Implement more aggressive eviction
4. Move to Redis for larger capacity

### Low Hit Rate

**Symptoms**: Hit rate <60%, many cache misses

**Solutions**:
1. Increase TTL for stable data
2. Implement cache warming
3. Check cache key consistency
4. Verify data is cacheable

### Slow Performance

**Symptoms**: Cache operations taking >1ms

**Solutions**:
1. Check for lock contention
2. Reduce cleanup frequency
3. Optimize entry size
4. Consider Redis for better performance

## Future Enhancements

1. **Redis Implementation**
   - Distributed caching
   - Persistence
   - Replication

2. **Advanced Eviction**
   - Adaptive LRU
   - Frequency-based eviction
   - Tiered storage

3. **Compression**
   - Compress large entries
   - Reduce memory usage
   - Trade CPU for memory

4. **Partitioning**
   - Multiple cache instances
   - Partition by key prefix
   - Reduce lock contention

## References

- [Cache Patterns](https://docs.microsoft.com/en-us/azure/architecture/patterns/cache-aside)
- [Redis Best Practices](https://redis.io/docs/manual/patterns/)
- [Go Sync Package](https://pkg.go.dev/sync)
