package meals

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock services for testing
type MockAICoordinator struct {
	ParseMealFunc func(ctx context.Context, description string, photos []string) ([]DraftMealItem, float64, float64, error)
}

func (m *MockAICoordinator) ParseMeal(ctx context.Context, description string, photos []string) ([]DraftMealItem, float64, float64, error) {
	if m.ParseMealFunc != nil {
		return m.ParseMealFunc(ctx, description, photos)
	}
	return nil, 0, 0, nil
}

type MockCache struct {
	GetFunc func(ctx context.Context, key string) (interface{}, error)
	SetFunc func(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

func (m *MockCache) Get(ctx context.Context, key string) (interface{}, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	return nil, errors.New("not found")
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, key, value, ttl)
	}
	return nil
}

type MockCostTracker struct {
	TrackCostFunc func(ctx context.Context, userID uuid.UUID, cost float64) error
}

func (m *MockCostTracker) TrackCost(ctx context.Context, userID uuid.UUID, cost float64) error {
	if m.TrackCostFunc != nil {
		return m.TrackCostFunc(ctx, userID, cost)
	}
	return nil
}

type MockPhotoStorage struct {
	ValidatePhotosFunc func(ctx context.Context, photoIDs []string) error
}

func (m *MockPhotoStorage) ValidatePhotos(ctx context.Context, photoIDs []string) error {
	if m.ValidatePhotosFunc != nil {
		return m.ValidatePhotosFunc(ctx, photoIDs)
	}
	return nil
}

func TestParseMeal_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockAI := &MockAICoordinator{
		ParseMealFunc: func(ctx context.Context, description string, photos []string) ([]DraftMealItem, float64, float64, error) {
			return []DraftMealItem{
				{Name: "Eggs", Quantity: 2, Unit: "large", Calories: 180, ProteinG: 12.6},
			}, 0.92, 0.0023, nil
		},
	}

	mockCache := &MockCache{
		GetFunc: func(ctx context.Context, key string) (interface{}, error) {
			return nil, errors.New("not found")
		},
		SetFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
			return nil
		},
	}

	mockCost := &MockCostTracker{
		TrackCostFunc: func(ctx context.Context, uid uuid.UUID, cost float64) error {
			assert.Equal(t, userID, uid)
			assert.Equal(t, 0.0023, cost)
			return nil
		},
	}

	mockPhotos := &MockPhotoStorage{
		ValidatePhotosFunc: func(ctx context.Context, photoIDs []string) error {
			return nil
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	service := &service{
		aiCoordinator: mockAI,
		cache:         mockCache,
		costTracker:   mockCost,
		photoStorage:  mockPhotos,
		logger:        logger,
	}

	req := &ParseMealRequest{
		Description:    "2 scrambled eggs",
		MealType:       MealTypeBreakfast,
		ConsumedAt:     time.Now().UTC(),
		Photos:         []string{},
		IdempotencyKey: "test-key-123",
	}

	result, err := service.ParseMeal(ctx, userID, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.DraftMeal.Items, 1)
	assert.Equal(t, "Eggs", result.DraftMeal.Items[0].Name)
}

func TestParseMeal_FutureDate(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	service := &service{}

	req := &ParseMealRequest{
		Description:    "2 scrambled eggs",
		MealType:       MealTypeBreakfast,
		ConsumedAt:     time.Now().UTC().Add(24 * time.Hour),
		IdempotencyKey: "test-key-123",
	}

	_, err := service.ParseMeal(ctx, userID, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "future")
}

func TestConfirmMeal_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now().UTC()

	mockRepo := &MockRepository{
		CreateMealWithItemsFunc: func(ctx context.Context, uid uuid.UUID, meal *Meal, items []MealItem) (*MealWithItems, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, MealTypeBreakfast, meal.MealType)
			assert.Len(t, items, 1)
			return &MealWithItems{
				Meal:  *meal,
				Items: items,
			}, nil
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	service := &service{
		repo:   mockRepo,
		logger: logger,
	}

	req := &ConfirmMealRequest{
		MealType:   MealTypeBreakfast,
		ConsumedAt: now,
		Items: []DraftMealItem{
			// Calories should be: (12.6 * 4) + (1.2 * 4) + (12.0 * 9) = 163.2
			{Name: "Eggs", Quantity: 2, Calories: 163, ProteinG: 12.6, CarbsG: 1.2, FatG: 12.0},
		},
	}

	result, err := service.ConfirmMeal(ctx, userID, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
