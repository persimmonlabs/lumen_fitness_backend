package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const (
	GroqAPIURL     = "https://api.groq.com/openai/v1/chat/completions"
	GroqModel      = "llama-3.3-70b-versatile"
	GroqTimeout    = 30 * time.Second
	MaxRetries     = 3
	RetryDelay     = 2 * time.Second
)

// GroqService implements AIService using Groq API
type GroqService struct {
	apiKey     string
	httpClient *http.Client
	model      string
	logger     *slog.Logger
}

// NewGroqService creates a new Groq service instance
func NewGroqService(apiKey string) *GroqService {
	return &GroqService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: GroqTimeout,
		},
		model: GroqModel,
		logger: slog.Default(),
	}
}

// NewGroqServiceWithLogger creates a new Groq service instance with custom logger
func NewGroqServiceWithLogger(apiKey string, logger *slog.Logger) *GroqService {
	return &GroqService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: GroqTimeout,
		},
		model: GroqModel,
		logger: logger,
	}
}

// GroqRequest represents the Groq API request structure
type groqRequest struct {
	Model       string          `json:"model"`
	Messages    []groqMessage   `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

// GroqResponse represents the Groq API response structure
type groqResponse struct {
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
	} `json:"error,omitempty"`
}

// AnalyzeNutrition implements AIService.AnalyzeNutrition for Groq
func (g *GroqService) AnalyzeNutrition(req NutritionRequest) (*NutritionResponse, error) {
	startTime := time.Now()

	// Validate input
	if err := g.validateRequest(req); err != nil {
		return nil, err
	}

	// Build prompt
	prompt, err := g.buildPrompt(req)
	if err != nil {
		return nil, NewAIError(ErrorTypeInvalidInput, "Failed to build prompt", err.Error(), false)
	}

	// Call Groq API with retries
	var response *groqResponse
	var lastErr error

	for attempt := 0; attempt < MaxRetries; attempt++ {
		response, lastErr = g.callAPI(prompt)
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
	nutritionResp, err := g.parseResponse(response, startTime)
	if err != nil {
		return nil, err
	}

	return nutritionResp, nil
}

// validateRequest validates the nutrition request
func (g *GroqService) validateRequest(req NutritionRequest) error {
	if g.apiKey == "" {
		return NewAIError(ErrorTypeNoAPIKey, "Groq API key is not configured", "", false)
	}

	// Groq doesn't support images
	if len(req.Images) > 0 {
		return NewAIError(ErrorTypeInvalidInput, "Groq does not support image analysis", "Use OpenRouter for vision capabilities", false)
	}

	// Need either text or audio
	if req.Text == "" && len(req.AudioData) == 0 {
		return NewAIError(ErrorTypeInvalidInput, "Request must contain text or audio data", "", false)
	}

	return nil
}

// buildPrompt constructs the prompt from the request
func (g *GroqService) buildPrompt(req NutritionRequest) (string, error) {
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

	// Handle audio (would need transcription, but for now treat as text)
	text := req.Text
	if text == "" && len(req.AudioData) > 0 {
		// In production, you'd transcribe audio first using Groq's Whisper API
		// For now, return error
		return "", fmt.Errorf("audio transcription not implemented yet")
	}

	// Build user prompt
	userPrompt := fmt.Sprintf(template.User, text, dietaryRestrictions, allergies, units)

	return userPrompt, nil
}

// callAPI makes the HTTP request to Groq API
func (g *GroqService) callAPI(prompt string) (*groqResponse, error) {
	template := GetNutritionPrompt()

	reqBody := groqRequest{
		Model: g.model,
		Messages: []groqMessage{
			{
				Role:    "system",
				Content: template.System,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3, // Lower temperature for more consistent nutrition data
		MaxTokens:   2000,
		ResponseFormat: &responseFormat{
			Type: "json_object",
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, NewAIError(ErrorTypeInvalidInput, "Failed to marshal request", err.Error(), false)
	}

	httpReq, err := http.NewRequest("POST", GroqAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, NewAIError(ErrorTypeAPIError, "Failed to create request", err.Error(), false)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, NewAIError(ErrorTypeTimeout, "Request to Groq API failed", err.Error(), true)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewAIError(ErrorTypeAPIError, "Failed to read response", err.Error(), true)
	}

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, g.handleHTTPError(resp.StatusCode, body)
	}

	var groqResp groqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		return nil, NewAIError(ErrorTypeInvalidResponse, "Failed to parse response", err.Error(), false)
	}

	// Check for API errors
	if groqResp.Error != nil {
		return nil, NewAIError(ErrorTypeAPIError, groqResp.Error.Message, groqResp.Error.Type, false)
	}

	return &groqResp, nil
}

// handleHTTPError converts HTTP errors to AIErrors
func (g *GroqService) handleHTTPError(statusCode int, body []byte) *AIError {
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
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return NewAIError(ErrorTypeAPIError, "Groq API error", errorMsg, true)
	default:
		return NewAIError(ErrorTypeAPIError, fmt.Sprintf("HTTP %d error", statusCode), errorMsg, false)
	}
}

// parseResponse converts Groq response to NutritionResponse
func (g *GroqService) parseResponse(resp *groqResponse, startTime time.Time) (*NutritionResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, NewAIError(ErrorTypeInvalidResponse, "No response from Groq", "", false)
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
		ModelUsed:      g.model,
		ProcessingTime: time.Since(startTime),
		Cost:           0.0, // Groq is free
		Warnings:       parsed.Warnings,
		Suggestions:    parsed.Suggestions,
		RawResponse:    content,
	}, nil
}

// SupportsVision returns false (Groq doesn't support image analysis)
func (g *GroqService) SupportsVision() bool {
	return false
}

// SupportsAudio returns true (Groq supports Whisper for transcription)
func (g *GroqService) SupportsAudio() bool {
	return true
}

// GetCostPerRequest returns 0.0 (Groq is free)
func (g *GroqService) GetCostPerRequest() float64 {
	return 0.0
}

// Name returns the service name
func (g *GroqService) Name() string {
	return "Groq"
}

// EstimateMeal provides fast, low-detail meal estimation using llama-3.1-8b
func (g *GroqService) EstimateMeal(description string) (*MealEstimation, error) {
	startTime := time.Now()

	if g.apiKey == "" {
		return nil, NewAIError(ErrorTypeNoAPIKey, "Groq API key is not configured", "", false)
	}

	if description == "" {
		return nil, NewAIError(ErrorTypeInvalidInput, "Description is required", "", false)
	}

	// Build estimation prompt
	prompt := fmt.Sprintf(`Estimate nutrition for: "%s"
Respond with ONLY valid JSON:
{"calories": 500, "protein": 30, "confidence": "medium"}
Confidence: low/medium/high based on description specificity.`, description)

	// Use fast model for quick estimations
	reqBody := groqRequest{
		Model: "llama-3.1-8b-instant", // Fast model
		Messages: []groqMessage{
			{
				Role:    "system",
				Content: "You are a nutrition expert. Provide quick calorie estimates. Return ONLY valid JSON.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3,
		MaxTokens:   200,
		ResponseFormat: &responseFormat{
			Type: "json_object",
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, NewAIError(ErrorTypeInvalidInput, "Failed to marshal request", err.Error(), false)
	}

	httpReq, err := http.NewRequest("POST", GroqAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, NewAIError(ErrorTypeAPIError, "Failed to create request", err.Error(), false)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, NewAIError(ErrorTypeTimeout, "Request to Groq API failed", err.Error(), true)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewAIError(ErrorTypeAPIError, "Failed to read response", err.Error(), true)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, g.handleHTTPError(resp.StatusCode, body)
	}

	var groqResp groqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		return nil, NewAIError(ErrorTypeInvalidResponse, "Failed to parse response", err.Error(), false)
	}

	if groqResp.Error != nil {
		return nil, NewAIError(ErrorTypeAPIError, groqResp.Error.Message, groqResp.Error.Type, false)
	}

	if len(groqResp.Choices) == 0 {
		return nil, NewAIError(ErrorTypeInvalidResponse, "No response from Groq", "", false)
	}

	content := groqResp.Choices[0].Message.Content

	// Parse estimation JSON
	var estimation MealEstimation
	if err := json.Unmarshal([]byte(content), &estimation); err != nil {
		return nil, NewAIError(ErrorTypeInvalidResponse, "Failed to parse estimation data", err.Error(), false)
	}

	estimation.ProcessingTime = time.Since(startTime)
	return &estimation, nil
}

// NormalizeMealDescription normalizes meal description using AI
func (g *GroqService) NormalizeMealDescription(description string) (string, error) {
	if g.apiKey == "" {
		return "", NewAIError(ErrorTypeNoAPIKey, "Groq API key is not configured", "", false)
	}

	if description == "" {
		return description, nil
	}

	// Build normalization prompt
	prompt := fmt.Sprintf(`Normalize this meal name to title case and fix typos: "%s"
Respond with ONLY the normalized text, no JSON or explanation.`, description)

	reqBody := groqRequest{
		Model: "llama-3.1-8b-instant", // Fast model
		Messages: []groqMessage{
			{
				Role:    "system",
				Content: "You are a text normalization expert. Return ONLY the normalized text.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1, // Very low for consistent normalization
		MaxTokens:   100,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return description, NewAIError(ErrorTypeInvalidInput, "Failed to marshal request", err.Error(), false)
	}

	httpReq, err := http.NewRequest("POST", GroqAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return description, NewAIError(ErrorTypeAPIError, "Failed to create request", err.Error(), false)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return description, NewAIError(ErrorTypeTimeout, "Request to Groq API failed", err.Error(), true)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return description, NewAIError(ErrorTypeAPIError, "Failed to read response", err.Error(), true)
	}

	if resp.StatusCode != http.StatusOK {
		return description, g.handleHTTPError(resp.StatusCode, body)
	}

	var groqResp groqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		return description, NewAIError(ErrorTypeInvalidResponse, "Failed to parse response", err.Error(), false)
	}

	if groqResp.Error != nil {
		return description, NewAIError(ErrorTypeAPIError, groqResp.Error.Message, groqResp.Error.Type, false)
	}

	if len(groqResp.Choices) == 0 {
		return description, NewAIError(ErrorTypeInvalidResponse, "No response from Groq", "", false)
	}

	normalized := groqResp.Choices[0].Message.Content
	// Trim whitespace and quotes
	normalized = strings.TrimSpace(normalized)
	normalized = strings.Trim(normalized, "\"'")

	return normalized, nil
}

// TranscribeAudio transcribes audio to text using Groq Whisper API
func (g *GroqService) TranscribeAudio(audioData []byte, contentType string) (string, error) {
	startTime := time.Now()

	if g.apiKey == "" {
		return "", NewAIError(ErrorTypeNoAPIKey, "Groq API key is not configured", "", false)
	}

	if len(audioData) == 0 {
		return "", NewAIError(ErrorTypeInvalidInput, "Audio data is required", "", false)
	}

	// Groq Whisper endpoint
	whisperURL := "https://api.groq.com/openai/v1/audio/transcriptions"

	// Create multipart form
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add file field
	fw, err := w.CreateFormFile("file", "audio.webm")
	if err != nil {
		return "", NewAIError(ErrorTypeInvalidInput, "Failed to create form file", err.Error(), false)
	}
	if _, err := fw.Write(audioData); err != nil {
		return "", NewAIError(ErrorTypeInvalidInput, "Failed to write audio data", err.Error(), false)
	}

	// Add model field
	if err := w.WriteField("model", "whisper-large-v3"); err != nil {
		return "", NewAIError(ErrorTypeInvalidInput, "Failed to write model field", err.Error(), false)
	}

	// Add language field (optional, auto-detect)
	if err := w.WriteField("language", "en"); err != nil {
		return "", NewAIError(ErrorTypeInvalidInput, "Failed to write language field", err.Error(), false)
	}

	w.Close()

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", whisperURL, &b)
	if err != nil {
		return "", NewAIError(ErrorTypeAPIError, "Failed to create request", err.Error(), false)
	}

	httpReq.Header.Set("Content-Type", w.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	// Make request with longer timeout for audio processing
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", NewAIError(ErrorTypeTimeout, "Request to Groq Whisper API failed", err.Error(), true)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", NewAIError(ErrorTypeAPIError, "Failed to read response", err.Error(), true)
	}

	if resp.StatusCode != http.StatusOK {
		return "", g.handleHTTPError(resp.StatusCode, body)
	}

	// Parse transcription response
	var transcriptionResp struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &transcriptionResp); err != nil {
		return "", NewAIError(ErrorTypeInvalidResponse, "Failed to parse transcription", err.Error(), false)
	}

	g.logger.Debug("audio transcription completed",
		slog.Duration("duration", time.Since(startTime)),
		slog.Int("audio_bytes", len(audioData)),
	)

	return transcriptionResp.Text, nil
}
