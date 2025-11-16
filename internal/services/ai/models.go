package ai

import (
	"time"
)

// NutritionRequest represents a request for nutritional analysis
type NutritionRequest struct {
	Text        string   `json:"text,omitempty"`         // Text description of food
	Images      [][]byte `json:"images,omitempty"`       // Image data (base64 encoded)
	AudioData   []byte   `json:"audio_data,omitempty"`   // Audio data for transcription
	UserID      string   `json:"user_id"`                // User identifier for cost tracking
	Preferences *UserPreferences `json:"preferences,omitempty"` // User dietary preferences
}

// UserPreferences contains user-specific dietary preferences
type UserPreferences struct {
	DietaryRestrictions []string `json:"dietary_restrictions,omitempty"` // e.g., "vegetarian", "gluten-free"
	AllergiesWarnings   []string `json:"allergies_warnings,omitempty"`   // e.g., "peanuts", "dairy"
	PreferredUnits      string   `json:"preferred_units,omitempty"`      // "metric" or "imperial"
}

// NutritionResponse represents the AI service response
type NutritionResponse struct {
	Items           []ParsedItem  `json:"items"`                      // Parsed food items
	TotalNutrition  NutritionData `json:"total_nutrition"`            // Aggregated nutrition
	Confidence      float64       `json:"confidence"`                 // 0.0-1.0
	ModelUsed       string        `json:"model_used"`                 // Which AI model was used
	ProcessingTime  time.Duration `json:"processing_time"`            // How long it took
	Cost            float64       `json:"cost,omitempty"`             // Cost in USD (if paid model)
	Warnings        []string      `json:"warnings,omitempty"`         // Allergy/dietary warnings
	Suggestions     []string      `json:"suggestions,omitempty"`      // AI-generated suggestions
	RawResponse     string        `json:"raw_response,omitempty"`     // Original AI response for debugging
}

// ParsedItem represents a single food item identified by AI
type ParsedItem struct {
	Name        string        `json:"name"`                   // Food name
	Quantity    float64       `json:"quantity"`               // Amount
	Unit        string        `json:"unit"`                   // Unit of measurement
	Nutrition   NutritionData `json:"nutrition"`              // Nutritional data
	Confidence  float64       `json:"confidence"`             // Item-specific confidence
	Description string        `json:"description,omitempty"`  // Additional context
}

// NutritionData represents nutritional information
type NutritionData struct {
	Calories      float64 `json:"calories"`       // kcal
	Protein       float64 `json:"protein"`        // grams
	Carbohydrates float64 `json:"carbohydrates"`  // grams
	Fat           float64 `json:"fat"`            // grams
	Fiber         float64 `json:"fiber,omitempty"` // grams
	Sugar         float64 `json:"sugar,omitempty"` // grams
	Sodium        float64 `json:"sodium,omitempty"` // mg
}

// AIService defines the interface for AI providers
type AIService interface {
	// AnalyzeNutrition processes a nutrition request
	AnalyzeNutrition(req NutritionRequest) (*NutritionResponse, error)

	// SupportsVision returns true if the service can process images
	SupportsVision() bool

	// SupportsAudio returns true if the service can process audio
	SupportsAudio() bool

	// GetCostPerRequest returns the estimated cost per request in USD
	GetCostPerRequest() float64

	// Name returns the service name
	Name() string
}

// ErrorType represents different types of AI service errors
type ErrorType string

const (
	ErrorTypeInvalidInput     ErrorType = "invalid_input"
	ErrorTypeAPIError         ErrorType = "api_error"
	ErrorTypeRateLimited      ErrorType = "rate_limited"
	ErrorTypeCostLimitExceeded ErrorType = "cost_limit_exceeded"
	ErrorTypeTimeout          ErrorType = "timeout"
	ErrorTypeInvalidResponse  ErrorType = "invalid_response"
	ErrorTypeNoAPIKey         ErrorType = "no_api_key"
)

// AIError represents a structured error from AI services
type AIError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Retryable bool    `json:"retryable"`
}

func (e *AIError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// NewAIError creates a new AI error
func NewAIError(errType ErrorType, message, details string, retryable bool) *AIError {
	return &AIError{
		Type:      errType,
		Message:   message,
		Details:   details,
		Retryable: retryable,
	}
}

// PromptTemplate contains the structured prompt for nutrition analysis
type PromptTemplate struct {
	System      string
	User        string
	ImagePrompt string
	AudioPrompt string
}

// MealEstimation represents a quick meal nutrition estimate
type MealEstimation struct {
	Calories       int           `json:"calories"`
	Protein        int           `json:"protein"`
	Confidence     string        `json:"confidence"`      // "low", "medium", "high"
	ProcessingTime time.Duration `json:"processing_time"`
}

// GetNutritionPrompt returns the standard nutrition analysis prompt
func GetNutritionPrompt() PromptTemplate {
	return PromptTemplate{
		System: `You are a nutrition analysis expert. Analyze food descriptions, images, or audio and return ONLY valid JSON with nutritional information.

Response format:
{
  "items": [
    {
      "name": "food name",
      "quantity": 1.0,
      "unit": "serving/cup/gram/etc",
      "nutrition": {
        "calories": 0,
        "protein": 0,
        "carbohydrates": 0,
        "fat": 0,
        "fiber": 0,
        "sugar": 0,
        "sodium": 0
      },
      "confidence": 0.0-1.0,
      "description": "optional context"
    }
  ],
  "warnings": ["allergy warnings if applicable"],
  "suggestions": ["dietary suggestions if applicable"]
}

Rules:
1. Return ONLY valid JSON, no markdown or extra text
2. Set confidence based on specificity (exact portions = high, vague = low)
3. Include warnings for common allergens
4. Provide helpful suggestions for healthier alternatives
5. If uncertain, err on the side of overestimating calories`,

		User: `Analyze this food and return nutrition data in JSON format:
%s

User preferences:
- Dietary restrictions: %v
- Allergies: %v
- Preferred units: %s`,

		ImagePrompt: `Analyze the food in this image and return nutrition data in JSON format.
Estimate portions based on visual cues (plate size, common serving sizes, etc.).

User preferences:
- Dietary restrictions: %v
- Allergies: %v
- Preferred units: %s`,

		AudioPrompt: `The user said: "%s"

Analyze this food description and return nutrition data in JSON format.

User preferences:
- Dietary restrictions: %v
- Allergies: %v
- Preferred units: %s`,
	}
}
