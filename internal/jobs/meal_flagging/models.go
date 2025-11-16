package meal_flagging

import (
	"time"

	"github.com/google/uuid"
)

// FlagType represents the type of quality issue detected
type FlagType string

const (
	FlagTypeUnusualPortion FlagType = "unusual_portion"
	FlagTypeMacroMismatch  FlagType = "macro_mismatch"
	FlagTypeDuplicate      FlagType = "duplicate"
	FlagTypeLowConfidence  FlagType = "low_confidence"
)

// MealFlag represents a quality issue detected in a meal
type MealFlag struct {
	ID          uuid.UUID `json:"id" db:"id"`
	MealID      uuid.UUID `json:"meal_id" db:"meal_id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	FlagType    FlagType  `json:"flag_type" db:"flag_type"`
	Severity    string    `json:"severity" db:"severity"` // low, medium, high
	Description string    `json:"description" db:"description"`
	Details     string    `json:"details" db:"details"` // JSON with specific info
	Resolved    bool      `json:"resolved" db:"resolved"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy  *uuid.UUID `json:"resolved_by,omitempty" db:"resolved_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// FlagResult contains the result of a flagging check
type FlagResult struct {
	ShouldFlag  bool
	FlagType    FlagType
	Severity    string
	Description string
	Details     map[string]interface{}
}

// FlagSummary provides a summary of flagging results
type FlagSummary struct {
	TotalMealsChecked int                  `json:"total_meals_checked"`
	TotalFlagsCreated int                  `json:"total_flags_created"`
	FlagsByType       map[FlagType]int     `json:"flags_by_type"`
	FlagsBySeverity   map[string]int       `json:"flags_by_severity"`
	ProcessedAt       time.Time            `json:"processed_at"`
	Duration          time.Duration        `json:"duration"`
	Errors            []string             `json:"errors,omitempty"`
}

// NewFlagSummary creates a new flag summary
func NewFlagSummary() *FlagSummary {
	return &FlagSummary{
		FlagsByType:     make(map[FlagType]int),
		FlagsBySeverity: make(map[string]int),
		Errors:          make([]string, 0),
		ProcessedAt:     time.Now(),
	}
}

// AddFlag increments counters for a flag
func (s *FlagSummary) AddFlag(flagType FlagType, severity string) {
	s.TotalFlagsCreated++
	s.FlagsByType[flagType]++
	s.FlagsBySeverity[severity]++
}

// AddError adds an error to the summary
func (s *FlagSummary) AddError(err string) {
	s.Errors = append(s.Errors, err)
}

// MealData represents the meal data needed for flagging checks
type MealData struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Name         string
	Description  string
	Calories     float64
	Protein      float64
	Carbs        float64
	Fat          float64
	LoggedAt     time.Time
	AIConfidence *float64
	Items        []MealItemData
}

// MealItemData represents an individual item in a meal
type MealItemData struct {
	ID          uuid.UUID
	Description string
	Quantity    float64
	Unit        string
	Calories    float64
	Protein     float64
	Carbs       float64
	Fat         float64
}

// PortionThreshold defines thresholds for unusual portions
type PortionThreshold struct {
	FoodCategory string
	MaxQuantity  float64
	Unit         string
	MaxCalories  float64
}

// DefaultPortionThresholds returns standard portion size limits
func DefaultPortionThresholds() []PortionThreshold {
	return []PortionThreshold{
		{FoodCategory: "chicken", MaxQuantity: 450, Unit: "g", MaxCalories: 1000},
		{FoodCategory: "beef", MaxQuantity: 450, Unit: "g", MaxCalories: 1000},
		{FoodCategory: "pork", MaxQuantity: 450, Unit: "g", MaxCalories: 1000},
		{FoodCategory: "fish", MaxQuantity: 500, Unit: "g", MaxCalories: 1000},
		{FoodCategory: "rice", MaxQuantity: 300, Unit: "g", MaxCalories: 1000},
		{FoodCategory: "pasta", MaxQuantity: 300, Unit: "g", MaxCalories: 1000},
		{FoodCategory: "bread", MaxQuantity: 400, Unit: "g", MaxCalories: 1000},
		{FoodCategory: "oil", MaxQuantity: 30, Unit: "ml", MaxCalories: 300},
		{FoodCategory: "butter", MaxQuantity: 50, Unit: "g", MaxCalories: 400},
		{FoodCategory: "cheese", MaxQuantity: 200, Unit: "g", MaxCalories: 800},
		{FoodCategory: "nuts", MaxQuantity: 100, Unit: "g", MaxCalories: 600},
	}
}

// Config holds the job configuration
type Config struct {
	Enabled      bool   `json:"enabled"`
	ScheduleTime string `json:"schedule_time"` // "02:00" (24-hour format)
	BatchSize    int    `json:"batch_size"`
	LookbackDays int    `json:"lookback_days"`

	// Tolerance settings
	MacroTolerancePercent float64 `json:"macro_tolerance_percent"` // Default: 10%
	ConfidenceThreshold   float64 `json:"confidence_threshold"`    // Default: 0.7
	DuplicateWindowMinutes int    `json:"duplicate_window_minutes"` // Default: 30
	DuplicateSimilarityPercent float64 `json:"duplicate_similarity_percent"` // Default: 80%
}

// DefaultConfig returns the default job configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:                    true,
		ScheduleTime:               "02:00",
		BatchSize:                  1000,
		LookbackDays:               1,
		MacroTolerancePercent:      10.0,
		ConfidenceThreshold:        0.7,
		DuplicateWindowMinutes:     30,
		DuplicateSimilarityPercent: 80.0,
	}
}
