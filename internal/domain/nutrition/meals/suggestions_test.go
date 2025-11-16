package meals

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestGetMealSuggestions_Success(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	userID := uuid.New()
	mealType := MealTypeLunch

	// Expected query (with normalized whitespace for matching)
	expectedQuery := `WITH meal_descriptions`

	// Mock rows
	rows := sqlmock.NewRows([]string{
		"description", "avg_calories", "avg_protein", "frequency", "last_consumed_at",
	}).
		AddRow("chicken salad", 350.5, 25.0, 5, time.Now()).
		AddRow("turkey sandwich", 420.0, 30.0, 4, time.Now()).
		AddRow("greek yogurt", 150.0, 15.0, 3, time.Now())

	// Expect query with hour range 11-13 (12:30 ±1 hour)
	mock.ExpectQuery(expectedQuery).
		WithArgs(userID, 11, 13).
		WillReturnRows(rows)

	// Execute
	suggestions, err := repo.GetMealSuggestions(context.Background(), userID, mealType)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(suggestions) != 3 {
		t.Errorf("expected 3 suggestions, got %d", len(suggestions))
	}

	// Verify first suggestion
	if suggestions[0].Description != "chicken salad" {
		t.Errorf("expected description 'chicken salad', got '%s'", suggestions[0].Description)
	}

	if suggestions[0].AvgCalories != 350.5 {
		t.Errorf("expected avg_calories 350.5, got %f", suggestions[0].AvgCalories)
	}

	if suggestions[0].Frequency != 5 {
		t.Errorf("expected frequency 5, got %d", suggestions[0].Frequency)
	}

	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetMealSuggestions_EarlyMorning(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	userID := uuid.New()
	mealType := MealTypeBreakfast

	expectedQuery := `WITH meal_descriptions`

	rows := sqlmock.NewRows([]string{
		"description", "avg_calories", "avg_protein", "frequency", "last_consumed_at",
	})

	// Expect hour range to be 0-1 (can't go negative)
	mock.ExpectQuery(expectedQuery).
		WithArgs(userID, 0, 1).
		WillReturnRows(rows)

	// Execute
	suggestions, err := repo.GetMealSuggestions(context.Background(), userID, mealType)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if suggestions == nil {
		t.Fatal("expected empty slice, got nil")
	}

	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(suggestions))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetMealSuggestions_LateEvening(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	userID := uuid.New()
	mealType := MealTypeSnack

	expectedQuery := `WITH meal_descriptions`

	rows := sqlmock.NewRows([]string{
		"description", "avg_calories", "avg_protein", "frequency", "last_consumed_at",
	}).AddRow("late night snack", 200.0, 10.0, 2, time.Now())

	// Expect hour range to be 22-23 (can't exceed 23)
	mock.ExpectQuery(expectedQuery).
		WithArgs(userID, 22, 23).
		WillReturnRows(rows)

	// Execute
	suggestions, err := repo.GetMealSuggestions(context.Background(), userID, mealType)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(suggestions) != 1 {
		t.Errorf("expected 1 suggestion, got %d", len(suggestions))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetMealSuggestions_NoResults(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	userID := uuid.New()
	mealType := MealTypeDinner

	expectedQuery := `WITH meal_descriptions`

	// Empty result set
	rows := sqlmock.NewRows([]string{
		"description", "avg_calories", "avg_protein", "frequency", "last_consumed_at",
	})

	mock.ExpectQuery(expectedQuery).
		WithArgs(userID, 14, 16).
		WillReturnRows(rows)

	// Execute
	suggestions, err := repo.GetMealSuggestions(context.Background(), userID, mealType)

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty slice, not nil
	if suggestions == nil {
		t.Error("expected empty slice, got nil")
	}

	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(suggestions))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetMealSuggestions_DatabaseError(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	userID := uuid.New()
	mealType := MealTypeLunch

	expectedQuery := `WITH meal_descriptions`

	// Simulate database error
	mock.ExpectQuery(expectedQuery).
		WithArgs(userID, 11, 13).
		WillReturnError(sqlmock.ErrCancelled)

	// Execute
	_, err = repo.GetMealSuggestions(context.Background(), userID, mealType)

	// Verify error is returned
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
