package meals

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockService is a mock implementation of Service for testing
type MockService struct {
	ParseMealFunc   func(ctx context.Context, userID uuid.UUID, req *ParseMealRequest) (*ParseMealResponse, error)
	ConfirmMealFunc func(ctx context.Context, userID uuid.UUID, req *ConfirmMealRequest) (*MealResponse, error)
	GetMealFunc     func(ctx context.Context, userID, mealID uuid.UUID) (*MealResponse, error)
	ListMealsFunc   func(ctx context.Context, userID uuid.UUID, filters ListMealFilters) (*MealListResponse, error)
	UpdateMealFunc  func(ctx context.Context, userID, mealID uuid.UUID, req *UpdateMealRequest) (*MealResponse, error)
	DeleteMealFunc  func(ctx context.Context, userID, mealID uuid.UUID) error
	CopyMealFunc    func(ctx context.Context, userID, mealID uuid.UUID, req *CopyMealRequest) (*MealResponse, error)
}

func (m *MockService) ParseMeal(ctx context.Context, userID uuid.UUID, req *ParseMealRequest) (*ParseMealResponse, error) {
	if m.ParseMealFunc != nil {
		return m.ParseMealFunc(ctx, userID, req)
	}
	return nil, nil
}

func (m *MockService) ConfirmMeal(ctx context.Context, userID uuid.UUID, req *ConfirmMealRequest) (*MealResponse, error) {
	if m.ConfirmMealFunc != nil {
		return m.ConfirmMealFunc(ctx, userID, req)
	}
	return nil, nil
}

func (m *MockService) GetMeal(ctx context.Context, userID, mealID uuid.UUID) (*MealResponse, error) {
	if m.GetMealFunc != nil {
		return m.GetMealFunc(ctx, userID, mealID)
	}
	return nil, nil
}

func (m *MockService) ListMeals(ctx context.Context, userID uuid.UUID, filters ListMealFilters) (*MealListResponse, error) {
	if m.ListMealsFunc != nil {
		return m.ListMealsFunc(ctx, userID, filters)
	}
	return nil, nil
}

func (m *MockService) UpdateMeal(ctx context.Context, userID, mealID uuid.UUID, req *UpdateMealRequest) (*MealResponse, error) {
	if m.UpdateMealFunc != nil {
		return m.UpdateMealFunc(ctx, userID, mealID, req)
	}
	return nil, nil
}

func (m *MockService) DeleteMeal(ctx context.Context, userID, mealID uuid.UUID) error {
	if m.DeleteMealFunc != nil {
		return m.DeleteMealFunc(ctx, userID, mealID)
	}
	return nil
}

func (m *MockService) CopyMeal(ctx context.Context, userID, mealID uuid.UUID, req *CopyMealRequest) (*MealResponse, error) {
	if m.CopyMealFunc != nil {
		return m.CopyMealFunc(ctx, userID, mealID, req)
	}
	return nil, nil
}

func (m *MockService) EstimateMeal(ctx context.Context, userID uuid.UUID, req *EstimateMealRequest) (*EstimateMealResponse, error) {
	return nil, nil
}

func (m *MockService) ParseVoice(ctx context.Context, userID uuid.UUID, audioData []byte, contentType string, mealType MealType, consumedAt time.Time, idempotencyKey string) (*ParseMealResponse, error) {
	return nil, nil
}

func (m *MockService) GetDraftStatus(ctx context.Context, userID, draftID uuid.UUID) (*DraftStatusResponse, error) {
	return nil, nil
}

func (m *MockService) GetMealSuggestions(ctx context.Context, userID uuid.UUID, mealType MealType) (*MealSuggestionsResponse, error) {
	return nil, nil
}

func TestParseMealHandler_Success(t *testing.T) {
	userID := uuid.New()
	mockService := &MockService{
		ParseMealFunc: func(ctx context.Context, uid uuid.UUID, req *ParseMealRequest) (*ParseMealResponse, error) {
			assert.Equal(t, userID, uid)
			resp := &ParseMealResponse{}
			resp.DraftMeal.Items = []DraftMealItem{
				{Name: "Eggs", Calories: 180},
			}
			resp.DraftMeal.Confidence = 0.92
			resp.DraftMeal.CostUSD = 0.0023
			return resp, nil
		},
	}

	handler := &Handler{service: mockService}

	reqBody := ParseMealRequest{
		Description:    "2 scrambled eggs",
		MealType:       MealTypeBreakfast,
		ConsumedAt:     time.Now().UTC(),
		IdempotencyKey: "test-key",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/meals/parse", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))
	w := httptest.NewRecorder()

	handler.ParseMeal(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ParseMealResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response.DraftMeal.Items, 1)
}

func TestConfirmMealHandler_Success(t *testing.T) {
	userID := uuid.New()
	mealID := uuid.New()

	mockService := &MockService{
		ConfirmMealFunc: func(ctx context.Context, uid uuid.UUID, req *ConfirmMealRequest) (*MealResponse, error) {
			assert.Equal(t, userID, uid)
			return &MealResponse{
				MealWithItems: MealWithItems{
					Meal: Meal{
						ID:       mealID,
						UserID:   userID,
						MealType: req.MealType,
					},
				},
			}, nil
		},
	}

	handler := &Handler{service: mockService}

	reqBody := ConfirmMealRequest{
		MealType:   MealTypeBreakfast,
		ConsumedAt: time.Now().UTC(),
		Items: []DraftMealItem{
			{Name: "Eggs", Quantity: 2, Calories: 180, ProteinG: 12.6, CarbsG: 1.2, FatG: 12.0},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/meals/confirm", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))
	w := httptest.NewRecorder()

	handler.ConfirmMeal(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestGetMealHandler_Success(t *testing.T) {
	userID := uuid.New()
	mealID := uuid.New()

	mockService := &MockService{
		GetMealFunc: func(ctx context.Context, uid, mid uuid.UUID) (*MealResponse, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, mealID, mid)
			return &MealResponse{
				MealWithItems: MealWithItems{
					Meal: Meal{
						ID:     mealID,
						UserID: userID,
					},
				},
			}, nil
		},
	}

	handler := &Handler{service: mockService}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meals/"+mealID.String(), nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", mealID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.GetMeal(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListMealsHandler_Success(t *testing.T) {
	userID := uuid.New()

	mockService := &MockService{
		ListMealsFunc: func(ctx context.Context, uid uuid.UUID, filters ListMealFilters) (*MealListResponse, error) {
			assert.Equal(t, userID, uid)
			return &MealListResponse{
				Meals: []MealListItem{
					{ID: uuid.New(), MealType: MealTypeBreakfast},
				},
				Pagination: Pagination{
					Page:  1,
					Limit: 20,
					Total: 1,
				},
			}, nil
		},
	}

	handler := &Handler{service: mockService}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meals", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))
	w := httptest.NewRecorder()

	handler.ListMeals(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateMealHandler_Success(t *testing.T) {
	userID := uuid.New()
	mealID := uuid.New()

	mockService := &MockService{
		UpdateMealFunc: func(ctx context.Context, uid, mid uuid.UUID, req *UpdateMealRequest) (*MealResponse, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, mealID, mid)
			return &MealResponse{}, nil
		},
	}

	handler := &Handler{service: mockService}

	reqBody := UpdateMealRequest{
		MealType:   MealTypeLunch,
		ConsumedAt: time.Now().UTC(),
		Items: []DraftMealItem{
			{Name: "Salad", Calories: 200, ProteinG: 10, CarbsG: 20, FatG: 5},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/meals/"+mealID.String(), bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", mealID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.UpdateMeal(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteMealHandler_Success(t *testing.T) {
	userID := uuid.New()
	mealID := uuid.New()

	mockService := &MockService{
		DeleteMealFunc: func(ctx context.Context, uid, mid uuid.UUID) error {
			assert.Equal(t, userID, uid)
			assert.Equal(t, mealID, mid)
			return nil
		},
	}

	handler := &Handler{service: mockService}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/meals/"+mealID.String(), nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", mealID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.DeleteMeal(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCopyMealHandler_Success(t *testing.T) {
	userID := uuid.New()
	mealID := uuid.New()

	mockService := &MockService{
		CopyMealFunc: func(ctx context.Context, uid, mid uuid.UUID, req *CopyMealRequest) (*MealResponse, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, mealID, mid)
			return &MealResponse{}, nil
		},
	}

	handler := &Handler{service: mockService}

	reqBody := CopyMealRequest{
		ConsumedAt: time.Now().UTC().Add(24 * time.Hour),
		MealType:   MealTypeBreakfast,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/meals/"+mealID.String()+"/copy", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", mealID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.CopyMeal(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}
