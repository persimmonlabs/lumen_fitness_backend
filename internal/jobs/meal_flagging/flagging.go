package meal_flagging

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Flagger handles meal quality flagging operations
type Flagger struct {
	db     *sql.DB
	config *Config
}

// NewFlagger creates a new meal flagger
func NewFlagger(db *sql.DB, config *Config) *Flagger {
	if config == nil {
		config = DefaultConfig()
	}
	return &Flagger{
		db:     db,
		config: config,
	}
}

// CheckUnusualPortion detects portions that are significantly larger than typical servings
func (f *Flagger) CheckUnusualPortion(meal *MealData) *FlagResult {
	thresholds := DefaultPortionThresholds()

	for _, item := range meal.Items {
		// Check against calorie threshold for any single item
		if item.Calories > 1000 {
			return &FlagResult{
				ShouldFlag:  true,
				FlagType:    FlagTypeUnusualPortion,
				Severity:    "high",
				Description: fmt.Sprintf("Single item '%s' has excessive calories (%.0f cal)", item.Description, item.Calories),
				Details: map[string]interface{}{
					"item_description": item.Description,
					"calories":         item.Calories,
					"threshold":        1000,
				},
			}
		}

		// Check against category-specific thresholds
		for _, threshold := range thresholds {
			if f.matchesCategory(item.Description, threshold.FoodCategory) {
				// Check quantity threshold
				if strings.EqualFold(item.Unit, threshold.Unit) && item.Quantity > threshold.MaxQuantity {
					return &FlagResult{
						ShouldFlag:  true,
						FlagType:    FlagTypeUnusualPortion,
						Severity:    f.calculatePortionSeverity(item.Quantity, threshold.MaxQuantity),
						Description: fmt.Sprintf("Unusual portion: '%s' (%.0f%s exceeds typical %.0f%s)",
							item.Description, item.Quantity, item.Unit, threshold.MaxQuantity, threshold.Unit),
						Details: map[string]interface{}{
							"item_description": item.Description,
							"quantity":         item.Quantity,
							"unit":             item.Unit,
							"threshold":        threshold.MaxQuantity,
							"category":         threshold.FoodCategory,
						},
					}
				}

				// Check calorie threshold for this category
				if item.Calories > threshold.MaxCalories {
					return &FlagResult{
						ShouldFlag:  true,
						FlagType:    FlagTypeUnusualPortion,
						Severity:    "high",
						Description: fmt.Sprintf("Unusual calories: '%s' (%.0f cal exceeds typical %.0f cal for %s)",
							item.Description, item.Calories, threshold.MaxCalories, threshold.FoodCategory),
						Details: map[string]interface{}{
							"item_description": item.Description,
							"calories":         item.Calories,
							"threshold":        threshold.MaxCalories,
							"category":         threshold.FoodCategory,
						},
					}
				}
			}
		}
	}

	return &FlagResult{ShouldFlag: false}
}

// CheckMacroMismatch validates that calories match macronutrient calculation
func (f *Flagger) CheckMacroMismatch(meal *MealData) *FlagResult {
	// Calculate expected calories: Protein*4 + Carbs*4 + Fat*9
	calculatedCalories := (meal.Protein * 4) + (meal.Carbs * 4) + (meal.Fat * 9)

	// Allow for tolerance (default 10%)
	tolerance := f.config.MacroTolerancePercent / 100.0
	lowerBound := calculatedCalories * (1 - tolerance)
	upperBound := calculatedCalories * (1 + tolerance)

	if meal.Calories < lowerBound || meal.Calories > upperBound {
		difference := math.Abs(meal.Calories - calculatedCalories)
		percentDiff := (difference / calculatedCalories) * 100

		return &FlagResult{
			ShouldFlag:  true,
			FlagType:    FlagTypeMacroMismatch,
			Severity:    f.calculateMacroSeverity(percentDiff),
			Description: fmt.Sprintf("Macro mismatch: reported %.0f cal vs calculated %.0f cal (%.1f%% difference)",
				meal.Calories, calculatedCalories, percentDiff),
			Details: map[string]interface{}{
				"reported_calories":   meal.Calories,
				"calculated_calories": calculatedCalories,
				"protein":             meal.Protein,
				"carbs":               meal.Carbs,
				"fat":                 meal.Fat,
				"difference":          difference,
				"percent_difference":  percentDiff,
				"tolerance_percent":   f.config.MacroTolerancePercent,
			},
		}
	}

	return &FlagResult{ShouldFlag: false}
}

// CheckDuplicate finds meals within 30 minutes with similar items
func (f *Flagger) CheckDuplicate(ctx context.Context, meal *MealData) *FlagResult {
	// Query for meals within the time window
	windowMinutes := f.config.DuplicateWindowMinutes
	startTime := meal.LoggedAt.Add(-time.Duration(windowMinutes) * time.Minute)
	endTime := meal.LoggedAt.Add(time.Duration(windowMinutes) * time.Minute)

	query := `
		SELECT m.id, m.name, m.description, m.logged_at
		FROM meals m
		WHERE m.user_id = $1
		AND m.id != $2
		AND m.logged_at BETWEEN $3 AND $4
		ORDER BY m.logged_at DESC
	`

	rows, err := f.db.QueryContext(ctx, query, meal.UserID, meal.ID, startTime, endTime)
	if err != nil {
		return &FlagResult{ShouldFlag: false}
	}
	defer rows.Close()

	for rows.Next() {
		var otherID uuid.UUID
		var otherName, otherDescription string
		var otherLoggedAt time.Time

		if err := rows.Scan(&otherID, &otherName, &otherDescription, &otherLoggedAt); err != nil {
			continue
		}

		// Get items for the other meal
		otherItems, err := f.getMealItems(ctx, otherID)
		if err != nil {
			continue
		}

		// Calculate similarity
		similarity := f.calculateMealSimilarity(meal.Items, otherItems)
		threshold := f.config.DuplicateSimilarityPercent

		if similarity >= threshold {
			timeDiff := meal.LoggedAt.Sub(otherLoggedAt)

			return &FlagResult{
				ShouldFlag:  true,
				FlagType:    FlagTypeDuplicate,
				Severity:    f.calculateDuplicateSeverity(similarity),
				Description: fmt.Sprintf("Possible duplicate: %.0f%% similar to meal logged %v earlier",
					similarity, timeDiff.Round(time.Minute)),
				Details: map[string]interface{}{
					"other_meal_id":   otherID,
					"other_meal_name": otherName,
					"similarity":      similarity,
					"time_difference": timeDiff.String(),
					"threshold":       threshold,
				},
			}
		}
	}

	return &FlagResult{ShouldFlag: false}
}

// CheckLowConfidence flags meals with AI confidence below threshold
func (f *Flagger) CheckLowConfidence(meal *MealData) *FlagResult {
	if meal.AIConfidence == nil {
		// No AI confidence data, skip this check
		return &FlagResult{ShouldFlag: false}
	}

	confidence := *meal.AIConfidence
	threshold := f.config.ConfidenceThreshold

	if confidence < threshold {
		return &FlagResult{
			ShouldFlag:  true,
			FlagType:    FlagTypeLowConfidence,
			Severity:    f.calculateConfidenceSeverity(confidence),
			Description: fmt.Sprintf("Low AI confidence: %.1f%% (threshold: %.1f%%)",
				confidence*100, threshold*100),
			Details: map[string]interface{}{
				"confidence": confidence,
				"threshold":  threshold,
				"suggestion": "Manual review recommended",
			},
		}
	}

	return &FlagResult{ShouldFlag: false}
}

// FlagMeal inserts a flag record into the database
func (f *Flagger) FlagMeal(ctx context.Context, mealID, userID uuid.UUID, result *FlagResult) error {
	if !result.ShouldFlag {
		return nil
	}

	detailsJSON, err := json.Marshal(result.Details)
	if err != nil {
		return fmt.Errorf("marshal details: %w", err)
	}

	query := `
		INSERT INTO meal_flags (id, meal_id, user_id, flag_type, severity, description, details, resolved, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = f.db.ExecContext(ctx, query,
		uuid.New(),
		mealID,
		userID,
		result.FlagType,
		result.Severity,
		result.Description,
		detailsJSON,
		false,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("insert meal flag: %w", err)
	}

	return nil
}

// Helper functions

func (f *Flagger) matchesCategory(description, category string) bool {
	descLower := strings.ToLower(description)
	catLower := strings.ToLower(category)
	return strings.Contains(descLower, catLower)
}

func (f *Flagger) calculatePortionSeverity(actual, threshold float64) string {
	ratio := actual / threshold
	if ratio > 5 {
		return "high"
	} else if ratio > 3 {
		return "medium"
	}
	return "low"
}

func (f *Flagger) calculateMacroSeverity(percentDiff float64) string {
	if percentDiff > 30 {
		return "high"
	} else if percentDiff > 20 {
		return "medium"
	}
	return "low"
}

func (f *Flagger) calculateDuplicateSeverity(similarity float64) string {
	if similarity > 95 {
		return "high"
	} else if similarity > 90 {
		return "medium"
	}
	return "low"
}

func (f *Flagger) calculateConfidenceSeverity(confidence float64) string {
	if confidence < 0.5 {
		return "high"
	} else if confidence < 0.6 {
		return "medium"
	}
	return "low"
}

func (f *Flagger) getMealItems(ctx context.Context, mealID uuid.UUID) ([]MealItemData, error) {
	query := `
		SELECT id, description, quantity, unit, calories, protein, carbs, fat
		FROM meal_items
		WHERE meal_id = $1
	`

	rows, err := f.db.QueryContext(ctx, query, mealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MealItemData
	for rows.Next() {
		var item MealItemData
		err := rows.Scan(&item.ID, &item.Description, &item.Quantity, &item.Unit,
			&item.Calories, &item.Protein, &item.Carbs, &item.Fat)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (f *Flagger) calculateMealSimilarity(items1, items2 []MealItemData) float64 {
	if len(items1) == 0 || len(items2) == 0 {
		return 0
	}

	matchCount := 0
	totalItems := len(items1)

	for _, item1 := range items1 {
		for _, item2 := range items2 {
			if f.itemsAreSimilar(item1, item2) {
				matchCount++
				break
			}
		}
	}

	return (float64(matchCount) / float64(totalItems)) * 100
}

func (f *Flagger) itemsAreSimilar(item1, item2 MealItemData) bool {
	// Normalize descriptions
	desc1 := strings.ToLower(strings.TrimSpace(item1.Description))
	desc2 := strings.ToLower(strings.TrimSpace(item2.Description))

	// Check for exact match or substring match
	if desc1 == desc2 {
		return true
	}

	// Check if one contains the other (handles variations like "chicken breast" vs "breast chicken")
	if strings.Contains(desc1, desc2) || strings.Contains(desc2, desc1) {
		return true
	}

	// Check word overlap (at least 50% of words match)
	words1 := strings.Fields(desc1)
	words2 := strings.Fields(desc2)

	if len(words1) == 0 || len(words2) == 0 {
		return false
	}

	matchingWords := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				matchingWords++
				break
			}
		}
	}

	similarity := (float64(matchingWords) / float64(len(words1))) * 100
	return similarity >= 50
}
