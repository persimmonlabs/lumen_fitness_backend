package meals

import (
	"time"

	"github.com/google/uuid"
)

// MealType represents the type of meal
type MealType string

const (
	MealTypeBreakfast MealType = "breakfast"
	MealTypeLunch     MealType = "lunch"
	MealTypeDinner    MealType = "dinner"
	MealTypeSnack     MealType = "snack"
)

// IsValid checks if the meal type is valid
func (mt MealType) IsValid() bool {
	switch mt {
	case MealTypeBreakfast, MealTypeLunch, MealTypeDinner, MealTypeSnack:
		return true
	default:
		return false
	}
}

// DraftStatus represents the processing status of a draft meal
type DraftStatus string

const (
	DraftStatusAnalyzing DraftStatus = "analyzing" // AI is processing the meal
	DraftStatusReady     DraftStatus = "ready"     // AI processing complete, ready for review
	DraftStatusError     DraftStatus = "error"     // AI processing failed
)

// IsValid checks if the draft status is valid
func (ds DraftStatus) IsValid() bool {
	switch ds {
	case DraftStatusAnalyzing, DraftStatusReady, DraftStatusError:
		return true
	default:
		return false
	}
}

// Meal represents a meal with nutrition totals
type Meal struct {
	ID            uuid.UUID    `json:"id" db:"id"`
	UserID        uuid.UUID    `json:"user_id" db:"user_id"`
	MealType      MealType     `json:"meal_type" db:"meal_type"`
	ConsumedAt    time.Time    `json:"consumed_at" db:"consumed_at"`
	Photos        []string     `json:"photos" db:"photos"`
	Notes         string       `json:"notes" db:"notes"`
	TotalCalories float64      `json:"total_calories" db:"total_calories"`
	TotalProteinG float64      `json:"total_protein_g" db:"total_protein_g"`
	TotalCarbsG   float64      `json:"total_carbs_g" db:"total_carbs_g"`
	TotalFatG     float64      `json:"total_fat_g" db:"total_fat_g"`
	TotalFiberG   float64      `json:"total_fiber_g" db:"total_fiber_g"`
	IsDraft       bool         `json:"is_draft" db:"is_draft"`
	DraftStatus   *DraftStatus `json:"draft_status,omitempty" db:"draft_status"`
	DraftError    *string      `json:"draft_error,omitempty" db:"draft_error"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt     *time.Time   `json:"deleted_at,omitempty" db:"deleted_at"`
}

// MealItem represents a single food item in a meal
type MealItem struct {
	ID        uuid.UUID `json:"id" db:"id"`
	MealID    uuid.UUID `json:"meal_id" db:"meal_id"`
	Name      string    `json:"name" db:"name"`
	Quantity  float64   `json:"quantity" db:"quantity"`
	Unit      string    `json:"unit" db:"unit"`
	Calories  float64   `json:"calories" db:"calories"`
	ProteinG  float64   `json:"protein_g" db:"protein"`
	CarbsG    float64   `json:"carbs_g" db:"carbs"`
	FatG      float64   `json:"fat_g" db:"fat"`
	FiberG    float64   `json:"fiber_g" db:"fiber"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// MealWithItems combines a meal with its items
type MealWithItems struct {
	Meal
	Items []MealItem `json:"items"`
}

// ParseMealRequest represents a request to parse a meal description
type ParseMealRequest struct {
	Description    string    `json:"description" validate:"required,max=500"`
	MealType       MealType  `json:"meal_type" validate:"required"`
	ConsumedAt     time.Time `json:"consumed_at" validate:"required"`
	Photos         []string  `json:"photos" validate:"max=3,dive,required"`
	IdempotencyKey string    `json:"idempotency_key" validate:"required,max=100"`
}

// DraftMealItem represents a parsed food item (not yet saved)
type DraftMealItem struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
	Calories float64 `json:"calories"`
	ProteinG float64 `json:"protein_g"`
	CarbsG   float64 `json:"carbs_g"`
	FatG     float64 `json:"fat_g"`
	FiberG   float64 `json:"fiber_g"`
}

// NutritionTotals represents aggregated nutrition data
type NutritionTotals struct {
	Calories float64 `json:"calories"`
	ProteinG float64 `json:"protein_g"`
	CarbsG   float64 `json:"carbs_g"`
	FatG     float64 `json:"fat_g"`
	FiberG   float64 `json:"fiber_g"`
}

// ParseMealResponse represents the response from parsing a meal
type ParseMealResponse struct {
	DraftID   uuid.UUID   `json:"draft_id"`   // ID of created draft meal
	Status    DraftStatus `json:"status"`     // Current processing status
	DraftMeal *struct {
		Items      []DraftMealItem `json:"items"`
		Total      NutritionTotals `json:"total"`
		Confidence float64         `json:"confidence"`
		CostUSD    float64         `json:"cost_usd"`
	} `json:"draft_meal,omitempty"` // Only present if status is 'ready'
}

// DraftStatusResponse represents the response for checking draft status
type DraftStatusResponse struct {
	DraftID uuid.UUID        `json:"draft_id"`
	Status  DraftStatus      `json:"status"`
	Items   []DraftMealItem  `json:"items,omitempty"`
	Total   *NutritionTotals `json:"total,omitempty"`
	Error   *string          `json:"error,omitempty"`
}

// ConfirmMealRequest represents a request to save a parsed meal
type ConfirmMealRequest struct {
	MealType   MealType        `json:"meal_type" validate:"required"`
	ConsumedAt time.Time       `json:"consumed_at" validate:"required"`
	Photos     []string        `json:"photos" validate:"max=3,dive,required"`
	Notes      string          `json:"notes" validate:"max=500"`
	Items      []DraftMealItem `json:"items" validate:"required,min=1,max=50,dive"`
}

// UpdateMealRequest represents a request to update a meal
type UpdateMealRequest struct {
	MealType   MealType        `json:"meal_type" validate:"required"`
	ConsumedAt time.Time       `json:"consumed_at" validate:"required"`
	Photos     []string        `json:"photos" validate:"max=3,dive,required"`
	Notes      string          `json:"notes" validate:"max=500"`
	Items      []DraftMealItem `json:"items" validate:"required,min=1,max=50,dive"`
}

// CopyMealRequest represents a request to duplicate a meal
type CopyMealRequest struct {
	ConsumedAt time.Time `json:"consumed_at" validate:"required"`
	MealType   MealType  `json:"meal_type" validate:"required"`
}

// MealResponse represents a single meal response
type MealResponse struct {
	MealWithItems
}

// MealListItem represents a summary of a meal in a list
type MealListItem struct {
	ID            uuid.UUID `json:"id" db:"id"`
	MealType      MealType  `json:"meal_type" db:"meal_type"`
	ConsumedAt    time.Time `json:"consumed_at" db:"consumed_at"`
	TotalCalories float64   `json:"total_calories" db:"total_calories"`
	TotalProteinG float64   `json:"total_protein_g" db:"total_protein_g"`
	TotalCarbsG   float64   `json:"total_carbs_g" db:"total_carbs_g"`
	TotalFatG     float64   `json:"total_fat_g" db:"total_fat_g"`
	TotalFiberG   float64   `json:"total_fiber_g" db:"total_fiber_g"`
	ItemCount     int       `json:"item_count" db:"item_count"`
	HasPhotos     bool      `json:"has_photos" db:"has_photos"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// MealListResponse represents a paginated list of meals
type MealListResponse struct {
	Meals      []MealListItem `json:"meals"`
	Pagination Pagination     `json:"pagination"`
}

// EstimateMealRequest represents a request to estimate meal nutrition
type EstimateMealRequest struct {
	Description string `json:"description" validate:"required,max=200"`
}

// EstimateMealResponse represents the response from estimating a meal
type EstimateMealResponse struct {
	Calories   int    `json:"calories"`
	Protein    int    `json:"protein"`
	Confidence string `json:"confidence"` // "low", "medium", "high"
}

// ListMealFilters represents filters for listing meals
type ListMealFilters struct {
	Page     int        `json:"page"`
	Limit    int        `json:"limit"`
	Date     *time.Time `json:"date,omitempty"`
	MealType *MealType  `json:"meal_type,omitempty"`
}

// MealSuggestion represents a frequently eaten meal at similar times
type MealSuggestion struct {
	Description     string  `json:"description"`
	AvgCalories     float64 `json:"avg_calories"`
	AvgProtein      float64 `json:"avg_protein"`
	Frequency       int     `json:"frequency"`
	LastConsumedAt  time.Time `json:"last_consumed_at"`
}

// MealSuggestionsResponse represents suggested meals based on time
type MealSuggestionsResponse struct {
	Suggestions []MealSuggestion `json:"suggestions"`
	QueryTime   string           `json:"query_time"`
	TimeWindow  string           `json:"time_window"`
}
