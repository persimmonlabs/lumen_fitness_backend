package meals

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// Shared test logger
var testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

// AIError represents an AI service error for testing
type AIError struct {
	Message string
}

func (e *AIError) Error() string {
	return e.Message
}

// MockRepository is a mock implementation of Repository for testing
type MockRepository struct {
	CreateMealWithItemsFunc func(ctx context.Context, userID uuid.UUID, meal *Meal, items []MealItem) (*MealWithItems, error)
	GetMealByIDFunc         func(ctx context.Context, userID, mealID uuid.UUID) (*MealWithItems, error)
	ListMealsByUserFunc     func(ctx context.Context, userID uuid.UUID, filters ListMealFilters) ([]MealListItem, int, error)
	UpdateMealFunc          func(ctx context.Context, userID uuid.UUID, meal *Meal, items []MealItem) (*MealWithItems, error)
	DeleteMealFunc          func(ctx context.Context, userID, mealID uuid.UUID) error
	DuplicateMealFunc       func(ctx context.Context, userID, mealID uuid.UUID, consumedAt time.Time, mealType MealType) (*MealWithItems, error)
}

func (m *MockRepository) CreateMealWithItems(ctx context.Context, userID uuid.UUID, meal *Meal, items []MealItem) (*MealWithItems, error) {
	if m.CreateMealWithItemsFunc != nil {
		return m.CreateMealWithItemsFunc(ctx, userID, meal, items)
	}
	return nil, nil
}

func (m *MockRepository) GetMealByID(ctx context.Context, userID, mealID uuid.UUID) (*MealWithItems, error) {
	if m.GetMealByIDFunc != nil {
		return m.GetMealByIDFunc(ctx, userID, mealID)
	}
	return nil, nil
}

func (m *MockRepository) ListMealsByUser(ctx context.Context, userID uuid.UUID, filters ListMealFilters) ([]MealListItem, int, error) {
	if m.ListMealsByUserFunc != nil {
		return m.ListMealsByUserFunc(ctx, userID, filters)
	}
	return nil, 0, nil
}

func (m *MockRepository) UpdateMeal(ctx context.Context, userID uuid.UUID, meal *Meal, items []MealItem) (*MealWithItems, error) {
	if m.UpdateMealFunc != nil {
		return m.UpdateMealFunc(ctx, userID, meal, items)
	}
	return nil, nil
}

func (m *MockRepository) DeleteMeal(ctx context.Context, userID, mealID uuid.UUID) error {
	if m.DeleteMealFunc != nil {
		return m.DeleteMealFunc(ctx, userID, mealID)
	}
	return nil
}

func (m *MockRepository) DuplicateMeal(ctx context.Context, userID, mealID uuid.UUID, consumedAt time.Time, mealType MealType) (*MealWithItems, error) {
	if m.DuplicateMealFunc != nil {
		return m.DuplicateMealFunc(ctx, userID, mealID, consumedAt, mealType)
	}
	return nil, nil
}

func (m *MockRepository) GetMealSuggestions(ctx context.Context, userID uuid.UUID, mealType MealType) ([]MealSuggestion, error) {
	return nil, nil
}

func (m *MockRepository) CreateDraftMeal(ctx context.Context, userID uuid.UUID, meal *Meal) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (m *MockRepository) UpdateDraftStatus(ctx context.Context, draftID uuid.UUID, status DraftStatus, items []MealItem, errMsg *string) error {
	return nil
}

func (m *MockRepository) GetDraftStatus(ctx context.Context, userID, draftID uuid.UUID) (*Meal, []MealItem, error) {
	return nil, nil, nil
}

// MockAICoordinator implements AICoordinator for testing
type MockAICoordinator struct {
	mock.Mock
	ParseMealFunc func(ctx context.Context, description string, photos []string) ([]DraftMealItem, float64, float64, error)
}

func (m *MockAICoordinator) ParseMeal(ctx context.Context, description string, photos []string) ([]DraftMealItem, float64, float64, error) {
	if m.ParseMealFunc != nil {
		return m.ParseMealFunc(ctx, description, photos)
	}
	args := m.Called(ctx, description, photos)
	if args.Get(0) == nil {
		return nil, args.Get(1).(float64), args.Get(2).(float64), args.Error(3)
	}
	return args.Get(0).([]DraftMealItem), args.Get(1).(float64), args.Get(2).(float64), args.Error(3)
}

// MockCache implements a simple cache for testing
type MockCache struct {
	mock.Mock
	GetFunc func(ctx context.Context, key string) (interface{}, error)
	SetFunc func(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	items   map[string]interface{}
}

func (m *MockCache) Get(ctx context.Context, key string) (interface{}, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	if m.items != nil {
		if val, ok := m.items[key]; ok {
			return val, nil
		}
	}
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, key, value, ttl)
	}
	if m.items == nil {
		m.items = make(map[string]interface{})
	}
	m.items[key] = value
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

// MockCostTracker implements cost tracking for testing
type MockCostTracker struct {
	mock.Mock
	TrackCostFunc func(ctx context.Context, userID uuid.UUID, cost float64) error
}

func (m *MockCostTracker) TrackCost(ctx context.Context, userID uuid.UUID, cost float64) error {
	if m.TrackCostFunc != nil {
		return m.TrackCostFunc(ctx, userID, cost)
	}
	args := m.Called(ctx, userID, cost)
	return args.Error(0)
}

// MockPhotoStorage implements photo storage for testing
type MockPhotoStorage struct {
	mock.Mock
	ValidatePhotosFunc func(ctx context.Context, photoIDs []string) error
}

func (m *MockPhotoStorage) ValidatePhotos(ctx context.Context, photoIDs []string) error {
	if m.ValidatePhotosFunc != nil {
		return m.ValidatePhotosFunc(ctx, photoIDs)
	}
	args := m.Called(ctx, photoIDs)
	return args.Error(0)
}

// MockAIEstimator implements AIEstimator for testing
type MockAIEstimator struct {
	EstimateFunc func(description string) (*AIEstimation, error)
}

func (m *MockAIEstimator) EstimateMeal(description string) (*AIEstimation, error) {
	if m.EstimateFunc != nil {
		return m.EstimateFunc(description)
	}
	return &AIEstimation{
		Calories:   500,
		Protein:    30,
		Confidence: "medium",
	}, nil
}

// MockAITranscriber implements AITranscriber for testing
type MockAITranscriber struct {
	TranscribeFunc func(audioData []byte, contentType string) (string, error)
}

func (m *MockAITranscriber) TranscribeAudio(audioData []byte, contentType string) (string, error) {
	if m.TranscribeFunc != nil {
		return m.TranscribeFunc(audioData, contentType)
	}
	return "chicken breast with rice and vegetables", nil
}

// MockAINormalizer implements AINormalizer for testing
type MockAINormalizer struct {
	NormalizeFunc func(description string) (string, error)
}

func (m *MockAINormalizer) NormalizeMealDescription(description string) (string, error) {
	if m.NormalizeFunc != nil {
		return m.NormalizeFunc(description)
	}
	return "Normalized " + description, nil
}
