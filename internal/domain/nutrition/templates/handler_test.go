package templates

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
	"github.com/stretchr/testify/mock"
)

// MockService is a mock implementation of Service
type MockService struct {
	mock.Mock
}

func (m *MockService) CreateTemplate(ctx context.Context, userID uuid.UUID, req CreateTemplateRequest) (*TemplateResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TemplateResponse), args.Error(1)
}

func (m *MockService) CreateFromMeal(ctx context.Context, userID uuid.UUID, req CreateTemplateFromMealRequest) (*TemplateResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TemplateResponse), args.Error(1)
}

func (m *MockService) UseTemplate(ctx context.Context, userID, templateID uuid.UUID, req UseTemplateRequest) (uuid.UUID, error) {
	args := m.Called(ctx, userID, templateID, req)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockService) GetTemplate(ctx context.Context, userID, templateID uuid.UUID) (*TemplateResponse, error) {
	args := m.Called(ctx, userID, templateID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TemplateResponse), args.Error(1)
}

func (m *MockService) ListTemplates(ctx context.Context, userID uuid.UUID, limit, offset int) (*TemplateListResponse, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TemplateListResponse), args.Error(1)
}

func (m *MockService) UpdateTemplate(ctx context.Context, userID, templateID uuid.UUID, req UpdateTemplateRequest) (*TemplateResponse, error) {
	args := m.Called(ctx, userID, templateID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TemplateResponse), args.Error(1)
}

func (m *MockService) DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error {
	args := m.Called(ctx, userID, templateID)
	return args.Error(0)
}

func setupHandlerTest() (*Handler, *MockService) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	return handler, mockService
}

func addUserIDToContext(req *http.Request, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(req.Context(), "user_id", userID)
	return req.WithContext(ctx)
}

func TestHandler_CreateTemplate(t *testing.T) {
	handler, mockService := setupHandlerTest()
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

	expectedResponse := &TemplateResponse{
		ID:            uuid.New(),
		Name:          "Breakfast Template",
		TotalCalories: 300.0,
		Items:         []TemplateItemResponse{},
	}

	mockService.On("CreateTemplate", mock.Anything, userID, req).Return(expectedResponse, nil)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
	httpReq = addUserIDToContext(httpReq, userID)
	w := httptest.NewRecorder()

	handler.CreateTemplate(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response TemplateResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "Breakfast Template", response.Name)
	mockService.AssertExpectations(t)
}

func TestHandler_CreateTemplate_InvalidRequest(t *testing.T) {
	handler, _ := setupHandlerTest()
	userID := uuid.New()

	httpReq := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader([]byte("invalid json")))
	httpReq = addUserIDToContext(httpReq, userID)
	w := httptest.NewRecorder()

	handler.CreateTemplate(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateTemplate_Unauthorized(t *testing.T) {
	handler, _ := setupHandlerTest()

	req := CreateTemplateRequest{Name: "Template"}
	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateTemplate(w, httpReq)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CreateFromMeal(t *testing.T) {
	handler, mockService := setupHandlerTest()
	userID := uuid.New()
	mealID := uuid.New()

	req := CreateTemplateFromMealRequest{
		MealID: mealID,
		Name:   "Lunch Template",
	}

	expectedResponse := &TemplateResponse{
		ID:   uuid.New(),
		Name: "Lunch Template",
	}

	mockService.On("CreateFromMeal", mock.Anything, userID, req).Return(expectedResponse, nil)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/v1/templates/from-meal", bytes.NewReader(body))
	httpReq = addUserIDToContext(httpReq, userID)
	w := httptest.NewRecorder()

	handler.CreateFromMeal(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UseTemplate(t *testing.T) {
	handler, mockService := setupHandlerTest()
	userID := uuid.New()
	templateID := uuid.New()
	mealID := uuid.New()

	req := UseTemplateRequest{
		MealType: "breakfast",
		MealTime: time.Now(),
	}

	mockService.On("UseTemplate", mock.Anything, userID, templateID, req).Return(mealID, nil)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/v1/templates/"+templateID.String()+"/use", bytes.NewReader(body))
	httpReq = addUserIDToContext(httpReq, userID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", templateID.String())
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.UseTemplate(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "meal created from template", response["message"])
	mockService.AssertExpectations(t)
}

func TestHandler_UseTemplate_InvalidTemplateID(t *testing.T) {
	handler, _ := setupHandlerTest()
	userID := uuid.New()

	req := UseTemplateRequest{MealType: "breakfast", MealTime: time.Now()}
	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/v1/templates/invalid-uuid/use", bytes.NewReader(body))
	httpReq = addUserIDToContext(httpReq, userID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid-uuid")
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.UseTemplate(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTemplate(t *testing.T) {
	handler, mockService := setupHandlerTest()
	userID := uuid.New()
	templateID := uuid.New()

	expectedResponse := &TemplateResponse{
		ID:   templateID,
		Name: "Test Template",
	}

	mockService.On("GetTemplate", mock.Anything, userID, templateID).Return(expectedResponse, nil)

	httpReq := httptest.NewRequest("GET", "/api/v1/templates/"+templateID.String(), nil)
	httpReq = addUserIDToContext(httpReq, userID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", templateID.String())
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.GetTemplate(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response TemplateResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "Test Template", response.Name)
	mockService.AssertExpectations(t)
}

func TestHandler_GetTemplate_NotFound(t *testing.T) {
	handler, mockService := setupHandlerTest()
	userID := uuid.New()
	templateID := uuid.New()

	mockService.On("GetTemplate", mock.Anything, userID, templateID).Return(nil, ErrTemplateNotFound)

	httpReq := httptest.NewRequest("GET", "/api/v1/templates/"+templateID.String(), nil)
	httpReq = addUserIDToContext(httpReq, userID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", templateID.String())
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.GetTemplate(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ListTemplates(t *testing.T) {
	handler, mockService := setupHandlerTest()
	userID := uuid.New()

	expectedResponse := &TemplateListResponse{
		Templates: []TemplateResponse{
			{ID: uuid.New(), Name: "Template 1"},
			{ID: uuid.New(), Name: "Template 2"},
		},
		Total: 2,
	}

	mockService.On("ListTemplates", mock.Anything, userID, 50, 0).Return(expectedResponse, nil)

	httpReq := httptest.NewRequest("GET", "/api/v1/templates", nil)
	httpReq = addUserIDToContext(httpReq, userID)
	w := httptest.NewRecorder()

	handler.ListTemplates(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response TemplateListResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response.Templates, 2)
	assert.Equal(t, 2, response.Total)
	mockService.AssertExpectations(t)
}

func TestHandler_ListTemplates_WithPagination(t *testing.T) {
	handler, mockService := setupHandlerTest()
	userID := uuid.New()

	expectedResponse := &TemplateListResponse{
		Templates: []TemplateResponse{},
		Total:     0,
	}

	mockService.On("ListTemplates", mock.Anything, userID, 10, 20).Return(expectedResponse, nil)

	httpReq := httptest.NewRequest("GET", "/api/v1/templates?limit=10&offset=20", nil)
	httpReq = addUserIDToContext(httpReq, userID)
	w := httptest.NewRecorder()

	handler.ListTemplates(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateTemplate(t *testing.T) {
	handler, mockService := setupHandlerTest()
	userID := uuid.New()
	templateID := uuid.New()

	newName := "Updated Template"
	req := UpdateTemplateRequest{
		Name: &newName,
	}

	expectedResponse := &TemplateResponse{
		ID:   templateID,
		Name: newName,
	}

	mockService.On("UpdateTemplate", mock.Anything, userID, templateID, req).Return(expectedResponse, nil)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("PUT", "/api/v1/templates/"+templateID.String(), bytes.NewReader(body))
	httpReq = addUserIDToContext(httpReq, userID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", templateID.String())
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.UpdateTemplate(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response TemplateResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, newName, response.Name)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateTemplate_InvalidRequest(t *testing.T) {
	handler, _ := setupHandlerTest()
	userID := uuid.New()
	templateID := uuid.New()

	httpReq := httptest.NewRequest("PUT", "/api/v1/templates/"+templateID.String(), bytes.NewReader([]byte("invalid json")))
	httpReq = addUserIDToContext(httpReq, userID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", templateID.String())
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.UpdateTemplate(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteTemplate(t *testing.T) {
	handler, mockService := setupHandlerTest()
	userID := uuid.New()
	templateID := uuid.New()

	mockService.On("DeleteTemplate", mock.Anything, userID, templateID).Return(nil)

	httpReq := httptest.NewRequest("DELETE", "/api/v1/templates/"+templateID.String(), nil)
	httpReq = addUserIDToContext(httpReq, userID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", templateID.String())
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.DeleteTemplate(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "template deleted", response["message"])
	mockService.AssertExpectations(t)
}

func TestHandler_DeleteTemplate_NotFound(t *testing.T) {
	handler, mockService := setupHandlerTest()
	userID := uuid.New()
	templateID := uuid.New()

	mockService.On("DeleteTemplate", mock.Anything, userID, templateID).Return(ErrTemplateNotFound)

	httpReq := httptest.NewRequest("DELETE", "/api/v1/templates/"+templateID.String(), nil)
	httpReq = addUserIDToContext(httpReq, userID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", templateID.String())
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.DeleteTemplate(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ErrorHandling(t *testing.T) {
	handler, mockService := setupHandlerTest()
	userID := uuid.New()
	templateID := uuid.New()

	tests := []struct {
		name           string
		setupMock      func()
		expectedStatus int
	}{
		{
			name: "template not found",
			setupMock: func() {
				mockService.On("GetTemplate", mock.Anything, userID, templateID).Return(nil, ErrTemplateNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "unauthorized",
			setupMock: func() {
				mockService.On("GetTemplate", mock.Anything, userID, templateID).Return(nil, ErrUnauthorized).Once()
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "validation error",
			setupMock: func() {
				mockService.On("GetTemplate", mock.Anything, userID, templateID).Return(nil, ErrTemplateNameRequired).Once()
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "internal error",
			setupMock: func() {
				mockService.On("GetTemplate", mock.Anything, userID, templateID).Return(nil, assert.AnError).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			httpReq := httptest.NewRequest("GET", "/api/v1/templates/"+templateID.String(), nil)
			httpReq = addUserIDToContext(httpReq, userID)
			rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", templateID.String())
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.GetTemplate(w, httpReq)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}

	mockService.AssertExpectations(t)
}
