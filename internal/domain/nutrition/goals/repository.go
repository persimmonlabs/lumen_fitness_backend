package goals

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository handles goals data persistence
type Repository interface {
	// UserGoals operations
	GetUserGoals(ctx context.Context, userID int64) (*UserGoals, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*UserGoals, error)
	UpdateUserGoals(ctx context.Context, goals *UserGoals) error
	CreateUserGoals(ctx context.Context, goals *UserGoals) error

	// DailyGoal operations
	GetDailyGoals(ctx context.Context, userID int64) ([]DailyGoal, error)
	GetDailyGoal(ctx context.Context, userID int64, day string) (*DailyGoal, error)
	SetDailyGoal(ctx context.Context, goal *DailyGoal) error
	DeleteDailyGoal(ctx context.Context, userID int64, day string) error
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates a new goals repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

// GetUserGoals retrieves user goals by user ID
func (r *repository) GetUserGoals(ctx context.Context, userID int64) (*UserGoals, error) {
	query := `
		SELECT id, user_id, daily_calories, protein_grams, carbs_grams, fat_grams,
		       auto_calculate, activity_level, created_at, updated_at, last_calculated_at
		FROM user_goals
		WHERE user_id = $1`

	var goals UserGoals
	err := r.db.GetContext(ctx, &goals, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGoalsNotFound
		}
		return nil, err
	}

	return &goals, nil
}

// GetByUserID retrieves user goals by UUID (wrapper for GetUserGoals)
func (r *repository) GetByUserID(ctx context.Context, userID uuid.UUID) (*UserGoals, error) {
	query := `
		SELECT id, user_id, daily_calories, protein_grams, carbs_grams, fat_grams,
		       auto_calculate, activity_level, created_at, updated_at, last_calculated_at
		FROM user_goals
		WHERE user_id = $1`

	var goals UserGoals
	err := r.db.GetContext(ctx, &goals, query, userID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGoalsNotFound
		}
		return nil, err
	}

	return &goals, nil
}

// UpdateUserGoals updates existing user goals
func (r *repository) UpdateUserGoals(ctx context.Context, goals *UserGoals) error {
	query := `
		UPDATE user_goals
		SET daily_calories = $1,
		    protein_grams = $2,
		    carbs_grams = $3,
		    fat_grams = $4,
		    auto_calculate = $5,
		    activity_level = $6,
		    last_calculated_at = $7,
		    updated_at = $8
		WHERE user_id = $9`

	goals.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(
		ctx,
		query,
		goals.DailyCalories,
		goals.ProteinGrams,
		goals.CarbsGrams,
		goals.FatGrams,
		goals.AutoCalculate,
		goals.ActivityLevel,
		goals.LastCalculatedAt,
		goals.UpdatedAt,
		goals.UserID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrGoalsNotFound
	}

	return nil
}

// CreateUserGoals creates new user goals
func (r *repository) CreateUserGoals(ctx context.Context, goals *UserGoals) error {
	query := `
		INSERT INTO user_goals (
			user_id, daily_calories, protein_grams, carbs_grams, fat_grams,
			auto_calculate, activity_level, created_at, updated_at, last_calculated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) RETURNING id`

	now := time.Now()
	goals.CreatedAt = now
	goals.UpdatedAt = now

	err := r.db.QueryRowContext(
		ctx,
		query,
		goals.UserID,
		goals.DailyCalories,
		goals.ProteinGrams,
		goals.CarbsGrams,
		goals.FatGrams,
		goals.AutoCalculate,
		goals.ActivityLevel,
		goals.CreatedAt,
		goals.UpdatedAt,
		goals.LastCalculatedAt,
	).Scan(&goals.ID)

	return err
}

// GetDailyGoals retrieves all daily goals for a user
func (r *repository) GetDailyGoals(ctx context.Context, userID int64) ([]DailyGoal, error) {
	query := `
		SELECT id, user_id, day_of_week, daily_calories, protein_grams,
		       carbs_grams, fat_grams, created_at, updated_at
		FROM daily_goals
		WHERE user_id = $1
		ORDER BY
		    CASE day_of_week
		        WHEN 'monday' THEN 1
		        WHEN 'tuesday' THEN 2
		        WHEN 'wednesday' THEN 3
		        WHEN 'thursday' THEN 4
		        WHEN 'friday' THEN 5
		        WHEN 'saturday' THEN 6
		        WHEN 'sunday' THEN 7
		    END`

	var goals []DailyGoal
	err := r.db.SelectContext(ctx, &goals, query, userID)
	if err != nil {
		return nil, err
	}

	return goals, nil
}

// GetDailyGoal retrieves a specific daily goal
func (r *repository) GetDailyGoal(ctx context.Context, userID int64, day string) (*DailyGoal, error) {
	if !IsValidDayOfWeek(day) {
		return nil, ErrInvalidDayOfWeek
	}

	query := `
		SELECT id, user_id, day_of_week, daily_calories, protein_grams,
		       carbs_grams, fat_grams, created_at, updated_at
		FROM daily_goals
		WHERE user_id = $1 AND day_of_week = $2`

	var goal DailyGoal
	err := r.db.GetContext(ctx, &goal, query, userID, day)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDailyGoalNotFound
		}
		return nil, err
	}

	return &goal, nil
}

// SetDailyGoal creates or updates a daily goal
func (r *repository) SetDailyGoal(ctx context.Context, goal *DailyGoal) error {
	if !IsValidDayOfWeek(goal.DayOfWeek) {
		return ErrInvalidDayOfWeek
	}

	// Try to update first
	updateQuery := `
		UPDATE daily_goals
		SET daily_calories = $1,
		    protein_grams = $2,
		    carbs_grams = $3,
		    fat_grams = $4,
		    updated_at = $5
		WHERE user_id = $6 AND day_of_week = $7`

	goal.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(
		ctx,
		updateQuery,
		goal.DailyCalories,
		goal.ProteinGrams,
		goal.CarbsGrams,
		goal.FatGrams,
		goal.UpdatedAt,
		goal.UserID,
		goal.DayOfWeek,
	)

	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		return nil
	}

	// If no rows updated, insert new
	insertQuery := `
		INSERT INTO daily_goals (
			user_id, day_of_week, daily_calories, protein_grams,
			carbs_grams, fat_grams, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING id`

	now := time.Now()
	goal.CreatedAt = now
	goal.UpdatedAt = now

	err = r.db.QueryRowContext(
		ctx,
		insertQuery,
		goal.UserID,
		goal.DayOfWeek,
		goal.DailyCalories,
		goal.ProteinGrams,
		goal.CarbsGrams,
		goal.FatGrams,
		goal.CreatedAt,
		goal.UpdatedAt,
	).Scan(&goal.ID)

	return err
}

// DeleteDailyGoal deletes a specific daily goal
func (r *repository) DeleteDailyGoal(ctx context.Context, userID int64, day string) error {
	if !IsValidDayOfWeek(day) {
		return ErrInvalidDayOfWeek
	}

	query := `DELETE FROM daily_goals WHERE user_id = $1 AND day_of_week = $2`

	result, err := r.db.ExecContext(ctx, query, userID, day)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrDailyGoalNotFound
	}

	return nil
}
