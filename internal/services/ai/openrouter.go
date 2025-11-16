package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	OpenRouterAPIURL     = "https://openrouter.ai/api/v1/chat/completions"
	OpenRouterModel      = "anthropic/claude-3.5-sonnet" // Vision-capable model
	OpenRouterTimeout    = 45 * time.Second
	OpenRouterCostPerRequest = 0.003 // Approximate cost in USD
)

// OpenRouterService implements AIService using OpenRouter API
type OpenRouterService struct {
	apiKey     string
	httpClient *http.Client
	model      string
	appName    string
	siteURL    string
}

// OpenRouterConfig contains configuration for OpenRouter service
type OpenRouterConfig struct {
	APIKey  string
	Model   string // Optional: override default model
	AppName string // Optional: for OpenRouter analytics
	SiteURL string // Optional: for OpenRouter analytics
}

// NewOpenRouterService creates a new OpenRouter service instance
func NewOpenRouterService(config OpenRouterConfig) *OpenRouterService {
	model := config.Model
	if model == "" {
		model = OpenRouterModel
	}

	return &OpenRouterService{
		apiKey: config.APIKey,
		httpClient: &http.Client{
			Timeout: OpenRouterTimeout,
		},
		model:   model,
		appName: config.AppName,
		siteURL: config.SiteURL,
	}
}

// openRouterRequest represents the OpenRouter API request structure
type openRouterRequest struct {
	Model       string                `json:"model"`
	Messages    []openRouterMessage   `json:"messages"`
	Temperature float64               `json:"temperature"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
	ResponseFormat *responseFormat    `json:"response_format,omitempty"`
}

type openRouterMessage struct {
	Role    string                 `json:"role"`
	Content interface{}            `json:"content"` // Can be string or array of content parts
}

type contentPart struct {
	Type     string    `json:"type"` // "text" or "image_url"
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"` // data:image/jpeg;base64,... or https://...
}

// openRouterResponse represents the OpenRouter API response structure
type openRouterResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

// AnalyzeNutrition implements AIService.AnalyzeNutrition for OpenRouter
func (o *OpenRouterService) AnalyzeNutrition(req NutritionRequest) (*NutritionResponse, error) {
	startTime := time.Now()

	// Validate input
	if err := o.validateRequest(req); err != nil {
		return nil, err
	}

	// Build messages with potential image content
	messages, err := o.buildMessages(req)
	if err != nil {
		return nil, NewAIError(ErrorTypeInvalidInput, "Failed to build messages", err.Error(), false)
	}

	// Call OpenRouter API with retries
	var response *openRouterResponse
	var lastErr error

	for attempt := 0; attempt < MaxRetries; attempt++ {
		response, lastErr = o.callAPI(messages)
		if lastErr == nil {
			break
		}

		// Check if error is retryable
		if aiErr, ok := lastErr.(*AIError); ok && !aiErr.Retryable {
			return nil, lastErr
		}

		// Wait before retrying
		if attempt < MaxRetries-1 {
			time.Sleep(RetryDelay * time.Duration(attempt+1))
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	// Parse response
	nutritionResp, err := o.parseResponse(response, startTime)
	if err != nil {
		return nil, err
	}

	// Calculate actual cost based on token usage
	cost := o.calculateCost(response.Usage.PromptTokens, response.Usage.CompletionTokens)
	nutritionResp.Cost = cost

	return nutritionResp, nil
}

// validateRequest validates the nutrition request
func (o *OpenRouterService) validateRequest(req NutritionRequest) error {
	if o.apiKey == "" {
		return NewAIError(ErrorTypeNoAPIKey, "OpenRouter API key is not configured", "", false)
	}

	// Need at least one input type
	if req.Text == "" && len(req.Images) == 0 && len(req.AudioData) == 0 {
		return NewAIError(ErrorTypeInvalidInput, "Request must contain text, images, or audio data", "", false)
	}

	return nil
}

// buildMessages constructs the messages array from the request
func (o *OpenRouterService) buildMessages(req NutritionRequest) ([]openRouterMessage, error) {
	template := GetNutritionPrompt()

	// Get user preferences or defaults
	dietaryRestrictions := []string{}
	allergies := []string{}
	units := "metric"

	if req.Preferences != nil {
		dietaryRestrictions = req.Preferences.DietaryRestrictions
		allergies = req.Preferences.AllergiesWarnings
		if req.Preferences.PreferredUnits != "" {
			units = req.Preferences.PreferredUnits
		}
	}

	messages := []openRouterMessage{
		{
			Role:    "system",
			Content: template.System,
		},
	}

	// Build user message based on input type
	if len(req.Images) > 0 {
		// Vision request - use content array with text and images
		contentParts := []contentPart{}

		// Add text prompt
		promptText := fmt.Sprintf(template.ImagePrompt, dietaryRestrictions, allergies, units)
		contentParts = append(contentParts, contentPart{
			Type: "text",
			Text: promptText,
		})

		// Add images
		for _, imgData := range req.Images {
			base64Img := base64.StdEncoding.EncodeToString(imgData)
			contentParts = append(contentParts, contentPart{
				Type: "image_url",
				ImageURL: &imageURL{
					URL: "data:image/jpeg;base64," + base64Img,
				},
			})
		}

		messages = append(messages, openRouterMessage{
			Role:    "user",
			Content: contentParts,
		})
	} else if len(req.AudioData) > 0 {
		// Audio request - would need transcription first
		// For now, return error as this requires additional processing
		return nil, fmt.Errorf("audio transcription not implemented for OpenRouter")
	} else {
		// Text-only request
		userPrompt := fmt.Sprintf(template.User, req.Text, dietaryRestrictions, allergies, units)
		messages = append(messages, openRouterMessage{
			Role:    "user",
			Content: userPrompt,
		})
	}

	return messages, nil
}

// callAPI makes the HTTP request to OpenRouter API
func (o *OpenRouterService) callAPI(messages []openRouterMessage) (*openRouterResponse, error) {
	reqBody := openRouterRequest{
		Model:       o.model,
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   2000,
		ResponseFormat: &responseFormat{
			Type: "json_object",
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, NewAIError(ErrorTypeInvalidInput, "Failed to marshal request", err.Error(), false)
	}

	httpReq, err := http.NewRequest("POST", OpenRouterAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, NewAIError(ErrorTypeAPIError, "Failed to create request", err.Error(), false)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	// Optional: Add app identification for OpenRouter analytics
	if o.appName != "" {
		httpReq.Header.Set("X-Title", o.appName)
	}
	if o.siteURL != "" {
		httpReq.Header.Set("HTTP-Referer", o.siteURL)
	}

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, NewAIError(ErrorTypeTimeout, "Request to OpenRouter API failed", err.Error(), true)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewAIError(ErrorTypeAPIError, "Failed to read response", err.Error(), true)
	}

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, o.handleHTTPError(resp.StatusCode, body)
	}

	var openRouterResp openRouterResponse
	if err := json.Unmarshal(body, &openRouterResp); err != nil {
		return nil, NewAIError(ErrorTypeInvalidResponse, "Failed to parse response", err.Error(), false)
	}

	// Check for API errors
	if openRouterResp.Error != nil {
		return nil, NewAIError(ErrorTypeAPIError, openRouterResp.Error.Message, openRouterResp.Error.Type, false)
	}

	return &openRouterResp, nil
}

// handleHTTPError converts HTTP errors to AIErrors
func (o *OpenRouterService) handleHTTPError(statusCode int, body []byte) *AIError {
	var errorMsg string
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		errorMsg = errResp.Error.Message
	} else {
		errorMsg = string(body)
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return NewAIError(ErrorTypeAPIError, "Invalid API key", errorMsg, false)
	case http.StatusTooManyRequests:
		return NewAIError(ErrorTypeRateLimited, "Rate limit exceeded", errorMsg, true)
	case http.StatusBadRequest:
		return NewAIError(ErrorTypeInvalidInput, "Invalid request", errorMsg, false)
	case http.StatusPaymentRequired:
		return NewAIError(ErrorTypeCostLimitExceeded, "Insufficient credits", errorMsg, false)
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return NewAIError(ErrorTypeAPIError, "OpenRouter API error", errorMsg, true)
	default:
		return NewAIError(ErrorTypeAPIError, fmt.Sprintf("HTTP %d error", statusCode), errorMsg, false)
	}
}

// parseResponse converts OpenRouter response to NutritionResponse
func (o *OpenRouterService) parseResponse(resp *openRouterResponse, startTime time.Time) (*NutritionResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, NewAIError(ErrorTypeInvalidResponse, "No response from OpenRouter", "", false)
	}

	content := resp.Choices[0].Message.Content

	// Parse JSON response
	var parsed struct {
		Items       []ParsedItem `json:"items"`
		Warnings    []string     `json:"warnings,omitempty"`
		Suggestions []string     `json:"suggestions,omitempty"`
	}

	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, NewAIError(ErrorTypeInvalidResponse, "Failed to parse nutrition data", err.Error(), false)
	}

	if len(parsed.Items) == 0 {
		return nil, NewAIError(ErrorTypeInvalidResponse, "No food items found in response", "", false)
	}

	// Calculate total nutrition and average confidence
	totalNutrition := NutritionData{}
	totalConfidence := 0.0

	for _, item := range parsed.Items {
		totalNutrition.Calories += item.Nutrition.Calories
		totalNutrition.Protein += item.Nutrition.Protein
		totalNutrition.Carbohydrates += item.Nutrition.Carbohydrates
		totalNutrition.Fat += item.Nutrition.Fat
		totalNutrition.Fiber += item.Nutrition.Fiber
		totalNutrition.Sugar += item.Nutrition.Sugar
		totalNutrition.Sodium += item.Nutrition.Sodium
		totalConfidence += item.Confidence
	}

	avgConfidence := totalConfidence / float64(len(parsed.Items))

	return &NutritionResponse{
		Items:          parsed.Items,
		TotalNutrition: totalNutrition,
		Confidence:     avgConfidence,
		ModelUsed:      o.model,
		ProcessingTime: time.Since(startTime),
		Warnings:       parsed.Warnings,
		Suggestions:    parsed.Suggestions,
		RawResponse:    content,
	}, nil
}

// calculateCost estimates the cost based on token usage
// Pricing varies by model, this is a rough estimate
func (o *OpenRouterService) calculateCost(promptTokens, completionTokens int) float64 {
	// Approximate pricing for Claude 3.5 Sonnet:
	// $3.00 per million input tokens, $15.00 per million output tokens
	inputCost := float64(promptTokens) * 3.0 / 1000000.0
	outputCost := float64(completionTokens) * 15.0 / 1000000.0
	return inputCost + outputCost
}

// SupportsVision returns true (OpenRouter supports vision models)
func (o *OpenRouterService) SupportsVision() bool {
	return true
}

// SupportsAudio returns false (requires separate transcription)
func (o *OpenRouterService) SupportsAudio() bool {
	return false
}

// GetCostPerRequest returns the estimated cost per request
func (o *OpenRouterService) GetCostPerRequest() float64 {
	return OpenRouterCostPerRequest
}

// Name returns the service name
func (o *OpenRouterService) Name() string {
	return "OpenRouter"
}
