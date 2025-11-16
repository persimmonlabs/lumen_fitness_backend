package ai

// This file contains example integration code for using the AI service layer
// in your HTTP handlers and business logic.

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

// Example: Initialize coordinator at application startup
func InitializeAICoordinator() (*Coordinator, error) {
	config := CoordinatorConfig{
		GroqAPIKey:       os.Getenv("GROQ_API_KEY"),
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:  getEnvOrDefault("OPENROUTER_MODEL", "anthropic/claude-3.5-sonnet"),
		AppName:          "Lumen Nutrition App",
		SiteURL:          "https://lumen.app",
		PreferGroq:       getEnvBool("AI_PREFER_GROQ", true),
		CostLimit:        getEnvFloat("AI_COST_LIMIT", 10.0),
	}

	coordinator, err := NewCoordinator(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AI coordinator: %w", err)
	}

	log.Printf("AI Coordinator initialized (Groq: %v, OpenRouter: %v)",
		coordinator.groq != nil, coordinator.openRouter != nil)

	return coordinator, nil
}

// Example: HTTP handler for text-based nutrition analysis
func HandleTextAnalysis(coordinator *Coordinator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var reqBody struct {
			Text                string   `json:"text"`
			UserID              string   `json:"user_id"`
			DietaryRestrictions []string `json:"dietary_restrictions,omitempty"`
			Allergies           []string `json:"allergies,omitempty"`
			PreferredUnits      string   `json:"preferred_units,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate input
		if reqBody.Text == "" {
			http.Error(w, "Text field is required", http.StatusBadRequest)
			return
		}

		if reqBody.UserID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// Build AI request
		aiReq := NutritionRequest{
			Text:   reqBody.Text,
			UserID: reqBody.UserID,
			Preferences: &UserPreferences{
				DietaryRestrictions: reqBody.DietaryRestrictions,
				AllergiesWarnings:   reqBody.Allergies,
				PreferredUnits:      getDefaultIfEmpty(reqBody.PreferredUnits, "metric"),
			},
		}

		// Call AI service
		resp, err := coordinator.AnalyzeNutrition(aiReq)
		if err != nil {
			handleAIError(w, err)
			return
		}

		// Log usage for monitoring
		log.Printf("AI Analysis: user=%s, model=%s, confidence=%.2f, cost=$%.4f",
			reqBody.UserID, resp.ModelUsed, resp.Confidence, resp.Cost)

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    resp,
		})
	}
}

// Example: HTTP handler for image-based nutrition analysis
func HandleImageAnalysis(coordinator *Coordinator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if OpenRouter is available (required for vision)
		if !coordinator.SupportsVision() {
			http.Error(w, "Image analysis not available - OpenRouter API key required", http.StatusServiceUnavailable)
			return
		}

		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20) // 10 MB max
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		userID := r.FormValue("user_id")
		if userID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// Get uploaded images
		files := r.MultipartForm.File["images"]
		if len(files) == 0 {
			http.Error(w, "At least one image is required", http.StatusBadRequest)
			return
		}

		if len(files) > 5 {
			http.Error(w, "Maximum 5 images allowed", http.StatusBadRequest)
			return
		}

		// Read image data
		var imageData [][]byte
		for _, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				http.Error(w, "Failed to read image", http.StatusBadRequest)
				return
			}
			defer file.Close()

			data, err := io.ReadAll(file)
			if err != nil {
				http.Error(w, "Failed to read image data", http.StatusBadRequest)
				return
			}

			imageData = append(imageData, data)
		}

		// Build AI request
		aiReq := NutritionRequest{
			Images: imageData,
			UserID: userID,
			Preferences: &UserPreferences{
				PreferredUnits: r.FormValue("preferred_units"),
			},
		}

		// Call AI service
		resp, err := coordinator.AnalyzeNutrition(aiReq)
		if err != nil {
			handleAIError(w, err)
			return
		}

		// Log usage
		log.Printf("Image Analysis: user=%s, images=%d, model=%s, cost=$%.4f",
			userID, len(imageData), resp.ModelUsed, resp.Cost)

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    resp,
		})
	}
}

// Example: HTTP handler to get usage statistics
func HandleGetStats(coordinator *Coordinator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := coordinator.GetStats()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    stats,
		})
	}
}

// Example: HTTP handler to update cost limit (admin only)
func HandleUpdateCostLimit(coordinator *Coordinator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var reqBody struct {
			CostLimit float64 `json:"cost_limit"`
		}

		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if reqBody.CostLimit <= 0 {
			http.Error(w, "Cost limit must be positive", http.StatusBadRequest)
			return
		}

		// Update limit
		coordinator.SetCostLimit(reqBody.CostLimit)

		log.Printf("Cost limit updated to $%.2f", reqBody.CostLimit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Cost limit updated to $%.2f", reqBody.CostLimit),
		})
	}
}

// handleAIError converts AI errors to appropriate HTTP responses
func handleAIError(w http.ResponseWriter, err error) {
	aiErr, ok := err.(*AIError)
	if !ok {
		// Generic error
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Printf("AI Error: %v", err)
		return
	}

	// Map AI error types to HTTP status codes
	statusCode := http.StatusInternalServerError
	canRetry := aiErr.Retryable

	switch aiErr.Type {
	case ErrorTypeInvalidInput:
		statusCode = http.StatusBadRequest
		canRetry = false
	case ErrorTypeNoAPIKey:
		statusCode = http.StatusServiceUnavailable
		canRetry = false
	case ErrorTypeRateLimited:
		statusCode = http.StatusTooManyRequests
		canRetry = true
	case ErrorTypeCostLimitExceeded:
		statusCode = http.StatusPaymentRequired
		canRetry = false
	case ErrorTypeTimeout:
		statusCode = http.StatusGatewayTimeout
		canRetry = true
	case ErrorTypeAPIError, ErrorTypeInvalidResponse:
		statusCode = http.StatusBadGateway
		canRetry = aiErr.Retryable
	}

	// Log error
	log.Printf("AI Error [%s]: %s - %s", aiErr.Type, aiErr.Message, aiErr.Details)

	// Return error response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   false,
		"error":     aiErr.Message,
		"details":   aiErr.Details,
		"retryable": canRetry,
		"type":      aiErr.Type,
	})
}

// Example: Background job to reset monthly costs
func ResetMonthlyCostsJob(coordinator *Coordinator) {
	// This would be called by a cron job or scheduler
	// The coordinator already handles this internally, but you could
	// explicitly reset if needed
	coordinator.ResetCosts()
	log.Println("Monthly AI costs reset")
}

// Utility functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		result, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return result
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		result, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return defaultValue
		}
		return result
	}
	return defaultValue
}

func getDefaultIfEmpty(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// Example: Main function showing complete setup
func ExampleMain() {
	// Initialize AI coordinator
	coordinator, err := InitializeAICoordinator()
	if err != nil {
		log.Fatal(err)
	}

	// Set up HTTP routes
	http.HandleFunc("/api/nutrition/analyze/text", HandleTextAnalysis(coordinator))
	http.HandleFunc("/api/nutrition/analyze/image", HandleImageAnalysis(coordinator))
	http.HandleFunc("/api/ai/stats", HandleGetStats(coordinator))
	http.HandleFunc("/api/ai/cost-limit", HandleUpdateCostLimit(coordinator))

	// Log initial stats
	stats := coordinator.GetStats()
	log.Printf("AI Services: Groq=%v, OpenRouter=%v, Cost Limit=$%.2f",
		stats.GroqAvailable, stats.OpenRouterAvailable, stats.CostLimit)

	// Start server
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Example: Using with dependency injection
type NutritionService struct {
	ai *Coordinator
	// ... other dependencies
}

func NewNutritionService(aiCoordinator *Coordinator) *NutritionService {
	return &NutritionService{
		ai: aiCoordinator,
	}
}

func (s *NutritionService) AnalyzeMeal(text string, userID string) (*NutritionResponse, error) {
	req := NutritionRequest{
		Text:   text,
		UserID: userID,
	}
	return s.ai.AnalyzeNutrition(req)
}

func (s *NutritionService) AnalyzeMealPhoto(imageData []byte, userID string) (*NutritionResponse, error) {
	req := NutritionRequest{
		Images: [][]byte{imageData},
		UserID: userID,
	}
	return s.ai.AnalyzeNutrition(req)
}

func (s *NutritionService) GetAIUsageStats() CoordinatorStats {
	return s.ai.GetStats()
}
