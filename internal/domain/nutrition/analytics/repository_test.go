package analytics

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, Repository) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewRepository(db)
	return db, mock, repo
}

func TestGetDailyNutrition(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	userID := "user-123"
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	timezone := "America/New_York"

	tests := []struct {
		name          string
		setupMock     func()
		expectedError bool
		validate      func(*testing.T, *DailyTotals)
	}{
		{
			name: "successful daily nutrition retrieval",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"meal_type", "calories", "protein", "carbs", "fat", "fiber"}).
					AddRow("breakfast", 450.0, 25.0, 50.0, 15.0, 8.0).
					AddRow("lunch", 600.0, 35.0, 60.0, 20.0, 10.0).
					AddRow("dinner", 550.0, 30.0, 55.0, 18.0, 9.0).
					AddRow("snack", 200.0, 10.0, 25.0, 7.0, 3.0)

				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition`).
					WithArgs(userID, date.Format("2006-01-02"), timezone).
					WillReturnRows(rows)
			},
			expectedError: false,
			validate: func(t *testing.T, result *DailyTotals) {
				assert.Equal(t, 1800.0, result.TotalCalories)
				assert.Equal(t, 100.0, result.TotalProtein)
				assert.Equal(t, 190.0, result.TotalCarbs)
				assert.Equal(t, 60.0, result.TotalFat)
				assert.Equal(t, 30.0, result.TotalFiber)
				assert.Len(t, result.MealBreakdown, 4)
				assert.Equal(t, timezone, result.Timezone)
			},
		},
		{
			name: "no data for date",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"meal_type", "calories", "protein", "carbs", "fat", "fiber"})

				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition`).
					WithArgs(userID, date.Format("2006-01-02"), timezone).
					WillReturnRows(rows)
			},
			expectedError: false,
			validate: func(t *testing.T, result *DailyTotals) {
				assert.Equal(t, 0.0, result.TotalCalories)
				assert.Len(t, result.MealBreakdown, 0)
			},
		},
		{
			name: "database error",
			setupMock: func() {
				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition`).
					WithArgs(userID, date.Format("2006-01-02"), timezone).
					WillReturnError(sql.ErrConnDone)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := repo.GetDailyNutrition(context.Background(), userID, date, timezone)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tt.validate(t, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetDateRangeNutrition(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	userID := "user-123"
	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	timezone := "America/New_York"

	// Mock 3 days of data
	for i := 0; i < 3; i++ {
		currentDate := startDate.AddDate(0, 0, i)
		rows := sqlmock.NewRows([]string{"meal_type", "calories", "protein", "carbs", "fat", "fiber"}).
			AddRow("breakfast", 400.0, 20.0, 45.0, 12.0, 6.0)

		mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition`).
			WithArgs(userID, currentDate.Format("2006-01-02"), timezone).
			WillReturnRows(rows)
	}

	result, err := repo.GetDateRangeNutrition(context.Background(), userID, startDate, endDate, timezone)

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserGoals(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	userID := "user-123"

	tests := []struct {
		name          string
		setupMock     func()
		expectedError bool
		validate      func(*testing.T, *UserGoals)
	}{
		{
			name: "user with custom goals",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"user_id", "calories_goal", "protein_goal", "carbs_goal", "fat_goal", "fiber_goal",
				}).AddRow(userID, 2200.0, 165.0, 220.0, 70.0, 35.0)

				mock.ExpectQuery(`SELECT.*FROM user_profiles`).
					WithArgs(userID).
					WillReturnRows(rows)
			},
			expectedError: false,
			validate: func(t *testing.T, goals *UserGoals) {
				assert.Equal(t, userID, goals.UserID)
				assert.Equal(t, 2200.0, goals.CaloriesGoal)
				assert.Equal(t, 165.0, goals.ProteinGoal)
				assert.Equal(t, 220.0, goals.CarbsGoal)
				assert.Equal(t, 70.0, goals.FatGoal)
				assert.Equal(t, 35.0, goals.FiberGoal)
			},
		},
		{
			name: "user without profile returns defaults",
			setupMock: func() {
				mock.ExpectQuery(`SELECT.*FROM user_profiles`).
					WithArgs(userID).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: false,
			validate: func(t *testing.T, goals *UserGoals) {
				assert.Equal(t, userID, goals.UserID)
				assert.Equal(t, 2000.0, goals.CaloriesGoal)
				assert.Equal(t, 150.0, goals.ProteinGoal)
				assert.Equal(t, 200.0, goals.CarbsGoal)
				assert.Equal(t, 65.0, goals.FatGoal)
				assert.Equal(t, 30.0, goals.FiberGoal)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := repo.GetUserGoals(context.Background(), userID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tt.validate(t, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetLoggingStreak(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	userID := "user-123"
	endDate := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	timezone := "America/New_York"

	tests := []struct {
		name          string
		setupMock     func()
		expectedStreak int
		expectedError bool
	}{
		{
			name: "7 day streak",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(7)
				mock.ExpectQuery(`WITH RECURSIVE date_series`).
					WithArgs(userID, endDate.Format("2006-01-02"), timezone).
					WillReturnRows(rows)
			},
			expectedStreak: 7,
			expectedError: false,
		},
		{
			name: "no streak",
			setupMock: func() {
				mock.ExpectQuery(`WITH RECURSIVE date_series`).
					WithArgs(userID, endDate.Format("2006-01-02"), timezone).
					WillReturnError(sql.ErrNoRows)
			},
			expectedStreak: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			streak, err := repo.GetLoggingStreak(context.Background(), userID, endDate, timezone)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStreak, streak)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
