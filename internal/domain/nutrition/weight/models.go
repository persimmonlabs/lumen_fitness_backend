package weight

import (
	"time"

	"github.com/google/uuid"
)

// WeightEntry represents a single weight measurement
type WeightEntry struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Weight      float64    `json:"weight" db:"weight"`                   // in kilograms
	MeasuredAt  time.Time  `json:"measured_at" db:"measured_at"`         // UTC timestamp
	Notes       *string    `json:"notes,omitempty" db:"notes"`           // optional notes
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateWeightRequest represents request to create a new weight entry
type CreateWeightRequest struct {
	Weight     float64   `json:"weight" validate:"required,min=1,max=500"`
	MeasuredAt time.Time `json:"measured_at" validate:"required"`
	Notes      *string   `json:"notes,omitempty"`
}

// UpdateWeightRequest represents request to update an existing weight entry
type UpdateWeightRequest struct {
	Weight     *float64   `json:"weight,omitempty" validate:"omitempty,min=1,max=500"`
	MeasuredAt *time.Time `json:"measured_at,omitempty"`
	Notes      *string    `json:"notes,omitempty"`
}

// WeightListResponse represents paginated list of weight entries
type WeightListResponse struct {
	Entries    []WeightEntry `json:"entries"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// WeightStats represents statistical calculations for weight entries
type WeightStats struct {
	Average7Day   *float64 `json:"average_7day,omitempty"`   // 7-day moving average
	Average30Day  *float64 `json:"average_30day,omitempty"`  // 30-day moving average
	RateOfChange  *float64 `json:"rate_of_change,omitempty"` // kg per week
	LatestWeight  float64  `json:"latest_weight"`
	LatestDate    time.Time `json:"latest_date"`
}

// WeightListFilter represents filters for querying weight entries
type WeightListFilter struct {
	UserID    uuid.UUID
	StartDate *time.Time
	EndDate   *time.Time
	Page      int
	PageSize  int
}
