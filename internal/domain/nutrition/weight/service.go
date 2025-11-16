package weight

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidWeight      = errors.New("weight must be between 1 and 500 kg")
	ErrFutureDate         = errors.New("measured_at cannot be in the future")
	ErrDuplicateEntry     = errors.New("weight entry already exists for this date")
	ErrUnauthorized       = errors.New("unauthorized access to weight entry")
)

// Service defines the business logic for weight tracking
type Service interface {
	// CreateEntry creates a new weight entry with validation
	CreateEntry(ctx context.Context, userID uuid.UUID, req CreateWeightRequest) (*WeightEntry, error)

	// GetEntry retrieves a weight entry by ID
	GetEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*WeightEntry, error)

	// ListEntries retrieves weight entries with filters
	ListEntries(ctx context.Context, filter WeightListFilter) (*WeightListResponse, error)

	// GetLatestEntry retrieves the most recent weight entry
	GetLatestEntry(ctx context.Context, userID uuid.UUID) (*WeightEntry, error)

	// UpdateEntry updates an existing weight entry
	UpdateEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, req UpdateWeightRequest) (*WeightEntry, error)

	// DeleteEntry deletes a weight entry
	DeleteEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) error

	// GetStats calculates weight statistics (moving averages, rate of change)
	GetStats(ctx context.Context, userID uuid.UUID) (*WeightStats, error)
}

type service struct {
	repo Repository
}

// NewService creates a new weight service
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) CreateEntry(ctx context.Context, userID uuid.UUID, req CreateWeightRequest) (*WeightEntry, error) {
	// Validate weight range
	if err := s.validateWeight(req.Weight); err != nil {
		return nil, err
	}

	// Validate measured_at is not in future
	if err := s.validateMeasuredAt(req.MeasuredAt); err != nil {
		return nil, err
	}

	// Check for duplicate entry on the same date
	exists, err := s.repo.CheckDuplicateDate(ctx, userID, req.MeasuredAt, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateEntry
	}

	// Create entry
	now := time.Now().UTC()
	entry := &WeightEntry{
		ID:         uuid.New(),
		UserID:     userID,
		Weight:     req.Weight,
		MeasuredAt: req.MeasuredAt.UTC(),
		Notes:      req.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		if errors.Is(err, ErrDuplicateDate) {
			return nil, ErrDuplicateEntry
		}
		return nil, err
	}

	return entry, nil
}

func (s *service) GetEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*WeightEntry, error) {
	entry, err := s.repo.GetByID(ctx, entryID, userID)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (s *service) ListEntries(ctx context.Context, filter WeightListFilter) (*WeightListResponse, error) {
	// Set default pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	entries, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PageSize)))

	return &WeightListResponse{
		Entries:    entries,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *service) GetLatestEntry(ctx context.Context, userID uuid.UUID) (*WeightEntry, error) {
	entry, err := s.repo.GetLatest(ctx, userID)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (s *service) UpdateEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, req UpdateWeightRequest) (*WeightEntry, error) {
	// Get existing entry
	entry, err := s.repo.GetByID(ctx, entryID, userID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Weight != nil {
		if err := s.validateWeight(*req.Weight); err != nil {
			return nil, err
		}
		entry.Weight = *req.Weight
	}

	if req.MeasuredAt != nil {
		if err := s.validateMeasuredAt(*req.MeasuredAt); err != nil {
			return nil, err
		}

		// Check for duplicate if date is changing
		if !entry.MeasuredAt.Equal(*req.MeasuredAt) {
			exists, err := s.repo.CheckDuplicateDate(ctx, userID, *req.MeasuredAt, &entryID)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ErrDuplicateEntry
			}
		}

		entry.MeasuredAt = req.MeasuredAt.UTC()
	}

	if req.Notes != nil {
		entry.Notes = req.Notes
	}

	entry.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, entry); err != nil {
		if errors.Is(err, ErrDuplicateDate) {
			return nil, ErrDuplicateEntry
		}
		return nil, err
	}

	return entry, nil
}

func (s *service) DeleteEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) error {
	return s.repo.Delete(ctx, entryID, userID)
}

func (s *service) GetStats(ctx context.Context, userID uuid.UUID) (*WeightStats, error) {
	// Get latest entry
	latest, err := s.repo.GetLatest(ctx, userID)
	if err != nil {
		return nil, err
	}

	stats := &WeightStats{
		LatestWeight: latest.Weight,
		LatestDate:   latest.MeasuredAt,
	}

	now := time.Now().UTC()

	// Calculate 7-day moving average
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	entries7, err := s.repo.GetByDateRange(ctx, userID, sevenDaysAgo, now)
	if err != nil {
		return nil, err
	}

	if len(entries7) > 0 {
		avg7 := s.calculateAverage(entries7)
		stats.Average7Day = &avg7
	}

	// Calculate 30-day moving average
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	entries30, err := s.repo.GetByDateRange(ctx, userID, thirtyDaysAgo, now)
	if err != nil {
		return nil, err
	}

	if len(entries30) > 0 {
		avg30 := s.calculateAverage(entries30)
		stats.Average30Day = &avg30
	}

	// Calculate rate of change (kg/week)
	if len(entries7) >= 2 {
		rateOfChange := s.calculateRateOfChange(entries7)
		stats.RateOfChange = &rateOfChange
	}

	return stats, nil
}

// Helper functions

func (s *service) validateWeight(weight float64) error {
	if weight < 1 || weight > 500 {
		return ErrInvalidWeight
	}
	return nil
}

func (s *service) validateMeasuredAt(measuredAt time.Time) error {
	now := time.Now().UTC()
	if measuredAt.After(now) {
		return ErrFutureDate
	}
	return nil
}

func (s *service) calculateAverage(entries []WeightEntry) float64 {
	if len(entries) == 0 {
		return 0
	}

	sum := 0.0
	for _, entry := range entries {
		sum += entry.Weight
	}

	return sum / float64(len(entries))
}

func (s *service) calculateRateOfChange(entries []WeightEntry) float64 {
	if len(entries) < 2 {
		return 0
	}

	// Get first and last entries (sorted by date ascending)
	first := entries[0]
	last := entries[len(entries)-1]

	// Calculate weight change
	weightChange := last.Weight - first.Weight

	// Calculate time difference in weeks
	timeDiff := last.MeasuredAt.Sub(first.MeasuredAt)
	weeks := timeDiff.Hours() / (24 * 7)

	if weeks == 0 {
		return 0
	}

	// Return kg per week
	return weightChange / weeks
}
