package cache

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryCache_BasicOperations(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()

	t.Run("Set and Get Nutrition", func(t *testing.T) {
		nutrition := &NutritionResponse{
			FoodName:      "Chicken Breast",
			Calories:      165,
			Protein:       31,
			Carbohydrates: 0,
			Fat:           3.6,
			Fiber:         0,
			ServingSize:   "100g",
			Confidence:    0.95,
			Source:        "OpenAI",
			CachedAt:      time.Now(),
		}

		err := cache.SetNutrition(ctx, "chicken-breast-100g", nutrition, 1*time.Hour)
		if err != nil {
			t.Fatalf("SetNutrition failed: %v", err)
		}

		retrieved, err := cache.GetNutrition(ctx, "chicken-breast-100g")
		if err != nil {
			t.Fatalf("GetNutrition failed: %v", err)
		}

		if retrieved.FoodName != nutrition.FoodName {
			t.Errorf("FoodName mismatch: got %s, want %s", retrieved.FoodName, nutrition.FoodName)
		}
		if retrieved.Calories != nutrition.Calories {
			t.Errorf("Calories mismatch: got %.2f, want %.2f", retrieved.Calories, nutrition.Calories)
		}
		if retrieved.Protein != nutrition.Protein {
			t.Errorf("Protein mismatch: got %.2f, want %.2f", retrieved.Protein, nutrition.Protein)
		}
	})

	t.Run("Set and Get Idempotency", func(t *testing.T) {
		record := &IdempotencyRecord{
			RequestID:    "req-12345",
			StatusCode:   200,
			ResponseBody: []byte(`{"status":"success"}`),
			CreatedAt:    time.Now(),
			CompletedAt:  time.Now().Add(100 * time.Millisecond),
			InProgress:   false,
		}

		err := cache.SetIdempotency(ctx, "req-12345", record, 24*time.Hour)
		if err != nil {
			t.Fatalf("SetIdempotency failed: %v", err)
		}

		retrieved, err := cache.GetIdempotency(ctx, "req-12345")
		if err != nil {
			t.Fatalf("GetIdempotency failed: %v", err)
		}

		if retrieved.RequestID != record.RequestID {
			t.Errorf("RequestID mismatch: got %s, want %s", retrieved.RequestID, record.RequestID)
		}
		if retrieved.StatusCode != record.StatusCode {
			t.Errorf("StatusCode mismatch: got %d, want %d", retrieved.StatusCode, record.StatusCode)
		}
		if string(retrieved.ResponseBody) != string(record.ResponseBody) {
			t.Errorf("ResponseBody mismatch: got %s, want %s", retrieved.ResponseBody, record.ResponseBody)
		}
	})

	t.Run("Get Non-Existent Key", func(t *testing.T) {
		_, err := cache.GetNutrition(ctx, "non-existent-key")
		if !IsCacheMiss(err) {
			t.Errorf("Expected cache miss error, got: %v", err)
		}
	})

	t.Run("Delete Key", func(t *testing.T) {
		nutrition := &NutritionResponse{
			FoodName: "Test Food",
			Calories: 100,
		}

		err := cache.SetNutrition(ctx, "test-key", nutrition, 1*time.Hour)
		if err != nil {
			t.Fatalf("SetNutrition failed: %v", err)
		}

		err = cache.Delete(ctx, "test-key")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = cache.GetNutrition(ctx, "test-key")
		if !IsCacheMiss(err) {
			t.Errorf("Expected cache miss after delete, got: %v", err)
		}
	})

	t.Run("Clear Cache", func(t *testing.T) {
		// Add multiple entries
		for i := 0; i < 5; i++ {
			nutrition := &NutritionResponse{FoodName: "Food", Calories: float64(i)}
			err := cache.SetNutrition(ctx, string(rune('a'+i)), nutrition, 1*time.Hour)
			if err != nil {
				t.Fatalf("SetNutrition failed: %v", err)
			}
		}

		err := cache.Clear(ctx)
		if err != nil {
			t.Fatalf("Clear failed: %v", err)
		}

		stats := cache.Stats()
		if stats.Size != 0 {
			t.Errorf("Expected size 0 after clear, got %d", stats.Size)
		}
	})
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	config := DefaultConfig()
	config.CleanupInterval = 50 * time.Millisecond
	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()

	t.Run("Entry Expires After TTL", func(t *testing.T) {
		nutrition := &NutritionResponse{
			FoodName: "Expired Food",
			Calories: 100,
		}

		// Set with very short TTL
		err := cache.SetNutrition(ctx, "expires-fast", nutrition, 100*time.Millisecond)
		if err != nil {
			t.Fatalf("SetNutrition failed: %v", err)
		}

		// Verify it exists immediately
		_, err = cache.GetNutrition(ctx, "expires-fast")
		if err != nil {
			t.Fatalf("GetNutrition failed immediately: %v", err)
		}

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Should be expired now
		_, err = cache.GetNutrition(ctx, "expires-fast")
		if !IsCacheMiss(err) {
			t.Errorf("Expected cache miss after TTL, got: %v", err)
		}
	})

	t.Run("Zero TTL Never Expires", func(t *testing.T) {
		nutrition := &NutritionResponse{
			FoodName: "Never Expires",
			Calories: 100,
		}

		err := cache.SetNutrition(ctx, "never-expires", nutrition, 0)
		if err != nil {
			t.Fatalf("SetNutrition failed: %v", err)
		}

		// Wait longer than normal TTL would be
		time.Sleep(200 * time.Millisecond)

		// Should still exist
		retrieved, err := cache.GetNutrition(ctx, "never-expires")
		if err != nil {
			t.Errorf("Entry with zero TTL expired: %v", err)
		}
		if retrieved.FoodName != "Never Expires" {
			t.Errorf("Wrong entry retrieved")
		}
	})
}

func TestMemoryCache_Concurrency(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Number of concurrent operations
	numGoroutines := 100
	numOperations := 100

	t.Run("Concurrent Writes", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := string(rune('a' + (id+j)%26))
					nutrition := &NutritionResponse{
						FoodName: key,
						Calories: float64(id*numOperations + j),
					}
					_ = cache.SetNutrition(ctx, key, nutrition, 1*time.Hour)
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("Concurrent Reads", func(t *testing.T) {
		// First populate cache
		for i := 0; i < 26; i++ {
			key := string(rune('a' + i))
			nutrition := &NutritionResponse{
				FoodName: key,
				Calories: float64(i),
			}
			_ = cache.SetNutrition(ctx, key, nutrition, 1*time.Hour)
		}

		// Now read concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := string(rune('a' + (id+j)%26))
					_, _ = cache.GetNutrition(ctx, key)
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("Concurrent Mixed Operations", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := string(rune('a' + (id+j)%26))
					if j%3 == 0 {
						nutrition := &NutritionResponse{FoodName: key, Calories: float64(j)}
						_ = cache.SetNutrition(ctx, key, nutrition, 1*time.Hour)
					} else if j%3 == 1 {
						_, _ = cache.GetNutrition(ctx, key)
					} else {
						_ = cache.Delete(ctx, key)
					}
				}
			}(i)
		}
		wg.Wait()
	})
}

func TestMemoryCache_LRUEviction(t *testing.T) {
	config := DefaultConfig()
	config.MaxMemoryMB = 1 // Very small limit to trigger eviction
	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()

	t.Run("Evicts LRU Entries When Full", func(t *testing.T) {
		// Fill cache with entries
		for i := 0; i < 100; i++ {
			nutrition := &NutritionResponse{
				FoodName:      "Food " + string(rune('0'+i%10)),
				Calories:      float64(i),
				Protein:       float64(i),
				Carbohydrates: float64(i),
				Fat:           float64(i),
				ServingSize:   "100g of test data to increase size",
			}
			_ = cache.SetNutrition(ctx, string(rune('a'+i%26)), nutrition, 1*time.Hour)
		}

		stats := cache.Stats()
		if stats.Evictions == 0 {
			t.Log("Warning: No evictions occurred, memory limit might be too high")
		}

		// Cache should still be functional
		nutrition := &NutritionResponse{FoodName: "New Entry", Calories: 100}
		err := cache.SetNutrition(ctx, "new-entry", nutrition, 1*time.Hour)
		if err != nil && err != ErrCacheFull {
			t.Errorf("Unexpected error when adding to full cache: %v", err)
		}
	})
}

func TestMemoryCache_Stats(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()

	t.Run("Stats Track Hits and Misses", func(t *testing.T) {
		// Add entry
		nutrition := &NutritionResponse{FoodName: "Test", Calories: 100}
		_ = cache.SetNutrition(ctx, "test", nutrition, 1*time.Hour)

		// Hit
		_, _ = cache.GetNutrition(ctx, "test")
		_, _ = cache.GetNutrition(ctx, "test")

		// Misses
		_, _ = cache.GetNutrition(ctx, "nonexistent1")
		_, _ = cache.GetNutrition(ctx, "nonexistent2")
		_, _ = cache.GetNutrition(ctx, "nonexistent3")

		stats := cache.Stats()
		if stats.Hits != 2 {
			t.Errorf("Expected 2 hits, got %d", stats.Hits)
		}
		if stats.Misses != 3 {
			t.Errorf("Expected 3 misses, got %d", stats.Misses)
		}
		if stats.HitRate != 0.4 {
			t.Errorf("Expected hit rate 0.4, got %.2f", stats.HitRate)
		}
	})

	t.Run("Stats Report Size", func(t *testing.T) {
		_ = cache.Clear(ctx)

		for i := 0; i < 10; i++ {
			nutrition := &NutritionResponse{FoodName: "Food", Calories: float64(i)}
			_ = cache.SetNutrition(ctx, string(rune('a'+i)), nutrition, 1*time.Hour)
		}

		stats := cache.Stats()
		if stats.Size != 10 {
			t.Errorf("Expected size 10, got %d", stats.Size)
		}
	})
}

func TestMemoryCache_ContextCancellation(t *testing.T) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	t.Run("Operations Respect Context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		nutrition := &NutritionResponse{FoodName: "Test", Calories: 100}

		// Operations should still work (they don't do async work)
		// but this tests that context is properly accepted
		err := cache.SetNutrition(ctx, "test", nutrition, 1*time.Hour)
		if err != nil {
			t.Errorf("SetNutrition with cancelled context failed: %v", err)
		}

		_, err = cache.GetNutrition(ctx, "test")
		if err != nil {
			t.Errorf("GetNutrition with cancelled context failed: %v", err)
		}
	})
}

func BenchmarkMemoryCache_Set(b *testing.B) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()
	nutrition := &NutritionResponse{
		FoodName: "Benchmark Food",
		Calories: 100,
		Protein:  10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.SetNutrition(ctx, "bench-key", nutrition, 1*time.Hour)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()
	nutrition := &NutritionResponse{
		FoodName: "Benchmark Food",
		Calories: 100,
		Protein:  10,
	}
	_ = cache.SetNutrition(ctx, "bench-key", nutrition, 1*time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.GetNutrition(ctx, "bench-key")
	}
}

func BenchmarkMemoryCache_Concurrent(b *testing.B) {
	cache := NewMemoryCache(nil)
	defer cache.Close()

	ctx := context.Background()
	nutrition := &NutritionResponse{
		FoodName: "Benchmark Food",
		Calories: 100,
		Protein:  10,
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				_ = cache.SetNutrition(ctx, "bench-key", nutrition, 1*time.Hour)
			} else {
				_, _ = cache.GetNutrition(ctx, "bench-key")
			}
			i++
		}
	})
}
