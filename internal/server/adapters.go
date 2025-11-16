// Package server provides adapter implementations to bridge services.
//
// This file contains adapters that connect the AI Coordinator to the meals
// service interfaces, allowing the AI features to work seamlessly with the
// nutrition tracking domain.
package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/meals"
	"github.com/pradord/lumen_final/backend/internal/services/ai"
	"github.com/pradord/lumen_final/backend/internal/services/cache"
)

// aiCoordinatorAdapter adapts ai.Coordinator to meals.AICoordinator interface
type aiCoordinatorAdapter struct {
	coordinator *ai.Coordinator
}

// ParseMeal implements meals.AICoordinator by calling the AI coordinator
func (a *aiCoordinatorAdapter) ParseMeal(ctx context.Context, description string, photos []string) ([]meals.DraftMealItem, float64, float64, error) {
	// Build AI request
	req := ai.NutritionRequest{
		Text:   description,
		UserID: "system", // Will be overridden by actual user context
	}

	// TODO: Add photo support when Supabase storage is integrated
	// For now, only text-based parsing is supported

	// Call AI coordinator
	resp, err := a.coordinator.AnalyzeNutrition(req)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("AI analysis failed: %w", err)
	}

	// Convert AI response to meals domain format
	items := make([]meals.DraftMealItem, len(resp.Items))
	for i, item := range resp.Items {
		items[i] = meals.DraftMealItem{
			Name:     item.Name,
			Quantity: item.Quantity,
			Unit:     item.Unit,
			Calories: item.Nutrition.Calories,
			ProteinG: item.Nutrition.Protein,
			CarbsG:   item.Nutrition.Carbohydrates,
			FatG:     item.Nutrition.Fat,
			FiberG:   item.Nutrition.Fiber,
		}
	}

	return items, resp.Confidence, resp.Cost, nil
}

// aiEstimatorAdapter adapts ai.Coordinator to meals.AIEstimator interface
type aiEstimatorAdapter struct {
	coordinator *ai.Coordinator
}

// EstimateMeal implements meals.AIEstimator by calling the AI coordinator
func (a *aiEstimatorAdapter) EstimateMeal(description string) (*meals.AIEstimation, error) {
	// Use AI coordinator for quick estimation
	req := ai.NutritionRequest{
		Text:   description,
		UserID: "system",
	}

	resp, err := a.coordinator.AnalyzeNutrition(req)
	if err != nil {
		return nil, fmt.Errorf("AI estimation failed: %w", err)
	}

	// Convert to estimation format
	totalCalories := int(resp.TotalNutrition.Calories)
	totalProtein := int(resp.TotalNutrition.Protein)

	// Map confidence to string
	confidence := "medium"
	if resp.Confidence >= 0.8 {
		confidence = "high"
	} else if resp.Confidence < 0.5 {
		confidence = "low"
	}

	return &meals.AIEstimation{
		Calories:   totalCalories,
		Protein:    totalProtein,
		Confidence: confidence,
	}, nil
}

// aiTranscriberAdapter adapts ai.Coordinator to meals.AITranscriber interface
type aiTranscriberAdapter struct {
	coordinator *ai.Coordinator
}

// TranscribeAudio implements meals.AITranscriber by calling the AI coordinator
func (a *aiTranscriberAdapter) TranscribeAudio(audioData []byte, contentType string) (string, error) {
	// Use AI coordinator for audio transcription
	req := ai.NutritionRequest{
		AudioData: audioData,
		UserID:    "system",
	}

	// Call AI service - it will handle audio transcription internally
	resp, err := a.coordinator.AnalyzeNutrition(req)
	if err != nil {
		return "", fmt.Errorf("audio transcription failed: %w", err)
	}

	// Extract transcription from response
	// For now, we'll reconstruct the description from parsed items
	var description strings.Builder
	for i, item := range resp.Items {
		if i > 0 {
			description.WriteString(", ")
		}
		description.WriteString(fmt.Sprintf("%.1f %s %s", item.Quantity, item.Unit, item.Name))
	}

	return description.String(), nil
}

// aiNormalizerAdapter adapts ai.Coordinator to meals.AINormalizer interface
type aiNormalizerAdapter struct {
	coordinator *ai.Coordinator
}

// NormalizeMealDescription implements meals.AINormalizer
func (a *aiNormalizerAdapter) NormalizeMealDescription(description string) (string, error) {
	// For now, return the description as-is
	// Future enhancement: use AI to normalize/clean up meal descriptions
	return description, nil
}

// cacheAdapter adapts cache.Cache to meals.Cache interface
type cacheAdapter struct {
	cache cache.Cache
}

// Get implements meals.Cache.Get
func (a *cacheAdapter) Get(ctx context.Context, key string) (interface{}, error) {
	// Try to get from cache - for now we use a generic approach
	// Future enhancement: implement type-safe cache operations
	return nil, fmt.Errorf("cache miss")
}

// Set implements meals.Cache.Set
func (a *cacheAdapter) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// For now, cache operations are no-op
	// Future enhancement: implement proper cache storage
	return nil
}
