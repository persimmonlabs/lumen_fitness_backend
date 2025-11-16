package templates

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateTemplate(ctx context.Context, userID uuid.UUID, req CreateTemplateRequest) (*Template, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Template), args.Error(1)
}

func (m *MockRepository) CreateFromMeal(ctx context.Context, userID uuid.UUID, req CreateTemplateFromMealRequest) (*Template, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Template), args.Error(1)
}

func (m *MockRepository) GetTemplateByID(ctx context.Context, userID, templateID uuid.UUID) (*Template, error) {
	args := m.Called(ctx, userID, templateID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Template), args.Error(1)
}

func (m *MockRepository) ListTemplates(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Template, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]Template), args.Int(1), args.Error(2)
}

func (m *MockRepository) UpdateTemplate(ctx context.Context, userID, templateID uuid.UUID, req UpdateTemplateRequest) (*Template, error) {
	args := m.Called(ctx, userID, templateID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Template), args.Error(1)
}

func (m *MockRepository) DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error {
	args := m.Called(ctx, userID, templateID)
	return args.Error(0)
}

func setupServiceTest(t *testing.T) (*service, *MockRepository, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	mockRepo := new(MockRepository)
	svc := &service{
		repo: mockRepo,
		db:   sqlxDB,
	}

	return svc, mockRepo, mock
}

func TestService_CreateTemplate(t *testing.T) {
	svc, mockRepo, _ := setupServiceTest(t)
	userID := uuid.New()
	foodID := uuid.New()

	req := CreateTemplateRequest{
		Name: "Breakfast Template",
		Items: []CreateTemplateItemRequest{
			{
				FoodID:      foodID,
				ServingSize: 1.0,
				ServingUnit: "cup",
			},
		},
	}

	expectedTemplate := &Template{
		ID:            uuid.New(),
		UserID:        userID,
		Name:          "Breakfast Template",
		TotalCalories: 300.0,
		Items: []TemplateItem{
			{
				FoodID:      foodID,
				FoodName:    "Oatmeal",
				ServingSize: 1.0,
				ServingUnit: "cup",
			},
		},
	}

	mockRepo.On("CreateTemplate", mock.Anything, userID, req).Return(expectedTemplate, nil)

	result, err := svc.CreateTemplate(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Breakfast Template", result.Name)
	mockRepo.AssertExpectations(t)
}

func TestService_CreateTemplate_ValidationErrors(t *testing.T) {
	svc, _, _ := setupServiceTest(t)
	userID := uuid.New()

	tests := []struct {
		name        string
		req         CreateTemplateRequest
		expectedErr error
	}{
		{
			name: "empty name",
			req: CreateTemplateRequest{
				Name: "",
				Items: []CreateTemplateItemRequest{
					{FoodID: uuid.New(), ServingSize: 1.0, ServingUnit: "cup"},
				},
			},
			expectedErr: ErrTemplateNameRequired,
		},
		{
			name: "name too long",
			req: CreateTemplateRequest{
				Name:  string(make([]byte, 101)),
				Items: []CreateTemplateItemRequest{{FoodID: uuid.New(), ServingSize: 1.0, ServingUnit: "cup"}},
			},
			expectedErr: ErrTemplateNameTooLong,
		},
		{
			name: "no items",
			req: CreateTemplateRequest{
				Name:  "Template",
				Items: []CreateTemplateItemRequest{},
			},
			expectedErr: ErrTemplateItemsRequired,
		},
		{
			name: "missing food_id",
			req: CreateTemplateRequest{
				Name: "Template",
				Items: []CreateTemplateItemRequest{
					{FoodID: uuid.Nil, ServingSize: 1.0, ServingUnit: "cup"},
				},
			},
			expectedErr: ErrFoodIDRequired,
		},
		{
			name: "invalid serving size",
			req: CreateTemplateRequest{
				Name: "Template",
				Items: []CreateTemplateItemRequest{
					{FoodID: uuid.New(), ServingSize: 0, ServingUnit: "cup"},
				},
			},
			expectedErr: ErrInvalidServingSize,
		},
		{
			name: "missing serving unit",
			req: CreateTemplateRequest{
				Name: "Template",
				Items: []CreateTemplateItemRequest{
					{FoodID: uuid.New(), ServingSize: 1.0, ServingUnit: ""},
				},
			},
			expectedErr: ErrServingUnitRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateTemplate(context.Background(), userID, tt.req)
			assert.ErrorContains(t, err, tt.expectedErr.Error())
		})
	}
}

func TestService_CreateFromMeal(t *testing.T) {
	svc, mockRepo, mock := setupServiceTest(t)
	userID := uuid.New()
	mealID := uuid.New()

	req := CreateTemplateFromMealRequest{
		MealID: mealID,
		Name:   "Lunch Template",
	}

	// Mock meal existence check
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(mealID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	expectedTemplate := &Template{
		ID:     uuid.New(),
		UserID: userID,
		Name:   "Lunch Template",
	}

	mockRepo.On("CreateFromMeal", mock.Anything, userID, req).Return(expectedTemplate, nil)

	result, err := svc.CreateFromMeal(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Lunch Template", result.Name)
	mockRepo.AssertExpectations(t)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestService_CreateFromMeal_MealNotFound(t *testing.T) {
	svc, _, mock := setupServiceTest(t)
	userID := uuid.New()
	mealID := uuid.New()

	req := CreateTemplateFromMealRequest{
		MealID: mealID,
		Name:   "Template",
	}

	// Mock meal existence check - meal doesn't exist
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(mealID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, err := svc.CreateFromMeal(context.Background(), userID, req)

	assert.ErrorIs(t, err, ErrMealNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestService_UseTemplate(t *testing.T) {
	svc, mockRepo, mock := setupServiceTest(t)
	userID := uuid.New()
	templateID := uuid.New()
	foodID := uuid.New()

	req := UseTemplateRequest{
		MealType: "breakfast",
		MealTime: time.Now(),
	}

	template := &Template{
		ID:            templateID,
		UserID:        userID,
		Name:          "Breakfast Template",
		TotalCalories: 300.0,
		Items: []TemplateItem{
			{
				ID:          uuid.New(),
				TemplateID:  templateID,
				FoodID:      foodID,
				FoodName:    "Oatmeal",
				ServingSize: 1.0,
				ServingUnit: "cup",
				Calories:    150.0,
				Protein:     5.0,
				Carbs:       27.0,
				Fat:         3.0,
			},
		},
	}

	mockRepo.On("GetTemplateByID", mock.Anything, userID, templateID).Return(template, nil)

	// Mock meal creation
	mock.ExpectExec(`INSERT INTO nutrition.meals`).
		WithArgs(sqlmock.AnyArg(), userID, req.MealType, req.MealTime, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock meal item creation
	mock.ExpectExec(`INSERT INTO nutrition.meal_items`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), foodID, "Oatmeal", 1.0, "cup", 150.0, 5.0, 27.0, 3.0).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mealID, err := svc.UseTemplate(context.Background(), userID, templateID, req)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, mealID)
	mockRepo.AssertExpectations(t)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestService_UseTemplate_ValidationErrors(t *testing.T) {
	svc, _, _ := setupServiceTest(t)
	userID := uuid.New()
	templateID := uuid.New()

	tests := []struct {
		name        string
		req         UseTemplateRequest
		expectedErr error
	}{
		{
			name: "invalid meal type",
			req: UseTemplateRequest{
				MealType: "invalid",
				MealTime: time.Now(),
			},
			expectedErr: ErrInvalidMealType,
		},
		{
			name: "future meal time",
			req: UseTemplateRequest{
				MealType: "breakfast",
				MealTime: time.Now().Add(48 * time.Hour),
			},
			expectedErr: ErrInvalidMealTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UseTemplate(context.Background(), userID, templateID, tt.req)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestService_GetTemplate(t *testing.T) {
	svc, mockRepo, _ := setupServiceTest(t)
	userID := uuid.New()
	templateID := uuid.New()

	template := &Template{
		ID:     templateID,
		UserID: userID,
		Name:   "Test Template",
	}

	mockRepo.On("GetTemplateByID", mock.Anything, userID, templateID).Return(template, nil)

	result, err := svc.GetTemplate(context.Background(), userID, templateID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Template", result.Name)
	mockRepo.AssertExpectations(t)
}

func TestService_ListTemplates(t *testing.T) {
	svc, mockRepo, _ := setupServiceTest(t)
	userID := uuid.New()

	templates := []Template{
		{ID: uuid.New(), UserID: userID, Name: "Template 1"},
		{ID: uuid.New(), UserID: userID, Name: "Template 2"},
	}

	mockRepo.On("ListTemplates", mock.Anything, userID, 50, 0).Return(templates, 2, nil)

	result, err := svc.ListTemplates(context.Background(), userID, 0, 0)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Templates, 2)
	assert.Equal(t, 2, result.Total)
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateTemplate(t *testing.T) {
	svc, mockRepo, _ := setupServiceTest(t)
	userID := uuid.New()
	templateID := uuid.New()

	newName := "Updated Template"
	req := UpdateTemplateRequest{
		Name: &newName,
	}

	updatedTemplate := &Template{
		ID:     templateID,
		UserID: userID,
		Name:   newName,
	}

	mockRepo.On("UpdateTemplate", mock.Anything, userID, templateID, req).Return(updatedTemplate, nil)

	result, err := svc.UpdateTemplate(context.Background(), userID, templateID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newName, result.Name)
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateTemplate_ValidationErrors(t *testing.T) {
	svc, _, _ := setupServiceTest(t)
	userID := uuid.New()
	templateID := uuid.New()

	tests := []struct {
		name        string
		req         UpdateTemplateRequest
		expectedErr error
	}{
		{
			name: "empty name",
			req: UpdateTemplateRequest{
				Name: stringPtr(""),
			},
			expectedErr: ErrTemplateNameRequired,
		},
		{
			name: "name too long",
			req: UpdateTemplateRequest{
				Name: stringPtr(string(make([]byte, 101))),
			},
			expectedErr: ErrTemplateNameTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UpdateTemplate(context.Background(), userID, templateID, tt.req)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestService_DeleteTemplate(t *testing.T) {
	svc, mockRepo, _ := setupServiceTest(t)
	userID := uuid.New()
	templateID := uuid.New()

	mockRepo.On("DeleteTemplate", mock.Anything, userID, templateID).Return(nil)

	err := svc.DeleteTemplate(context.Background(), userID, templateID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func stringPtr(s string) *string {
	return &s
}
