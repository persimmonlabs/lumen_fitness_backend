package meal_flagging

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// MealFlaggingJob handles scheduled meal quality checks
type MealFlaggingJob struct {
	db       *sql.DB
	config   *Config
	flagger  *Flagger
	cron     *cron.Cron
	logger   *log.Logger
	isRunning bool
}

// NewMealFlaggingJob creates a new job instance
func NewMealFlaggingJob(db *sql.DB, config *Config, logger *log.Logger) *MealFlaggingJob {
	if config == nil {
		config = DefaultConfig()
	}
	if logger == nil {
		logger = log.Default()
	}

	return &MealFlaggingJob{
		db:      db,
		config:  config,
		flagger: NewFlagger(db, config),
		cron:    cron.New(),
		logger:  logger,
	}
}

// Start begins the scheduled job
func (j *MealFlaggingJob) Start() error {
	if !j.config.Enabled {
		j.logger.Println("Meal flagging job is disabled")
		return nil
	}

	// Parse schedule time (e.g., "02:00" for 2am)
	cronExpr, err := j.parseCronSchedule(j.config.ScheduleTime)
	if err != nil {
		return fmt.Errorf("parse schedule: %w", err)
	}

	// Add the job to cron
	_, err = j.cron.AddFunc(cronExpr, func() {
		if err := j.Run(context.Background()); err != nil {
			j.logger.Printf("Job execution failed: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("add cron job: %w", err)
	}

	j.cron.Start()
	j.logger.Printf("Meal flagging job started, scheduled for %s daily", j.config.ScheduleTime)
	return nil
}

// Stop halts the scheduled job
func (j *MealFlaggingJob) Stop() {
	j.cron.Stop()
	j.logger.Println("Meal flagging job stopped")
}

// Run executes the job immediately
func (j *MealFlaggingJob) Run(ctx context.Context) error {
	if j.isRunning {
		return fmt.Errorf("job is already running")
	}

	j.isRunning = true
	defer func() { j.isRunning = false }()

	startTime := time.Now()
	j.logger.Println("Starting meal flagging job")

	summary := NewFlagSummary()

	// Calculate time range
	endTime := time.Now()
	startTime = endTime.Add(-time.Duration(j.config.LookbackDays) * 24 * time.Hour)

	// Process meals in batches
	offset := 0
	for {
		meals, err := j.fetchMealBatch(ctx, startTime, endTime, offset)
		if err != nil {
			summary.AddError(fmt.Sprintf("fetch batch at offset %d: %v", offset, err))
			break
		}

		if len(meals) == 0 {
			break
		}

		for _, meal := range meals {
			if err := j.processMeal(ctx, &meal, summary); err != nil {
				summary.AddError(fmt.Sprintf("process meal %s: %v", meal.ID, err))
			}
			summary.TotalMealsChecked++
		}

		offset += j.config.BatchSize

		// Check if we've processed fewer than batch size (last batch)
		if len(meals) < j.config.BatchSize {
			break
		}
	}

	summary.Duration = time.Since(startTime)
	j.logSummary(summary)

	return nil
}

// RunForUser processes meals for a specific user
func (j *MealFlaggingJob) RunForUser(ctx context.Context, userID uuid.UUID) (*FlagSummary, error) {
	j.logger.Printf("Running meal flagging for user %s", userID)

	startTime := time.Now()
	summary := NewFlagSummary()

	// Calculate time range
	endTime := time.Now()
	lookbackStart := endTime.Add(-time.Duration(j.config.LookbackDays) * 24 * time.Hour)

	// Process meals in batches
	offset := 0
	for {
		meals, err := j.fetchUserMealBatch(ctx, userID, lookbackStart, endTime, offset)
		if err != nil {
			summary.AddError(fmt.Sprintf("fetch batch at offset %d: %v", offset, err))
			break
		}

		if len(meals) == 0 {
			break
		}

		for _, meal := range meals {
			if err := j.processMeal(ctx, &meal, summary); err != nil {
				summary.AddError(fmt.Sprintf("process meal %s: %v", meal.ID, err))
			}
			summary.TotalMealsChecked++
		}

		offset += j.config.BatchSize

		if len(meals) < j.config.BatchSize {
			break
		}
	}

	summary.Duration = time.Since(startTime)
	return summary, nil
}

// processMeal runs all flagging checks on a single meal
func (j *MealFlaggingJob) processMeal(ctx context.Context, meal *MealData, summary *FlagSummary) error {
	// Run all checks
	checks := []func(*MealData) *FlagResult{
		j.flagger.CheckUnusualPortion,
		j.flagger.CheckMacroMismatch,
		j.flagger.CheckLowConfidence,
	}

	for _, check := range checks {
		result := check(meal)
		if result.ShouldFlag {
			if err := j.flagger.FlagMeal(ctx, meal.ID, meal.UserID, result); err != nil {
				return err
			}
			summary.AddFlag(result.FlagType, result.Severity)
		}
	}

	// Check for duplicates (requires database query)
	duplicateResult := j.flagger.CheckDuplicate(ctx, meal)
	if duplicateResult.ShouldFlag {
		if err := j.flagger.FlagMeal(ctx, meal.ID, meal.UserID, duplicateResult); err != nil {
			return err
		}
		summary.AddFlag(duplicateResult.FlagType, duplicateResult.Severity)
	}

	return nil
}

// fetchMealBatch retrieves a batch of meals from the database
func (j *MealFlaggingJob) fetchMealBatch(ctx context.Context, startTime, endTime time.Time, offset int) ([]MealData, error) {
	query := `
		SELECT
			m.id, m.user_id, m.name, m.description,
			m.calories, m.protein, m.carbs, m.fat,
			m.logged_at, m.ai_confidence
		FROM meals m
		WHERE m.logged_at BETWEEN $1 AND $2
		AND m.deleted_at IS NULL
		ORDER BY m.logged_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := j.db.QueryContext(ctx, query, startTime, endTime, j.config.BatchSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meals []MealData
	for rows.Next() {
		var meal MealData
		var description sql.NullString
		var aiConfidence sql.NullFloat64

		err := rows.Scan(
			&meal.ID, &meal.UserID, &meal.Name, &description,
			&meal.Calories, &meal.Protein, &meal.Carbs, &meal.Fat,
			&meal.LoggedAt, &aiConfidence,
		)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			meal.Description = description.String
		}
		if aiConfidence.Valid {
			conf := aiConfidence.Float64
			meal.AIConfidence = &conf
		}

		// Load meal items
		items, err := j.flagger.getMealItems(ctx, meal.ID)
		if err != nil {
			return nil, err
		}
		meal.Items = items

		meals = append(meals, meal)
	}

	return meals, nil
}

// fetchUserMealBatch retrieves a batch of meals for a specific user
func (j *MealFlaggingJob) fetchUserMealBatch(ctx context.Context, userID uuid.UUID, startTime, endTime time.Time, offset int) ([]MealData, error) {
	query := `
		SELECT
			m.id, m.user_id, m.name, m.description,
			m.calories, m.protein, m.carbs, m.fat,
			m.logged_at, m.ai_confidence
		FROM meals m
		WHERE m.user_id = $1
		AND m.logged_at BETWEEN $2 AND $3
		AND m.deleted_at IS NULL
		ORDER BY m.logged_at DESC
		LIMIT $4 OFFSET $5
	`

	rows, err := j.db.QueryContext(ctx, query, userID, startTime, endTime, j.config.BatchSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meals []MealData
	for rows.Next() {
		var meal MealData
		var description sql.NullString
		var aiConfidence sql.NullFloat64

		err := rows.Scan(
			&meal.ID, &meal.UserID, &meal.Name, &description,
			&meal.Calories, &meal.Protein, &meal.Carbs, &meal.Fat,
			&meal.LoggedAt, &aiConfidence,
		)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			meal.Description = description.String
		}
		if aiConfidence.Valid {
			conf := aiConfidence.Float64
			meal.AIConfidence = &conf
		}

		// Load meal items
		items, err := j.flagger.getMealItems(ctx, meal.ID)
		if err != nil {
			return nil, err
		}
		meal.Items = items

		meals = append(meals, meal)
	}

	return meals, nil
}

// parseCronSchedule converts a time string like "02:00" to a cron expression
func (j *MealFlaggingJob) parseCronSchedule(timeStr string) (string, error) {
	// Parse time in format "HH:MM"
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return "", err
	}

	// Create cron expression: "minute hour * * *"
	// Example: "02:00" becomes "0 2 * * *" (run at 2am daily)
	return fmt.Sprintf("%d %d * * *", t.Minute(), t.Hour()), nil
}

// logSummary logs the job execution summary
func (j *MealFlaggingJob) logSummary(summary *FlagSummary) {
	j.logger.Printf("=== Meal Flagging Job Summary ===")
	j.logger.Printf("Meals checked: %d", summary.TotalMealsChecked)
	j.logger.Printf("Flags created: %d", summary.TotalFlagsCreated)
	j.logger.Printf("Duration: %v", summary.Duration)

	if len(summary.FlagsByType) > 0 {
		j.logger.Printf("Flags by type:")
		for flagType, count := range summary.FlagsByType {
			j.logger.Printf("  - %s: %d", flagType, count)
		}
	}

	if len(summary.FlagsBySeverity) > 0 {
		j.logger.Printf("Flags by severity:")
		for severity, count := range summary.FlagsBySeverity {
			j.logger.Printf("  - %s: %d", severity, count)
		}
	}

	if len(summary.Errors) > 0 {
		j.logger.Printf("Errors encountered: %d", len(summary.Errors))
		for i, err := range summary.Errors {
			if i < 10 { // Log first 10 errors
				j.logger.Printf("  - %s", err)
			}
		}
		if len(summary.Errors) > 10 {
			j.logger.Printf("  ... and %d more errors", len(summary.Errors)-10)
		}
	}

	j.logger.Printf("================================")
}
