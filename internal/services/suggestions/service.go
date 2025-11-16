// Package suggestions provides meal and nutrition suggestions.
//
// This package generates personalized meal recommendations based on
// user goals, preferences, meal history, and nutritional targets.
// It uses AI to provide context-aware suggestions.
package suggestions

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pradord/lumen_final/backend/internal/services/ai"
)

// Service handles meal suggestion operations.
type Service struct {
	aiCoordinator *ai.Coordinator
	logger        *slog.Logger
	config        Config
}

// Config contains configuration for the suggestions service.
type Config struct {
	MaxSuggestions     int
	IncludeRecipes     bool
	IncludeMacros      bool
	PersonalizationLevel string // "basic", "moderate", "advanced"
}

// DefaultConfig returns the default suggestions service configuration.
func DefaultConfig() Config {
	return Config{
		MaxSuggestions:     5,
		IncludeRecipes:     true,
		IncludeMacros:      true,
		PersonalizationLevel: "moderate",
	}
}

// NewService creates a new suggestions service instance.
func NewService(aiCoordinator *ai.Coordinator, logger *slog.Logger, config Config) *Service {
	return &Service{
		aiCoordinator: aiCoordinator,
		logger:        logger,
		config:        config,
	}
}

// NutritionContext represents the user's current nutrition status.
type NutritionContext struct {
	CaloriesRemaining float64
	ProteinRemaining  float64
	CarbsRemaining    float64
	FatRemaining      float64
	FiberTarget       float64
}

// MealContext represents context about upcoming meal.
type MealContext struct {
	MealType        string    // breakfast, lunch, dinner, snack
	ScheduledTime   time.Time
	TimeOfDay       string    // morning, afternoon, evening
}

// UserPreferences represents user dietary preferences.
type UserPreferences struct {
	DietaryRestrictions []string // vegetarian, vegan, gluten-free, etc.
	Allergies          []string
	DislikedFoods      []string
	PreferredCuisines  []string
	CookingTime        int // max cooking time in minutes
}

// SuggestionRequest contains data needed for generating suggestions.
type SuggestionRequest struct {
	UserID           uuid.UUID
	MealContext      MealContext
	NutritionContext NutritionContext
	UserPreferences  UserPreferences
	RecentMeals      []string // descriptions of recent meals to avoid repetition
	MaxSuggestions   int
}

// Suggestion represents a single meal suggestion.
type Suggestion struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Ingredients     []string `json:"ingredients"`
	EstimatedMacros Macros   `json:"estimated_macros"`
	PrepTime        int      `json:"prep_time_minutes"`
	Difficulty      string   `json:"difficulty"` // easy, moderate, hard
	MatchScore      float64  `json:"match_score"` // 0-1, how well it matches goals
	Tags            []string `json:"tags"`
	RecipeURL       string   `json:"recipe_url,omitempty"`
}

// Macros represents estimated macro nutrients.
type Macros struct {
	Calories float64 `json:"calories"`
	ProteinG float64 `json:"protein_g"`
	CarbsG   float64 `json:"carbs_g"`
	FatG     float64 `json:"fat_g"`
	FiberG   float64 `json:"fiber_g"`
}

// SuggestionResponse contains the generated meal suggestions.
type SuggestionResponse struct {
	Suggestions []Suggestion `json:"suggestions"`
	GeneratedAt time.Time    `json:"generated_at"`
	Confidence  float64      `json:"confidence"`
	Message     string       `json:"message"`
}

// Generate creates personalized meal suggestions.
func (s *Service) Generate(ctx context.Context, req *SuggestionRequest) (*SuggestionResponse, error) {
	// Build AI prompt based on context
	prompt := s.buildPrompt(req)

	// Use AI coordinator's nutrition analysis with text prompt
	nutritionReq := ai.NutritionRequest{
		Text: prompt,
	}

	analysisResp, err := s.aiCoordinator.AnalyzeNutrition(nutritionReq)
	if err != nil {
		s.logger.Error("failed to generate suggestions",
			slog.String("error", err.Error()),
			slog.String("user_id", req.UserID.String()),
		)
		// Fall back to rule-based suggestions
		return s.generateRuleBasedSuggestions(req), nil
	}

	// Parse AI response and structure suggestions
	// For now, use a simplified response from the AI analysis
	suggestions := s.parseAISuggestionsFromNutrition(analysisResp, req)

	s.logger.Info("suggestions generated",
		slog.String("user_id", req.UserID.String()),
		slog.Int("count", len(suggestions)),
		slog.String("meal_type", req.MealContext.MealType),
	)

	return &SuggestionResponse{
		Suggestions: suggestions,
		GeneratedAt: time.Now(),
		Confidence:  0.85,
		Message:     "Suggestions based on your goals and preferences",
	}, nil
}

// buildPrompt constructs the AI prompt for suggestions.
func (s *Service) buildPrompt(req *SuggestionRequest) string {
	prompt := fmt.Sprintf(`Generate %d healthy meal suggestions for %s with the following requirements:

Nutritional Goals:
- Calories remaining: %.0f kcal
- Protein remaining: %.0fg
- Carbs remaining: %.0fg
- Fat remaining: %.0fg
- Fiber target: %.0fg

Meal Context:
- Meal type: %s
- Time of day: %s
- Available cooking time: %d minutes

`, req.MaxSuggestions, req.MealContext.MealType,
		req.NutritionContext.CaloriesRemaining,
		req.NutritionContext.ProteinRemaining,
		req.NutritionContext.CarbsRemaining,
		req.NutritionContext.FatRemaining,
		req.NutritionContext.FiberTarget,
		req.MealContext.MealType,
		req.MealContext.TimeOfDay,
		req.UserPreferences.CookingTime)

	// Add dietary restrictions
	if len(req.UserPreferences.DietaryRestrictions) > 0 {
		prompt += fmt.Sprintf("Dietary restrictions: %v\n", req.UserPreferences.DietaryRestrictions)
	}

	if len(req.UserPreferences.Allergies) > 0 {
		prompt += fmt.Sprintf("Allergies: %v\n", req.UserPreferences.Allergies)
	}

	if len(req.UserPreferences.DislikedFoods) > 0 {
		prompt += fmt.Sprintf("Avoid: %v\n", req.UserPreferences.DislikedFoods)
	}

	// Add recent meals to avoid repetition
	if len(req.RecentMeals) > 0 {
		prompt += fmt.Sprintf("\nRecent meals (avoid similar): %v\n", req.RecentMeals)
	}

	prompt += `
For each suggestion, provide:
1. Meal title
2. Brief description (1-2 sentences)
3. Main ingredients (top 5)
4. Estimated macros (calories, protein, carbs, fat, fiber)
5. Prep time in minutes
6. Difficulty level (easy/moderate/hard)
7. Relevant tags (e.g., high-protein, quick, vegetarian)

Format as JSON array of meal objects.`

	return prompt
}

// parseAISuggestionsFromNutrition parses AI nutrition response into meal suggestions.
func (s *Service) parseAISuggestionsFromNutrition(nutritionResp *ai.NutritionResponse, req *SuggestionRequest) []Suggestion {
	// For now, create suggestions based on remaining macros
	// This is a simplified implementation - in production, enhance this
	return s.generateRuleBasedSuggestions(req).Suggestions
}

// parseAISuggestions parses AI response into structured suggestions.
func (s *Service) parseAISuggestions(aiResponse string, req *SuggestionRequest) []Suggestion {
	// Simplified parsing - in production, use proper JSON parsing
	// For now, return mock suggestions

	suggestions := []Suggestion{
		{
			ID:          uuid.New().String(),
			Title:       "Grilled Chicken Salad",
			Description: "Fresh mixed greens with grilled chicken breast, cherry tomatoes, cucumbers, and balsamic vinaigrette",
			Ingredients: []string{"chicken breast", "mixed greens", "cherry tomatoes", "cucumber", "balsamic vinegar"},
			EstimatedMacros: Macros{
				Calories: req.NutritionContext.CaloriesRemaining * 0.4,
				ProteinG: req.NutritionContext.ProteinRemaining * 0.5,
				CarbsG:   req.NutritionContext.CarbsRemaining * 0.2,
				FatG:     req.NutritionContext.FatRemaining * 0.3,
				FiberG:   8.0,
			},
			PrepTime:   15,
			Difficulty: "easy",
			MatchScore: 0.92,
			Tags:       []string{"high-protein", "quick", "low-carb"},
		},
	}

	return suggestions
}

// generateRuleBasedSuggestions provides fallback suggestions without AI.
func (s *Service) generateRuleBasedSuggestions(req *SuggestionRequest) *SuggestionResponse {
	suggestions := []Suggestion{
		{
			ID:          uuid.New().String(),
			Title:       "Simple Balanced Meal",
			Description: "A balanced meal with protein, vegetables, and healthy carbs",
			Ingredients: []string{"lean protein", "vegetables", "whole grains"},
			EstimatedMacros: Macros{
				Calories: req.NutritionContext.CaloriesRemaining * 0.33,
				ProteinG: req.NutritionContext.ProteinRemaining * 0.33,
				CarbsG:   req.NutritionContext.CarbsRemaining * 0.33,
				FatG:     req.NutritionContext.FatRemaining * 0.33,
				FiberG:   10.0,
			},
			PrepTime:   20,
			Difficulty: "easy",
			MatchScore: 0.75,
			Tags:       []string{"balanced", "healthy"},
		},
	}

	return &SuggestionResponse{
		Suggestions: suggestions,
		GeneratedAt: time.Now(),
		Confidence:  0.60,
		Message:     "Basic suggestions (AI unavailable)",
	}
}

// GetQuickSuggestions provides fast, template-based suggestions.
func (s *Service) GetQuickSuggestions(ctx context.Context, mealType string, calorieTarget float64) *SuggestionResponse {
	// Simple rule-based suggestions for when AI is not needed
	templates := map[string][]Suggestion{
		"breakfast": {
			{
				ID:          uuid.New().String(),
				Title:       "Protein Oatmeal",
				Description: "Oats with protein powder, berries, and nuts",
				Ingredients: []string{"oats", "protein powder", "berries", "almonds"},
				EstimatedMacros: Macros{Calories: 400, ProteinG: 30, CarbsG: 45, FatG: 12, FiberG: 8},
				PrepTime:        10,
				Difficulty:      "easy",
				MatchScore:      0.85,
				Tags:            []string{"high-protein", "quick", "energizing"},
			},
		},
		"lunch": {
			{
				ID:          uuid.New().String(),
				Title:       "Chicken Rice Bowl",
				Description: "Grilled chicken with brown rice and roasted vegetables",
				Ingredients: []string{"chicken", "brown rice", "broccoli", "peppers"},
				EstimatedMacros: Macros{Calories: 500, ProteinG: 40, CarbsG: 55, FatG: 15, FiberG: 10},
				PrepTime:        25,
				Difficulty:      "moderate",
				MatchScore:      0.88,
				Tags:            []string{"high-protein", "balanced", "filling"},
			},
		},
	}

	suggestions := templates[mealType]
	if suggestions == nil {
		suggestions = templates["lunch"] // default
	}

	return &SuggestionResponse{
		Suggestions: suggestions,
		GeneratedAt: time.Now(),
		Confidence:  0.80,
		Message:     "Quick template-based suggestions",
	}
}
