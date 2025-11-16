package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenRouterService_AnalyzeNutrition(t *testing.T) {
	tests := []struct {
		name           string
		request        NutritionRequest
		mockResponse   openRouterResponse
		mockStatusCode int
		wantError      bool
		errorType      ErrorType
	}{
		{
			name: "successful text analysis",
			request: NutritionRequest{
				Text:   "salmon and quinoa",
				UserID: "test-user",
			},
			mockResponse: openRouterResponse{
				ID: "test-id",
				Choices: []struct {
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Message: struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						}{
							Role: "assistant",
							Content: `{
								"items": [
									{
										"name": "Salmon",
										"quantity": 6,
										"unit": "oz",
										"nutrition": {
											"calories": 350,
											"protein": 40,
											"carbohydrates": 0,
											"fat": 20
										},
										"confidence": 0.88
									}
								]
							}`,
						},
					},
				},
				Usage: struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				}{
					PromptTokens:     150,
					CompletionTokens: 250,
					TotalTokens:      400,
				},
			},
			mockStatusCode: http.StatusOK,
			wantError:      false,
		},
		{
			name: "successful image analysis",
			request: NutritionRequest{
				Images: [][]byte{{255, 216, 255}}, // JPEG header
				UserID: "test-user",
			},
			mockResponse: openRouterResponse{
				ID: "test-id",
				Choices: []struct {
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Message: struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						}{
							Content: `{
								"items": [
									{
										"name": "Pizza Slice",
										"quantity": 2,
										"unit": "slices",
										"nutrition": {
											"calories": 570,
											"protein": 24,
											"carbohydrates": 70,
											"fat": 22
										},
										"confidence": 0.75
									}
								]
							}`,
						},
					},
				},
				Usage: struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				}{
					PromptTokens:     500,
					CompletionTokens: 300,
					TotalTokens:      800,
				},
			},
			mockStatusCode: http.StatusOK,
			wantError:      false,
		},
		{
			name: "no API key",
			request: NutritionRequest{
				Text: "test",
			},
			wantError: true,
			errorType: ErrorTypeNoAPIKey,
		},
		{
			name: "payment required error",
			request: NutritionRequest{
				Text: "test",
			},
			mockStatusCode: http.StatusPaymentRequired,
			wantError:      true,
			errorType:      ErrorTypeCostLimitExceeded,
		},
		{
			name: "rate limit error",
			request: NutritionRequest{
				Text: "test",
			},
			mockStatusCode: http.StatusTooManyRequests,
			wantError:      true,
			errorType:      ErrorTypeRateLimited,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			var server *httptest.Server
			if tt.mockStatusCode != 0 {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify headers
					if r.Header.Get("Authorization") == "" {
						t.Error("Missing Authorization header")
					}

					w.WriteHeader(tt.mockStatusCode)
					if tt.mockStatusCode == http.StatusOK {
						json.NewEncoder(w).Encode(tt.mockResponse)
					} else {
						json.NewEncoder(w).Encode(map[string]interface{}{
							"error": map[string]string{
								"message": "Mock error",
								"type":    "test_error",
							},
						})
					}
				}))
				defer server.Close()
			}

			// Create service
			service := NewOpenRouterService(OpenRouterConfig{
				APIKey:  "test-api-key",
				AppName: "test-app",
				SiteURL: "https://test.com",
			})

			if tt.name == "no API key" {
				service.apiKey = ""
			}

			// Call method
			resp, err := service.AnalyzeNutrition(tt.request)

			// Check error
			if tt.wantError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if aiErr, ok := err.(*AIError); ok {
					if aiErr.Type != tt.errorType {
						t.Errorf("Expected error type %v, got %v", tt.errorType, aiErr.Type)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Validate response
			if resp != nil {
				if resp.Cost == 0.0 {
					t.Error("Expected non-zero cost for OpenRouter")
				}
				if len(resp.Items) == 0 {
					t.Error("Expected items in response")
				}
			}
		})
	}
}

func TestOpenRouterService_Capabilities(t *testing.T) {
	service := NewOpenRouterService(OpenRouterConfig{APIKey: "test"})

	if !service.SupportsVision() {
		t.Error("OpenRouter should support vision")
	}

	if service.SupportsAudio() {
		t.Error("OpenRouter should not support audio directly")
	}

	cost := service.GetCostPerRequest()
	if cost == 0.0 {
		t.Error("OpenRouter should have non-zero estimated cost")
	}

	if service.Name() != "OpenRouter" {
		t.Errorf("Expected name 'OpenRouter', got '%s'", service.Name())
	}
}

func TestOpenRouterService_BuildMessages(t *testing.T) {
	service := NewOpenRouterService(OpenRouterConfig{APIKey: "test"})

	tests := []struct {
		name           string
		request        NutritionRequest
		wantErr        bool
		expectImageMsg bool
	}{
		{
			name: "text only",
			request: NutritionRequest{
				Text: "apple",
			},
			wantErr:        false,
			expectImageMsg: false,
		},
		{
			name: "with image",
			request: NutritionRequest{
				Images: [][]byte{{1, 2, 3}},
			},
			wantErr:        false,
			expectImageMsg: true,
		},
		{
			name: "with preferences",
			request: NutritionRequest{
				Text: "pasta",
				Preferences: &UserPreferences{
					DietaryRestrictions: []string{"gluten-free"},
					AllergiesWarnings:   []string{"wheat"},
					PreferredUnits:      "imperial",
				},
			},
			wantErr:        false,
			expectImageMsg: false,
		},
		{
			name: "audio not supported",
			request: NutritionRequest{
				AudioData: []byte("audio"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := service.buildMessages(tt.request)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error for unsupported input type")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(messages) < 2 {
				t.Error("Expected at least system and user messages")
			}

			// Check if image message structure is correct
			if tt.expectImageMsg {
				userMsg := messages[1]
				if contentParts, ok := userMsg.Content.([]contentPart); ok {
					foundImage := false
					for _, part := range contentParts {
						if part.Type == "image_url" {
							foundImage = true
							break
						}
					}
					if !foundImage {
						t.Error("Expected image content part in message")
					}
				} else {
					t.Error("Expected content to be array of contentPart for image messages")
				}
			}
		})
	}
}

func TestOpenRouterService_CalculateCost(t *testing.T) {
	service := NewOpenRouterService(OpenRouterConfig{APIKey: "test"})

	tests := []struct {
		name             string
		promptTokens     int
		completionTokens int
		wantCost         bool // Just check if cost > 0
	}{
		{
			name:             "typical request",
			promptTokens:     100,
			completionTokens: 200,
			wantCost:         true,
		},
		{
			name:             "large request",
			promptTokens:     1000,
			completionTokens: 2000,
			wantCost:         true,
		},
		{
			name:             "zero tokens",
			promptTokens:     0,
			completionTokens: 0,
			wantCost:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := service.calculateCost(tt.promptTokens, tt.completionTokens)
			if tt.wantCost && cost <= 0 {
				t.Error("Expected positive cost for token usage")
			}
			if !tt.wantCost && cost != 0 {
				t.Error("Expected zero cost for zero tokens")
			}
		})
	}
}

func TestOpenRouterService_CustomModel(t *testing.T) {
	customModel := "gpt-4-vision-preview"
	service := NewOpenRouterService(OpenRouterConfig{
		APIKey: "test",
		Model:  customModel,
	})

	if service.model != customModel {
		t.Errorf("Expected model %s, got %s", customModel, service.model)
	}
}
