package cost

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewTracker(t *testing.T) {
	config := DefaultConfig()
	tracker := NewTracker(config)
	defer tracker.Close()

	if tracker == nil {
		t.Fatal("expected non-nil tracker")
	}

	stats := tracker.GetStats()
	if stats["total_users"].(int) != 0 {
		t.Errorf("expected 0 users, got %d", stats["total_users"].(int))
	}
}

func TestRecord(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	tests := []struct {
		name      string
		userID    string
		service   string
		cost      float64
		wantError error
	}{
		{
			name:      "valid record",
			userID:    "user1",
			service:   ServiceGroq,
			cost:      0.50,
			wantError: nil,
		},
		{
			name:      "zero cost",
			userID:    "user1",
			service:   ServiceOpenRouter,
			cost:      0,
			wantError: nil,
		},
		{
			name:      "invalid user ID",
			userID:    "",
			service:   ServiceGroq,
			cost:      0.50,
			wantError: ErrInvalidUserID,
		},
		{
			name:      "invalid service",
			userID:    "user1",
			service:   "",
			cost:      0.50,
			wantError: ErrInvalidService,
		},
		{
			name:      "negative cost",
			userID:    "user1",
			service:   ServiceGroq,
			cost:      -0.50,
			wantError: ErrInvalidCost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tracker.Record(ctx, tt.userID, tt.service, tt.cost)
			if err != tt.wantError {
				t.Errorf("expected error %v, got %v", tt.wantError, err)
			}
		})
	}
}

func TestRecordWithMetadata(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	metadata := map[string]interface{}{
		"model":   "llama-3.1-70b",
		"tokens":  1500,
		"request": "chat-completion",
	}

	err := tracker.RecordWithMetadata(ctx, "user1", ServiceGroq, 0.75, metadata)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records, err := tracker.GetRecords(ctx, "user1", time.Now().Add(-1*time.Hour), time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	if records[0].Metadata["model"] != "llama-3.1-70b" {
		t.Errorf("expected model metadata to be preserved")
	}
}

func TestGetMonthlyUsage(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	// Record some usage
	err := tracker.Record(ctx, "user1", ServiceGroq, 10.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = tracker.Record(ctx, "user1", ServiceOpenRouter, 15.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	usage, err := tracker.GetMonthlyUsage(ctx, "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 25.0
	if usage != expected {
		t.Errorf("expected usage %.2f, got %.2f", expected, usage)
	}
}

func TestGetRolling30DayUsage(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	// Record current usage
	err := tracker.Record(ctx, "user1", ServiceGroq, 10.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Manually add an old record (should not be counted)
	oldRecord := UsageRecord{
		ID:        "old-1",
		UserID:    "user1",
		Service:   ServiceGroq,
		Cost:      50.0,
		Timestamp: time.Now().AddDate(0, 0, -31),
	}
	tracker.mu.Lock()
	tracker.storage["user1"] = append(tracker.storage["user1"], oldRecord)
	tracker.mu.Unlock()

	usage, err := tracker.GetRolling30DayUsage(ctx, "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only count the recent 10.0, not the 50.0 from 31 days ago
	if usage >= 15.0 {
		t.Errorf("expected usage around 10.0, got %.2f (old records should be excluded)", usage)
	}
}

func TestGetGlobalMonthlyUsage(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	// Record usage for multiple users
	tracker.Record(ctx, "user1", ServiceGroq, 10.0)
	tracker.Record(ctx, "user2", ServiceOpenRouter, 20.0)
	tracker.Record(ctx, "user3", ServiceClaude, 30.0)

	usage, err := tracker.GetGlobalMonthlyUsage(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 60.0
	if usage != expected {
		t.Errorf("expected global usage %.2f, got %.2f", expected, usage)
	}
}

func TestCanMakeRequest(t *testing.T) {
	config := DefaultConfig()
	config.DefaultUserMonthlyLimit = 100.0
	tracker := NewTracker(config)
	defer tracker.Close()
	ctx := context.Background()

	tests := []struct {
		name          string
		existingUsage float64
		estimatedCost float64
		want          bool
	}{
		{
			name:          "within limit",
			existingUsage: 50.0,
			estimatedCost: 30.0,
			want:          true,
		},
		{
			name:          "exactly at limit",
			existingUsage: 70.0,
			estimatedCost: 30.0,
			want:          true,
		},
		{
			name:          "exceeds limit",
			existingUsage: 80.0,
			estimatedCost: 30.0,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := "user-" + tt.name

			// Record existing usage
			if tt.existingUsage > 0 {
				tracker.Record(ctx, userID, ServiceGroq, tt.existingUsage)
			}

			can, err := tracker.CanMakeRequest(ctx, userID, tt.estimatedCost)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if can != tt.want {
				t.Errorf("expected %v, got %v", tt.want, can)
			}
		})
	}
}

func TestCheckLimits(t *testing.T) {
	config := DefaultConfig()
	config.DefaultUserMonthlyLimit = 100.0
	tracker := NewTracker(config)
	defer tracker.Close()
	ctx := context.Background()

	// Record existing usage
	tracker.Record(ctx, "user1", ServiceGroq, 75.0)

	result, err := tracker.CheckLimits(ctx, "user1", 30.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Allowed {
		t.Error("expected request to be denied")
	}

	if result.CurrentUsage != 75.0 {
		t.Errorf("expected current usage 75.0, got %.2f", result.CurrentUsage)
	}

	if result.Limit != 100.0 {
		t.Errorf("expected limit 100.0, got %.2f", result.Limit)
	}

	if result.EstimatedTotal != 105.0 {
		t.Errorf("expected estimated total 105.0, got %.2f", result.EstimatedTotal)
	}

	if result.RemainingBudget != 25.0 {
		t.Errorf("expected remaining budget 25.0, got %.2f", result.RemainingBudget)
	}
}

func TestGlobalLimitEnforcement(t *testing.T) {
	config := DefaultConfig()
	config.GlobalMonthlyLimit = 100.0
	config.DefaultUserMonthlyLimit = 50.0
	tracker := NewTracker(config)
	defer tracker.Close()
	ctx := context.Background()

	// Record usage that's within individual limits but exceeds global
	tracker.Record(ctx, "user1", ServiceGroq, 40.0)
	tracker.Record(ctx, "user2", ServiceOpenRouter, 40.0)

	// User3 is within their limit but would exceed global
	can, err := tracker.CanMakeRequest(ctx, "user3", 25.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if can {
		t.Error("expected request to be denied due to global limit")
	}
}

func TestGetUsageByService(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	// Record usage across different services
	tracker.Record(ctx, "user1", ServiceGroq, 10.0)
	tracker.Record(ctx, "user1", ServiceGroq, 15.0)
	tracker.Record(ctx, "user1", ServiceOpenRouter, 20.0)
	tracker.Record(ctx, "user1", ServiceClaude, 30.0)

	now := time.Now()
	breakdown, err := tracker.GetUsageByService(ctx, "user1", now.Add(-1*time.Hour), now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedGroq := 25.0
	if breakdown[ServiceGroq] != expectedGroq {
		t.Errorf("expected groq usage %.2f, got %.2f", expectedGroq, breakdown[ServiceGroq])
	}

	expectedOpenRouter := 20.0
	if breakdown[ServiceOpenRouter] != expectedOpenRouter {
		t.Errorf("expected openrouter usage %.2f, got %.2f", expectedOpenRouter, breakdown[ServiceOpenRouter])
	}

	expectedClaude := 30.0
	if breakdown[ServiceClaude] != expectedClaude {
		t.Errorf("expected claude usage %.2f, got %.2f", expectedClaude, breakdown[ServiceClaude])
	}
}

func TestGetSummary(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	// Record usage
	tracker.Record(ctx, "user1", ServiceGroq, 10.0)
	tracker.Record(ctx, "user1", ServiceOpenRouter, 20.0)
	tracker.Record(ctx, "user1", ServiceClaude, 30.0)

	now := time.Now()
	summary, err := tracker.GetSummary(ctx, "user1", now.Add(-1*time.Hour), now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.TotalCost != 60.0 {
		t.Errorf("expected total cost 60.0, got %.2f", summary.TotalCost)
	}

	if summary.RecordCount != 3 {
		t.Errorf("expected 3 records, got %d", summary.RecordCount)
	}

	if len(summary.ByService) != 3 {
		t.Errorf("expected 3 services, got %d", len(summary.ByService))
	}
}

func TestSetUserLimit(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	// Set custom limit
	err := tracker.SetUserLimit("user1", 200.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Record usage up to 150
	tracker.Record(ctx, "user1", ServiceGroq, 150.0)

	// Should still be allowed (custom limit is 200)
	can, err := tracker.CanMakeRequest(ctx, "user1", 40.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !can {
		t.Error("expected request to be allowed with custom limit")
	}

	// Should be denied
	can, err = tracker.CanMakeRequest(ctx, "user1", 60.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if can {
		t.Error("expected request to be denied")
	}
}

func TestSetGlobalLimit(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()

	err := tracker.SetGlobalLimit(500.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tracker.globalLimits.MonthlyLimit != 500.0 {
		t.Errorf("expected global limit 500.0, got %.2f", tracker.globalLimits.MonthlyLimit)
	}

	// Test invalid limit
	err = tracker.SetGlobalLimit(-100.0)
	if err != ErrInvalidCost {
		t.Errorf("expected ErrInvalidCost, got %v", err)
	}
}

func TestConcurrentRecording(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 100
	recordsPerGoroutine := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			userID := "user1"
			for j := 0; j < recordsPerGoroutine; j++ {
				err := tracker.Record(ctx, userID, ServiceGroq, 1.0)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify total usage
	usage, err := tracker.GetMonthlyUsage(ctx, "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := float64(numGoroutines * recordsPerGoroutine)
	if usage != expected {
		t.Errorf("expected usage %.2f, got %.2f", expected, usage)
	}
}

func TestConcurrentLimitChecks(t *testing.T) {
	config := DefaultConfig()
	config.DefaultUserMonthlyLimit = 1000.0
	tracker := NewTracker(config)
	defer tracker.Close()
	ctx := context.Background()

	// Pre-populate with some usage
	tracker.Record(ctx, "user1", ServiceGroq, 500.0)

	var wg sync.WaitGroup
	numChecks := 50

	wg.Add(numChecks)
	for i := 0; i < numChecks; i++ {
		go func() {
			defer wg.Done()
			_, err := tracker.CanMakeRequest(ctx, "user1", 100.0)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestCleanup(t *testing.T) {
	config := DefaultConfig()
	config.CleanupRetentionDays = 1
	tracker := NewTracker(config)
	defer tracker.Close()

	// Add old record
	oldRecord := UsageRecord{
		ID:        "old-1",
		UserID:    "user1",
		Service:   ServiceGroq,
		Cost:      50.0,
		Timestamp: time.Now().AddDate(0, 0, -2), // 2 days old
	}

	tracker.mu.Lock()
	tracker.storage["user1"] = []UsageRecord{oldRecord}
	tracker.mu.Unlock()

	// Add recent record
	ctx := context.Background()
	tracker.Record(ctx, "user2", ServiceGroq, 10.0)

	// Run cleanup
	tracker.cleanup()

	// Check that old record was removed
	tracker.mu.RLock()
	user1Records := tracker.storage["user1"]
	user2Records := tracker.storage["user2"]
	tracker.mu.RUnlock()

	if len(user1Records) != 0 {
		t.Errorf("expected old records to be cleaned up, got %d records", len(user1Records))
	}

	if len(user2Records) != 1 {
		t.Errorf("expected recent record to be kept, got %d records", len(user2Records))
	}
}

func TestGetRecords(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	// Record some usage
	tracker.Record(ctx, "user1", ServiceGroq, 10.0)
	tracker.Record(ctx, "user1", ServiceOpenRouter, 20.0)

	now := time.Now()
	records, err := tracker.GetRecords(ctx, "user1", now.Add(-1*time.Hour), now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}

	// Test empty result
	records, err = tracker.GetRecords(ctx, "nonexistent", now.Add(-1*time.Hour), now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestInvalidTimeRange(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	now := time.Now()
	start := now
	end := now.Add(-1 * time.Hour) // end before start

	_, err := tracker.GetUsageByService(ctx, "user1", start, end)
	if err != ErrInvalidTimeRange {
		t.Errorf("expected ErrInvalidTimeRange, got %v", err)
	}

	_, err = tracker.GetSummary(ctx, "user1", start, end)
	if err != ErrInvalidTimeRange {
		t.Errorf("expected ErrInvalidTimeRange, got %v", err)
	}

	_, err = tracker.GetRecords(ctx, "user1", start, end)
	if err != ErrInvalidTimeRange {
		t.Errorf("expected ErrInvalidTimeRange, got %v", err)
	}
}

func TestGetStats(t *testing.T) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	// Initial stats
	stats := tracker.GetStats()
	if stats["total_users"].(int) != 0 {
		t.Errorf("expected 0 users initially")
	}

	// Add some records
	tracker.Record(ctx, "user1", ServiceGroq, 10.0)
	tracker.Record(ctx, "user1", ServiceOpenRouter, 20.0)
	tracker.Record(ctx, "user2", ServiceClaude, 30.0)

	// Set custom limit
	tracker.SetUserLimit("user1", 200.0)

	stats = tracker.GetStats()
	if stats["total_users"].(int) != 2 {
		t.Errorf("expected 2 users, got %d", stats["total_users"].(int))
	}

	if stats["total_records"].(int) != 3 {
		t.Errorf("expected 3 records, got %d", stats["total_records"].(int))
	}

	if stats["custom_limits"].(int) != 1 {
		t.Errorf("expected 1 custom limit, got %d", stats["custom_limits"].(int))
	}
}

func BenchmarkRecord(b *testing.B) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.Record(ctx, "user1", ServiceGroq, 1.0)
	}
}

func BenchmarkGetMonthlyUsage(b *testing.B) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		tracker.Record(ctx, "user1", ServiceGroq, 1.0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.GetMonthlyUsage(ctx, "user1")
	}
}

func BenchmarkCanMakeRequest(b *testing.B) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	// Pre-populate with data
	tracker.Record(ctx, "user1", ServiceGroq, 50.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.CanMakeRequest(ctx, "user1", 10.0)
	}
}

func BenchmarkConcurrentRecords(b *testing.B) {
	tracker := NewTracker(DefaultConfig())
	defer tracker.Close()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tracker.Record(ctx, "user1", ServiceGroq, 1.0)
		}
	})
}
