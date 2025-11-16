package meal_flagging

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return db, mock
}

func TestCheckUnusualPortion(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	flagger := NewFlagger(db, DefaultConfig())

	tests := []struct {
		name       string
		meal       *MealData
		shouldFlag bool
		severity   string
	}{
		{
			name: "Normal portion - no flag",
			meal: &MealData{
				ID:     uuid.New(),
				UserID: uuid.New(),
				Items: []MealItemData{
					{
						Description: "chicken breast",
						Quantity:    200,
						Unit:        "g",
						Calories:    330,
					},
				},
			},
			shouldFlag: false,
		},
		{
			name: "Excessive chicken portion - high severity",
			meal: &MealData{
				ID:     uuid.New(),
				UserID: uuid.New(),
				Items: []MealItemData{
					{
						Description: "grilled chicken breast",
						Quantity:    600,
						Unit:        "g",
						Calories:    990,
					},
				},
			},
			shouldFlag: true,
			severity:   "low", // 600/450 = 1.33, not > 3
		},
		{
			name: "Very excessive portion - high severity",
			meal: &MealData{
				ID:     uuid.New(),
				UserID: uuid.New(),
				Items: []MealItemData{
					{
						Description: "chicken",
						Quantity:    1500,
						Unit:        "g",
						Calories:    2475,
					},
				},
			},
			shouldFlag: true,
			severity:   "high", // > 1000 calories triggers high severity
		},
		{
			name: "Excessive calories for single item",
			meal: &MealData{
				ID:     uuid.New(),
				UserID: uuid.New(),
				Items: []MealItemData{
					{
						Description: "fried rice",
						Quantity:    500,
						Unit:        "g",
						Calories:    1200,
					},
				},
			},
			shouldFlag: true,
			severity:   "high",
		},
		{
			name: "Excessive oil portion",
			meal: &MealData{
				ID:     uuid.New(),
				UserID: uuid.New(),
				Items: []MealItemData{
					{
						Description: "olive oil",
						Quantity:    50,
						Unit:        "ml",
						Calories:    440,
					},
				},
			},
			shouldFlag: true,
			severity:   "low",
		},
		{
			name: "Multiple normal items - no flag",
			meal: &MealData{
				ID:     uuid.New(),
				UserID: uuid.New(),
				Items: []MealItemData{
					{Description: "chicken breast", Quantity: 150, Unit: "g", Calories: 248},
					{Description: "brown rice", Quantity: 100, Unit: "g", Calories: 112},
					{Description: "broccoli", Quantity: 100, Unit: "g", Calories: 34},
				},
			},
			shouldFlag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flagger.CheckUnusualPortion(tt.meal)
			assert.Equal(t, tt.shouldFlag, result.ShouldFlag, "shouldFlag mismatch")
			if tt.shouldFlag {
				assert.Equal(t, FlagTypeUnusualPortion, result.FlagType)
				assert.Equal(t, tt.severity, result.Severity, "severity mismatch")
				assert.NotEmpty(t, result.Description)
				assert.NotEmpty(t, result.Details)
			}
		})
	}
}

func TestCheckMacroMismatch(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	flagger := NewFlagger(db, DefaultConfig())

	tests := []struct {
		name       string
		meal       *MealData
		shouldFlag bool
		severity   string
	}{
		{
			name: "Perfect macro match - no flag",
			meal: &MealData{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Protein:  30, // 120 cal
				Carbs:    40, // 160 cal
				Fat:      10, // 90 cal
				Calories: 370, // Total: 370
			},
			shouldFlag: false,
		},
		{
			name: "Within tolerance (5% diff) - no flag",
			meal: &MealData{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Protein:  30, // 120 cal
				Carbs:    40, // 160 cal
				Fat:      10, // 90 cal
				Calories: 385, // 370 + 4% = 384.8
			},
			shouldFlag: false,
		},
		{
			name: "Slight mismatch (15% diff) - low severity",
			meal: &MealData{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Protein:  30, // 120 cal
				Carbs:    40, // 160 cal
				Fat:      10, // 90 cal
				Calories: 425, // 370 + 15% = 425.5
			},
			shouldFlag: true,
			severity:   "low",
		},
		{
			name: "Moderate mismatch (25% diff) - medium severity",
			meal: &MealData{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Protein:  30, // 120 cal
				Carbs:    40, // 160 cal
				Fat:      10, // 90 cal
				Calories: 463, // 370 + 25% = 462.5
			},
			shouldFlag: true,
			severity:   "medium",
		},
		{
			name: "Large mismatch (35% diff) - high severity",
			meal: &MealData{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Protein:  30, // 120 cal
				Carbs:    40, // 160 cal
				Fat:      10, // 90 cal
				Calories: 500, // 370 + 35% = 499.5
			},
			shouldFlag: true,
			severity:   "high",
		},
		{
			name: "Underreported calories - medium severity",
			meal: &MealData{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Protein:  30, // 120 cal
				Carbs:    40, // 160 cal
				Fat:      10, // 90 cal
				Calories: 280, // 370 - 24% = 281.2
			},
			shouldFlag: true,
			severity:   "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flagger.CheckMacroMismatch(tt.meal)
			assert.Equal(t, tt.shouldFlag, result.ShouldFlag, "shouldFlag mismatch")
			if tt.shouldFlag {
				assert.Equal(t, FlagTypeMacroMismatch, result.FlagType)
				assert.Equal(t, tt.severity, result.Severity, "severity mismatch")
				assert.NotEmpty(t, result.Description)
				assert.Contains(t, result.Details, "reported_calories")
				assert.Contains(t, result.Details, "calculated_calories")
			}
		})
	}
}

func TestCheckLowConfidence(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	flagger := NewFlagger(db, DefaultConfig())

	tests := []struct {
		name       string
		confidence *float64
		shouldFlag bool
		severity   string
	}{
		{
			name:       "High confidence - no flag",
			confidence: floatPtr(0.95),
			shouldFlag: false,
		},
		{
			name:       "Confidence at threshold - no flag",
			confidence: floatPtr(0.7),
			shouldFlag: false,
		},
		{
			name:       "Low confidence (0.65) - low severity",
			confidence: floatPtr(0.65),
			shouldFlag: true,
			severity:   "low",
		},
		{
			name:       "Medium confidence (0.55) - medium severity",
			confidence: floatPtr(0.55),
			shouldFlag: true,
			severity:   "medium",
		},
		{
			name:       "Very low confidence (0.3) - high severity",
			confidence: floatPtr(0.3),
			shouldFlag: true,
			severity:   "high",
		},
		{
			name:       "No confidence data - no flag",
			confidence: nil,
			shouldFlag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meal := &MealData{
				ID:           uuid.New(),
				UserID:       uuid.New(),
				AIConfidence: tt.confidence,
			}

			result := flagger.CheckLowConfidence(meal)
			assert.Equal(t, tt.shouldFlag, result.ShouldFlag, "shouldFlag mismatch")
			if tt.shouldFlag {
				assert.Equal(t, FlagTypeLowConfidence, result.FlagType)
				assert.Equal(t, tt.severity, result.Severity, "severity mismatch")
				assert.NotEmpty(t, result.Description)
			}
		})
	}
}

func TestCheckDuplicate(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	config := DefaultConfig()
	flagger := NewFlagger(db, config)

	userID := uuid.New()
	mealID := uuid.New()
	otherMealID := uuid.New()
	loggedAt := time.Now()

	meal := &MealData{
		ID:       mealID,
		UserID:   userID,
		LoggedAt: loggedAt,
		Items: []MealItemData{
			{Description: "chicken breast", Quantity: 200, Unit: "g"},
			{Description: "rice", Quantity: 150, Unit: "g"},
			{Description: "broccoli", Quantity: 100, Unit: "g"},
		},
	}

	t.Run("Duplicate found - high similarity", func(t *testing.T) {
		// Mock the query for nearby meals
		rows := sqlmock.NewRows([]string{"id", "name", "description", "logged_at"}).
			AddRow(otherMealID, "Lunch", "Chicken and rice", loggedAt.Add(-15*time.Minute))

		mock.ExpectQuery("SELECT m.id, m.name, m.description, m.logged_at").
			WithArgs(userID, mealID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		// Mock the meal items query
		itemRows := sqlmock.NewRows([]string{"id", "description", "quantity", "unit", "calories", "protein", "carbs", "fat"}).
			AddRow(uuid.New(), "chicken breast", 200.0, "g", 330.0, 62.0, 0.0, 7.0).
			AddRow(uuid.New(), "rice", 150.0, "g", 195.0, 4.0, 43.0, 0.4).
			AddRow(uuid.New(), "broccoli", 100.0, "g", 34.0, 2.8, 7.0, 0.4)

		mock.ExpectQuery("SELECT id, description, quantity, unit").
			WithArgs(otherMealID).
			WillReturnRows(itemRows)

		result := flagger.CheckDuplicate(context.Background(), meal)
		assert.True(t, result.ShouldFlag)
		assert.Equal(t, FlagTypeDuplicate, result.FlagType)
		assert.NotEmpty(t, result.Description)
	})

	t.Run("No duplicates found", func(t *testing.T) {
		// Mock empty result
		rows := sqlmock.NewRows([]string{"id", "name", "description", "logged_at"})

		mock.ExpectQuery("SELECT m.id, m.name, m.description, m.logged_at").
			WithArgs(userID, mealID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		result := flagger.CheckDuplicate(context.Background(), meal)
		assert.False(t, result.ShouldFlag)
	})
}

func TestFlagMeal(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	flagger := NewFlagger(db, DefaultConfig())

	mealID := uuid.New()
	userID := uuid.New()

	result := &FlagResult{
		ShouldFlag:  true,
		FlagType:    FlagTypeUnusualPortion,
		Severity:    "high",
		Description: "Test flag",
		Details: map[string]interface{}{
			"test": "data",
		},
	}

	t.Run("Successfully insert flag", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO meal_flags").
			WithArgs(
				sqlmock.AnyArg(), // id
				mealID,
				userID,
				FlagTypeUnusualPortion,
				"high",
				"Test flag",
				sqlmock.AnyArg(), // details JSON
				false,
				sqlmock.AnyArg(), // created_at
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := flagger.FlagMeal(context.Background(), mealID, userID, result)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Do not insert if shouldFlag is false", func(t *testing.T) {
		noFlagResult := &FlagResult{ShouldFlag: false}
		err := flagger.FlagMeal(context.Background(), mealID, userID, noFlagResult)
		assert.NoError(t, err)
		// No expectations set, so this should pass
	})
}

func TestItemSimilarity(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	flagger := NewFlagger(db, DefaultConfig())

	tests := []struct {
		name     string
		item1    MealItemData
		item2    MealItemData
		expected bool
	}{
		{
			name:     "Exact match",
			item1:    MealItemData{Description: "chicken breast"},
			item2:    MealItemData{Description: "chicken breast"},
			expected: true,
		},
		{
			name:     "Case insensitive match",
			item1:    MealItemData{Description: "Chicken Breast"},
			item2:    MealItemData{Description: "chicken breast"},
			expected: true,
		},
		{
			name:     "Substring match",
			item1:    MealItemData{Description: "grilled chicken breast"},
			item2:    MealItemData{Description: "chicken breast"},
			expected: true,
		},
		{
			name:     "Word order variation",
			item1:    MealItemData{Description: "chicken breast grilled"},
			item2:    MealItemData{Description: "grilled chicken breast"},
			expected: true,
		},
		{
			name:     "Different items",
			item1:    MealItemData{Description: "chicken breast"},
			item2:    MealItemData{Description: "beef steak"},
			expected: false,
		},
		{
			name:     "Partial word match (50% threshold)",
			item1:    MealItemData{Description: "chicken rice bowl"},
			item2:    MealItemData{Description: "chicken pasta salad"},
			expected: false, // "chicken" matches, 1/3 = 33% < 50%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flagger.itemsAreSimilar(tt.item1, tt.item2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMealSimilarity(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	flagger := NewFlagger(db, DefaultConfig())

	items1 := []MealItemData{
		{Description: "chicken breast"},
		{Description: "rice"},
		{Description: "broccoli"},
	}

	tests := []struct {
		name            string
		items2          []MealItemData
		expectedMin     float64
		expectedMax     float64
	}{
		{
			name: "Identical meals - 100% similarity",
			items2: []MealItemData{
				{Description: "chicken breast"},
				{Description: "rice"},
				{Description: "broccoli"},
			},
			expectedMin: 100,
			expectedMax: 100,
		},
		{
			name: "Two out of three match - 66% similarity",
			items2: []MealItemData{
				{Description: "chicken breast"},
				{Description: "rice"},
				{Description: "carrots"},
			},
			expectedMin: 60,
			expectedMax: 70,
		},
		{
			name: "Completely different - 0% similarity",
			items2: []MealItemData{
				{Description: "salmon"},
				{Description: "pasta"},
				{Description: "spinach"},
			},
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			name:        "Empty meal - 0% similarity",
			items2:      []MealItemData{},
			expectedMin: 0,
			expectedMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := flagger.calculateMealSimilarity(items1, tt.items2)
			assert.GreaterOrEqual(t, similarity, tt.expectedMin)
			assert.LessOrEqual(t, similarity, tt.expectedMax)
		})
	}
}

// Helper function
func floatPtr(f float64) *float64 {
	return &f
}
