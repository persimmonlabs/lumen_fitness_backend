package ai

import (
	"log/slog"
)

// GroqAdapter wraps GroqService and implements all AI-related interfaces
type GroqAdapter struct {
	*GroqService
}

// NewGroqAdapter creates a new Groq adapter with all capabilities
func NewGroqAdapter(apiKey string, logger *slog.Logger) *GroqAdapter {
	return &GroqAdapter{
		GroqService: NewGroqServiceWithLogger(apiKey, logger),
	}
}

// Ensure GroqAdapter implements all required interfaces at compile time
var (
	_ AIService = (*GroqAdapter)(nil)
)

// ConvertEstimationToAIEstimation converts MealEstimation to AIEstimation
func ConvertEstimationToAIEstimation(est *MealEstimation) *AIEstimation {
	return &AIEstimation{
		Calories:   est.Calories,
		Protein:    est.Protein,
		Confidence: est.Confidence,
	}
}

// AIEstimation represents a quick estimation result
type AIEstimation struct {
	Calories   int    `json:"calories"`
	Protein    int    `json:"protein"`
	Confidence string `json:"confidence"`
}

// EstimateMeal wraps the Groq estimation method to return AIEstimation
func (g *GroqAdapter) EstimateMealAI(description string) (*AIEstimation, error) {
	result, err := g.EstimateMeal(description)
	if err != nil {
		return nil, err
	}
	return ConvertEstimationToAIEstimation(result), nil
}
