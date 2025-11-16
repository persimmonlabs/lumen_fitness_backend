package cost

import (
	"errors"
	"time"
)

// Common errors
var (
	ErrCostLimitExceeded       = errors.New("user monthly cost limit exceeded")
	ErrGlobalCostLimitExceeded = errors.New("global monthly cost limit exceeded")
	ErrInvalidUserID           = errors.New("invalid user ID")
	ErrInvalidService          = errors.New("invalid service name")
	ErrInvalidCost             = errors.New("invalid cost value")
	ErrInvalidTimeRange        = errors.New("invalid time range")
)

// Service names
const (
	ServiceGroq       = "groq"
	ServiceOpenRouter = "openrouter"
	ServiceClaude     = "claude"
	ServiceOpenAI     = "openai"
)

// UsageRecord represents a single AI service usage event
type UsageRecord struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Service   string    `json:"service"`
	Cost      float64   `json:"cost"`
	Timestamp time.Time `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CostSummary provides aggregated cost information
type CostSummary struct {
	UserID       string             `json:"user_id"`
	TotalCost    float64            `json:"total_cost"`
	ByService    map[string]float64 `json:"by_service"`
	RecordCount  int                `json:"record_count"`
	PeriodStart  time.Time          `json:"period_start"`
	PeriodEnd    time.Time          `json:"period_end"`
}

// UserLimits defines cost limits for a user
type UserLimits struct {
	UserID       string  `json:"user_id"`
	MonthlyLimit float64 `json:"monthly_limit"`
	DailyLimit   float64 `json:"daily_limit,omitempty"`
}

// GlobalLimits defines organization-wide cost limits
type GlobalLimits struct {
	MonthlyLimit float64 `json:"monthly_limit"`
	DailyLimit   float64 `json:"daily_limit,omitempty"`
}

// Config holds configuration for the cost tracker
type Config struct {
	DefaultUserMonthlyLimit float64
	GlobalMonthlyLimit      float64
	CleanupRetentionDays    int
	CleanupInterval         time.Duration
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		DefaultUserMonthlyLimit: 100.0, // $100 per user per month
		GlobalMonthlyLimit:      10000.0, // $10,000 organization-wide
		CleanupRetentionDays:    90,      // Keep 90 days of history
		CleanupInterval:         24 * time.Hour,
	}
}

// UsageQuery defines parameters for querying usage records
type UsageQuery struct {
	UserID   string
	Service  string
	Start    time.Time
	End      time.Time
	Limit    int
}

// LimitCheckResult contains the result of a limit check
type LimitCheckResult struct {
	Allowed         bool    `json:"allowed"`
	CurrentUsage    float64 `json:"current_usage"`
	Limit           float64 `json:"limit"`
	EstimatedTotal  float64 `json:"estimated_total"`
	RemainingBudget float64 `json:"remaining_budget"`
	Reason          string  `json:"reason,omitempty"`
}
