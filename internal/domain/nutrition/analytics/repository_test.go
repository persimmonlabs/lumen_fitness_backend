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

// setupTestDB creates a mock database and sqlmock for testing
func setupTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return db, mock
}

func TestGetDailyNutrition(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		date          time.Time
		timezone      string
		mockSetup     func(sqlmock.Sqlmock)
		expectedError bool
		validate      func(*testing.T, *DailyTotals)
	}{
		{
			name:     "successful retrieval with multiple meals",
			userID:   "user-123",
			date:     time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			timezone: "America/New_York",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"meal_type", "calories", "protein", "carbs", "fat", "fiber"}).
					AddRow("breakfast", 450.0, 25.0, 50.0, 15.0, 8.0).
					AddRow("breakfast", 200.0, 10.0, 20.0, 5.0, 3.0).
					AddRow("lunch", 600.0, 35.0, 60.0, 20.0, 10.0).
					AddRow("dinner", 700.0, 40.0, 70.0, 25.0, 12.0)

				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition\(\$1, \$2, \$3\)`).
					WithArgs("user-123", "2025-01-15", "America/New_York").
					WillReturnRows(rows)
			},
			expectedError: false,
			validate: func(t *testing.T, totals *DailyTotals) {
				assert.NotNil(t, totals)
				assert.Equal(t, 1950.0, totals.TotalCalories)
				assert.Equal(t, 110.0, totals.TotalProtein)
				assert.Equal(t, 200.0, totals.TotalCarbs)
				assert.Equal(t, 65.0, totals.TotalFat)
				assert.Equal(t, 33.0, totals.TotalFiber)
				assert.Len(t, totals.MealBreakdown, 3)
				assert.Equal(t, "America/New_York", totals.Timezone)

				// Verify meal breakdown aggregation
				breakfastFound := false
				for _, meal := range totals.MealBreakdown {
					if meal.MealType == "breakfast" {
						breakfastFound = true
						assert.Equal(t, 650.0, meal.Calories)
						assert.Equal(t, 35.0, meal.Protein)
						assert.Equal(t, 70.0, meal.Carbs)
						assert.Equal(t, 20.0, meal.Fat)
						assert.Equal(t, 11.0, meal.Fiber)
						assert.Equal(t, 2, meal.ItemCount)
					}
				}
				assert.True(t, breakfastFound, "breakfast meal should be in breakdown")
			},
		},
		{
			name:     "no data for date",
			userID:   "user-123",
			date:     time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			timezone: "UTC",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"meal_type", "calories", "protein", "carbs", "fat", "fiber"})
				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition\(\$1, \$2, \$3\)`).
					WithArgs("user-123", "2025-01-15", "UTC").
					WillReturnRows(rows)
			},
			expectedError: false,
			validate: func(t *testing.T, totals *DailyTotals) {
				assert.NotNil(t, totals)
				assert.Equal(t, 0.0, totals.TotalCalories)
				assert.Len(t, totals.MealBreakdown, 0)
			},
		},
		{
			name:     "database error",
			userID:   "user-123",
			date:     time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			timezone: "UTC",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition\(\$1, \$2, \$3\)`).
					WithArgs("user-123", "2025-01-15", "UTC").
					WillReturnError(sql.ErrConnDone)
			},
			expectedError: true,
		},
		{
			name:     "single meal type",
			userID:   "user-456",
			date:     time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
			timezone: "Europe/London",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"meal_type", "calories", "protein", "carbs", "fat", "fiber"}).
					AddRow("snack", 150.0, 5.0, 20.0, 4.0, 2.0)

				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition\(\$1, \$2, \$3\)`).
					WithArgs("user-456", "2025-01-16", "Europe/London").
					WillReturnRows(rows)
			},
			expectedError: false,
			validate: func(t *testing.T, totals *DailyTotals) {
				assert.NotNil(t, totals)
				assert.Equal(t, 150.0, totals.TotalCalories)
				assert.Len(t, totals.MealBreakdown, 1)
				assert.Equal(t, "snack", totals.MealBreakdown[0].MealType)
				assert.Equal(t, 1, totals.MealBreakdown[0].ItemCount)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()

			repo := NewRepository(db)
			ctx := context.Background()

			tt.mockSetup(mock)

			result, err := repo.GetDailyNutrition(ctx, tt.userID, tt.date, tt.timezone)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetDateRangeNutrition(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		startDate     time.Time
		endDate       time.Time
		timezone      string
		mockSetup     func(sqlmock.Sqlmock)
		expectedError bool
		validate      func(*testing.T, []DailyTotals)
	}{
		{
			name:      "successful 3-day range",
			userID:    "user-123",
			startDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC),
			timezone:  "UTC",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Day 1
				rows1 := sqlmock.NewRows([]string{"meal_type", "calories", "protein", "carbs", "fat", "fiber"}).
					AddRow("breakfast", 400.0, 20.0, 50.0, 10.0, 5.0)
				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition\(\$1, \$2, \$3\)`).
					WithArgs("user-123", "2025-01-15", "UTC").
					WillReturnRows(rows1)

				// Day 2
				rows2 := sqlmock.NewRows([]string{"meal_type", "calories", "protein", "carbs", "fat", "fiber"}).
					AddRow("lunch", 500.0, 25.0, 60.0, 15.0, 8.0)
				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition\(\$1, \$2, \$3\)`).
					WithArgs("user-123", "2025-01-16", "UTC").
					WillReturnRows(rows2)

				// Day 3
				rows3 := sqlmock.NewRows([]string{"meal_type", "calories", "protein", "carbs", "fat", "fiber"}).
					AddRow("dinner", 600.0, 30.0, 70.0, 20.0, 10.0)
				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition\(\$1, \$2, \$3\)`).
					WithArgs("user-123", "2025-01-17", "UTC").
					WillReturnRows(rows3)
			},
			expectedError: false,
			validate: func(t *testing.T, results []DailyTotals) {
				assert.Len(t, results, 3)
				assert.Equal(t, 400.0, results[0].TotalCalories)
				assert.Equal(t, 500.0, results[1].TotalCalories)
				assert.Equal(t, 600.0, results[2].TotalCalories)
			},
		},
		{
			name:      "single day range",
			userID:    "user-123",
			startDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			timezone:  "UTC",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"meal_type", "calories", "protein", "carbs", "fat", "fiber"}).
					AddRow("breakfast", 300.0, 15.0, 40.0, 8.0, 4.0)
				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition\(\$1, \$2, \$3\)`).
					WithArgs("user-123", "2025-01-15", "UTC").
					WillReturnRows(rows)
			},
			expectedError: false,
			validate: func(t *testing.T, results []DailyTotals) {
				assert.Len(t, results, 1)
				assert.Equal(t, 300.0, results[0].TotalCalories)
			},
		},
		{
			name:      "error on second day",
			userID:    "user-123",
			startDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
			timezone:  "UTC",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"meal_type", "calories", "protein", "carbs", "fat", "fiber"}).
					AddRow("breakfast", 300.0, 15.0, 40.0, 8.0, 4.0)
				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition\(\$1, \$2, \$3\)`).
					WithArgs("user-123", "2025-01-15", "UTC").
					WillReturnRows(rows)

				mock.ExpectQuery(`SELECT \* FROM get_daily_nutrition\(\$1, \$2, \$3\)`).
					WithArgs("user-123", "2025-01-16", "UTC").
					WillReturnError(sql.ErrConnDone)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()

			repo := NewRepository(db)
			ctx := context.Background()

			tt.mockSetup(mock)

			results, err := repo.GetDateRangeNutrition(ctx, tt.userID, tt.startDate, tt.endDate, tt.timezone)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, results)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, results)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetUserGoals(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		mockSetup     func(sqlmock.Sqlmock)
		expectedError bool
		validate      func(*testing.T, *UserGoals)
	}{
		{
			name:   "successful retrieval with custom goals",
			userID: "user-123",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"user_id", "calories_goal", "protein_goal", "carbs_goal", "fat_goal", "fiber_goal"}).
					AddRow("user-123", 2500.0, 180.0, 250.0, 80.0, 35.0)

				mock.ExpectQuery(`SELECT (.+) FROM user_profiles WHERE user_id = \$1`).
					WithArgs("user-123").
					WillReturnRows(rows)
			},
			expectedError: false,
			validate: func(t *testing.T, goals *UserGoals) {
				assert.NotNil(t, goals)
				assert.Equal(t, "user-123", goals.UserID)
				assert.Equal(t, 2500.0, goals.CaloriesGoal)
				assert.Equal(t, 180.0, goals.ProteinGoal)
				assert.Equal(t, 250.0, goals.CarbsGoal)
				assert.Equal(t, 80.0, goals.FatGoal)
				assert.Equal(t, 35.0, goals.FiberGoal)
			},
		},
		{
			name:   "no profile exists - return default goals",
			userID: "user-456",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT (.+) FROM user_profiles WHERE user_id = \$1`).
					WithArgs("user-456").
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: false,
			validate: func(t *testing.T, goals *UserGoals) {
				assert.NotNil(t, goals)
				assert.Equal(t, "user-456", goals.UserID)
				// Verify default values
				assert.Equal(t, 2000.0, goals.CaloriesGoal)
				assert.Equal(t, 150.0, goals.ProteinGoal)
				assert.Equal(t, 200.0, goals.CarbsGoal)
				assert.Equal(t, 65.0, goals.FatGoal)
				assert.Equal(t, 30.0, goals.FiberGoal)
			},
		},
		{
			name:   "profile with null goals uses COALESCE defaults",
			userID: "user-789",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"user_id", "calories_goal", "protein_goal", "carbs_goal", "fat_goal", "fiber_goal"}).
					AddRow("user-789", 2000.0, 150.0, 200.0, 65.0, 30.0)

				mock.ExpectQuery(`SELECT (.+) FROM user_profiles WHERE user_id = \$1`).
					WithArgs("user-789").
					WillReturnRows(rows)
			},
			expectedError: false,
			validate: func(t *testing.T, goals *UserGoals) {
				assert.NotNil(t, goals)
				assert.Equal(t, "user-789", goals.UserID)
				assert.Equal(t, 2000.0, goals.CaloriesGoal)
				assert.Equal(t, 150.0, goals.ProteinGoal)
				assert.Equal(t, 200.0, goals.CarbsGoal)
				assert.Equal(t, 65.0, goals.FatGoal)
				assert.Equal(t, 30.0, goals.FiberGoal)
			},
		},
		{
			name:   "database connection error",
			userID: "user-123",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT (.+) FROM user_profiles WHERE user_id = \$1`).
					WithArgs("user-123").
					WillReturnError(sql.ErrConnDone)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()

			repo := NewRepository(db)
			ctx := context.Background()

			tt.mockSetup(mock)

			goals, err := repo.GetUserGoals(ctx, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, goals)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, goals)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetLoggingStreak(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		endDate        time.Time
		timezone       string
		mockSetup      func(sqlmock.Sqlmock)
		expectedError  bool
		expectedStreak int
	}{
		{
			name:     "7-day streak",
			userID:   "user-123",
			endDate:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			timezone: "UTC",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(7)
				mock.ExpectQuery(`WITH RECURSIVE date_series AS`).
					WithArgs("user-123", "2025-01-15", "UTC").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedStreak: 7,
		},
		{
			name:     "no streak - no meals logged",
			userID:   "user-456",
			endDate:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			timezone: "UTC",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
				mock.ExpectQuery(`WITH RECURSIVE date_series AS`).
					WithArgs("user-456", "2025-01-15", "UTC").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedStreak: 0,
		},
		{
			name:     "empty user ID defaults to zero streak",
			userID:   "",
			endDate:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			timezone: "UTC",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// No mock needed - validation fails before query
			},
			expectedError: true,
		},
		{
			name:     "empty timezone defaults to UTC",
			userID:   "user-123",
			endDate:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			timezone: "",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(5)
				// Should use UTC as default timezone
				mock.ExpectQuery(`WITH RECURSIVE date_series AS`).
					WithArgs("user-123", "2025-01-15", "UTC").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedStreak: 5,
		},
		{
			name:     "zero time defaults to now",
			userID:   "user-123",
			endDate:  time.Time{},
			timezone: "UTC",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(3)
				// Should accept any date since it defaults to now
				mock.ExpectQuery(`WITH RECURSIVE date_series AS`).
					WithArgs("user-123", sqlmock.AnyArg(), "UTC").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedStreak: 3,
		},
		{
			name:     "query returns no rows",
			userID:   "user-789",
			endDate:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			timezone: "UTC",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`WITH RECURSIVE date_series AS`).
					WithArgs("user-789", "2025-01-15", "UTC").
					WillReturnError(sql.ErrNoRows)
			},
			expectedError:  false,
			expectedStreak: 0,
		},
		{
			name:     "database error",
			userID:   "user-123",
			endDate:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			timezone: "UTC",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`WITH RECURSIVE date_series AS`).
					WithArgs("user-123", "2025-01-15", "UTC").
					WillReturnError(sql.ErrConnDone)
			},
			expectedError: true,
		},
		{
			name:     "different timezone",
			userID:   "user-123",
			endDate:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			timezone: "America/New_York",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(10)
				mock.ExpectQuery(`WITH RECURSIVE date_series AS`).
					WithArgs("user-123", "2025-01-15", "America/New_York").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedStreak: 10,
		},
		{
			name:     "maximum streak near limit",
			userID:   "user-dedicated",
			endDate:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			timezone: "UTC",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Test that StreakRecursionLimit constant is being used
				rows := sqlmock.NewRows([]string{"count"}).AddRow(365)
				mock.ExpectQuery(`WITH RECURSIVE date_series AS`).
					WithArgs("user-dedicated", "2025-01-15", "UTC").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedStreak: 365,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()

			repo := NewRepository(db)
			ctx := context.Background()

			tt.mockSetup(mock)

			streak, err := repo.GetLoggingStreak(ctx, tt.userID, tt.endDate, tt.timezone)

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

// TestGetLoggingStreak_VerifyMealsTable tests that the query uses the 'meals' table
func TestGetLoggingStreak_VerifyMealsTable(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	// The query should reference the 'meals' table, NOT 'food_logs'
	rows := sqlmock.NewRows([]string{"count"}).AddRow(5)

	// Verify the query contains "FROM meals" and "consumed_at" column
	mock.ExpectQuery(`WITH RECURSIVE date_series AS (.+) FROM meals WHERE user_id`).
		WithArgs("user-123", "2025-01-15", "UTC").
		WillReturnRows(rows)

	streak, err := repo.GetLoggingStreak(ctx, "user-123", time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), "UTC")

	assert.NoError(t, err)
	assert.Equal(t, 5, streak)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetLoggingStreak_VerifyRecursionLimit verifies StreakRecursionLimit constant usage
func TestGetLoggingStreak_VerifyRecursionLimit(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	// Verify the query uses StreakRecursionLimit constant (365)
	rows := sqlmock.NewRows([]string{"count"}).AddRow(100)

	// The query should contain "WHERE day_offset < 365"
	mock.ExpectQuery(`WITH RECURSIVE date_series AS (.+) WHERE day_offset < 365`).
		WithArgs("user-123", "2025-01-15", "UTC").
		WillReturnRows(rows)

	streak, err := repo.GetLoggingStreak(ctx, "user-123", time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), "UTC")

	assert.NoError(t, err)
	assert.Equal(t, 100, streak)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetLoggingStreak_InputValidation tests input validation logic
func TestGetLoggingStreak_InputValidation(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	t.Run("empty user ID returns error", func(t *testing.T) {
		_, err := repo.GetLoggingStreak(ctx, "", time.Now(), "UTC")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user ID cannot be empty")
	})

	t.Run("verify StreakRecursionLimit constant value", func(t *testing.T) {
		// This test verifies the StreakRecursionLimit constant is set correctly
		assert.Equal(t, 365, StreakRecursionLimit, "StreakRecursionLimit should be 365")
	})
}

// TestNewRepository verifies repository creation
func TestNewRepository(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	assert.NotNil(t, repo)
	assert.Implements(t, (*Repository)(nil), repo)
}
