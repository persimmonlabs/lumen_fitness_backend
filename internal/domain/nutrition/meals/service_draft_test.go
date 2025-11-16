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
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateMealWithItems(ctx context.Context, userID uuid.UUID, meal *Meal, items []MealItem) (*MealWithItems, error) {
	args := m.Called(ctx, userID, meal, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MealWithItems), args.Error(1)
}

func (m *MockRepository) CreateDraftMeal(ctx context.Context, userID uuid.UUID, meal *Meal) (uuid.UUID, error) {
	args := m.Called(ctx, userID, meal)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockRepository) UpdateDraftStatus(ctx context.Context, draftID uuid.UUID, status DraftStatus, items []MealItem, errMsg *string) error {
	args := m.Called(ctx, draftID, status, items, errMsg)
	return args.Error(0)
}

func (m *MockRepository) GetDraftStatus(ctx context.Context, userID, draftID uuid.UUID) (*Meal, []MealItem, error) {
	args := m.Called(ctx, userID, draftID)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*Meal), args.Get(1).([]MealItem), args.Error(2)
}

func (m *MockRepository) GetMealByID(ctx context.Context, userID, mealID uuid.UUID) (*MealWithItems, error) {
	args := m.Called(ctx, userID, mealID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MealWithItems), args.Error(1)
}

func (m *MockRepository) ListMealsByUser(ctx context.Context, userID uuid.UUID, filters ListMealFilters) ([]MealListItem, int, error) {
	args := m.Called(ctx, userID, filters)
	return args.Get(0).([]MealListItem), args.Int(1), args.Error(2)
}

func (m *MockRepository) UpdateMeal(ctx context.Context, userID uuid.UUID, meal *Meal, items []MealItem) (*MealWithItems, error) {
	args := m.Called(ctx, userID, meal, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MealWithItems), args.Error(1)
}

func (m *MockRepository) DeleteMeal(ctx context.Context, userID, mealID uuid.UUID) error {
	args := m.Called(ctx, userID, mealID)
	return args.Error(0)
}

func (m *MockRepository) DuplicateMeal(ctx context.Context, userID, mealID uuid.UUID, consumedAt time.Time, mealType MealType) (*MealWithItems, error) {
	args := m.Called(ctx, userID, mealID, consumedAt, mealType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MealWithItems), args.Error(1)
}

// MockAICoordinator is a mock implementation of AICoordinator
type MockAICoordinator struct {
	mock.Mock
}

func (m *MockAICoordinator) ParseMeal(ctx context.Context, description string, photos []string) ([]DraftMealItem, float64, float64, error) {
	args := m.Called(ctx, description, photos)
	if args.Get(0) == nil {
		return nil, 0, 0, args.Error(3)
	}
	return args.Get(0).([]DraftMealItem), args.Get(1).(float64), args.Get(2).(float64), args.Error(3)
}

// MockCache is a mock implementation of Cache
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(ctx context.Context, key string) (interface{}, error) {
	args := m.Called(ctx, key)
	return args.Get(0), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

// MockCostTracker is a mock implementation of CostTracker
type MockCostTracker struct {
	mock.Mock
}

func (m *MockCostTracker) TrackCost(ctx context.Context, userID uuid.UUID, cost float64) error {
	args := m.Called(ctx, userID, cost)
	return args.Error(0)
}

// MockPhotoStorage is a mock implementation of PhotoStorage
type MockPhotoStorage struct {
	mock.Mock
}

func (m *MockPhotoStorage) ValidatePhotos(ctx context.Context, photoIDs []string) error {
	args := m.Called(ctx, photoIDs)
	return args.Error(0)
}

func setupTestService() (*service, *MockRepository, *MockAICoordinator, *MockCache) {
	repo := new(MockRepository)
	aiCoordinator := new(MockAICoordinator)
	cache := new(MockCache)
	costTracker := new(MockCostTracker)
	photoStorage := new(MockPhotoStorage)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc := &service{
		repo:          repo,
		aiCoordinator: aiCoordinator,
		cache:         cache,
		costTracker:   costTracker,
		photoStorage:  photoStorage,
		logger:        logger,
	}

	return svc, repo, aiCoordinator, cache
}

func TestCreateDraftMeal(t *testing.T) {
	svc, repo, _, _ := setupTestService()
	ctx := context.Background()
	userID := uuid.New()
	draftID := uuid.New()

	req := &ParseMealRequest{
		Description:    "2 eggs and toast",
		MealType:       MealTypeBreakfast,
		ConsumedAt:     time.Now().UTC(),
		Photos:         []string{},
		IdempotencyKey: "test-key-123",
	}

	// Mock repository to return draft ID
	repo.On("CreateDraftMeal", ctx, userID, mock.AnythingOfType("*meals.Meal")).Return(draftID, nil)

	// Mock photo validation
	photoStorage := svc.photoStorage.(*MockPhotoStorage)
	photoStorage.On("ValidatePhotos", ctx, req.Photos).Return(nil)

	result, err := svc.ParseMealAsync(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, draftID, result.DraftID)
	assert.Equal(t, DraftStatusAnalyzing, result.Status)
	assert.Nil(t, result.DraftMeal)

	repo.AssertExpectations(t)
}

func TestCreateDraftMeal_FutureDate(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()
	userID := uuid.New()

	req := &ParseMealRequest{
		Description:    "future meal",
		MealType:       MealTypeBreakfast,
		ConsumedAt:     time.Now().UTC().Add(24 * time.Hour), // Tomorrow
		Photos:         []string{},
		IdempotencyKey: "test-key-123",
	}

	result, err := svc.ParseMealAsync(ctx, userID, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cannot log meals in the future")
}

func TestCreateDraftMeal_InvalidPhotos(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()
	userID := uuid.New()

	req := &ParseMealRequest{
		Description:    "meal with photos",
		MealType:       MealTypeBreakfast,
		ConsumedAt:     time.Now().UTC(),
		Photos:         []string{"photo1", "photo2"},
		IdempotencyKey: "test-key-123",
	}

	// Mock photo validation to fail
	photoStorage := svc.photoStorage.(*MockPhotoStorage)
	photoStorage.On("ValidatePhotos", ctx, req.Photos).Return(errors.New("invalid photo"))

	result, err := svc.ParseMealAsync(ctx, userID, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid photos")
}

func TestGetDraftStatus_Analyzing(t *testing.T) {
	svc, repo, _, _ := setupTestService()
	ctx := context.Background()
	userID := uuid.New()
	draftID := uuid.New()

	status := DraftStatusAnalyzing
	meal := &Meal{
		ID:          draftID,
		UserID:      userID,
		IsDraft:     true,
		DraftStatus: &status,
	}

	repo.On("GetDraftStatus", ctx, userID, draftID).Return(meal, []MealItem{}, nil)

	result, err := svc.GetDraftStatus(ctx, userID, draftID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, draftID, result.DraftID)
	assert.Equal(t, DraftStatusAnalyzing, result.Status)
	assert.Nil(t, result.Items)
	assert.Nil(t, result.Total)
	assert.Nil(t, result.Error)

	repo.AssertExpectations(t)
}

func TestGetDraftStatus_Ready(t *testing.T) {
	svc, repo, _, _ := setupTestService()
	ctx := context.Background()
	userID := uuid.New()
	draftID := uuid.New()

	status := DraftStatusReady
	meal := &Meal{
		ID:          draftID,
		UserID:      userID,
		IsDraft:     true,
		DraftStatus: &status,
	}

	items := []MealItem{
		{
			ID:       uuid.New(),
			MealID:   draftID,
			Name:     "Eggs",
			Quantity: 2,
			Unit:     "whole",
			Calories: 140,
			ProteinG: 12,
			CarbsG:   2,
			FatG:     10,
			FiberG:   0,
		},
		{
			ID:       uuid.New(),
			MealID:   draftID,
			Name:     "Toast",
			Quantity: 2,
			Unit:     "slice",
			Calories: 160,
			ProteinG: 6,
			CarbsG:   30,
			FatG:     2,
			FiberG:   2,
		},
	}

	repo.On("GetDraftStatus", ctx, userID, draftID).Return(meal, items, nil)

	result, err := svc.GetDraftStatus(ctx, userID, draftID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, draftID, result.DraftID)
	assert.Equal(t, DraftStatusReady, result.Status)
	assert.Len(t, result.Items, 2)
	assert.NotNil(t, result.Total)
	assert.Equal(t, 300.0, result.Total.Calories)
	assert.Equal(t, 18.0, result.Total.ProteinG)
	assert.Equal(t, 32.0, result.Total.CarbsG)
	assert.Equal(t, 12.0, result.Total.FatG)
	assert.Equal(t, 2.0, result.Total.FiberG)
	assert.Nil(t, result.Error)

	repo.AssertExpectations(t)
}

func TestGetDraftStatus_Error(t *testing.T) {
	svc, repo, _, _ := setupTestService()
	ctx := context.Background()
	userID := uuid.New()
	draftID := uuid.New()

	status := DraftStatusError
	errMsg := "AI service timeout"
	meal := &Meal{
		ID:          draftID,
		UserID:      userID,
		IsDraft:     true,
		DraftStatus: &status,
		DraftError:  &errMsg,
	}

	repo.On("GetDraftStatus", ctx, userID, draftID).Return(meal, []MealItem{}, nil)

	result, err := svc.GetDraftStatus(ctx, userID, draftID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, draftID, result.DraftID)
	assert.Equal(t, DraftStatusError, result.Status)
	assert.NotNil(t, result.Error)
	assert.Equal(t, "AI service timeout", *result.Error)

	repo.AssertExpectations(t)
}

func TestGetDraftStatus_NotFound(t *testing.T) {
	svc, repo, _, _ := setupTestService()
	ctx := context.Background()
	userID := uuid.New()
	draftID := uuid.New()

	repo.On("GetDraftStatus", ctx, userID, draftID).Return(nil, nil, errors.New("draft meal not found"))

	result, err := svc.GetDraftStatus(ctx, userID, draftID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get draft status")

	repo.AssertExpectations(t)
}

func TestUpdateDraftWithResults(t *testing.T) {
	svc, repo, _, _ := setupTestService()
	ctx := context.Background()
	draftID := uuid.New()

	items := []DraftMealItem{
		{
			Name:     "Chicken Breast",
			Quantity: 200,
			Unit:     "g",
			Calories: 330,
			ProteinG: 62,
			CarbsG:   0,
			FatG:     7,
			FiberG:   0,
		},
	}

	repo.On("UpdateDraftStatus", ctx, draftID, DraftStatusReady, mock.AnythingOfType("[]meals.MealItem"), (*string)(nil)).Return(nil)

	svc.updateDraftWithResults(ctx, draftID, items, 0.95, 0.002)

	repo.AssertExpectations(t)
}

func TestBackgroundProcessing_Success(t *testing.T) {
	svc, repo, aiCoordinator, cache := setupTestService()
	userID := uuid.New()
	draftID := uuid.New()
	description := "grilled chicken and vegetables"
	photos := []string{}
	idempotencyKey := "test-key-456"

	items := []DraftMealItem{
		{
			Name:     "Grilled Chicken",
			Quantity: 200,
			Unit:     "g",
			Calories: 330,
			ProteinG: 62,
			CarbsG:   0,
			FatG:     7,
			FiberG:   0,
		},
	}

	// Mock cache miss
	cache.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("cache miss"))
	cache.On("Set", mock.Anything, mock.Anything, items, 24*time.Hour).Return(nil)

	// Mock AI coordinator
	aiCoordinator.On("ParseMeal", mock.Anything, description, photos).Return(items, 0.92, 0.003, nil)

	// Mock cost tracker
	costTracker := svc.costTracker.(*MockCostTracker)
	costTracker.On("TrackCost", mock.Anything, userID, 0.003).Return(nil)

	// Mock repository update
	repo.On("UpdateDraftStatus", mock.Anything, draftID, DraftStatusReady, mock.AnythingOfType("[]meals.MealItem"), (*string)(nil)).Return(nil)

	// Run background processing
	svc.processAIInBackground(userID, draftID, description, photos, idempotencyKey)

	// Give goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	repo.AssertExpectations(t)
	aiCoordinator.AssertExpectations(t)
}

func TestBackgroundProcessing_AIFailure(t *testing.T) {
	svc, repo, aiCoordinator, cache := setupTestService()
	userID := uuid.New()
	draftID := uuid.New()
	description := "unknown meal"
	photos := []string{}
	idempotencyKey := "test-key-789"

	// Mock cache miss
	cache.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("cache miss"))

	// Mock AI coordinator failure
	aiErr := errors.New("AI service unavailable")
	aiCoordinator.On("ParseMeal", mock.Anything, description, photos).Return(nil, 0.0, 0.0, aiErr)

	// Mock repository error update
	errMsg := "AI service unavailable"
	repo.On("UpdateDraftStatus", mock.Anything, draftID, DraftStatusError, []MealItem(nil), &errMsg).Return(nil)

	// Run background processing
	svc.processAIInBackground(userID, draftID, description, photos, idempotencyKey)

	// Give goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	repo.AssertExpectations(t)
	aiCoordinator.AssertExpectations(t)
}
