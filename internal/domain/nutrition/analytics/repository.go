package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository handles data access for nutrition analytics
type Repository interface {
	GetDailyNutrition(ctx context.Context, userID string, date time.Time, timezone string) (*DailyTotals, error)
	GetDateRangeNutrition(ctx context.Context, userID string, startDate, endDate time.Time, timezone string) ([]DailyTotals, error)
	GetUserGoals(ctx context.Context, userID string) (*UserGoals, error)
	GetLoggingStreak(ctx context.Context, userID string, endDate time.Time, timezone string) (int, error)
}

type repository struct {
	db *sql.DB
}

// NewRepository creates a new analytics repository
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// GetDailyNutrition retrieves aggregated nutrition data for a specific day using RPC
func (r *repository) GetDailyNutrition(ctx context.Context, userID string, date time.Time, timezone string) (*DailyTotals, error) {
	// Use RPC function get_daily_nutrition
	query := `
		SELECT * FROM get_daily_nutrition($1, $2, $3)
	`

	rows, err := r.db.QueryContext(ctx, query, userID, date.Format("2006-01-02"), timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily nutrition: %w", err)
	}
	defer rows.Close()

	dailyTotals := &DailyTotals{
		Date:          date,
		Timezone:      timezone,
		MealBreakdown: make([]MealBreakdown, 0),
	}

	mealMap := make(map[string]*MealBreakdown)

	for rows.Next() {
		var (
			mealType      string
			calories      float64
			protein       float64
			carbs         float64
			fat           float64
			fiber         float64
		)

		if err := rows.Scan(&mealType, &calories, &protein, &carbs, &fat, &fiber); err != nil {
			return nil, fmt.Errorf("failed to scan daily nutrition row: %w", err)
		}

		// Aggregate totals
		dailyTotals.TotalCalories += calories
		dailyTotals.TotalProtein += protein
		dailyTotals.TotalCarbs += carbs
		dailyTotals.TotalFat += fat
		dailyTotals.TotalFiber += fiber

		// Track meal breakdown
		if meal, exists := mealMap[mealType]; exists {
			meal.Calories += calories
			meal.Protein += protein
			meal.Carbs += carbs
			meal.Fat += fat
			meal.Fiber += fiber
			meal.ItemCount++
		} else {
			mealMap[mealType] = &MealBreakdown{
				MealType:  mealType,
				Calories:  calories,
				Protein:   protein,
				Carbs:     carbs,
				Fat:       fat,
				Fiber:     fiber,
				ItemCount: 1,
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating daily nutrition rows: %w", err)
	}

	// Convert map to slice
	for _, meal := range mealMap {
		dailyTotals.MealBreakdown = append(dailyTotals.MealBreakdown, *meal)
	}

	return dailyTotals, nil
}

// GetDateRangeNutrition retrieves nutrition data for a date range
func (r *repository) GetDateRangeNutrition(ctx context.Context, userID string, startDate, endDate time.Time, timezone string) ([]DailyTotals, error) {
	result := make([]DailyTotals, 0)

	// Iterate through each day in the range
	currentDate := startDate
	for !currentDate.After(endDate) {
		dailyData, err := r.GetDailyNutrition(ctx, userID, currentDate, timezone)
		if err != nil {
			return nil, fmt.Errorf("failed to get nutrition for %s: %w", currentDate.Format("2006-01-02"), err)
		}

		result = append(result, *dailyData)
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return result, nil
}

// GetUserGoals retrieves user's nutrition goals from profile
func (r *repository) GetUserGoals(ctx context.Context, userID string) (*UserGoals, error) {
	query := `
		SELECT
			user_id,
			COALESCE(calorie_goal, 2000) as calories_goal,
			COALESCE(protein_goal, 150) as protein_goal,
			COALESCE(carb_goal, 200) as carbs_goal,
			COALESCE(fat_goal, 65) as fat_goal,
			COALESCE(fiber_goal, 30) as fiber_goal
		FROM user_profiles
		WHERE user_id = $1
	`

	goals := &UserGoals{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&goals.UserID,
		&goals.CaloriesGoal,
		&goals.ProteinGoal,
		&goals.CarbsGoal,
		&goals.FatGoal,
		&goals.FiberGoal,
	)

	if err == sql.ErrNoRows {
		// Return default goals if profile doesn't exist
		return &UserGoals{
			UserID:        userID,
			CaloriesGoal:  2000,
			ProteinGoal:   150,
			CarbsGoal:     200,
			FatGoal:       65,
			FiberGoal:     30,
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user goals: %w", err)
	}

	return goals, nil
}

// GetLoggingStreak calculates consecutive days with logged nutrition
func (r *repository) GetLoggingStreak(ctx context.Context, userID string, endDate time.Time, timezone string) (int, error) {
	query := `
		WITH RECURSIVE date_series AS (
			SELECT $2::date as log_date, 0 as day_offset
			UNION ALL
			SELECT (log_date - INTERVAL '1 day')::date, day_offset + 1
			FROM date_series
			WHERE day_offset < 365
		),
		logged_days AS (
			SELECT DISTINCT DATE(consumed_at AT TIME ZONE $3) as log_date
			FROM food_logs
			WHERE user_id = $1
			  AND consumed_at AT TIME ZONE $3 <= $2::date
		)
		SELECT COUNT(*)
		FROM date_series ds
		INNER JOIN logged_days ld ON ds.log_date = ld.log_date
		WHERE ds.log_date = (
			SELECT MIN(ds2.log_date)
			FROM date_series ds2
			LEFT JOIN logged_days ld2 ON ds2.log_date = ld2.log_date
			WHERE ld2.log_date IS NULL AND ds2.log_date <= $2::date
		) + INTERVAL '1 day'
		   OR NOT EXISTS (
			SELECT 1
			FROM date_series ds2
			LEFT JOIN logged_days ld2 ON ds2.log_date = ld2.log_date
			WHERE ld2.log_date IS NULL AND ds2.log_date <= $2::date
		)
	`

	var streak int
	err := r.db.QueryRowContext(ctx, query, userID, endDate.Format("2006-01-02"), timezone).Scan(&streak)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to calculate logging streak: %w", err)
	}

	return streak, nil
}
