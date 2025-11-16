package templates

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestRepository_CreateTemplate(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	userID := uuid.New()
	templateID := uuid.New()

	req := CreateTemplateRequest{
		Name:     "My Breakfast",
		PhotoURL: stringPtr("https://example.com/photo.jpg"),
		Items: []CreateTemplateItemRequest{
			{
				FoodID:      uuid.New(),
				ServingSize: 1.0,
				ServingUnit: "cup",
			},
		},
	}

	// Mock RPC function call
	mock.ExpectQuery(`SELECT create_template`).
		WithArgs(userID, req.Name, req.PhotoURL, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"create_template"}).AddRow(templateID))

	// Mock GetTemplateByID queries
	mock.ExpectQuery(`SELECT id, user_id, name`).
		WithArgs(templateID, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "photo_url", "total_calories", "total_protein", "total_carbs", "total_fat", "created_at", "updated_at",
		}).AddRow(templateID, userID, "My Breakfast", "https://example.com/photo.jpg", 300.0, 20.0, 30.0, 10.0, time.Now(), time.Now()))

	mock.ExpectQuery(`SELECT id, template_id, food_id`).
		WithArgs(templateID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "template_id", "food_id", "food_name", "serving_size", "serving_unit", "calories", "protein", "carbs", "fat", "food_photo_url", "created_at",
		}))

	template, err := repo.CreateTemplate(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "My Breakfast", template.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateFromMeal(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	userID := uuid.New()
	mealID := uuid.New()
	templateID := uuid.New()

	req := CreateTemplateFromMealRequest{
		MealID: mealID,
		Name:   "Lunch Template",
	}

	// Mock RPC function call
	mock.ExpectQuery(`SELECT create_template_from_meal`).
		WithArgs(userID, req.MealID, req.Name).
		WillReturnRows(sqlmock.NewRows([]string{"create_template_from_meal"}).AddRow(templateID))

	// Mock GetTemplateByID
	mock.ExpectQuery(`SELECT id, user_id, name`).
		WithArgs(templateID, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "photo_url", "total_calories", "total_protein", "total_carbs", "total_fat", "created_at", "updated_at",
		}).AddRow(templateID, userID, "Lunch Template", nil, 500.0, 30.0, 40.0, 15.0, time.Now(), time.Now()))

	mock.ExpectQuery(`SELECT id, template_id, food_id`).
		WithArgs(templateID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "template_id", "food_id", "food_name", "serving_size", "serving_unit", "calories", "protein", "carbs", "fat", "food_photo_url", "created_at",
		}))

	template, err := repo.CreateFromMeal(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "Lunch Template", template.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetTemplateByID(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	userID := uuid.New()
	templateID := uuid.New()
	foodID := uuid.New()

	// Mock template query
	mock.ExpectQuery(`SELECT id, user_id, name`).
		WithArgs(templateID, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "photo_url", "total_calories", "total_protein", "total_carbs", "total_fat", "created_at", "updated_at",
		}).AddRow(templateID, userID, "Test Template", nil, 400.0, 25.0, 35.0, 12.0, time.Now(), time.Now()))

	// Mock items query
	mock.ExpectQuery(`SELECT id, template_id, food_id`).
		WithArgs(templateID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "template_id", "food_id", "food_name", "serving_size", "serving_unit", "calories", "protein", "carbs", "fat", "food_photo_url", "created_at",
		}).AddRow(uuid.New(), templateID, foodID, "Chicken Breast", 100.0, "g", 165.0, 31.0, 0.0, 3.6, nil, time.Now()))

	template, err := repo.GetTemplateByID(context.Background(), userID, templateID)

	assert.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "Test Template", template.Name)
	assert.Len(t, template.Items, 1)
	assert.Equal(t, "Chicken Breast", template.Items[0].FoodName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_ListTemplates(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	userID := uuid.New()

	// Mock count query
	mock.ExpectQuery(`SELECT COUNT`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Mock templates query
	templateID1 := uuid.New()
	templateID2 := uuid.New()
	mock.ExpectQuery(`SELECT id, user_id, name`).
		WithArgs(userID, 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "photo_url", "total_calories", "total_protein", "total_carbs", "total_fat", "created_at", "updated_at",
		}).
			AddRow(templateID1, userID, "Template 1", nil, 300.0, 20.0, 30.0, 10.0, time.Now(), time.Now()).
			AddRow(templateID2, userID, "Template 2", nil, 400.0, 25.0, 35.0, 12.0, time.Now(), time.Now()))

	// Mock items queries for each template
	mock.ExpectQuery(`SELECT id, template_id, food_id`).
		WithArgs(templateID1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "template_id", "food_id", "food_name", "serving_size", "serving_unit", "calories", "protein", "carbs", "fat", "food_photo_url", "created_at",
		}))

	mock.ExpectQuery(`SELECT id, template_id, food_id`).
		WithArgs(templateID2).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "template_id", "food_id", "food_name", "serving_size", "serving_unit", "calories", "protein", "carbs", "fat", "food_photo_url", "created_at",
		}))

	templates, total, err := repo.ListTemplates(context.Background(), userID, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, templates, 2)
	assert.Equal(t, 2, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateTemplate(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	userID := uuid.New()
	templateID := uuid.New()

	newName := "Updated Template"
	req := UpdateTemplateRequest{
		Name: &newName,
	}

	// Mock update query
	mock.ExpectExec(`UPDATE nutrition.templates`).
		WithArgs(newName, templateID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Mock GetTemplateByID
	mock.ExpectQuery(`SELECT id, user_id, name`).
		WithArgs(templateID, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "photo_url", "total_calories", "total_protein", "total_carbs", "total_fat", "created_at", "updated_at",
		}).AddRow(templateID, userID, newName, nil, 300.0, 20.0, 30.0, 10.0, time.Now(), time.Now()))

	mock.ExpectQuery(`SELECT id, template_id, food_id`).
		WithArgs(templateID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "template_id", "food_id", "food_name", "serving_size", "serving_unit", "calories", "protein", "carbs", "fat", "food_photo_url", "created_at",
		}))

	template, err := repo.UpdateTemplate(context.Background(), userID, templateID, req)

	assert.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, newName, template.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteTemplate(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	userID := uuid.New()
	templateID := uuid.New()

	// Mock delete query
	mock.ExpectExec(`UPDATE nutrition.templates`).
		WithArgs(templateID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteTemplate(context.Background(), userID, templateID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteTemplate_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	userID := uuid.New()
	templateID := uuid.New()

	// Mock delete query with no rows affected
	mock.ExpectExec(`UPDATE nutrition.templates`).
		WithArgs(templateID, userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.DeleteTemplate(context.Background(), userID, templateID)

	assert.ErrorIs(t, err, ErrTemplateNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
