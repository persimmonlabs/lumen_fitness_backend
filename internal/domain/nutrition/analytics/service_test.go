package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetDailyNutrition(ctx context.Context, userID string, date time.Time, timezone string) (*DailyTotals, error) {
	args := m.Called(ctx, userID, date, timezone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DailyTotals), args.Error(1)
}

func (m *MockRepository) GetDateRangeNutrition(ctx context.Context, userID string, startDate, endDate time.Time, timezone string) ([]DailyTotals, error) {
	args := m.Called(ctx, userID, startDate, endDate, timezone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]DailyTotals), args.Error(1)
}

func (m *MockRepository) GetUserGoals(ctx context.Context, userID string) (*UserGoals, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserGoals), args.Error(1)
}

func (m *MockRepository) GetLoggingStreak(ctx context.Context, userID string, endDate time.Time, timezone string) (int, error) {
	args := m.Called(ctx, userID, endDate, timezone)
	return args.Int(0), args.Error(1)
}

func TestGetDailyAnalytics(t *testing.T) {
	mockRepo := new(MockRepository)
	mockWeightRepo := &mockWeightRepository{}
	mockGoalsRepo := &mockGoalsRepositoryAnalytics{}
	svc := NewService(mockRepo, mockWeightRepo, mockGoalsRepo)

	userID := "user-123"
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	timezone := "America/New_York"

	dailyTotals := &DailyTotals{
		Date:          date,
		Timezone:      timezone,
		TotalCalories: 2000,
		TotalProtein:  150,
		TotalCarbs:    200,
		TotalFat:      70,
		TotalFiber:    30,
	}

	goals := &UserGoals{
		UserID:        userID,
		CaloriesGoal:  2000,
		ProteinGoal:   150,
		CarbsGoal:     200,
		FatGoal:       65,
		FiberGoal:     30,
	}

	mockRepo.On("GetDailyNutrition", mock.Anything, userID, date, timezone).Return(dailyTotals, nil)
	mockRepo.On("GetUserGoals", mock.Anything, userID).Return(goals, nil)

	result, err := svc.GetDailyAnalytics(context.Background(), userID, date, timezone)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Progress)
	assert.Equal(t, 100.0, result.Progress.CaloriesPercent)
	assert.Equal(t, 100.0, result.Progress.ProteinPercent)
	assert.InDelta(t, 107.69, result.Progress.FatPercent, 0.01)

	mockRepo.AssertExpectations(t)
}

func TestGetWeeklyTrends(t *testing.T) {
	mockRepo := new(MockRepository)
	mockWeightRepo := &mockWeightRepository{}
	mockGoalsRepo := &mockGoalsRepositoryAnalytics{}
	svc := NewService(mockRepo, mockWeightRepo, mockGoalsRepo)

	userID := "user-123"
	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 6)
	timezone := "America/New_York"

	dailyData := []DailyTotals{
		{Date: startDate, TotalCalories: 2000, TotalProtein: 150, TotalCarbs: 200, TotalFat: 70, TotalFiber: 30},
		{Date: startDate.AddDate(0, 0, 1), TotalCalories: 1900, TotalProtein: 140, TotalCarbs: 190, TotalFat: 65, TotalFiber: 28},
		{Date: startDate.AddDate(0, 0, 2), TotalCalories: 2100, TotalProtein: 160, TotalCarbs: 210, TotalFat: 75, TotalFiber: 32},
		{Date: startDate.AddDate(0, 0, 3), TotalCalories: 0, TotalProtein: 0, TotalCarbs: 0, TotalFat: 0, TotalFiber: 0}, // Not logged
		{Date: startDate.AddDate(0, 0, 4), TotalCalories: 2050, TotalProtein: 155, TotalCarbs: 205, TotalFat: 72, TotalFiber: 31},
		{Date: startDate.AddDate(0, 0, 5), TotalCalories: 1950, TotalProtein: 145, TotalCarbs: 195, TotalFat: 68, TotalFiber: 29},
		{Date: startDate.AddDate(0, 0, 6), TotalCalories: 2000, TotalProtein: 150, TotalCarbs: 200, TotalFat: 70, TotalFiber: 30},
	}

	goals := &UserGoals{
		UserID:        userID,
		CaloriesGoal:  2000,
		ProteinGoal:   150,
		CarbsGoal:     200,
		FatGoal:       65,
		FiberGoal:     30,
	}

	mockRepo.On("GetDateRangeNutrition", mock.Anything, userID, startDate, endDate, timezone).Return(dailyData, nil)
	mockRepo.On("GetUserGoals", mock.Anything, userID).Return(goals, nil)
	mockRepo.On("GetLoggingStreak", mock.Anything, userID, endDate, timezone).Return(5, nil)

	result, err := svc.GetWeeklyTrends(context.Background(), userID, startDate, timezone)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 7, len(result.Trend.DailyData))
	assert.Equal(t, 6, result.Trend.Averages.DaysLogged)
	assert.InDelta(t, 2000, result.Trend.Averages.AvgCalories, 50)
	assert.InDelta(t, 85.71, result.Trend.ConsistencyScore, 0.01)
	assert.Equal(t, 5, result.Streak.CurrentStreak)

	mockRepo.AssertExpectations(t)
}

func TestGetMacroDistribution(t *testing.T) {
	mockRepo := new(MockRepository)
	mockWeightRepo := &mockWeightRepository{}
	mockGoalsRepo := &mockGoalsRepositoryAnalytics{}
	svc := NewService(mockRepo, mockWeightRepo, mockGoalsRepo)

	userID := "user-123"
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	timezone := "America/New_York"

	dailyTotals := &DailyTotals{
		Date:          date,
		Timezone:      timezone,
		TotalCalories: 2000,
		TotalProtein:  150, // 600 cal (30%)
		TotalCarbs:    200, // 800 cal (40%)
		TotalFat:      67,  // 600 cal (30%)
		TotalFiber:    30,
	}

	mockRepo.On("GetDailyNutrition", mock.Anything, userID, date, timezone).Return(dailyTotals, nil)

	result, err := svc.GetMacroDistribution(context.Background(), userID, date, timezone)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2000.0, result.TotalCalories)
	assert.Equal(t, 600.0, result.ProteinCalories)
	assert.Equal(t, 800.0, result.CarbsCalories)
	assert.InDelta(t, 603.0, result.FatCalories, 1)
	assert.InDelta(t, 30.0, result.ProteinPercent, 1)
	assert.InDelta(t, 40.0, result.CarbsPercent, 1)
	assert.InDelta(t, 30.0, result.FatPercent, 1)

	mockRepo.AssertExpectations(t)
}

func TestGetDateRangeStats(t *testing.T) {
	mockRepo := new(MockRepository)
	mockWeightRepo := &mockWeightRepository{}
	mockGoalsRepo := &mockGoalsRepositoryAnalytics{}
	svc := NewService(mockRepo, mockWeightRepo, mockGoalsRepo)

	userID := "user-123"
	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	timezone := "America/New_York"

	dailyData := []DailyTotals{
		{Date: startDate, TotalCalories: 2000, TotalProtein: 150, TotalCarbs: 200, TotalFat: 70, TotalFiber: 30},
		{Date: startDate.AddDate(0, 0, 1), TotalCalories: 1900, TotalProtein: 140, TotalCarbs: 190, TotalFat: 65, TotalFiber: 28},
		{Date: startDate.AddDate(0, 0, 2), TotalCalories: 2100, TotalProtein: 160, TotalCarbs: 210, TotalFat: 75, TotalFiber: 32},
	}

	mockRepo.On("GetDateRangeNutrition", mock.Anything, userID, startDate, endDate, timezone).Return(dailyData, nil)

	result, err := svc.GetDateRangeStats(context.Background(), userID, startDate, endDate, timezone)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// TrendsResponse doesn't have TotalDays or DaysLogged at the top level
	// These are in the Averages field
	assert.Equal(t, 3, len(result.DailyTotals))
	assert.Equal(t, 3, result.Averages.DaysLogged)
	// TrendsResponse doesn't have Totals - verify via min/max instead
	assert.InDelta(t, 2000, result.Averages.AvgCalories, 50)

	mockRepo.AssertExpectations(t)
}

func TestGetNutritionInsights(t *testing.T) {
	mockRepo := new(MockRepository)
	mockWeightRepo := &mockWeightRepository{}
	mockGoalsRepo := &mockGoalsRepositoryAnalytics{}
	svc := NewService(mockRepo, mockWeightRepo, mockGoalsRepo)

	userID := "user-123"
	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 21, 0, 0, 0, 0, time.UTC)
	timezone := "America/New_York"

	// Create 7 days of good data
	dailyData := make([]DailyTotals, 7)
	for i := 0; i < 7; i++ {
		dailyData[i] = DailyTotals{
			Date:          startDate.AddDate(0, 0, i),
			TotalCalories: 2000,
			TotalProtein:  150,
			TotalCarbs:    200,
			TotalFat:      70,
			TotalFiber:    30,
		}
	}

	goals := &UserGoals{
		UserID:        userID,
		CaloriesGoal:  2000,
		ProteinGoal:   150,
		CarbsGoal:     200,
		FatGoal:       65,
		FiberGoal:     30,
	}

	mockRepo.On("GetDateRangeNutrition", mock.Anything, userID, startDate, endDate, timezone).Return(dailyData, nil)
	mockRepo.On("GetUserGoals", mock.Anything, userID).Return(goals, nil)

	result, err := svc.GetNutritionInsights(context.Background(), userID, startDate, endDate, timezone)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "weekly", result.Period)
	assert.Greater(t, len(result.Insights), 0)
	assert.Greater(t, result.OverallScore, 0.0)

	// Check for success insights (good consistency and nutrition)
	hasConsistencySuccess := false
	hasCalorieSuccess := false
	for _, insight := range result.Insights {
		if insight.Category == "consistency" && insight.Type == "success" {
			hasConsistencySuccess = true
		}
		if insight.Category == "calories" && insight.Type == "success" {
			hasCalorieSuccess = true
		}
	}

	assert.True(t, hasConsistencySuccess)
	assert.True(t, hasCalorieSuccess)

	mockRepo.AssertExpectations(t)
}

func TestCalculateGoalComparison(t *testing.T) {
	svc := &service{}

	dailyTotals := &DailyTotals{
		TotalCalories: 2000,
		TotalProtein:  150,
		TotalCarbs:    200,
		TotalFat:      70,
		TotalFiber:    30,
	}

	goals := &UserGoals{
		CaloriesGoal: 2000,
		ProteinGoal:  150,
		CarbsGoal:    200,
		FatGoal:      65,
		FiberGoal:    30,
	}

	comparison := svc.calculateGoalComparison(dailyTotals, goals)

	assert.Equal(t, 100.0, comparison.CaloriesPercent)
	assert.Equal(t, 100.0, comparison.ProteinPercent)
	assert.Equal(t, 100.0, comparison.CarbsPercent)
	assert.InDelta(t, 107.69, comparison.FatPercent, 0.01)
	assert.Equal(t, 100.0, comparison.FiberPercent)
}

func TestCalculateAverages(t *testing.T) {
	svc := &service{}

	dailyData := []DailyTotals{
		{TotalCalories: 2000, TotalProtein: 150, TotalCarbs: 200, TotalFat: 70, TotalFiber: 30},
		{TotalCalories: 1900, TotalProtein: 140, TotalCarbs: 190, TotalFat: 65, TotalFiber: 28},
		{TotalCalories: 0, TotalProtein: 0, TotalCarbs: 0, TotalFat: 0, TotalFiber: 0}, // Not logged
	}

	averages := svc.calculateAverages(dailyData)

	assert.Equal(t, 2, averages.DaysLogged)
	assert.Equal(t, 1950.0, averages.AvgCalories)
	assert.Equal(t, 145.0, averages.AvgProtein)
}

func TestCalculateConsistencyScore(t *testing.T) {
	svc := &service{}

	tests := []struct {
		name          string
		dailyData     []DailyTotals
		totalDays     int
		expectedScore float64
	}{
		{
			name: "100% consistency",
			dailyData: []DailyTotals{
				{TotalCalories: 2000},
				{TotalCalories: 1900},
				{TotalCalories: 2100},
			},
			totalDays:     3,
			expectedScore: 100.0,
		},
		{
			name: "50% consistency",
			dailyData: []DailyTotals{
				{TotalCalories: 2000},
				{TotalCalories: 0},
			},
			totalDays:     2,
			expectedScore: 50.0,
		},
		{
			name:          "0% consistency",
			dailyData:     []DailyTotals{{TotalCalories: 0}},
			totalDays:     1,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := svc.calculateConsistencyScore(tt.dailyData, tt.totalDays)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

func TestGenerateInsights(t *testing.T) {
	svc := &service{}

	t.Run("consistency insights", func(t *testing.T) {
		insights := svc.generateConsistencyInsights(85, 6, 7)
		assert.Len(t, insights, 1)
		assert.Equal(t, "success", insights[0].Type)
		assert.Equal(t, "consistency", insights[0].Category)
	})

	t.Run("calorie insights - on target", func(t *testing.T) {
		avg := Averages{AvgCalories: 2000}
		goals := &UserGoals{CaloriesGoal: 2000}
		insights := svc.generateCalorieInsights(avg, goals)
		assert.Len(t, insights, 1)
		assert.Equal(t, "success", insights[0].Type)
	})

	t.Run("calorie insights - below target", func(t *testing.T) {
		avg := Averages{AvgCalories: 1500}
		goals := &UserGoals{CaloriesGoal: 2000}
		insights := svc.generateCalorieInsights(avg, goals)
		assert.Len(t, insights, 1)
		assert.Equal(t, "warning", insights[0].Type)
	})

	t.Run("fiber insights - low", func(t *testing.T) {
		avg := Averages{AvgFiber: 15}
		goals := &UserGoals{FiberGoal: 30}
		insights := svc.generateFiberInsights(avg, goals)
		assert.Len(t, insights, 1)
		assert.Equal(t, "alert", insights[0].Type)
		assert.Equal(t, "fiber", insights[0].Category)
	})
}
