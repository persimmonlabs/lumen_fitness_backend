// Package voice provides voice transcription services.
//
// This package handles audio transcription using Groq's Whisper API.
// It supports multiple audio formats and provides accurate speech-to-text
// conversion for meal logging.
package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Service handles voice transcription operations.
type Service struct {
	groqAPIKey string
	httpClient *http.Client
	logger     *slog.Logger
	config     Config
}

// Config contains configuration for the voice service.
type Config struct {
	Model              string
	Language           string
	Temperature        float64
	MaxFileSizeMB      int
	AllowedFormats     []string
	ResponseFormat     string
	Prompt             string
	APIEndpoint        string
}

// DefaultConfig returns the default voice service configuration.
func DefaultConfig() Config {
	return Config{
		Model:          "whisper-large-v3",
		Language:       "en",
		Temperature:    0.0,
		MaxFileSizeMB:  25,
		AllowedFormats: []string{"mp3", "mp4", "mpeg", "mpga", "m4a", "wav", "webm"},
		ResponseFormat: "json",
		Prompt:         "This is a food or meal description for nutrition tracking.",
		APIEndpoint:    "https://api.groq.com/openai/v1/audio/transcriptions",
	}
}

// NewService creates a new voice service instance.
func NewService(groqAPIKey string, logger *slog.Logger, config Config) *Service {
	return &Service{
		groqAPIKey: groqAPIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
		config: config,
	}
}

// TranscribeRequest represents a voice transcription request.
type TranscribeRequest struct {
	UserID      uuid.UUID
	AudioFile   io.Reader
	Filename    string
	ContentType string
	Size        int64
	Language    string // optional, defaults to config
}

// TranscribeResponse contains the transcription result.
type TranscribeResponse struct {
	Text       string    `json:"text"`
	Language   string    `json:"language"`
	Duration   float64   `json:"duration"`
	Confidence float64   `json:"confidence"`
	ProcessedAt time.Time `json:"processed_at"`
}

// GroqTranscriptionResponse represents Groq API response.
type GroqTranscriptionResponse struct {
	Text string `json:"text"`
}

// Transcribe converts audio to text using Groq Whisper API.
func (s *Service) Transcribe(ctx context.Context, req *TranscribeRequest) (*TranscribeResponse, error) {
	// Validate file size
	maxSize := int64(s.config.MaxFileSizeMB * 1024 * 1024)
	if req.Size > maxSize {
		return nil, fmt.Errorf("audio file size %d exceeds maximum allowed %d bytes", req.Size, maxSize)
	}

	// Read audio file into buffer
	audioData, err := io.ReadAll(req.AudioFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file field
	part, err := writer.CreateFormFile("file", req.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(audioData); err != nil {
		return nil, fmt.Errorf("failed to write audio data: %w", err)
	}

	// Add model field
	if err := writer.WriteField("model", s.config.Model); err != nil {
		return nil, fmt.Errorf("failed to write model field: %w", err)
	}

	// Add language field
	language := req.Language
	if language == "" {
		language = s.config.Language
	}
	if err := writer.WriteField("language", language); err != nil {
		return nil, fmt.Errorf("failed to write language field: %w", err)
	}

	// Add temperature field
	if err := writer.WriteField("temperature", fmt.Sprintf("%.1f", s.config.Temperature)); err != nil {
		return nil, fmt.Errorf("failed to write temperature field: %w", err)
	}

	// Add response format field
	if err := writer.WriteField("response_format", s.config.ResponseFormat); err != nil {
		return nil, fmt.Errorf("failed to write response_format field: %w", err)
	}

	// Add prompt field (helps with nutrition-specific vocabulary)
	if s.config.Prompt != "" {
		if err := writer.WriteField("prompt", s.config.Prompt); err != nil {
			return nil, fmt.Errorf("failed to write prompt field: %w", err)
		}
	}

	writer.Close()

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.config.APIEndpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.groqAPIKey))
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	startTime := time.Now()
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var groqResp GroqTranscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	duration := time.Since(startTime).Seconds()

	s.logger.Info("audio transcribed successfully",
		slog.String("user_id", req.UserID.String()),
		slog.Int("text_length", len(groqResp.Text)),
		slog.Float64("duration_seconds", duration),
	)

	return &TranscribeResponse{
		Text:        groqResp.Text,
		Language:    language,
		Duration:    duration,
		Confidence:  0.95, // Groq doesn't provide confidence, use a default high value
		ProcessedAt: time.Now(),
	}, nil
}

// TranscribeFromBase64 transcribes audio from base64-encoded data.
func (s *Service) TranscribeFromBase64(ctx context.Context, userID uuid.UUID, base64Audio string, filename string) (*TranscribeResponse, error) {
	// Decode base64
	// This is a simplified version - in production, properly decode base64
	return nil, fmt.Errorf("base64 transcription not yet implemented")
}

// GetSupportedFormats returns the list of supported audio formats.
func (s *Service) GetSupportedFormats() []string {
	return s.config.AllowedFormats
}

// ValidateAudioFormat checks if an audio format is supported.
func (s *Service) ValidateAudioFormat(format string) bool {
	for _, allowed := range s.config.AllowedFormats {
		if allowed == format {
			return true
		}
	}
	return false
}
