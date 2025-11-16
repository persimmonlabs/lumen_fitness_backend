package analytics_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/analytics"
)

// mockService implements analytics.Service for testing
type mockService struct {
	getDailyAnalyticsFunc    func(ctx context.Context, userID string, date time.Time, timezone string) (*analytics.DailyAnalyticsResponse, error)
	getWeeklyTrendsFunc      func(ctx context.Context, userID string, startDate time.Time, timezone string) (*analytics.WeeklyAnalyticsResponse, error)
	getDateRangeStatsFunc    func(ctx context.Context, userID string, startDate, endDate time.Time, timezone string) (*analytics.TrendsResponse, error)
	getMacroDistributionFunc func(ctx context.Context, userID string, date time.Time, timezone string) (*analytics.MacroDistribution, error)
}

func (m *mockService) GetDailyAnalytics(ctx context.Context, userID string, date time.Time, timezone string) (*analytics.DailyAnalyticsResponse, error) {
	if m.getDailyAnalyticsFunc != nil {
		return m.getDailyAnalyticsFunc(ctx, userID, date, timezone)
	}
	return &analytics.DailyAnalyticsResponse{}, nil
}

func (m *mockService) GetWeeklyTrends(ctx context.Context, userID string, startDate time.Time, timezone string) (*analytics.WeeklyAnalyticsResponse, error) {
	if m.getWeeklyTrendsFunc != nil {
		return m.getWeeklyTrendsFunc(ctx, userID, startDate, timezone)
	}
	return &analytics.WeeklyAnalyticsResponse{}, nil
}

func (m *mockService) GetDateRangeStats(ctx context.Context, userID string, startDate, endDate time.Time, timezone string) (*analytics.TrendsResponse, error) {
	if m.getDateRangeStatsFunc != nil {
		return m.getDateRangeStatsFunc(ctx, userID, startDate, endDate, timezone)
	}
	return &analytics.TrendsResponse{}, nil
}

func (m *mockService) GetMacroDistribution(ctx context.Context, userID string, date time.Time, timezone string) (*analytics.MacroDistribution, error) {
	if m.getMacroDistributionFunc != nil {
		return m.getMacroDistributionFunc(ctx, userID, date, timezone)
	}
	return &analytics.MacroDistribution{}, nil
}

func (m *mockService) GetNutritionInsights(ctx context.Context, userID string, startDate, endDate time.Time, timezone string) (*analytics.NutritionInsights, error) {
	return &analytics.NutritionInsights{}, nil
}

// contextWithUserID adds user ID to context for testing
func contextWithUserID(userID string) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestGetDailyAnalytics(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		queryParams    string
		mockResponse   *analytics.DailyAnalyticsResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:   "success - default parameters",
			userID: "user-123",
			mockResponse: &analytics.DailyAnalyticsResponse{
				Date: time.Now(),
				Totals: analytics.DailyTotals{
					TotalCalories: 2000,
					TotalProtein:  150,
					TotalCarbs:    200,
					TotalFat:      65,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with custom date and timezone",
			userID:         "user-123",
			queryParams:    "?date=2024-01-15&timezone=America/New_York",
			mockResponse:   &analytics.DailyAnalyticsResponse{},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - unauthorized",
			userID:         "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "error - invalid date format",
			userID:         "user-123",
			queryParams:    "?date=invalid-date",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockSvc := &mockService{
				getDailyAnalyticsFunc: func(ctx context.Context, userID string, date time.Time, timezone string) (*analytics.DailyAnalyticsResponse, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			// Create handler
			handler := analytics.NewHandler(mockSvc)

			// Create router
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/analytics/daily"+tt.queryParams, nil)
			if tt.userID != "" {
				req = req.WithContext(contextWithUserID(tt.userID))
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Serve request
			r.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response for success cases
			if tt.expectedStatus == http.StatusOK {
				var response analytics.SuccessResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if response.Status != "success" {
					t.Errorf("expected status 'success', got '%s'", response.Status)
				}
			}
		})
	}
}

func TestGetWeeklyAnalytics(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		queryParams    string
		mockResponse   *analytics.WeeklyAnalyticsResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:   "success - default parameters",
			userID: "user-123",
			mockResponse: &analytics.WeeklyAnalyticsResponse{
				StartDate: time.Now().AddDate(0, 0, -6),
				EndDate:   time.Now(),
				Trend: analytics.WeeklyTrends{
					DailyData: make([]analytics.DailyTotals, 7),
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with custom start date",
			userID:         "user-123",
			queryParams:    "?start_date=2024-01-01&timezone=America/New_York",
			mockResponse:   &analytics.WeeklyAnalyticsResponse{},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - unauthorized",
			userID:         "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "error - invalid start date",
			userID:         "user-123",
			queryParams:    "?start_date=invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockService{
				getWeeklyTrendsFunc: func(ctx context.Context, userID string, startDate time.Time, timezone string) (*analytics.WeeklyAnalyticsResponse, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			handler := analytics.NewHandler(mockSvc)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodGet, "/analytics/weekly"+tt.queryParams, nil)
			if tt.userID != "" {
				req = req.WithContext(contextWithUserID(tt.userID))
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetTrends(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		queryParams    string
		mockResponse   *analytics.TrendsResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:   "success - valid date range",
			userID: "user-123",
			queryParams: "?start_date=2024-01-01&end_date=2024-01-31&timezone=UTC",
			mockResponse: &analytics.TrendsResponse{
				StartDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:     time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
				DailyTotals: make([]analytics.DailyTotals, 31),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - missing start date",
			userID:         "user-123",
			queryParams:    "?end_date=2024-01-31",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "error - missing end date",
			userID:         "user-123",
			queryParams:    "?start_date=2024-01-01",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "error - invalid start date format",
			userID:         "user-123",
			queryParams:    "?start_date=invalid&end_date=2024-01-31",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "error - end date before start date",
			userID:         "user-123",
			queryParams:    "?start_date=2024-01-31&end_date=2024-01-01",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "error - unauthorized",
			userID:         "",
			queryParams:    "?start_date=2024-01-01&end_date=2024-01-31",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockService{
				getDateRangeStatsFunc: func(ctx context.Context, userID string, startDate, endDate time.Time, timezone string) (*analytics.TrendsResponse, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			handler := analytics.NewHandler(mockSvc)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodGet, "/analytics/trends"+tt.queryParams, nil)
			if tt.userID != "" {
				req = req.WithContext(contextWithUserID(tt.userID))
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetMacroDistribution(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		queryParams    string
		mockResponse   *analytics.MacroDistribution
		mockError      error
		expectedStatus int
	}{
		{
			name:   "success - default parameters",
			userID: "user-123",
			mockResponse: &analytics.MacroDistribution{
				Date:           time.Now(),
				ProteinPercent: 30,
				CarbsPercent:   40,
				FatPercent:     30,
				TotalCalories:  2000,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with custom date",
			userID:         "user-123",
			queryParams:    "?date=2024-01-15&timezone=America/New_York",
			mockResponse:   &analytics.MacroDistribution{},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - unauthorized",
			userID:         "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "error - invalid date",
			userID:         "user-123",
			queryParams:    "?date=invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockService{
				getMacroDistributionFunc: func(ctx context.Context, userID string, date time.Time, timezone string) (*analytics.MacroDistribution, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			handler := analytics.NewHandler(mockSvc)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodGet, "/analytics/distribution"+tt.queryParams, nil)
			if tt.userID != "" {
				req = req.WithContext(contextWithUserID(tt.userID))
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetGoalProgress(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		queryParams    string
		mockResponse   *analytics.DailyAnalyticsResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:   "success - default parameters",
			userID: "user-123",
			mockResponse: &analytics.DailyAnalyticsResponse{
				Date: time.Now(),
				Totals: analytics.DailyTotals{
					TotalCalories: 2000,
					TotalProtein:  150,
				},
				Progress: &analytics.GoalComparison{
					CaloriesGoal:    2000,
					CaloriesPercent: 100,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - unauthorized",
			userID:         "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockService{
				getDailyAnalyticsFunc: func(ctx context.Context, userID string, date time.Time, timezone string) (*analytics.DailyAnalyticsResponse, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			handler := analytics.NewHandler(mockSvc)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodGet, "/analytics/progress"+tt.queryParams, nil)
			if tt.userID != "" {
				req = req.WithContext(contextWithUserID(tt.userID))
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify response structure for success cases
			if tt.expectedStatus == http.StatusOK {
				var response analytics.SuccessResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				// Verify it's a ProgressResponse structure
				dataJSON, _ := json.Marshal(response.Data)
				var progressResp analytics.ProgressResponse
				if err := json.Unmarshal(dataJSON, &progressResp); err != nil {
					t.Fatalf("failed to unmarshal progress response: %v", err)
				}
			}
		})
	}
}

func TestRegisterRoutes(t *testing.T) {
	mockSvc := &mockService{}
	handler := analytics.NewHandler(mockSvc)
	r := chi.NewRouter()

	// Register routes
	handler.RegisterRoutes(r)

	// Test that routes are registered
	routes := []string{
		"/analytics/daily",
		"/analytics/weekly",
		"/analytics/trends",
		"/analytics/distribution",
		"/analytics/progress",
	}

	for _, route := range routes {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		req = req.WithContext(contextWithUserID("user-123"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Should not return 404 (route not found)
		if w.Code == http.StatusNotFound {
			t.Errorf("route %s not registered", route)
		}
	}
}
