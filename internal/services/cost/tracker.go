package cost

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Tracker manages cost tracking and limit enforcement
type Tracker struct {
	config       Config
	storage      map[string][]UsageRecord // userID -> records
	userLimits   map[string]UserLimits    // userID -> limits
	globalLimits GlobalLimits
	mu           sync.RWMutex
	stopCleanup  chan struct{}
	cleanupDone  chan struct{}
}

// NewTracker creates a new cost tracker instance
func NewTracker(config Config) *Tracker {
	t := &Tracker{
		config:       config,
		storage:      make(map[string][]UsageRecord),
		userLimits:   make(map[string]UserLimits),
		globalLimits: GlobalLimits{MonthlyLimit: config.GlobalMonthlyLimit},
		stopCleanup:  make(chan struct{}),
		cleanupDone:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go t.cleanupLoop()

	return t
}

// Close stops the tracker and cleanup goroutine
func (t *Tracker) Close() {
	close(t.stopCleanup)
	<-t.cleanupDone
}

// Record records a new AI usage event
func (t *Tracker) Record(ctx context.Context, userID, service string, cost float64) error {
	if userID == "" {
		return ErrInvalidUserID
	}
	if service == "" {
		return ErrInvalidService
	}
	if cost < 0 {
		return ErrInvalidCost
	}

	record := UsageRecord{
		ID:        uuid.New().String(),
		UserID:    userID,
		Service:   service,
		Cost:      cost,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.storage[userID] = append(t.storage[userID], record)

	return nil
}

// RecordWithMetadata records usage with additional metadata
func (t *Tracker) RecordWithMetadata(ctx context.Context, userID, service string, cost float64, metadata map[string]interface{}) error {
	if userID == "" {
		return ErrInvalidUserID
	}
	if service == "" {
		return ErrInvalidService
	}
	if cost < 0 {
		return ErrInvalidCost
	}

	record := UsageRecord{
		ID:        uuid.New().String(),
		UserID:    userID,
		Service:   service,
		Cost:      cost,
		Timestamp: time.Now().UTC(),
		Metadata:  metadata,
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.storage[userID] = append(t.storage[userID], record)

	return nil
}

// GetMonthlyUsage returns the total cost for a user in the current month
func (t *Tracker) GetMonthlyUsage(ctx context.Context, userID string) (float64, error) {
	if userID == "" {
		return 0, ErrInvalidUserID
	}

	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	return t.getUserUsageInRange(userID, start, end), nil
}

// GetRolling30DayUsage returns usage for the last 30 days
func (t *Tracker) GetRolling30DayUsage(ctx context.Context, userID string) (float64, error) {
	if userID == "" {
		return 0, ErrInvalidUserID
	}

	now := time.Now().UTC()
	start := now.AddDate(0, 0, -30)

	return t.getUserUsageInRange(userID, start, now), nil
}

// GetGlobalMonthlyUsage returns the total cost across all users for the current month
func (t *Tracker) GetGlobalMonthlyUsage(ctx context.Context) (float64, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	t.mu.RLock()
	defer t.mu.RUnlock()

	var total float64
	for _, records := range t.storage {
		for _, record := range records {
			if record.Timestamp.After(start) && record.Timestamp.Before(end) {
				total += record.Cost
			}
		}
	}

	return total, nil
}

// GetGlobalRolling30DayUsage returns global usage for the last 30 days
func (t *Tracker) GetGlobalRolling30DayUsage(ctx context.Context) (float64, error) {
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -30)

	t.mu.RLock()
	defer t.mu.RUnlock()

	var total float64
	for _, records := range t.storage {
		for _, record := range records {
			if record.Timestamp.After(start) && record.Timestamp.Before(now) {
				total += record.Cost
			}
		}
	}

	return total, nil
}

// CanMakeRequest checks if a user can make a request given the estimated cost
func (t *Tracker) CanMakeRequest(ctx context.Context, userID string, estimatedCost float64) (bool, error) {
	if userID == "" {
		return false, ErrInvalidUserID
	}
	if estimatedCost < 0 {
		return false, ErrInvalidCost
	}

	result, err := t.CheckLimits(ctx, userID, estimatedCost)
	if err != nil {
		return false, err
	}

	return result.Allowed, nil
}

// CheckLimits performs a comprehensive limit check with detailed results
func (t *Tracker) CheckLimits(ctx context.Context, userID string, estimatedCost float64) (*LimitCheckResult, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}
	if estimatedCost < 0 {
		return nil, ErrInvalidCost
	}

	// Get current usage (rolling 30 day)
	currentUsage, err := t.GetRolling30DayUsage(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get user limit
	userLimit := t.getUserLimit(userID)
	estimatedTotal := currentUsage + estimatedCost

	// Check user limit
	if estimatedTotal > userLimit {
		return &LimitCheckResult{
			Allowed:         false,
			CurrentUsage:    currentUsage,
			Limit:           userLimit,
			EstimatedTotal:  estimatedTotal,
			RemainingBudget: userLimit - currentUsage,
			Reason:          fmt.Sprintf("user monthly limit exceeded (%.2f + %.2f > %.2f)", currentUsage, estimatedCost, userLimit),
		}, nil
	}

	// Check global limit
	globalUsage, err := t.GetGlobalRolling30DayUsage(ctx)
	if err != nil {
		return nil, err
	}

	globalEstimated := globalUsage + estimatedCost
	if globalEstimated > t.globalLimits.MonthlyLimit {
		return &LimitCheckResult{
			Allowed:         false,
			CurrentUsage:    currentUsage,
			Limit:           userLimit,
			EstimatedTotal:  estimatedTotal,
			RemainingBudget: userLimit - currentUsage,
			Reason:          fmt.Sprintf("global monthly limit exceeded (%.2f + %.2f > %.2f)", globalUsage, estimatedCost, t.globalLimits.MonthlyLimit),
		}, nil
	}

	return &LimitCheckResult{
		Allowed:         true,
		CurrentUsage:    currentUsage,
		Limit:           userLimit,
		EstimatedTotal:  estimatedTotal,
		RemainingBudget: userLimit - currentUsage,
	}, nil
}

// GetUsageByService returns usage breakdown by service for a time range
func (t *Tracker) GetUsageByService(ctx context.Context, userID string, start, end time.Time) (map[string]float64, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}
	if end.Before(start) {
		return nil, ErrInvalidTimeRange
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	breakdown := make(map[string]float64)
	records, ok := t.storage[userID]
	if !ok {
		return breakdown, nil
	}

	for _, record := range records {
		if record.Timestamp.After(start) && record.Timestamp.Before(end) {
			breakdown[record.Service] += record.Cost
		}
	}

	return breakdown, nil
}

// GetSummary returns a comprehensive cost summary for a user
func (t *Tracker) GetSummary(ctx context.Context, userID string, start, end time.Time) (*CostSummary, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}
	if end.Before(start) {
		return nil, ErrInvalidTimeRange
	}

	breakdown, err := t.GetUsageByService(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}

	var totalCost float64
	var recordCount int

	t.mu.RLock()
	records := t.storage[userID]
	for _, record := range records {
		if record.Timestamp.After(start) && record.Timestamp.Before(end) {
			totalCost += record.Cost
			recordCount++
		}
	}
	t.mu.RUnlock()

	return &CostSummary{
		UserID:      userID,
		TotalCost:   totalCost,
		ByService:   breakdown,
		RecordCount: recordCount,
		PeriodStart: start,
		PeriodEnd:   end,
	}, nil
}

// SetUserLimit sets a custom limit for a specific user
func (t *Tracker) SetUserLimit(userID string, monthlyLimit float64) error {
	if userID == "" {
		return ErrInvalidUserID
	}
	if monthlyLimit < 0 {
		return ErrInvalidCost
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.userLimits[userID] = UserLimits{
		UserID:       userID,
		MonthlyLimit: monthlyLimit,
	}

	return nil
}

// SetGlobalLimit sets the global monthly limit
func (t *Tracker) SetGlobalLimit(monthlyLimit float64) error {
	if monthlyLimit < 0 {
		return ErrInvalidCost
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.globalLimits.MonthlyLimit = monthlyLimit

	return nil
}

// GetRecords returns usage records for a user within a time range
func (t *Tracker) GetRecords(ctx context.Context, userID string, start, end time.Time) ([]UsageRecord, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}
	if end.Before(start) {
		return nil, ErrInvalidTimeRange
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	records, ok := t.storage[userID]
	if !ok {
		return []UsageRecord{}, nil
	}

	var filtered []UsageRecord
	for _, record := range records {
		if record.Timestamp.After(start) && record.Timestamp.Before(end) {
			filtered = append(filtered, record)
		}
	}

	return filtered, nil
}

// getUserLimit returns the limit for a user (custom or default)
func (t *Tracker) getUserLimit(userID string) float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if limit, ok := t.userLimits[userID]; ok {
		return limit.MonthlyLimit
	}

	return t.config.DefaultUserMonthlyLimit
}

// getUserUsageInRange calculates total usage for a user in a time range
func (t *Tracker) getUserUsageInRange(userID string, start, end time.Time) float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	records, ok := t.storage[userID]
	if !ok {
		return 0
	}

	var total float64
	for _, record := range records {
		if record.Timestamp.After(start) && record.Timestamp.Before(end) {
			total += record.Cost
		}
	}

	return total
}

// cleanupLoop periodically removes old records
func (t *Tracker) cleanupLoop() {
	ticker := time.NewTicker(t.config.CleanupInterval)
	defer ticker.Stop()
	defer close(t.cleanupDone)

	for {
		select {
		case <-ticker.C:
			t.cleanup()
		case <-t.stopCleanup:
			return
		}
	}
}

// cleanup removes records older than the retention period
func (t *Tracker) cleanup() {
	cutoff := time.Now().UTC().AddDate(0, 0, -t.config.CleanupRetentionDays)

	t.mu.Lock()
	defer t.mu.Unlock()

	for userID, records := range t.storage {
		var kept []UsageRecord
		for _, record := range records {
			if record.Timestamp.After(cutoff) {
				kept = append(kept, record)
			}
		}

		if len(kept) == 0 {
			delete(t.storage, userID)
		} else {
			t.storage[userID] = kept
		}
	}
}

// GetStats returns statistics about the tracker
func (t *Tracker) GetStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var totalRecords int
	for _, records := range t.storage {
		totalRecords += len(records)
	}

	return map[string]interface{}{
		"total_users":   len(t.storage),
		"total_records": totalRecords,
		"custom_limits": len(t.userLimits),
	}
}
