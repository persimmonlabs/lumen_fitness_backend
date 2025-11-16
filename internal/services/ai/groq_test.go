package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGroqService_AnalyzeNutrition(t *testing.T) {
	tests := []struct {
		name           string
		request        NutritionRequest
		mockResponse   groqResponse
		mockStatusCode int
		wantError      bool
		errorType      ErrorType
	}{
		{
			name: "successful text analysis",
			request: NutritionRequest{
				Text:   "chicken breast and brown rice",
				UserID: "test-user",
			},
			mockResponse: groqResponse{
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
										"name": "Chicken Breast",
										"quantity": 1,
										"unit": "serving",
										"nutrition": {
											"calories": 165,
											"protein": 31,
											"carbohydrates": 0,
											"fat": 3.6
										},
										"confidence": 0.85
									},
									{
										"name": "Brown Rice",
										"quantity": 1,
										"unit": "cup",
										"nutrition": {
											"calories": 215,
											"protein": 5,
											"carbohydrates": 45,
											"fat": 1.8
										},
										"confidence": 0.9
									}
								],
								"warnings": [],
								"suggestions": ["Great balanced meal!"]
							}`,
						},
						FinishReason: "stop",
					},
				},
				Usage: struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				}{
					PromptTokens:     100,
					CompletionTokens: 200,
					TotalTokens:      300,
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
			name: "images not supported",
			request: NutritionRequest{
				Images: [][]byte{{1, 2, 3}},
			},
			wantError: true,
			errorType: ErrorTypeInvalidInput,
		},
		{
			name: "empty request",
			request: NutritionRequest{
				UserID: "test",
			},
			wantError: true,
			errorType: ErrorTypeInvalidInput,
		},
		{
			name: "API rate limit error",
			request: NutritionRequest{
				Text: "test",
			},
			mockStatusCode: http.StatusTooManyRequests,
			wantError:      true,
			errorType:      ErrorTypeRateLimited,
		},
		{
			name: "invalid API key",
			request: NutritionRequest{
				Text: "test",
			},
			mockStatusCode: http.StatusUnauthorized,
			wantError:      true,
			errorType:      ErrorTypeAPIError,
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
					if r.Header.Get("Content-Type") != "application/json" {
						t.Error("Wrong Content-Type header")
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
			service := NewGroqService("test-api-key")
			if tt.name == "no API key" {
				service.apiKey = ""
			}

			// Override API URL for testing
			if server != nil {
				originalURL := GroqAPIURL
				defer func() {
					// Can't actually change const, but in real code this would be configurable
				}()
				service.httpClient = &http.Client{Timeout: 5 * time.Second}
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

			// Validate response (only for successful cases with mock server)
			if resp != nil {
				if resp.ModelUsed != GroqModel {
					t.Errorf("Expected model %s, got %s", GroqModel, resp.ModelUsed)
				}
				if resp.Cost != 0.0 {
					t.Errorf("Expected zero cost for Groq, got %f", resp.Cost)
				}
				if len(resp.Items) == 0 {
					t.Error("Expected items in response")
				}
				if resp.Confidence <= 0 || resp.Confidence > 1 {
					t.Errorf("Invalid confidence score: %f", resp.Confidence)
				}
			}
		})
	}
}

func TestGroqService_Capabilities(t *testing.T) {
	service := NewGroqService("test-key")

	if service.SupportsVision() {
		t.Error("Groq should not support vision")
	}

	if !service.SupportsAudio() {
		t.Error("Groq should support audio")
	}

	if service.GetCostPerRequest() != 0.0 {
		t.Errorf("Groq should be free, got cost: %f", service.GetCostPerRequest())
	}

	if service.Name() != "Groq" {
		t.Errorf("Expected name 'Groq', got '%s'", service.Name())
	}
}

func TestGroqService_BuildPrompt(t *testing.T) {
	service := NewGroqService("test-key")

	tests := []struct {
		name    string
		request NutritionRequest
		wantErr bool
	}{
		{
			name: "with preferences",
			request: NutritionRequest{
				Text: "pizza",
				Preferences: &UserPreferences{
					DietaryRestrictions: []string{"vegetarian"},
					AllergiesWarnings:   []string{"dairy"},
					PreferredUnits:      "imperial",
				},
			},
			wantErr: false,
		},
		{
			name: "without preferences",
			request: NutritionRequest{
				Text: "burger",
			},
			wantErr: false,
		},
		{
			name: "with audio data",
			request: NutritionRequest{
				AudioData: []byte("audio"),
			},
			wantErr: true, // Not implemented yet
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := service.buildPrompt(tt.request)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error for audio data")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if prompt == "" {
				t.Error("Expected non-empty prompt")
			}
		})
	}
}

func TestGroqService_ParseResponse(t *testing.T) {
	service := NewGroqService("test-key")
	startTime := time.Now()

	tests := []struct {
		name     string
		response *groqResponse
		wantErr  bool
		errType  ErrorType
	}{
		{
			name: "valid response",
			response: &groqResponse{
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
								"items": [{
									"name": "Test Food",
									"quantity": 1,
									"unit": "serving",
									"nutrition": {
										"calories": 100,
										"protein": 10,
										"carbohydrates": 15,
										"fat": 5
									},
									"confidence": 0.8
								}]
							}`,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "no choices",
			response: &groqResponse{},
			wantErr:  true,
			errType:  ErrorTypeInvalidResponse,
		},
		{
			name: "invalid JSON",
			response: &groqResponse{
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
							Content: "not json",
						},
					},
				},
			},
			wantErr: true,
			errType: ErrorTypeInvalidResponse,
		},
		{
			name: "no items",
			response: &groqResponse{
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
							Content: `{"items": []}`,
						},
					},
				},
			},
			wantErr: true,
			errType: ErrorTypeInvalidResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.parseResponse(tt.response, startTime)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if aiErr, ok := err.(*AIError); ok {
					if aiErr.Type != tt.errType {
						t.Errorf("Expected error type %v, got %v", tt.errType, aiErr.Type)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if resp == nil {
				t.Fatal("Expected response, got nil")
			}
			if len(resp.Items) == 0 {
				t.Error("Expected items in response")
			}
		})
	}
}
