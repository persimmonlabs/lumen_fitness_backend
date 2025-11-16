package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/goals"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/weight"
)

// mockWeightRepository implements weight.Repository for testing
type mockWeightRepository struct {
	entries []weight.WeightEntry
	err     error
}

func (m *mockWeightRepository) GetByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]weight.WeightEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.entries, nil
}

func (m *mockWeightRepository) Create(ctx context.Context, entry *weight.WeightEntry) error {
	return nil
}

func (m *mockWeightRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*weight.WeightEntry, error) {
	return nil, nil
}

func (m *mockWeightRepository) List(ctx context.Context, filter weight.WeightListFilter) ([]weight.WeightEntry, int, error) {
	return nil, 0, nil
}

func (m *mockWeightRepository) GetLatest(ctx context.Context, userID uuid.UUID) (*weight.WeightEntry, error) {
	return nil, nil
}

func (m *mockWeightRepository) Update(ctx context.Context, entry *weight.WeightEntry) error {
	return nil
}

func (m *mockWeightRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return nil
}

func (m *mockWeightRepository) CheckDuplicateDate(ctx context.Context, userID uuid.UUID, date time.Time, excludeID *uuid.UUID) (bool, error) {
	return false, nil
}

// mockGoalsRepositoryTrajectory implements a minimal goals repository for trajectory testing
type mockGoalsRepositoryTrajectory struct {
	mockGoalsRepositoryAnalytics
	targetWeight float64
	targetDate   time.Time
	weeklyGoal   float64
	err          error
}

func (m *mockGoalsRepositoryTrajectory) GetByUserID(ctx context.Context, userID uuid.UUID) (*goals.UserGoals, error) {
	if m.err != nil {
		return nil, m.err
	}
	// Return a UserGoals with the target weight info
	return &goals.UserGoals{
		ID:           0,
		UserID:       0, // int64 user ID not used in trajectory calculation
		TargetWeight: m.targetWeight,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

func TestCalculateTrajectory_Success(t *testing.T) {
	// Setup
	userID := uuid.New()
	now := time.Now()

	// Create weight entries showing a downward trend (weight loss)
	entries := make([]weight.WeightEntry, 30)
	for i := 0; i < 30; i++ {
		entries[i] = weight.WeightEntry{
			ID:         uuid.New(),
			UserID:     userID,
			Weight:     85.0 - float64(i)*0.3, // Losing 0.3 kg per day
			MeasuredAt: now.AddDate(0, 0, -30+i),
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	}

	// Setup repositories
	weightRepo := &mockWeightRepository{entries: entries}
	goalsRepo := &mockGoalsRepositoryTrajectory{
		targetWeight: 75.0,
		targetDate:   now.AddDate(0, 0, 30),
		weeklyGoal:   -0.5,
	}

	// Create service
	svc := &service{
		weightRepo: weightRepo,
		goalsRepo:  goalsRepo,
	}

	// Execute
	trajectory, err := svc.CalculateTrajectory(context.Background(), userID.String())

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if trajectory == nil {
		t.Fatal("expected trajectory, got nil")
	}

	if trajectory.DaysAnalyzed != 30 {
		t.Errorf("expected 30 days analyzed, got %d", trajectory.DaysAnalyzed)
	}

	if trajectory.TargetWeight != 75.0 {
		t.Errorf("expected target weight 75.0, got %f", trajectory.TargetWeight)
	}

	if trajectory.CurrentWeight < 70 || trajectory.CurrentWeight > 90 {
		t.Errorf("unexpected current weight: %f", trajectory.CurrentWeight)
	}

	if trajectory.WeeklyRate >= 0 {
		t.Errorf("expected negative weekly rate (weight loss), got %f", trajectory.WeeklyRate)
	}

	if trajectory.Confidence < 0 || trajectory.Confidence > 1 {
		t.Errorf("confidence should be between 0 and 1, got %f", trajectory.Confidence)
	}
}

func TestCalculateTrajectory_InsufficientData(t *testing.T) {
	// Setup
	userID := uuid.New()
	now := time.Now()

	// Only 5 entries (need at least 7)
	entries := make([]weight.WeightEntry, 5)
	for i := 0; i < 5; i++ {
		entries[i] = weight.WeightEntry{
			ID:         uuid.New(),
			UserID:     userID,
			Weight:     85.0,
			MeasuredAt: now.AddDate(0, 0, -5+i),
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	}

	// Setup repositories
	weightRepo := &mockWeightRepository{entries: entries}
	goalsRepo := &mockGoalsRepositoryTrajectory{
		targetWeight: 75.0,
		targetDate:   now.AddDate(0, 0, 30),
		weeklyGoal:   0,
	}

	// Create service
	svc := &service{
		weightRepo: weightRepo,
		goalsRepo:  goalsRepo,
	}

	// Execute
	_, err := svc.CalculateTrajectory(context.Background(), userID.String())

	// Verify
	if err == nil {
		t.Fatal("expected error for insufficient data, got nil")
	}

	expectedError := "insufficient data: need at least 7 weight entries"
	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}

func TestApplyMovingAverage(t *testing.T) {
	tests := []struct {
		name     string
		data     []float64
		window   int
		expected []float64
	}{
		{
			name:     "simple average",
			data:     []float64{1, 2, 3, 4, 5},
			window:   3,
			expected: []float64{1, 1.5, 2, 3, 4},
		},
		{
			name:     "window larger than data",
			data:     []float64{1, 2, 3},
			window:   5,
			expected: []float64{1, 1.5, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyMovingAverage(tt.data, tt.window)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected length %d, got %d", len(tt.expected), len(result))
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("at index %d: expected %f, got %f", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestCalculateConfidence(t *testing.T) {
	tests := []struct {
		name       string
		dataPoints int
		original   []float64
		smoothed   []float64
		wantMin    float64
		wantMax    float64
	}{
		{
			name:       "high confidence with many points",
			dataPoints: 60,
			original:   []float64{85, 84.5, 84, 83.5, 83},
			smoothed:   []float64{85, 84.7, 84.3, 83.9, 83.5},
			wantMin:    0.7,
			wantMax:    1.0,
		},
		{
			name:       "low confidence with few points",
			dataPoints: 10,
			original:   []float64{85, 84, 86, 83, 85},
			smoothed:   []float64{85, 84.5, 85, 84.5, 84.5},
			wantMin:    0.0,
			wantMax:    0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := calculateConfidence(tt.dataPoints, tt.original, tt.smoothed)

			if confidence < tt.wantMin || confidence > tt.wantMax {
				t.Errorf("confidence %f not in expected range [%f, %f]", confidence, tt.wantMin, tt.wantMax)
			}
		})
	}
}
