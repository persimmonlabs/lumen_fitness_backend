package ai

import (
	"fmt"
	"math/rand"
	"time"
)

// MockService implements AIService for testing without API keys
type MockService struct {
	supportsVision bool
	supportsAudio  bool
	simulateDelay  time.Duration
	failureRate    float64 // 0.0 to 1.0
}

// MockConfig contains configuration for mock service
type MockConfig struct {
	SupportsVision bool
	SupportsAudio  bool
	SimulateDelay  time.Duration
	FailureRate    float64 // Simulate random failures
}

// NewMockService creates a new mock AI service
func NewMockService(config MockConfig) *MockService {
	return &MockService{
		supportsVision: config.SupportsVision,
		supportsAudio:  config.SupportsAudio,
		simulateDelay:  config.SimulateDelay,
		failureRate:    config.FailureRate,
	}
}

// AnalyzeNutrition implements AIService.AnalyzeNutrition with mock data
func (m *MockService) AnalyzeNutrition(req NutritionRequest) (*NutritionResponse, error) {
	startTime := time.Now()

	// Simulate processing delay
	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	// Simulate random failures
	if m.failureRate > 0 && rand.Float64() < m.failureRate {
		return nil, NewAIError(
			ErrorTypeAPIError,
			"Mock service simulated failure",
			fmt.Sprintf("Random failure (rate: %.2f)", m.failureRate),
			true,
		)
	}

	// Check if we support the input type
	if len(req.Images) > 0 && !m.supportsVision {
		return nil, NewAIError(
			ErrorTypeInvalidInput,
			"Mock service does not support vision",
			"Enable vision support in mock config",
			false,
		)
	}

	if len(req.AudioData) > 0 && !m.supportsAudio {
		return nil, NewAIError(
			ErrorTypeInvalidInput,
			"Mock service does not support audio",
			"Enable audio support in mock config",
			false,
		)
	}

	// Generate mock nutrition data based on input
	items := m.generateMockItems(req)

	// Calculate totals
	totalNutrition := NutritionData{}
	totalConfidence := 0.0

	for _, item := range items {
		totalNutrition.Calories += item.Nutrition.Calories
		totalNutrition.Protein += item.Nutrition.Protein
		totalNutrition.Carbohydrates += item.Nutrition.Carbohydrates
		totalNutrition.Fat += item.Nutrition.Fat
		totalNutrition.Fiber += item.Nutrition.Fiber
		totalNutrition.Sugar += item.Nutrition.Sugar
		totalNutrition.Sodium += item.Nutrition.Sodium
		totalConfidence += item.Confidence
	}

	avgConfidence := totalConfidence / float64(len(items))

	// Generate mock warnings and suggestions
	warnings := m.generateMockWarnings(req, items)
	suggestions := m.generateMockSuggestions(items)

	return &NutritionResponse{
		Items:          items,
		TotalNutrition: totalNutrition,
		Confidence:     avgConfidence,
		ModelUsed:      "Mock AI Service",
		ProcessingTime: time.Since(startTime),
		Cost:           0.0,
		Warnings:       warnings,
		Suggestions:    suggestions,
		RawResponse:    fmt.Sprintf("Mock response for: %s", req.Text),
	}, nil
}

// generateMockItems creates realistic mock food items
func (m *MockService) generateMockItems(req NutritionRequest) []ParsedItem {
	// Common food items for mock data
	mockFoods := []struct {
		name        string
		nutrition   NutritionData
		unit        string
		quantity    float64
		confidence  float64
		description string
	}{
		{
			name: "Grilled Chicken Breast",
			nutrition: NutritionData{
				Calories:      165,
				Protein:       31,
				Carbohydrates: 0,
				Fat:           3.6,
				Fiber:         0,
				Sugar:         0,
				Sodium:        74,
			},
			unit:        "serving",
			quantity:    1,
			confidence:  0.85,
			description: "4 oz boneless, skinless chicken breast",
		},
		{
			name: "Brown Rice",
			nutrition: NutritionData{
				Calories:      215,
				Protein:       5,
				Carbohydrates: 45,
				Fat:           1.8,
				Fiber:         3.5,
				Sugar:         0.7,
				Sodium:        10,
			},
			unit:        "cup",
			quantity:    1,
			confidence:  0.9,
			description: "Cooked long-grain brown rice",
		},
		{
			name: "Mixed Green Salad",
			nutrition: NutritionData{
				Calories:      50,
				Protein:       2,
				Carbohydrates: 8,
				Fat:           2.5,
				Fiber:         3,
				Sugar:         3,
				Sodium:        35,
			},
			unit:        "bowl",
			quantity:    1,
			confidence:  0.75,
			description: "Lettuce, tomatoes, cucumbers with olive oil",
		},
		{
			name: "Greek Yogurt",
			nutrition: NutritionData{
				Calories:      100,
				Protein:       17,
				Carbohydrates: 6,
				Fat:           0.7,
				Fiber:         0,
				Sugar:         4,
				Sodium:        65,
			},
			unit:        "cup",
			quantity:    1,
			confidence:  0.9,
			description: "Plain non-fat Greek yogurt",
		},
		{
			name: "Banana",
			nutrition: NutritionData{
				Calories:      105,
				Protein:       1.3,
				Carbohydrates: 27,
				Fat:           0.4,
				Fiber:         3.1,
				Sugar:         14,
				Sodium:        1,
			},
			unit:        "medium",
			quantity:    1,
			confidence:  0.95,
			description: "Fresh banana, about 7 inches",
		},
	}

	// Determine number of items based on input
	numItems := 1 + rand.Intn(3) // 1-3 items
	if len(req.Images) > 0 {
		numItems = 2 + rand.Intn(2) // 2-3 items for images
	}

	items := make([]ParsedItem, numItems)
	for i := 0; i < numItems; i++ {
		food := mockFoods[rand.Intn(len(mockFoods))]
		items[i] = ParsedItem{
			Name:        food.name,
			Quantity:    food.quantity,
			Unit:        food.unit,
			Nutrition:   food.nutrition,
			Confidence:  food.confidence,
			Description: food.description,
		}
	}

	return items
}

// generateMockWarnings creates relevant warnings based on food items
func (m *MockService) generateMockWarnings(req NutritionRequest, items []ParsedItem) []string {
	var warnings []string

	// Check user allergies
	if req.Preferences != nil && len(req.Preferences.AllergiesWarnings) > 0 {
		for _, allergy := range req.Preferences.AllergiesWarnings {
			warnings = append(warnings,
				fmt.Sprintf("Note: User has allergy to %s. Please verify ingredients.", allergy))
		}
	}

	// Check for high sodium
	totalSodium := 0.0
	for _, item := range items {
		totalSodium += item.Nutrition.Sodium
	}
	if totalSodium > 500 {
		warnings = append(warnings, "High sodium content detected. Consider lower-sodium alternatives.")
	}

	// Check for dietary restrictions
	if req.Preferences != nil {
		for _, restriction := range req.Preferences.DietaryRestrictions {
			if restriction == "vegetarian" {
				warnings = append(warnings, "Vegetarian diet: Please verify no meat products are included.")
			}
		}
	}

	return warnings
}

// generateMockSuggestions creates helpful suggestions
func (m *MockService) generateMockSuggestions(items []ParsedItem) []string {
	suggestions := []string{
		"Consider adding more vegetables for additional fiber and nutrients.",
		"Stay hydrated! Aim for 8 glasses of water per day.",
	}

	// Check protein content
	totalProtein := 0.0
	for _, item := range items {
		totalProtein += item.Nutrition.Protein
	}
	if totalProtein < 20 {
		suggestions = append(suggestions, "Add a protein source to make this meal more balanced.")
	}

	// Check fiber
	totalFiber := 0.0
	for _, item := range items {
		totalFiber += item.Nutrition.Fiber
	}
	if totalFiber < 5 {
		suggestions = append(suggestions, "Increase fiber intake with whole grains or legumes.")
	}

	return suggestions
}

// SupportsVision returns the configured vision support
func (m *MockService) SupportsVision() bool {
	return m.supportsVision
}

// SupportsAudio returns the configured audio support
func (m *MockService) SupportsAudio() bool {
	return m.supportsAudio
}

// GetCostPerRequest returns 0 (mock is free)
func (m *MockService) GetCostPerRequest() float64 {
	return 0.0
}

// Name returns the service name
func (m *MockService) Name() string {
	return "Mock AI Service"
}
