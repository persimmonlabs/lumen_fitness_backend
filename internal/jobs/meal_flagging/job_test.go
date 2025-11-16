package meal_flagging

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewMealFlaggingJob(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	t.Run("Create with default config", func(t *testing.T) {
		job := NewMealFlaggingJob(db, nil, nil)
		assert.NotNil(t, job)
		assert.NotNil(t, job.config)
		assert.NotNil(t, job.flagger)
		assert.NotNil(t, job.logger)
		assert.Equal(t, DefaultConfig().ScheduleTime, job.config.ScheduleTime)
	})

	t.Run("Create with custom config", func(t *testing.T) {
		customConfig := &Config{
			Enabled:      true,
			ScheduleTime: "03:00",
			BatchSize:    500,
			LookbackDays: 2,
		}
		job := NewMealFlaggingJob(db, customConfig, nil)
		assert.Equal(t, "03:00", job.config.ScheduleTime)
		assert.Equal(t, 500, job.config.BatchSize)
		assert.Equal(t, 2, job.config.LookbackDays)
	})
}

func TestParseCronSchedule(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	job := NewMealFlaggingJob(db, nil, nil)

	tests := []struct {
		name     string
		timeStr  string
		expected string
		hasError bool
	}{
		{
			name:     "2am schedule",
			timeStr:  "02:00",
			expected: "0 2 * * *",
			hasError: false,
		},
		{
			name:     "3:30am schedule",
			timeStr:  "03:30",
			expected: "30 3 * * *",
			hasError: false,
		},
		{
			name:     "Midnight schedule",
			timeStr:  "00:00",
			expected: "0 0 * * *",
			hasError: false,
		},
		{
			name:     "Invalid format",
			timeStr:  "invalid",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := job.parseCronSchedule(tt.timeStr)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFetchMealBatch(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	job := NewMealFlaggingJob(db, DefaultConfig(), nil)

	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	t.Run("Fetch meals successfully", func(t *testing.T) {
		mealID := uuid.New()
		userID := uuid.New()

		// Mock meals query
		mealRows := sqlmock.NewRows([]string{
			"id", "user_id", "name", "description",
			"calories", "protein", "carbs", "fat",
			"logged_at", "ai_confidence",
		}).AddRow(
			mealID, userID, "Breakfast", "Eggs and toast",
			400.0, 20.0, 40.0, 15.0,
			time.Now(), 0.85,
		)

		mock.ExpectQuery("SELECT (.+) FROM meals m WHERE").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1000, 0).
			WillReturnRows(mealRows)

		// Mock meal items query
		itemRows := sqlmock.NewRows([]string{
			"id", "description", "quantity", "unit",
			"calories", "protein", "carbs", "fat",
		}).AddRow(
			uuid.New(), "eggs", 2.0, "pieces",
			140.0, 12.0, 1.0, 10.0,
		).AddRow(
			uuid.New(), "toast", 2.0, "slices",
			160.0, 6.0, 30.0, 2.0,
		)

		mock.ExpectQuery("SELECT id, description, quantity, unit").
			WithArgs(mealID).
			WillReturnRows(itemRows)

		meals, err := job.fetchMealBatch(context.Background(), startTime, endTime, 0)
		assert.NoError(t, err)
		assert.Len(t, meals, 1)
		assert.Equal(t, mealID, meals[0].ID)
		assert.Equal(t, userID, meals[0].UserID)
		assert.Len(t, meals[0].Items, 2)
	})

	t.Run("Handle empty result", func(t *testing.T) {
		emptyRows := sqlmock.NewRows([]string{
			"id", "user_id", "name", "description",
			"calories", "protein", "carbs", "fat",
			"logged_at", "ai_confidence",
		})

		mock.ExpectQuery("SELECT (.+) FROM meals m WHERE").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1000, 0).
			WillReturnRows(emptyRows)

		meals, err := job.fetchMealBatch(context.Background(), startTime, endTime, 0)
		assert.NoError(t, err)
		assert.Len(t, meals, 0)
	})
}

func TestFetchUserMealBatch(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	job := NewMealFlaggingJob(db, DefaultConfig(), nil)

	userID := uuid.New()
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	t.Run("Fetch user meals successfully", func(t *testing.T) {
		mealID := uuid.New()

		// Mock user meals query
		mealRows := sqlmock.NewRows([]string{
			"id", "user_id", "name", "description",
			"calories", "protein", "carbs", "fat",
			"logged_at", "ai_confidence",
		}).AddRow(
			mealID, userID, "Lunch", "Chicken salad",
			350.0, 30.0, 20.0, 12.0,
			time.Now(), 0.9,
		)

		mock.ExpectQuery("SELECT (.+) FROM meals m WHERE m.user_id").
			WithArgs(userID, sqlmock.AnyArg(), sqlmock.AnyArg(), 1000, 0).
			WillReturnRows(mealRows)

		// Mock meal items query
		itemRows := sqlmock.NewRows([]string{
			"id", "description", "quantity", "unit",
			"calories", "protein", "carbs", "fat",
		}).AddRow(
			uuid.New(), "chicken breast", 150.0, "g",
			248.0, 25.0, 0.0, 5.0,
		)

		mock.ExpectQuery("SELECT id, description, quantity, unit").
			WithArgs(mealID).
			WillReturnRows(itemRows)

		meals, err := job.fetchUserMealBatch(context.Background(), userID, startTime, endTime, 0)
		assert.NoError(t, err)
		assert.Len(t, meals, 1)
		assert.Equal(t, userID, meals[0].UserID)
	})
}

func TestProcessMeal(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	job := NewMealFlaggingJob(db, DefaultConfig(), log.New(os.Stdout, "", 0))

	t.Run("Flag unusual portion", func(t *testing.T) {
		meal := &MealData{
			ID:       uuid.New(),
			UserID:   uuid.New(),
			Calories: 415, // Matches macros: 30*4 + 40*4 + 15*9 = 415
			Protein:  30,
			Carbs:    40,
			Fat:      15,
			LoggedAt: time.Now(),
			Items: []MealItemData{
				{
					Description: "chicken breast",
					Quantity:    600,
					Unit:        "g",
					Calories:    990,
				},
			},
		}

		summary := NewFlagSummary()

		// Expect flag insertion FIRST (unusual portion is checked first)
		mock.ExpectExec("INSERT INTO meal_flags").
			WithArgs(sqlmock.AnyArg(), meal.ID, meal.UserID, FlagTypeUnusualPortion,
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), false, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// THEN expect duplicate check query (happens last)
		mock.ExpectQuery("SELECT m.id, m.name, m.description, m.logged_at").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "logged_at"}))

		err := job.processMeal(context.Background(), meal, summary)
		assert.NoError(t, err)
		assert.Equal(t, 1, summary.TotalFlagsCreated)
		assert.Equal(t, 1, summary.FlagsByType[FlagTypeUnusualPortion])
	})

	t.Run("Flag macro mismatch", func(t *testing.T) {
		meal := &MealData{
			ID:       uuid.New(),
			UserID:   uuid.New(),
			Calories: 600, // Should be ~370
			Protein:  30,  // 120 cal
			Carbs:    40,  // 160 cal
			Fat:      10,  // 90 cal
			LoggedAt: time.Now(),
			Items:    []MealItemData{},
		}

		summary := NewFlagSummary()

		// Expect flag insertion for macro mismatch (checked second)
		mock.ExpectExec("INSERT INTO meal_flags").
			WithArgs(sqlmock.AnyArg(), meal.ID, meal.UserID, FlagTypeMacroMismatch,
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), false, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// THEN expect duplicate check query (happens last)
		mock.ExpectQuery("SELECT m.id, m.name, m.description, m.logged_at").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "logged_at"}))

		err := job.processMeal(context.Background(), meal, summary)
		assert.NoError(t, err)
		assert.Equal(t, 1, summary.TotalFlagsCreated)
		assert.Equal(t, 1, summary.FlagsByType[FlagTypeMacroMismatch])
	})

	t.Run("No flags for normal meal", func(t *testing.T) {
		meal := &MealData{
			ID:       uuid.New(),
			UserID:   uuid.New(),
			Calories: 370,
			Protein:  30,
			Carbs:    40,
			Fat:      10,
			LoggedAt: time.Now(),
			Items: []MealItemData{
				{Description: "chicken", Quantity: 150, Unit: "g", Calories: 248},
			},
		}

		summary := NewFlagSummary()

		// Mock duplicate check query
		mock.ExpectQuery("SELECT m.id, m.name, m.description, m.logged_at").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "logged_at"}))

		err := job.processMeal(context.Background(), meal, summary)
		assert.NoError(t, err)
		assert.Equal(t, 0, summary.TotalFlagsCreated)
	})
}

func TestRunForUser(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	logger := log.New(os.Stdout, "[TEST] ", 0)
	job := NewMealFlaggingJob(db, DefaultConfig(), logger)

	userID := uuid.New()

	t.Run("Process user meals successfully", func(t *testing.T) {
		mealID := uuid.New()

		// Mock user meals query
		mealRows := sqlmock.NewRows([]string{
			"id", "user_id", "name", "description",
			"calories", "protein", "carbs", "fat",
			"logged_at", "ai_confidence",
		}).AddRow(
			mealID, userID, "Test Meal", "Description",
			400.0, 30.0, 40.0, 10.0,
			time.Now(), nil,
		)

		mock.ExpectQuery("SELECT (.+) FROM meals m WHERE m.user_id").
			WithArgs(userID, sqlmock.AnyArg(), sqlmock.AnyArg(), 1000, 0).
			WillReturnRows(mealRows)

		// Mock meal items query
		itemRows := sqlmock.NewRows([]string{
			"id", "description", "quantity", "unit",
			"calories", "protein", "carbs", "fat",
		}).AddRow(
			uuid.New(), "test item", 100.0, "g",
			200.0, 10.0, 20.0, 5.0,
		)

		mock.ExpectQuery("SELECT id, description, quantity, unit").
			WithArgs(mealID).
			WillReturnRows(itemRows)

		// Mock duplicate check
		mock.ExpectQuery("SELECT m.id, m.name, m.description, m.logged_at").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "logged_at"}))

		summary, err := job.RunForUser(context.Background(), userID)
		assert.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Equal(t, 1, summary.TotalMealsChecked)
	})
}

func TestFlagSummary(t *testing.T) {
	t.Run("Create new summary", func(t *testing.T) {
		summary := NewFlagSummary()
		assert.NotNil(t, summary)
		assert.Equal(t, 0, summary.TotalMealsChecked)
		assert.Equal(t, 0, summary.TotalFlagsCreated)
		assert.NotNil(t, summary.FlagsByType)
		assert.NotNil(t, summary.FlagsBySeverity)
	})

	t.Run("Add flags", func(t *testing.T) {
		summary := NewFlagSummary()
		summary.AddFlag(FlagTypeUnusualPortion, "high")
		summary.AddFlag(FlagTypeMacroMismatch, "medium")
		summary.AddFlag(FlagTypeUnusualPortion, "low")

		assert.Equal(t, 3, summary.TotalFlagsCreated)
		assert.Equal(t, 2, summary.FlagsByType[FlagTypeUnusualPortion])
		assert.Equal(t, 1, summary.FlagsByType[FlagTypeMacroMismatch])
		assert.Equal(t, 1, summary.FlagsBySeverity["high"])
		assert.Equal(t, 1, summary.FlagsBySeverity["medium"])
		assert.Equal(t, 1, summary.FlagsBySeverity["low"])
	})

	t.Run("Add errors", func(t *testing.T) {
		summary := NewFlagSummary()
		summary.AddError("error 1")
		summary.AddError("error 2")

		assert.Len(t, summary.Errors, 2)
		assert.Contains(t, summary.Errors, "error 1")
		assert.Contains(t, summary.Errors, "error 2")
	})
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, "02:00", config.ScheduleTime)
	assert.Equal(t, 1000, config.BatchSize)
	assert.Equal(t, 1, config.LookbackDays)
	assert.Equal(t, 10.0, config.MacroTolerancePercent)
	assert.Equal(t, 0.7, config.ConfidenceThreshold)
	assert.Equal(t, 30, config.DuplicateWindowMinutes)
	assert.Equal(t, 80.0, config.DuplicateSimilarityPercent)
}

func TestStartStopJob(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	t.Run("Start disabled job", func(t *testing.T) {
		config := DefaultConfig()
		config.Enabled = false

		job := NewMealFlaggingJob(db, config, nil)
		err := job.Start()
		assert.NoError(t, err)
	})

	t.Run("Start and stop job", func(t *testing.T) {
		config := DefaultConfig()
		job := NewMealFlaggingJob(db, config, nil)

		err := job.Start()
		assert.NoError(t, err)

		// Give it a moment to start
		time.Sleep(100 * time.Millisecond)

		job.Stop()
	})

	t.Run("Invalid schedule time", func(t *testing.T) {
		config := DefaultConfig()
		config.ScheduleTime = "invalid"

		job := NewMealFlaggingJob(db, config, nil)
		err := job.Start()
		assert.Error(t, err)
	})
}

func TestConcurrentJobExecution(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	job := NewMealFlaggingJob(db, DefaultConfig(), nil)

	// Manually set isRunning to true
	job.isRunning = true

	err := job.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}
