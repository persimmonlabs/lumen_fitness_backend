package meals

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for meal data access
type Repository interface {
	CreateMealWithItems(ctx context.Context, userID uuid.UUID, meal *Meal, items []MealItem) (*MealWithItems, error)
	CreateDraftMeal(ctx context.Context, userID uuid.UUID, meal *Meal) (uuid.UUID, error)
	UpdateDraftStatus(ctx context.Context, draftID uuid.UUID, status DraftStatus, items []MealItem, errMsg *string) error
	GetDraftStatus(ctx context.Context, userID, draftID uuid.UUID) (*Meal, []MealItem, error)
	GetMealByID(ctx context.Context, userID, mealID uuid.UUID) (*MealWithItems, error)
	ListMealsByUser(ctx context.Context, userID uuid.UUID, filters ListMealFilters) ([]MealListItem, int, error)
	UpdateMeal(ctx context.Context, userID uuid.UUID, meal *Meal, items []MealItem) (*MealWithItems, error)
	DeleteMeal(ctx context.Context, userID, mealID uuid.UUID) error
	DuplicateMeal(ctx context.Context, userID, mealID uuid.UUID, consumedAt time.Time, mealType MealType) (*MealWithItems, error)
	GetMealSuggestions(ctx context.Context, userID uuid.UUID, mealType MealType) ([]MealSuggestion, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates a new meal repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

// CreateMealWithItems creates a meal with items.
// NOTE: Database triggers (migration 012) automatically calculate total_* fields from meal_items.
// DO NOT pass total values - they will be overwritten by triggers after item insertion.
func (r *repository) CreateMealWithItems(ctx context.Context, userID uuid.UUID, meal *Meal, items []MealItem) (*MealWithItems, error) {
	// Convert items to JSON for RPC function
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("marshal items: %w", err)
	}

	photosJSON, err := json.Marshal(meal.Photos)
	if err != nil {
		return nil, fmt.Errorf("marshal photos: %w", err)
	}

	// NOTE: Total nutrition values (params $6-$10) are set to 0
	// Database triggers will automatically calculate them from meal_items
	query := `
		SELECT create_meal_with_items(
			$1::uuid, $2::text, $3::timestamptz, $4::jsonb,
			$5::text, $6::float, $7::float, $8::float,
			$9::float, $10::float, $11::jsonb
		)
	`

	var mealID uuid.UUID
	err = r.db.QueryRowContext(ctx, query,
		userID,
		meal.MealType,
		meal.ConsumedAt,
		photosJSON,
		meal.Notes,
		0.0, // total_calories - triggers will calculate
		0.0, // total_protein_g - triggers will calculate
		0.0, // total_carbs_g - triggers will calculate
		0.0, // total_fat_g - triggers will calculate
		0.0, // total_fiber_g - triggers will calculate
		itemsJSON,
	).Scan(&mealID)
	if err != nil {
		return nil, fmt.Errorf("create meal: %w", err)
	}

	// Fetch the created meal with items (totals will be calculated by triggers)
	return r.GetMealByID(ctx, userID, mealID)
}

// GetMealByID retrieves a meal with its items
func (r *repository) GetMealByID(ctx context.Context, userID, mealID uuid.UUID) (*MealWithItems, error) {
	var meal Meal
	query := `
		SELECT id, user_id, meal_type, consumed_at, photos, notes,
			   total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
			   created_at, updated_at, deleted_at
		FROM meals
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	var photosJSON []byte
	err := r.db.QueryRowContext(ctx, query, mealID, userID).Scan(
		&meal.ID, &meal.UserID, &meal.MealType, &meal.ConsumedAt,
		&photosJSON, &meal.Notes, &meal.TotalCalories, &meal.TotalProteinG,
		&meal.TotalCarbsG, &meal.TotalFatG, &meal.TotalFiberG,
		&meal.CreatedAt, &meal.UpdatedAt, &meal.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("meal not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get meal: %w", err)
	}

	if err := json.Unmarshal(photosJSON, &meal.Photos); err != nil {
		return nil, fmt.Errorf("unmarshal photos: %w", err)
	}

	// Fetch items
	itemsQuery := `
		SELECT id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber, created_at
		FROM meal_items
		WHERE meal_id = $1
		ORDER BY created_at
	`

	var items []MealItem
	err = r.db.SelectContext(ctx, &items, itemsQuery, mealID)
	if err != nil {
		return nil, fmt.Errorf("get items: %w", err)
	}

	return &MealWithItems{
		Meal:  meal,
		Items: items,
	}, nil
}

// ListMealsByUser retrieves paginated meals for a user
func (r *repository) ListMealsByUser(ctx context.Context, userID uuid.UUID, filters ListMealFilters) ([]MealListItem, int, error) {
	query := `
		SELECT m.id, m.meal_type, m.consumed_at, m.total_calories, m.total_protein_g,
			   m.total_carbs_g, m.total_fat_g, m.total_fiber_g,
			   COUNT(mi.id) as item_count,
			   CASE WHEN jsonb_array_length(m.photos) > 0 THEN true ELSE false END as has_photos,
			   m.created_at
		FROM meals m
		LEFT JOIN meal_items mi ON m.id = mi.meal_id
		WHERE m.user_id = $1 AND m.deleted_at IS NULL
	`

	args := []interface{}{userID}

	// Use len(args) to track parameter position dynamically
	if filters.Date != nil {
		query += fmt.Sprintf(" AND DATE(m.consumed_at) = DATE($%d)", len(args)+1)
		args = append(args, *filters.Date)
	}

	if filters.MealType != nil {
		query += fmt.Sprintf(" AND m.meal_type = $%d", len(args)+1)
		args = append(args, *filters.MealType)
	}

	query += " GROUP BY m.id ORDER BY m.consumed_at DESC"

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") as total"
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count meals: %w", err)
	}

	// Add pagination - use len(args) for correct parameter numbering
	offset := (filters.Page - 1) * filters.Limit
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, filters.Limit, offset)

	var meals []MealListItem
	err = r.db.SelectContext(ctx, &meals, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list meals: %w", err)
	}

	return meals, total, nil
}

// UpdateMeal updates a meal and its items.
// NOTE: Database triggers (migration 012) automatically calculate total_* fields from meal_items.
// DO NOT manually set totals - they are recalculated after item INSERT/UPDATE/DELETE.
func (r *repository) UpdateMeal(ctx context.Context, userID uuid.UUID, meal *Meal, items []MealItem) (*MealWithItems, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	photosJSON, err := json.Marshal(meal.Photos)
	if err != nil {
		return nil, fmt.Errorf("marshal photos: %w", err)
	}

	// Update meal metadata only (NOT totals - triggers handle those)
	updateQuery := `
		UPDATE meals
		SET meal_type = $1, consumed_at = $2, photos = $3, notes = $4, updated_at = NOW()
		WHERE id = $5 AND user_id = $6 AND deleted_at IS NULL
	`

	result, err := tx.ExecContext(ctx, updateQuery,
		meal.MealType, meal.ConsumedAt, photosJSON, meal.Notes,
		meal.ID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("update meal: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("meal not found")
	}

	// Delete existing items (triggers will update meal totals to 0)
	_, err = tx.ExecContext(ctx, "DELETE FROM meal_items WHERE meal_id = $1", meal.ID)
	if err != nil {
		return nil, fmt.Errorf("delete items: %w", err)
	}

	// Insert new items (triggers will recalculate meal totals)
	for _, item := range items {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, item.ID, meal.ID, item.Name, item.Quantity, item.Unit,
			item.Calories, item.ProteinG, item.CarbsG, item.FatG, item.FiberG)
		if err != nil {
			return nil, fmt.Errorf("insert item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// Fetch updated meal with correctly calculated totals from database triggers
	return r.GetMealByID(ctx, userID, meal.ID)
}

// DeleteMeal soft deletes a meal
func (r *repository) DeleteMeal(ctx context.Context, userID, mealID uuid.UUID) error {
	query := `UPDATE meals SET deleted_at = NOW() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, mealID, userID)
	if err != nil {
		return fmt.Errorf("delete meal: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("meal not found")
	}

	return nil
}

// DuplicateMeal creates a copy of a meal with new consumed_at
func (r *repository) DuplicateMeal(ctx context.Context, userID, mealID uuid.UUID, consumedAt time.Time, mealType MealType) (*MealWithItems, error) {
	// Fetch original meal
	original, err := r.GetMealByID(ctx, userID, mealID)
	if err != nil {
		return nil, err
	}

	// Create new meal with copied data
	newMeal := original.Meal
	newMeal.ID = uuid.New()
	newMeal.ConsumedAt = consumedAt
	newMeal.MealType = mealType
	newMeal.CreatedAt = time.Now().UTC()
	newMeal.UpdatedAt = time.Now().UTC()

	// Create new items
	newItems := make([]MealItem, len(original.Items))
	for i, item := range original.Items {
		newItems[i] = item
		newItems[i].ID = uuid.New()
		newItems[i].MealID = newMeal.ID
	}

	return r.CreateMealWithItems(ctx, userID, &newMeal, newItems)
}

// CreateDraftMeal creates a draft meal with status='analyzing'
func (r *repository) CreateDraftMeal(ctx context.Context, userID uuid.UUID, meal *Meal) (uuid.UUID, error) {
	draftID := uuid.New()
	status := DraftStatusAnalyzing

	photosJSON, err := json.Marshal(meal.Photos)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal photos: %w", err)
	}

	query := `
		INSERT INTO meals (
			id, user_id, meal_type, consumed_at, photos, notes,
			total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
			is_draft, draft_status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW()
		)
	`

	_, err = r.db.ExecContext(ctx, query,
		draftID, userID, meal.MealType, meal.ConsumedAt, photosJSON, meal.Notes,
		0, 0, 0, 0, 0, // Zero nutrition values initially
		true, status, // is_draft=true, status='analyzing'
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create draft meal: %w", err)
	}

	return draftID, nil
}

// UpdateDraftStatus updates the draft status and optionally adds items.
// NOTE: Database triggers (migration 012) automatically calculate meal totals from items.
// DO NOT manually calculate totals - insert items and let triggers handle it.
func (r *repository) UpdateDraftStatus(ctx context.Context, draftID uuid.UUID, status DraftStatus, items []MealItem, errMsg *string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Update meal status only (NOT totals - triggers calculate those from items)
	updateQuery := `
		UPDATE meals
		SET draft_status = $1, draft_error = $2, updated_at = NOW()
		WHERE id = $3 AND is_draft = TRUE
	`

	result, err := tx.ExecContext(ctx, updateQuery,
		status, errMsg, draftID,
	)
	if err != nil {
		return fmt.Errorf("update draft status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("draft meal not found")
	}

	// Insert items if provided and status is ready
	// Database triggers will automatically calculate and update meal totals
	if status == DraftStatusReady && len(items) > 0 {
		for _, item := range items {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			`, item.ID, draftID, item.Name, item.Quantity, item.Unit,
				item.Calories, item.ProteinG, item.CarbsG, item.FatG, item.FiberG)
			if err != nil {
				return fmt.Errorf("insert item: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// GetDraftStatus retrieves a draft meal and its items
func (r *repository) GetDraftStatus(ctx context.Context, userID, draftID uuid.UUID) (*Meal, []MealItem, error) {
	var meal Meal
	query := `
		SELECT id, user_id, meal_type, consumed_at, photos, notes,
		       total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
		       is_draft, draft_status, draft_error,
		       created_at, updated_at, deleted_at
		FROM meals
		WHERE id = $1 AND user_id = $2 AND is_draft = TRUE AND deleted_at IS NULL
	`

	var photosJSON []byte
	err := r.db.QueryRowContext(ctx, query, draftID, userID).Scan(
		&meal.ID, &meal.UserID, &meal.MealType, &meal.ConsumedAt,
		&photosJSON, &meal.Notes, &meal.TotalCalories, &meal.TotalProteinG,
		&meal.TotalCarbsG, &meal.TotalFatG, &meal.TotalFiberG,
		&meal.IsDraft, &meal.DraftStatus, &meal.DraftError,
		&meal.CreatedAt, &meal.UpdatedAt, &meal.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("draft meal not found")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get draft meal: %w", err)
	}

	if err := json.Unmarshal(photosJSON, &meal.Photos); err != nil {
		return nil, nil, fmt.Errorf("unmarshal photos: %w", err)
	}

	// Fetch items if ready
	var items []MealItem
	if meal.DraftStatus != nil && *meal.DraftStatus == DraftStatusReady {
		itemsQuery := `
			SELECT id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber, created_at
			FROM meal_items
			WHERE meal_id = $1
			ORDER BY created_at
		`
		err = r.db.SelectContext(ctx, &items, itemsQuery, draftID)
		if err != nil {
			return nil, nil, fmt.Errorf("get items: %w", err)
		}
	}

	return &meal, items, nil
}
