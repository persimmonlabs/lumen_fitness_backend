package weight

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

func (m *MockService) CreateEntry(ctx context.Context, userID uuid.UUID, req CreateWeightRequest) (*WeightEntry, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WeightEntry), args.Error(1)
}

func (m *MockService) GetEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*WeightEntry, error) {
	args := m.Called(ctx, userID, entryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WeightEntry), args.Error(1)
}

func (m *MockService) ListEntries(ctx context.Context, filter WeightListFilter) (*WeightListResponse, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WeightListResponse), args.Error(1)
}

func (m *MockService) GetLatestEntry(ctx context.Context, userID uuid.UUID) (*WeightEntry, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WeightEntry), args.Error(1)
}

func (m *MockService) UpdateEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, req UpdateWeightRequest) (*WeightEntry, error) {
	args := m.Called(ctx, userID, entryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WeightEntry), args.Error(1)
}

func (m *MockService) DeleteEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) error {
	args := m.Called(ctx, userID, entryID)
	return args.Error(0)
}

func (m *MockService) GetStats(ctx context.Context, userID uuid.UUID) (*WeightStats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WeightStats), args.Error(1)
}

func contextWithUserID(userID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestHandler_RecordWeight(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()

	t.Run("successful weight recording", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		req := CreateWeightRequest{
			Weight:     75.5,
			MeasuredAt: now,
		}

		expectedEntry := &WeightEntry{
			ID:         uuid.New(),
			UserID:     userID,
			Weight:     75.5,
			MeasuredAt: now,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		mockService.On("CreateEntry", mock.Anything, userID, req).Return(expectedEntry, nil)

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/weight", bytes.NewReader(body))
		httpReq = httpReq.WithContext(contextWithUserID(userID))
		w := httptest.NewRecorder()

		handler.RecordWeight(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		httpReq := httptest.NewRequest(http.MethodPost, "/weight", bytes.NewReader([]byte("invalid json")))
		httpReq = httpReq.WithContext(contextWithUserID(userID))
		w := httptest.NewRecorder()

		handler.RecordWeight(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unauthorized - no user ID", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		req := CreateWeightRequest{
			Weight:     75.5,
			MeasuredAt: now,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/weight", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.RecordWeight(w, httpReq)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("duplicate entry error", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		req := CreateWeightRequest{
			Weight:     75.5,
			MeasuredAt: now,
		}

		mockService.On("CreateEntry", mock.Anything, userID, req).Return(nil, ErrDuplicateEntry)

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/weight", bytes.NewReader(body))
		httpReq = httpReq.WithContext(contextWithUserID(userID))
		w := httptest.NewRecorder()

		handler.RecordWeight(w, httpReq)

		assert.Equal(t, http.StatusConflict, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetWeight(t *testing.T) {
	userID := uuid.New()
	weightID := uuid.New()
	now := time.Now().UTC()

	t.Run("successful retrieval", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		expectedEntry := &WeightEntry{
			ID:         weightID,
			UserID:     userID,
			Weight:     75.5,
			MeasuredAt: now,
		}

		mockService.On("GetEntry", mock.Anything, userID, weightID).Return(expectedEntry, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/weight/"+weightID.String(), nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", weightID.String())
		httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.GetWeight(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid weight ID", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		httpReq := httptest.NewRequest(http.MethodGet, "/weight/invalid", nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "invalid")
		httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.GetWeight(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		mockService.On("GetEntry", mock.Anything, userID, weightID).Return(nil, ErrNotFound)

		httpReq := httptest.NewRequest(http.MethodGet, "/weight/"+weightID.String(), nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", weightID.String())
		httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.GetWeight(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_ListWeights(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()

	t.Run("successful list", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		expectedResponse := &WeightListResponse{
			Entries: []WeightEntry{
				{ID: uuid.New(), UserID: userID, Weight: 75.5, MeasuredAt: now},
			},
			Total:      1,
			Page:       1,
			PageSize:   20,
			TotalPages: 1,
		}

		mockService.On("ListEntries", mock.Anything, mock.MatchedBy(func(f WeightListFilter) bool {
			return f.UserID == userID && f.Page == 1 && f.PageSize == 20
		})).Return(expectedResponse, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/weight", nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))
		w := httptest.NewRecorder()

		handler.ListWeights(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("with pagination parameters", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		expectedResponse := &WeightListResponse{
			Entries:    []WeightEntry{},
			Total:      0,
			Page:       2,
			PageSize:   10,
			TotalPages: 0,
		}

		mockService.On("ListEntries", mock.Anything, mock.MatchedBy(func(f WeightListFilter) bool {
			return f.Page == 2 && f.PageSize == 10
		})).Return(expectedResponse, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/weight?page=2&page_size=10", nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))
		w := httptest.NewRecorder()

		handler.ListWeights(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("with date filters", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		expectedResponse := &WeightListResponse{
			Entries:    []WeightEntry{},
			Total:      0,
			Page:       1,
			PageSize:   20,
			TotalPages: 0,
		}

		mockService.On("ListEntries", mock.Anything, mock.MatchedBy(func(f WeightListFilter) bool {
			return f.StartDate != nil && f.EndDate != nil
		})).Return(expectedResponse, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/weight?start_date=2024-01-01&end_date=2024-12-31", nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))
		w := httptest.NewRecorder()

		handler.ListWeights(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_UpdateWeight(t *testing.T) {
	userID := uuid.New()
	weightID := uuid.New()
	now := time.Now().UTC()

	t.Run("successful update", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		newWeight := 76.0
		req := UpdateWeightRequest{
			Weight: &newWeight,
		}

		expectedEntry := &WeightEntry{
			ID:         weightID,
			UserID:     userID,
			Weight:     76.0,
			MeasuredAt: now,
			UpdatedAt:  now,
		}

		mockService.On("UpdateEntry", mock.Anything, userID, weightID, req).Return(expectedEntry, nil)

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPut, "/weight/"+weightID.String(), bytes.NewReader(body))
		httpReq = httpReq.WithContext(contextWithUserID(userID))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", weightID.String())
		httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateWeight(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid weight ID", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		req := UpdateWeightRequest{Weight: new(float64)}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPut, "/weight/invalid", bytes.NewReader(body))
		httpReq = httpReq.WithContext(contextWithUserID(userID))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "invalid")
		httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateWeight(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		httpReq := httptest.NewRequest(http.MethodPut, "/weight/"+weightID.String(), bytes.NewReader([]byte("invalid")))
		httpReq = httpReq.WithContext(contextWithUserID(userID))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", weightID.String())
		httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateWeight(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_DeleteWeight(t *testing.T) {
	userID := uuid.New()
	weightID := uuid.New()

	t.Run("successful deletion", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		mockService.On("DeleteEntry", mock.Anything, userID, weightID).Return(nil)

		httpReq := httptest.NewRequest(http.MethodDelete, "/weight/"+weightID.String(), nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", weightID.String())
		httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.DeleteWeight(w, httpReq)

		assert.Equal(t, http.StatusNoContent, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid weight ID", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		httpReq := httptest.NewRequest(http.MethodDelete, "/weight/invalid", nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "invalid")
		httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.DeleteWeight(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		mockService.On("DeleteEntry", mock.Anything, userID, weightID).Return(ErrNotFound)

		httpReq := httptest.NewRequest(http.MethodDelete, "/weight/"+weightID.String(), nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", weightID.String())
		httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.DeleteWeight(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetTrend(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()

	t.Run("successful trend retrieval", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		avg7 := 75.5
		avg30 := 74.8
		rate := 0.5
		expectedStats := &WeightStats{
			LatestWeight: 76.0,
			LatestDate:   now,
			Average7Day:  &avg7,
			Average30Day: &avg30,
			RateOfChange: &rate,
		}

		mockService.On("GetStats", mock.Anything, userID).Return(expectedStats, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/weight/trend", nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))
		w := httptest.NewRecorder()

		handler.GetTrend(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("no data found", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		mockService.On("GetStats", mock.Anything, userID).Return(nil, ErrNotFound)

		httpReq := httptest.NewRequest(http.MethodGet, "/weight/trend", nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))
		w := httptest.NewRecorder()

		handler.GetTrend(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetLatest(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()

	t.Run("successful retrieval", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		expectedEntry := &WeightEntry{
			ID:         uuid.New(),
			UserID:     userID,
			Weight:     75.5,
			MeasuredAt: now,
		}

		mockService.On("GetLatestEntry", mock.Anything, userID).Return(expectedEntry, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/weight/latest", nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))
		w := httptest.NewRecorder()

		handler.GetLatest(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("no entries found", func(t *testing.T) {
		mockService := new(MockService)
		handler := NewHandler(mockService)

		mockService.On("GetLatestEntry", mock.Anything, userID).Return(nil, ErrNotFound)

		httpReq := httptest.NewRequest(http.MethodGet, "/weight/latest", nil)
		httpReq = httpReq.WithContext(contextWithUserID(userID))
		w := httptest.NewRecorder()

		handler.GetLatest(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandleServiceError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{"not found", ErrNotFound, http.StatusNotFound},
		{"invalid weight", ErrInvalidWeight, http.StatusBadRequest},
		{"future date", ErrFutureDate, http.StatusBadRequest},
		{"duplicate entry", ErrDuplicateEntry, http.StatusConflict},
		{"unauthorized", ErrUnauthorized, http.StatusForbidden},
		{"unknown error", assert.AnError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handleServiceError(w, tt.err)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
