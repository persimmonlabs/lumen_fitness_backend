package weight

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, entry *WeightEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*WeightEntry, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WeightEntry), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context, filter WeightListFilter) ([]WeightEntry, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]WeightEntry), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetLatest(ctx context.Context, userID uuid.UUID) (*WeightEntry, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WeightEntry), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, entry *WeightEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockRepository) GetByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]WeightEntry, error) {
	args := m.Called(ctx, userID, startDate, endDate)
	return args.Get(0).([]WeightEntry), args.Error(1)
}

func (m *MockRepository) CheckDuplicateDate(ctx context.Context, userID uuid.UUID, date time.Time, excludeID *uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, date, excludeID)
	return args.Bool(0), args.Error(1)
}

func TestService_CreateEntry(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now().UTC()

	t.Run("successful creation", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		req := CreateWeightRequest{
			Weight:     75.5,
			MeasuredAt: now,
		}

		mockRepo.On("CheckDuplicateDate", ctx, userID, req.MeasuredAt, (*uuid.UUID)(nil)).Return(false, nil)
		mockRepo.On("Create", ctx, mock.AnythingOfType("*weight.WeightEntry")).Return(nil)

		entry, err := service.CreateEntry(ctx, userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, userID, entry.UserID)
		assert.Equal(t, 75.5, entry.Weight)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid weight - too low", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		req := CreateWeightRequest{
			Weight:     0.5,
			MeasuredAt: now,
		}

		entry, err := service.CreateEntry(ctx, userID, req)

		assert.Error(t, err)
		assert.Nil(t, entry)
		assert.Equal(t, ErrInvalidWeight, err)
	})

	t.Run("invalid weight - too high", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		req := CreateWeightRequest{
			Weight:     501,
			MeasuredAt: now,
		}

		entry, err := service.CreateEntry(ctx, userID, req)

		assert.Error(t, err)
		assert.Nil(t, entry)
		assert.Equal(t, ErrInvalidWeight, err)
	})

	t.Run("future date", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		futureDate := now.Add(24 * time.Hour)
		req := CreateWeightRequest{
			Weight:     75.5,
			MeasuredAt: futureDate,
		}

		entry, err := service.CreateEntry(ctx, userID, req)

		assert.Error(t, err)
		assert.Nil(t, entry)
		assert.Equal(t, ErrFutureDate, err)
	})

	t.Run("duplicate date", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		req := CreateWeightRequest{
			Weight:     75.5,
			MeasuredAt: now,
		}

		mockRepo.On("CheckDuplicateDate", ctx, userID, req.MeasuredAt, (*uuid.UUID)(nil)).Return(true, nil)

		entry, err := service.CreateEntry(ctx, userID, req)

		assert.Error(t, err)
		assert.Nil(t, entry)
		assert.Equal(t, ErrDuplicateEntry, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetEntry(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	entryID := uuid.New()
	now := time.Now().UTC()

	t.Run("successful retrieval", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		expectedEntry := &WeightEntry{
			ID:         entryID,
			UserID:     userID,
			Weight:     75.5,
			MeasuredAt: now,
		}

		mockRepo.On("GetByID", ctx, entryID, userID).Return(expectedEntry, nil)

		entry, err := service.GetEntry(ctx, userID, entryID)

		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, entryID, entry.ID)
		assert.Equal(t, 75.5, entry.Weight)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		mockRepo.On("GetByID", ctx, entryID, userID).Return(nil, ErrNotFound)

		entry, err := service.GetEntry(ctx, userID, entryID)

		assert.Error(t, err)
		assert.Nil(t, entry)
		assert.Equal(t, ErrNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_ListEntries(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now().UTC()

	t.Run("successful list with pagination", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		filter := WeightListFilter{
			UserID:   userID,
			Page:     1,
			PageSize: 20,
		}

		entries := []WeightEntry{
			{ID: uuid.New(), UserID: userID, Weight: 75.5, MeasuredAt: now},
			{ID: uuid.New(), UserID: userID, Weight: 76.0, MeasuredAt: now.Add(-24 * time.Hour)},
		}

		mockRepo.On("List", ctx, filter).Return(entries, 2, nil)

		response, err := service.ListEntries(ctx, filter)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 2, response.Total)
		assert.Len(t, response.Entries, 2)
		assert.Equal(t, 1, response.Page)
		assert.Equal(t, 20, response.PageSize)
		assert.Equal(t, 1, response.TotalPages)
		mockRepo.AssertExpectations(t)
	})

	t.Run("default pagination values", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		filter := WeightListFilter{
			UserID: userID,
			// No page/pageSize set
		}

		mockRepo.On("List", ctx, mock.MatchedBy(func(f WeightListFilter) bool {
			return f.Page == 1 && f.PageSize == 20
		})).Return([]WeightEntry{}, 0, nil)

		response, err := service.ListEntries(ctx, filter)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		mockRepo.AssertExpectations(t)
	})

	t.Run("cap page size at 100", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		filter := WeightListFilter{
			UserID:   userID,
			Page:     1,
			PageSize: 200, // Over limit
		}

		mockRepo.On("List", ctx, mock.MatchedBy(func(f WeightListFilter) bool {
			return f.PageSize == 20 // Should be capped
		})).Return([]WeightEntry{}, 0, nil)

		response, err := service.ListEntries(ctx, filter)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_UpdateEntry(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	entryID := uuid.New()
	now := time.Now().UTC()

	t.Run("successful update", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		existingEntry := &WeightEntry{
			ID:         entryID,
			UserID:     userID,
			Weight:     75.5,
			MeasuredAt: now,
		}

		newWeight := 76.0
		req := UpdateWeightRequest{
			Weight: &newWeight,
		}

		mockRepo.On("GetByID", ctx, entryID, userID).Return(existingEntry, nil)
		mockRepo.On("Update", ctx, mock.MatchedBy(func(e *WeightEntry) bool {
			return e.Weight == 76.0
		})).Return(nil)

		entry, err := service.UpdateEntry(ctx, userID, entryID, req)

		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, 76.0, entry.Weight)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update with invalid weight", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		existingEntry := &WeightEntry{
			ID:         entryID,
			UserID:     userID,
			Weight:     75.5,
			MeasuredAt: now,
		}

		invalidWeight := 600.0
		req := UpdateWeightRequest{
			Weight: &invalidWeight,
		}

		mockRepo.On("GetByID", ctx, entryID, userID).Return(existingEntry, nil)

		entry, err := service.UpdateEntry(ctx, userID, entryID, req)

		assert.Error(t, err)
		assert.Nil(t, entry)
		assert.Equal(t, ErrInvalidWeight, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("entry not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		newWeight := 76.0
		req := UpdateWeightRequest{
			Weight: &newWeight,
		}

		mockRepo.On("GetByID", ctx, entryID, userID).Return(nil, ErrNotFound)

		entry, err := service.UpdateEntry(ctx, userID, entryID, req)

		assert.Error(t, err)
		assert.Nil(t, entry)
		assert.Equal(t, ErrNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_DeleteEntry(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	entryID := uuid.New()

	t.Run("successful deletion", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		mockRepo.On("Delete", ctx, entryID, userID).Return(nil)

		err := service.DeleteEntry(ctx, userID, entryID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		mockRepo.On("Delete", ctx, entryID, userID).Return(ErrNotFound)

		err := service.DeleteEntry(ctx, userID, entryID)

		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetStats(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now().UTC()

	t.Run("successful stats calculation", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		latestEntry := &WeightEntry{
			ID:         uuid.New(),
			UserID:     userID,
			Weight:     76.0,
			MeasuredAt: now,
		}

		entries7Days := []WeightEntry{
			{Weight: 75.0, MeasuredAt: now.Add(-6 * 24 * time.Hour)},
			{Weight: 75.5, MeasuredAt: now.Add(-3 * 24 * time.Hour)},
			{Weight: 76.0, MeasuredAt: now},
		}

		entries30Days := []WeightEntry{
			{Weight: 74.0, MeasuredAt: now.Add(-29 * 24 * time.Hour)},
			{Weight: 75.0, MeasuredAt: now.Add(-15 * 24 * time.Hour)},
			{Weight: 76.0, MeasuredAt: now},
		}

		mockRepo.On("GetLatest", ctx, userID).Return(latestEntry, nil)
		mockRepo.On("GetByDateRange", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(entries7Days, nil).Once()
		mockRepo.On("GetByDateRange", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(entries30Days, nil).Once()

		stats, err := service.GetStats(ctx, userID)

		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, 76.0, stats.LatestWeight)
		assert.NotNil(t, stats.Average7Day)
		assert.NotNil(t, stats.Average30Day)
		assert.NotNil(t, stats.RateOfChange)
		mockRepo.AssertExpectations(t)
	})

	t.Run("no entries found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		mockRepo.On("GetLatest", ctx, userID).Return(nil, ErrNotFound)

		stats, err := service.GetStats(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Equal(t, ErrNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_ValidateWeight(t *testing.T) {
	service := &service{}

	tests := []struct {
		name    string
		weight  float64
		wantErr bool
	}{
		{"valid minimum", 1.0, false},
		{"valid middle", 75.5, false},
		{"valid maximum", 500.0, false},
		{"invalid too low", 0.5, true},
		{"invalid zero", 0.0, true},
		{"invalid negative", -10.0, true},
		{"invalid too high", 501.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateWeight(tt.weight)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrInvalidWeight, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_ValidateMeasuredAt(t *testing.T) {
	service := &service{}
	now := time.Now().UTC()

	tests := []struct {
		name        string
		measuredAt  time.Time
		wantErr     bool
	}{
		{"valid past date", now.Add(-24 * time.Hour), false},
		{"valid current time", now, false},
		{"invalid future date", now.Add(24 * time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateMeasuredAt(tt.measuredAt)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrFutureDate, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_CalculateAverage(t *testing.T) {
	service := &service{}

	tests := []struct {
		name     string
		entries  []WeightEntry
		expected float64
	}{
		{
			name:     "empty entries",
			entries:  []WeightEntry{},
			expected: 0.0,
		},
		{
			name: "single entry",
			entries: []WeightEntry{
				{Weight: 75.5},
			},
			expected: 75.5,
		},
		{
			name: "multiple entries",
			entries: []WeightEntry{
				{Weight: 75.0},
				{Weight: 76.0},
				{Weight: 74.0},
			},
			expected: 75.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateAverage(tt.entries)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestService_CalculateRateOfChange(t *testing.T) {
	service := &service{}
	now := time.Now().UTC()

	tests := []struct {
		name     string
		entries  []WeightEntry
		expected float64
	}{
		{
			name:     "empty entries",
			entries:  []WeightEntry{},
			expected: 0.0,
		},
		{
			name: "single entry",
			entries: []WeightEntry{
				{Weight: 75.0, MeasuredAt: now},
			},
			expected: 0.0,
		},
		{
			name: "weight gain over 1 week",
			entries: []WeightEntry{
				{Weight: 75.0, MeasuredAt: now.Add(-7 * 24 * time.Hour)},
				{Weight: 76.0, MeasuredAt: now},
			},
			expected: 1.0, // 1 kg per week
		},
		{
			name: "weight loss over 2 weeks",
			entries: []WeightEntry{
				{Weight: 76.0, MeasuredAt: now.Add(-14 * 24 * time.Hour)},
				{Weight: 74.0, MeasuredAt: now},
			},
			expected: -1.0, // -1 kg per week
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateRateOfChange(tt.entries)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}
