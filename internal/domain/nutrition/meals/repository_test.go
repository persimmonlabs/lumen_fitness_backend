package meals

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMockDB creates a new mock database and sqlmock instance
func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestListMealsByUser_NoFilters(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()

	filters := ListMealFilters{
		Page:  1,
		Limit: 20,
	}

	// Expected SQL query - verify parameter count and column names
	expectedQuery := `
		SELECT m.id, m.meal_type, m.consumed_at, m.total_calories, m.total_protein_g,
			   m.total_carbs_g, m.total_fat_g, m.total_fiber_g,
			   COUNT\(mi.id\) as item_count,
			   CASE WHEN jsonb_array_length\(m.photos\) > 0 THEN true ELSE false END as has_photos,
			   m.created_at
		FROM meals m
		LEFT JOIN meal_items mi ON m.id = mi.meal_id
		WHERE m.user_id = \$1 AND m.deleted_at IS NULL
		 GROUP BY m.id ORDER BY m.consumed_at DESC
	`

	// Count query should have only userID parameter
	countQuery := "SELECT COUNT\\(\\*\\) FROM \\(" + expectedQuery + "\\) as total"

	// Mock count query - expects 1 parameter (userID)
	mock.ExpectQuery(countQuery).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// Mock main query with pagination - expects 3 parameters (userID, LIMIT, OFFSET)
	mealID := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "meal_type", "consumed_at", "total_calories", "total_protein_g",
		"total_carbs_g", "total_fat_g", "total_fiber_g", "item_count", "has_photos", "created_at",
	}).AddRow(
		mealID, MealTypeBreakfast, now, 350.5, 25.3, 40.2, 15.1, 5.2, 3, true, now,
	)

	mock.ExpectQuery(expectedQuery + " LIMIT \\$2 OFFSET \\$3").
		WithArgs(userID, 20, 0). // userID=$1, LIMIT=$2, OFFSET=$3
		WillReturnRows(rows)

	// Execute
	meals, total, err := repo.ListMealsByUser(ctx, userID, filters)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, meals, 1)
	assert.Equal(t, mealID, meals[0].ID)
	assert.Equal(t, MealTypeBreakfast, meals[0].MealType)
	assert.Equal(t, 350.5, meals[0].TotalCalories)
	assert.Equal(t, 3, meals[0].ItemCount)
	assert.True(t, meals[0].HasPhotos)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListMealsByUser_WithDateFilter(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	filterDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	filters := ListMealFilters{
		Page:  1,
		Limit: 20,
		Date:  &filterDate,
	}

	// Query with Date filter - parameter numbering: $1=userID, $2=Date, $3=LIMIT, $4=OFFSET
	baseQuery := `
		SELECT m.id, m.meal_type, m.consumed_at, m.total_calories, m.total_protein_g,
			   m.total_carbs_g, m.total_fat_g, m.total_fiber_g,
			   COUNT\(mi.id\) as item_count,
			   CASE WHEN jsonb_array_length\(m.photos\) > 0 THEN true ELSE false END as has_photos,
			   m.created_at
		FROM meals m
		LEFT JOIN meal_items mi ON m.id = mi.meal_id
		WHERE m.user_id = \$1 AND m.deleted_at IS NULL
		 AND DATE\(m.consumed_at\) = DATE\(\$2\) GROUP BY m.id ORDER BY m.consumed_at DESC
	`

	// Count query - expects 2 parameters (userID, Date)
	countQuery := "SELECT COUNT\\(\\*\\) FROM \\(" + baseQuery + "\\) as total"
	mock.ExpectQuery(countQuery).
		WithArgs(userID, filterDate).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Main query with pagination - expects 4 parameters (userID, Date, LIMIT, OFFSET)
	mealID := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "meal_type", "consumed_at", "total_calories", "total_protein_g",
		"total_carbs_g", "total_fat_g", "total_fiber_g", "item_count", "has_photos", "created_at",
	}).AddRow(
		mealID, MealTypeLunch, now, 450.0, 30.0, 50.0, 18.0, 6.0, 4, false, now,
	)

	mock.ExpectQuery(baseQuery + " LIMIT \\$3 OFFSET \\$4").
		WithArgs(userID, filterDate, 20, 0). // $1=userID, $2=Date, $3=LIMIT, $4=OFFSET
		WillReturnRows(rows)

	// Execute
	meals, total, err := repo.ListMealsByUser(ctx, userID, filters)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, meals, 1)
	assert.Equal(t, MealTypeLunch, meals[0].MealType)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListMealsByUser_WithMealTypeFilter(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	mealType := MealTypeDinner

	filters := ListMealFilters{
		Page:     1,
		Limit:    20,
		MealType: &mealType,
	}

	// Query with MealType filter - parameter numbering: $1=userID, $2=MealType, $3=LIMIT, $4=OFFSET
	baseQuery := `
		SELECT m.id, m.meal_type, m.consumed_at, m.total_calories, m.total_protein_g,
			   m.total_carbs_g, m.total_fat_g, m.total_fiber_g,
			   COUNT\(mi.id\) as item_count,
			   CASE WHEN jsonb_array_length\(m.photos\) > 0 THEN true ELSE false END as has_photos,
			   m.created_at
		FROM meals m
		LEFT JOIN meal_items mi ON m.id = mi.meal_id
		WHERE m.user_id = \$1 AND m.deleted_at IS NULL
		 AND m.meal_type = \$2 GROUP BY m.id ORDER BY m.consumed_at DESC
	`

	// Count query - expects 2 parameters (userID, MealType)
	countQuery := "SELECT COUNT\\(\\*\\) FROM \\(" + baseQuery + "\\) as total"
	mock.ExpectQuery(countQuery).
		WithArgs(userID, mealType).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	// Main query with pagination - expects 4 parameters (userID, MealType, LIMIT, OFFSET)
	mealID := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "meal_type", "consumed_at", "total_calories", "total_protein_g",
		"total_carbs_g", "total_fat_g", "total_fiber_g", "item_count", "has_photos", "created_at",
	}).AddRow(
		mealID, MealTypeDinner, now, 600.0, 35.0, 60.0, 25.0, 8.0, 5, true, now,
	)

	mock.ExpectQuery(baseQuery + " LIMIT \\$3 OFFSET \\$4").
		WithArgs(userID, mealType, 20, 0). // $1=userID, $2=MealType, $3=LIMIT, $4=OFFSET
		WillReturnRows(rows)

	// Execute
	meals, total, err := repo.ListMealsByUser(ctx, userID, filters)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, meals, 1)
	assert.Equal(t, MealTypeDinner, meals[0].MealType)
	assert.Equal(t, 600.0, meals[0].TotalCalories)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListMealsByUser_WithBothFilters(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	filterDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	mealType := MealTypeBreakfast

	filters := ListMealFilters{
		Page:     2,
		Limit:    10,
		Date:     &filterDate,
		MealType: &mealType,
	}

	// Query with both filters - parameter numbering: $1=userID, $2=Date, $3=MealType, $4=LIMIT, $5=OFFSET
	baseQuery := `
		SELECT m.id, m.meal_type, m.consumed_at, m.total_calories, m.total_protein_g,
			   m.total_carbs_g, m.total_fat_g, m.total_fiber_g,
			   COUNT\(mi.id\) as item_count,
			   CASE WHEN jsonb_array_length\(m.photos\) > 0 THEN true ELSE false END as has_photos,
			   m.created_at
		FROM meals m
		LEFT JOIN meal_items mi ON m.id = mi.meal_id
		WHERE m.user_id = \$1 AND m.deleted_at IS NULL
		 AND DATE\(m.consumed_at\) = DATE\(\$2\) AND m.meal_type = \$3 GROUP BY m.id ORDER BY m.consumed_at DESC
	`

	// Count query - expects 3 parameters (userID, Date, MealType)
	countQuery := "SELECT COUNT\\(\\*\\) FROM \\(" + baseQuery + "\\) as total"
	mock.ExpectQuery(countQuery).
		WithArgs(userID, filterDate, mealType).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Main query with pagination - expects 5 parameters (userID, Date, MealType, LIMIT, OFFSET)
	// Page 2, Limit 10 means offset = (2-1) * 10 = 10
	mealID := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "meal_type", "consumed_at", "total_calories", "total_protein_g",
		"total_carbs_g", "total_fat_g", "total_fiber_g", "item_count", "has_photos", "created_at",
	}).AddRow(
		mealID, MealTypeBreakfast, now, 320.0, 22.0, 38.0, 12.0, 4.5, 2, false, now,
	)

	mock.ExpectQuery(baseQuery + " LIMIT \\$4 OFFSET \\$5").
		WithArgs(userID, filterDate, mealType, 10, 10). // $1=userID, $2=Date, $3=MealType, $4=LIMIT, $5=OFFSET
		WillReturnRows(rows)

	// Execute
	meals, total, err := repo.ListMealsByUser(ctx, userID, filters)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, meals, 1)
	assert.Equal(t, MealTypeBreakfast, meals[0].MealType)
	assert.Equal(t, 2, meals[0].ItemCount)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListMealsByUser_CountQueryError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()

	filters := ListMealFilters{
		Page:  1,
		Limit: 20,
	}

	expectedQuery := `
		SELECT m.id, m.meal_type, m.consumed_at, m.total_calories, m.total_protein_g,
			   m.total_carbs_g, m.total_fat_g, m.total_fiber_g,
			   COUNT\(mi.id\) as item_count,
			   CASE WHEN jsonb_array_length\(m.photos\) > 0 THEN true ELSE false END as has_photos,
			   m.created_at
		FROM meals m
		LEFT JOIN meal_items mi ON m.id = mi.meal_id
		WHERE m.user_id = \$1 AND m.deleted_at IS NULL
		 GROUP BY m.id ORDER BY m.consumed_at DESC
	`

	countQuery := "SELECT COUNT\\(\\*\\) FROM \\(" + expectedQuery + "\\) as total"

	// Mock count query returning an error
	mock.ExpectQuery(countQuery).
		WithArgs(userID).
		WillReturnError(fmt.Errorf("database error"))

	// Execute
	meals, total, err := repo.ListMealsByUser(ctx, userID, filters)

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "count meals")
	assert.Nil(t, meals)
	assert.Equal(t, 0, total)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMealByID_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	mealID := uuid.New()
	now := time.Now().UTC()

	photos := []string{"photo1.jpg", "photo2.jpg"}
	photosJSON, _ := json.Marshal(photos)

	// Mock meal query
	mealQuery := `
		SELECT id, user_id, meal_type, consumed_at, photos, notes,
			   total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
			   created_at, updated_at, deleted_at
		FROM meals
		WHERE id = \$1 AND user_id = \$2 AND deleted_at IS NULL
	`

	mealRows := sqlmock.NewRows([]string{
		"id", "user_id", "meal_type", "consumed_at", "photos", "notes",
		"total_calories", "total_protein_g", "total_carbs_g", "total_fat_g", "total_fiber_g",
		"created_at", "updated_at", "deleted_at",
	}).AddRow(
		mealID, userID, MealTypeBreakfast, now, photosJSON, "Test notes",
		450.5, 30.2, 45.3, 20.1, 5.5,
		now, now, nil,
	)

	mock.ExpectQuery(mealQuery).
		WithArgs(mealID, userID).
		WillReturnRows(mealRows)

	// Mock items query - CRITICAL: Test correct column names (protein, carbs, fat, fiber - NOT protein_g)
	itemsQuery := `
		SELECT id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber, created_at
		FROM meal_items
		WHERE meal_id = \$1
		ORDER BY created_at
	`

	itemID1 := uuid.New()
	itemID2 := uuid.New()
	itemRows := sqlmock.NewRows([]string{
		"id", "meal_id", "name", "quantity", "unit", "calories", "protein", "carbs", "fat", "fiber", "created_at",
	}).AddRow(
		itemID1, mealID, "Eggs", 2.0, "large", 180.0, 12.6, 1.2, 10.0, 0.0, now,
	).AddRow(
		itemID2, mealID, "Toast", 2.0, "slice", 270.5, 17.6, 44.1, 10.1, 5.5, now,
	)

	mock.ExpectQuery(itemsQuery).
		WithArgs(mealID).
		WillReturnRows(itemRows)

	// Execute
	result, err := repo.GetMealByID(ctx, userID, mealID)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, mealID, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, MealTypeBreakfast, result.MealType)
	assert.Equal(t, "Test notes", result.Notes)
	assert.Equal(t, 450.5, result.TotalCalories)
	assert.Equal(t, 30.2, result.TotalProteinG)
	assert.Equal(t, photos, result.Photos)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, "Eggs", result.Items[0].Name)
	assert.Equal(t, 12.6, result.Items[0].ProteinG)
	assert.Equal(t, "Toast", result.Items[1].Name)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMealByID_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	mealID := uuid.New()

	mealQuery := `
		SELECT id, user_id, meal_type, consumed_at, photos, notes,
			   total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
			   created_at, updated_at, deleted_at
		FROM meals
		WHERE id = \$1 AND user_id = \$2 AND deleted_at IS NULL
	`

	mock.ExpectQuery(mealQuery).
		WithArgs(mealID, userID).
		WillReturnError(sql.ErrNoRows)

	// Execute
	result, err := repo.GetMealByID(ctx, userID, mealID)

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "meal not found")
	assert.Nil(t, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMealByID_InvalidPhotosJSON(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	mealID := uuid.New()
	now := time.Now().UTC()

	mealQuery := `
		SELECT id, user_id, meal_type, consumed_at, photos, notes,
			   total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
			   created_at, updated_at, deleted_at
		FROM meals
		WHERE id = \$1 AND user_id = \$2 AND deleted_at IS NULL
	`

	// Return invalid JSON for photos
	invalidJSON := []byte("invalid json")

	mealRows := sqlmock.NewRows([]string{
		"id", "user_id", "meal_type", "consumed_at", "photos", "notes",
		"total_calories", "total_protein_g", "total_carbs_g", "total_fat_g", "total_fiber_g",
		"created_at", "updated_at", "deleted_at",
	}).AddRow(
		mealID, userID, MealTypeBreakfast, now, invalidJSON, "Test notes",
		450.5, 30.2, 45.3, 20.1, 5.5,
		now, now, nil,
	)

	mock.ExpectQuery(mealQuery).
		WithArgs(mealID, userID).
		WillReturnRows(mealRows)

	// Execute
	result, err := repo.GetMealByID(ctx, userID, mealID)

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal photos")
	assert.Nil(t, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMeal_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	mealID := uuid.New()
	now := time.Now().UTC()

	meal := &Meal{
		ID:            mealID,
		UserID:        userID,
		MealType:      MealTypeLunch,
		ConsumedAt:    now,
		Photos:        []string{"photo1.jpg"},
		Notes:         "Updated notes",
		TotalCalories: 500.0,
		TotalProteinG: 35.0,
		TotalCarbsG:   50.0,
		TotalFatG:     20.0,
		TotalFiberG:   7.0,
	}

	items := []MealItem{
		{
			ID:       uuid.New(),
			MealID:   mealID,
			Name:     "Chicken",
			Quantity: 150.0,
			Unit:     "g",
			Calories: 250.0,
			ProteinG: 30.0,
			CarbsG:   0.0,
			FatG:     12.0,
			FiberG:   0.0,
		},
	}

	photosJSON, _ := json.Marshal(meal.Photos)

	// Begin transaction
	mock.ExpectBegin()

	// Update meal
	updateQuery := `
		UPDATE meals
		SET meal_type = \$1, consumed_at = \$2, photos = \$3, notes = \$4,
		    total_calories = \$5, total_protein_g = \$6, total_carbs_g = \$7,
		    total_fat_g = \$8, total_fiber_g = \$9, updated_at = NOW\(\)
		WHERE id = \$10 AND user_id = \$11 AND deleted_at IS NULL
	`

	mock.ExpectExec(updateQuery).
		WithArgs(
			meal.MealType, meal.ConsumedAt, photosJSON, meal.Notes,
			meal.TotalCalories, meal.TotalProteinG, meal.TotalCarbsG,
			meal.TotalFatG, meal.TotalFiberG, mealID, userID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Delete existing items
	deleteQuery := "DELETE FROM meal_items WHERE meal_id = \\$1"
	mock.ExpectExec(deleteQuery).
		WithArgs(mealID).
		WillReturnResult(sqlmock.NewResult(0, 2))

	// Insert new items - CRITICAL: Test correct column names (protein, carbs, fat, fiber)
	insertQuery := `
		INSERT INTO meal_items \(id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber\)
		VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, \$9, \$10\)
	`

	mock.ExpectExec(insertQuery).
		WithArgs(
			items[0].ID, mealID, items[0].Name, items[0].Quantity, items[0].Unit,
			items[0].Calories, items[0].ProteinG, items[0].CarbsG, items[0].FatG, items[0].FiberG,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Commit transaction
	mock.ExpectCommit()

	// GetMealByID after update
	mealQuery := `
		SELECT id, user_id, meal_type, consumed_at, photos, notes,
			   total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
			   created_at, updated_at, deleted_at
		FROM meals
		WHERE id = \$1 AND user_id = \$2 AND deleted_at IS NULL
	`

	mealRows := sqlmock.NewRows([]string{
		"id", "user_id", "meal_type", "consumed_at", "photos", "notes",
		"total_calories", "total_protein_g", "total_carbs_g", "total_fat_g", "total_fiber_g",
		"created_at", "updated_at", "deleted_at",
	}).AddRow(
		mealID, userID, meal.MealType, meal.ConsumedAt, photosJSON, meal.Notes,
		meal.TotalCalories, meal.TotalProteinG, meal.TotalCarbsG, meal.TotalFatG, meal.TotalFiberG,
		now, now, nil,
	)

	mock.ExpectQuery(mealQuery).
		WithArgs(mealID, userID).
		WillReturnRows(mealRows)

	itemsQuery := `
		SELECT id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber, created_at
		FROM meal_items
		WHERE meal_id = \$1
		ORDER BY created_at
	`

	itemRows := sqlmock.NewRows([]string{
		"id", "meal_id", "name", "quantity", "unit", "calories", "protein", "carbs", "fat", "fiber", "created_at",
	}).AddRow(
		items[0].ID, mealID, items[0].Name, items[0].Quantity, items[0].Unit,
		items[0].Calories, items[0].ProteinG, items[0].CarbsG, items[0].FatG, items[0].FiberG, now,
	)

	mock.ExpectQuery(itemsQuery).
		WithArgs(mealID).
		WillReturnRows(itemRows)

	// Execute
	result, err := repo.UpdateMeal(ctx, userID, meal, items)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, mealID, result.ID)
	assert.Equal(t, "Updated notes", result.Notes)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, "Chicken", result.Items[0].Name)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMeal_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	mealID := uuid.New()

	meal := &Meal{
		ID:            mealID,
		MealType:      MealTypeLunch,
		Photos:        []string{},
		TotalCalories: 500.0,
	}

	photosJSON, _ := json.Marshal(meal.Photos)

	// Begin transaction
	mock.ExpectBegin()

	// Update meal - no rows affected
	updateQuery := `
		UPDATE meals
		SET meal_type = \$1, consumed_at = \$2, photos = \$3, notes = \$4,
		    total_calories = \$5, total_protein_g = \$6, total_carbs_g = \$7,
		    total_fat_g = \$8, total_fiber_g = \$9, updated_at = NOW\(\)
		WHERE id = \$10 AND user_id = \$11 AND deleted_at IS NULL
	`

	mock.ExpectExec(updateQuery).
		WithArgs(
			meal.MealType, meal.ConsumedAt, photosJSON, meal.Notes,
			meal.TotalCalories, meal.TotalProteinG, meal.TotalCarbsG,
			meal.TotalFatG, meal.TotalFiberG, mealID, userID,
		).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Rollback
	mock.ExpectRollback()

	// Execute
	result, err := repo.UpdateMeal(ctx, userID, meal, []MealItem{})

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "meal not found")
	assert.Nil(t, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMealWithItems_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	mealID := uuid.New()
	now := time.Now().UTC()

	meal := &Meal{
		MealType:      MealTypeBreakfast,
		ConsumedAt:    now,
		Photos:        []string{"photo1.jpg"},
		Notes:         "Breakfast meal",
		TotalCalories: 400.0,
		TotalProteinG: 25.0,
		TotalCarbsG:   40.0,
		TotalFatG:     15.0,
		TotalFiberG:   5.0,
	}

	items := []MealItem{
		{
			ID:       uuid.New(),
			Name:     "Oatmeal",
			Quantity: 1.0,
			Unit:     "cup",
			Calories: 150.0,
			ProteinG: 5.0,
			CarbsG:   27.0,
			FatG:     3.0,
			FiberG:   4.0,
		},
	}

	photosJSON, _ := json.Marshal(meal.Photos)
	itemsJSON, _ := json.Marshal(items)

	// Mock RPC function call
	rpcQuery := `
		SELECT create_meal_with_items\(
			\$1::uuid, \$2::text, \$3::timestamptz, \$4::jsonb,
			\$5::text, \$6::float, \$7::float, \$8::float,
			\$9::float, \$10::float, \$11::jsonb
		\)
	`

	mock.ExpectQuery(rpcQuery).
		WithArgs(
			userID, meal.MealType, meal.ConsumedAt, photosJSON,
			meal.Notes, meal.TotalCalories, meal.TotalProteinG, meal.TotalCarbsG,
			meal.TotalFatG, meal.TotalFiberG, itemsJSON,
		).
		WillReturnRows(sqlmock.NewRows([]string{"create_meal_with_items"}).AddRow(mealID))

	// GetMealByID after creation
	mealQuery := `
		SELECT id, user_id, meal_type, consumed_at, photos, notes,
			   total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
			   created_at, updated_at, deleted_at
		FROM meals
		WHERE id = \$1 AND user_id = \$2 AND deleted_at IS NULL
	`

	mealRows := sqlmock.NewRows([]string{
		"id", "user_id", "meal_type", "consumed_at", "photos", "notes",
		"total_calories", "total_protein_g", "total_carbs_g", "total_fat_g", "total_fiber_g",
		"created_at", "updated_at", "deleted_at",
	}).AddRow(
		mealID, userID, meal.MealType, meal.ConsumedAt, photosJSON, meal.Notes,
		meal.TotalCalories, meal.TotalProteinG, meal.TotalCarbsG, meal.TotalFatG, meal.TotalFiberG,
		now, now, nil,
	)

	mock.ExpectQuery(mealQuery).
		WithArgs(mealID, userID).
		WillReturnRows(mealRows)

	itemsQuery := `
		SELECT id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber, created_at
		FROM meal_items
		WHERE meal_id = \$1
		ORDER BY created_at
	`

	itemRows := sqlmock.NewRows([]string{
		"id", "meal_id", "name", "quantity", "unit", "calories", "protein", "carbs", "fat", "fiber", "created_at",
	}).AddRow(
		items[0].ID, mealID, items[0].Name, items[0].Quantity, items[0].Unit,
		items[0].Calories, items[0].ProteinG, items[0].CarbsG, items[0].FatG, items[0].FiberG, now,
	)

	mock.ExpectQuery(itemsQuery).
		WithArgs(mealID).
		WillReturnRows(itemRows)

	// Execute
	result, err := repo.CreateMealWithItems(ctx, userID, meal, items)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, mealID, result.ID)
	assert.Equal(t, MealTypeBreakfast, result.MealType)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, "Oatmeal", result.Items[0].Name)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMealWithItems_RPCError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now().UTC()

	meal := &Meal{
		MealType:   MealTypeBreakfast,
		ConsumedAt: now,
		Photos:     []string{},
	}

	photosJSON, _ := json.Marshal(meal.Photos)
	itemsJSON, _ := json.Marshal([]MealItem{})

	rpcQuery := `
		SELECT create_meal_with_items\(
			\$1::uuid, \$2::text, \$3::timestamptz, \$4::jsonb,
			\$5::text, \$6::float, \$7::float, \$8::float,
			\$9::float, \$10::float, \$11::jsonb
		\)
	`

	mock.ExpectQuery(rpcQuery).
		WithArgs(
			userID, meal.MealType, meal.ConsumedAt, photosJSON,
			meal.Notes, meal.TotalCalories, meal.TotalProteinG, meal.TotalCarbsG,
			meal.TotalFatG, meal.TotalFiberG, itemsJSON,
		).
		WillReturnError(fmt.Errorf("RPC function error"))

	// Execute
	result, err := repo.CreateMealWithItems(ctx, userID, meal, []MealItem{})

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create meal")
	assert.Nil(t, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListMealsByUser_EmptyResult(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uuid.New()

	filters := ListMealFilters{
		Page:  1,
		Limit: 20,
	}

	expectedQuery := `
		SELECT m.id, m.meal_type, m.consumed_at, m.total_calories, m.total_protein_g,
			   m.total_carbs_g, m.total_fat_g, m.total_fiber_g,
			   COUNT\(mi.id\) as item_count,
			   CASE WHEN jsonb_array_length\(m.photos\) > 0 THEN true ELSE false END as has_photos,
			   m.created_at
		FROM meals m
		LEFT JOIN meal_items mi ON m.id = mi.meal_id
		WHERE m.user_id = \$1 AND m.deleted_at IS NULL
		 GROUP BY m.id ORDER BY m.consumed_at DESC
	`

	countQuery := "SELECT COUNT\\(\\*\\) FROM \\(" + expectedQuery + "\\) as total"

	mock.ExpectQuery(countQuery).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	rows := sqlmock.NewRows([]string{
		"id", "meal_type", "consumed_at", "total_calories", "total_protein_g",
		"total_carbs_g", "total_fat_g", "total_fiber_g", "item_count", "has_photos", "created_at",
	})

	mock.ExpectQuery(expectedQuery + " LIMIT \\$2 OFFSET \\$3").
		WithArgs(userID, 20, 0).
		WillReturnRows(rows)

	// Execute
	meals, total, err := repo.ListMealsByUser(ctx, userID, filters)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, meals, 0)

	assert.NoError(t, mock.ExpectationsWereMet())
}
