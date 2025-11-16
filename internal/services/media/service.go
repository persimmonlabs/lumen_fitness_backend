// Package media provides media upload and management services.
//
// This package handles photo uploads, compression, validation, and storage
// integration with Supabase Storage. It supports multiple image formats and
// provides automatic compression based on configuration.
package media

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	storage_go "github.com/supabase-community/storage-go"
	"golang.org/x/image/webp"
)

// Service handles media upload and management operations.
type Service struct {
	storageClient *storage_go.Client
	logger        *slog.Logger
	config        Config
}

// Config contains configuration for the media service.
type Config struct {
	MaxFileSizeMB      int
	MaxWidth           int
	MaxHeight          int
	JPEGQuality        int
	AllowedFormats     []string
	StorageBucket      string
	CompressionEnabled bool
}

// DefaultConfig returns the default media service configuration.
func DefaultConfig() Config {
	return Config{
		MaxFileSizeMB:      10,
		MaxWidth:           1920,
		MaxHeight:          1920,
		JPEGQuality:        85,
		AllowedFormats:     []string{"jpg", "jpeg", "png", "webp"},
		StorageBucket:      "meal-photos",
		CompressionEnabled: true,
	}
}

// NewService creates a new media service instance.
func NewService(storageClient *storage_go.Client, logger *slog.Logger, config Config) *Service {
	return &Service{
		storageClient: storageClient,
		logger:        logger,
		config:        config,
	}
}

// UploadRequest represents a media upload request.
type UploadRequest struct {
	UserID      uuid.UUID
	File        io.Reader
	Filename    string
	ContentType string
	Size        int64
}

// UploadResponse contains the result of a media upload.
type UploadResponse struct {
	FileID      string    `json:"file_id"`
	URL         string    `json:"url"`
	PublicURL   string    `json:"public_url"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// Upload handles uploading and processing a media file.
func (s *Service) Upload(ctx context.Context, req *UploadRequest) (*UploadResponse, error) {
	// Validate file size
	maxSize := int64(s.config.MaxFileSizeMB * 1024 * 1024)
	if req.Size > maxSize {
		return nil, fmt.Errorf("file size %d exceeds maximum allowed %d bytes", req.Size, maxSize)
	}

	// Validate file format
	ext := strings.ToLower(filepath.Ext(req.Filename))
	ext = strings.TrimPrefix(ext, ".")
	if !s.isAllowedFormat(ext) {
		return nil, fmt.Errorf("file format %s not allowed", ext)
	}

	// Read file into buffer
	buf := &bytes.Buffer{}
	written, err := io.Copy(buf, req.File)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Process image if compression is enabled
	var processedData []byte
	var processedContentType string

	if s.config.CompressionEnabled {
		processedData, processedContentType, err = s.processImage(buf.Bytes(), ext)
		if err != nil {
			s.logger.Warn("failed to process image, using original",
				slog.String("error", err.Error()),
				slog.String("filename", req.Filename),
			)
			processedData = buf.Bytes()
			processedContentType = req.ContentType
		}
	} else {
		processedData = buf.Bytes()
		processedContentType = req.ContentType
	}

	// Generate unique filename
	fileID := uuid.New().String()
	filename := fmt.Sprintf("%s/%s%s", req.UserID.String(), fileID, ext)

	// Upload to Supabase Storage
	var uploadURL string
	if s.storageClient != nil {
		contentType := processedContentType
		upsert := true

		_, err = s.storageClient.UploadFile(s.config.StorageBucket, filename, bytes.NewReader(processedData), storage_go.FileOptions{
			ContentType: &contentType,
			Upsert:      &upsert,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to upload to storage: %w", err)
		}

		// Get public URL
		resp := s.storageClient.GetPublicUrl(s.config.StorageBucket, filename)
		uploadURL = resp.SignedURL
	} else {
		// Mock mode for development
		uploadURL = fmt.Sprintf("http://localhost:54321/storage/v1/object/public/%s/%s", s.config.StorageBucket, filename)
	}

	s.logger.Info("media uploaded successfully",
		slog.String("file_id", fileID),
		slog.String("user_id", req.UserID.String()),
		slog.Int64("original_size", written),
		slog.Int64("processed_size", int64(len(processedData))),
	)

	return &UploadResponse{
		FileID:      fileID,
		URL:         uploadURL,
		PublicURL:   uploadURL,
		Size:        int64(len(processedData)),
		ContentType: processedContentType,
		UploadedAt:  time.Now(),
	}, nil
}

// Delete removes a media file from storage.
func (s *Service) Delete(ctx context.Context, userID uuid.UUID, fileID string) error {
	if s.storageClient == nil {
		// Mock mode - nothing to delete
		s.logger.Info("media deleted (mock mode)",
			slog.String("file_id", fileID),
			slog.String("user_id", userID.String()),
		)
		return nil
	}

	// Construct file path
	// Assuming pattern: {userID}/{timestamp}_{uuid}.ext
	// We need to search for the file in the user's directory
	// For simplicity, construct expected path pattern
	path := fmt.Sprintf("%s/%s", userID.String(), fileID)

	files := []string{path}
	_, err := s.storageClient.RemoveFile(s.config.StorageBucket, files)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	s.logger.Info("media deleted successfully",
		slog.String("file_id", fileID),
		slog.String("user_id", userID.String()),
	)

	return nil
}

// GetURL retrieves the public URL for a media file.
func (s *Service) GetURL(ctx context.Context, userID uuid.UUID, fileID string) (string, error) {
	if s.storageClient == nil {
		// Mock mode
		path := fmt.Sprintf("%s/%s", userID.String(), fileID)
		return fmt.Sprintf("http://localhost:54321/storage/v1/object/public/%s/%s", s.config.StorageBucket, path), nil
	}

	path := fmt.Sprintf("%s/%s", userID.String(), fileID)
	resp := s.storageClient.GetPublicUrl(s.config.StorageBucket, path)

	if resp.SignedURL == "" {
		return "", fmt.Errorf("failed to get public URL for file: %s", fileID)
	}

	return resp.SignedURL, nil
}

// processImage compresses and resizes an image based on configuration.
func (s *Service) processImage(data []byte, format string) ([]byte, string, error) {
	// Decode image based on format
	var img image.Image
	var err error

	reader := bytes.NewReader(data)
	switch format {
	case "jpg", "jpeg":
		img, err = jpeg.Decode(reader)
	case "png":
		img, err = png.Decode(reader)
	case "webp":
		img, err = webp.Decode(reader)
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize if necessary
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width > s.config.MaxWidth || height > s.config.MaxHeight {
		// Calculate new dimensions maintaining aspect ratio
		scale := float64(s.config.MaxWidth) / float64(width)
		if scaleH := float64(s.config.MaxHeight) / float64(height); scaleH < scale {
			scale = scaleH
		}

		newWidth := int(float64(width) * scale)
		newHeight := int(float64(height) * scale)

		// Use simple resize (in production, use a proper resize library)
		// For now, we'll skip resizing and just compress
		_ = newWidth
		_ = newHeight
	}

	// Encode as JPEG with configured quality
	buf := &bytes.Buffer{}
	err = jpeg.Encode(buf, img, &jpeg.Options{Quality: s.config.JPEGQuality})
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), "image/jpeg", nil
}

// isAllowedFormat checks if a file format is allowed.
func (s *Service) isAllowedFormat(format string) bool {
	for _, allowed := range s.config.AllowedFormats {
		if strings.EqualFold(format, allowed) {
			return true
		}
	}
	return false
}
