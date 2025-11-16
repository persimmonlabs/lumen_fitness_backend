package weight

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for weight entry data access
type Repository interface {
	// Create creates a new weight entry
	Create(ctx context.Context, entry *WeightEntry) error

	// GetByID retrieves a weight entry by ID
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*WeightEntry, error)

	// List retrieves weight entries with filters and pagination
	List(ctx context.Context, filter WeightListFilter) ([]WeightEntry, int, error)

	// GetLatest retrieves the most recent weight entry for a user
	GetLatest(ctx context.Context, userID uuid.UUID) (*WeightEntry, error)

	// Update updates an existing weight entry
	Update(ctx context.Context, entry *WeightEntry) error

	// Delete deletes a weight entry
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	// GetByDateRange retrieves entries within a date range for statistics
	GetByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]WeightEntry, error)

	// CheckDuplicateDate checks if an entry exists for a specific date
	CheckDuplicateDate(ctx context.Context, userID uuid.UUID, date time.Time, excludeID *uuid.UUID) (bool, error)
}
