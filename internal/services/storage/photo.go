package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	storage_go "github.com/supabase-community/storage-go"
)

// PhotoService handles photo upload, storage, and retrieval with Supabase Storage
type PhotoService struct {
	storage *storage_go.Client
	bucket  string
	maxSize int64 // Maximum file size in bytes
	enabled bool  // Whether Supabase Storage is enabled
}

// PhotoServiceConfig holds configuration for PhotoService
type PhotoServiceConfig struct {
	Storage *storage_go.Client
	Bucket  string
	MaxSize int64 // Maximum file size in bytes (default: 10MB)
}

// NewPhotoService creates a new PhotoService instance
func NewPhotoService(config PhotoServiceConfig) *PhotoService {
	maxSize := config.MaxSize
	if maxSize == 0 {
		maxSize = 10 * 1024 * 1024 // 10MB default
	}

	enabled := config.Storage != nil && config.Bucket != ""

	return &PhotoService{
		storage: config.Storage,
		bucket:  config.Bucket,
		maxSize: maxSize,
		enabled: enabled,
	}
}

// Upload handles photo upload with automatic compression and validation
func (s *PhotoService) Upload(ctx context.Context, userID string, photo io.Reader, filename string) (string, error) {
	// Read photo data
	data, err := io.ReadAll(photo)
	if err != nil {
		return "", fmt.Errorf("failed to read photo: %w", err)
	}

	// Validate photo
	if err := s.Validate(bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	// Compress photo
	compressed, err := CompressImage(bytes.NewReader(data), CompressionOptions{
		MaxWidth:      2000,
		MaxHeight:     2000,
		Quality:       85,
		TargetSize:    1024 * 1024, // 1MB
		MinQuality:    75,
		FallbackWidth: 1500,
	})
	if err != nil {
		return "", fmt.Errorf("compression failed: %w", err)
	}

	// Generate unique filename
	storagePath := s.generateStoragePath(userID, filename)

	// Upload to Supabase Storage if enabled, otherwise return mock URL
	if !s.enabled {
		return s.getMockURL(storagePath), nil
	}

	// Upload to Supabase Storage
	if err := s.uploadToSupabase(ctx, storagePath, compressed); err != nil {
		return "", fmt.Errorf("upload to Supabase failed: %w", err)
	}

	// Get public URL
	url, err := s.GetURL(ctx, storagePath)
	if err != nil {
		return "", fmt.Errorf("failed to get URL: %w", err)
	}

	return url, nil
}

// GetURL retrieves the public URL for a stored photo
func (s *PhotoService) GetURL(ctx context.Context, path string) (string, error) {
	if !s.enabled {
		return s.getMockURL(path), nil
	}

	// Get public URL from Supabase Storage
	resp := s.storage.GetPublicUrl(s.bucket, path)
	if resp.SignedURL == "" {
		return "", fmt.Errorf("failed to get public URL for path: %s", path)
	}

	return resp.SignedURL, nil
}

// Delete removes a photo from storage
func (s *PhotoService) Delete(ctx context.Context, path string) error {
	if !s.enabled {
		return nil // Nothing to delete in mock mode
	}

	// Delete from Supabase Storage
	files := []string{path}
	_, err := s.storage.RemoveFile(s.bucket, files)
	if err != nil {
		return fmt.Errorf("failed to delete photo: %w", err)
	}

	return nil
}

// Validate checks if the photo meets requirements
func (s *PhotoService) Validate(photo io.Reader) error {
	// Read first 512 bytes for validation
	buffer := make([]byte, 512)
	n, err := photo.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read photo: %w", err)
	}

	// Validate file signature
	if err := ValidateImageSignature(buffer[:n]); err != nil {
		return err
	}

	// Check file size (we need to read the entire file)
	var size int64
	if seeker, ok := photo.(io.Seeker); ok {
		// If photo is seekable, get size efficiently
		currentPos, _ := seeker.Seek(0, io.SeekCurrent)
		endPos, err := seeker.Seek(0, io.SeekEnd)
		if err != nil {
			return fmt.Errorf("failed to get file size: %w", err)
		}
		size = endPos
		seeker.Seek(currentPos, io.SeekStart) // Reset position
	} else {
		// Otherwise, read entire file to check size
		data, err := io.ReadAll(photo)
		if err != nil {
			return fmt.Errorf("failed to read photo: %w", err)
		}
		size = int64(len(data))
	}

	if size > s.maxSize {
		return fmt.Errorf("file size %d bytes exceeds maximum %d bytes", size, s.maxSize)
	}

	return nil
}

// uploadToSupabase handles the actual upload to Supabase Storage
func (s *PhotoService) uploadToSupabase(ctx context.Context, path string, data []byte) error {
	contentType := "image/jpeg"
	upsert := true

	_, err := s.storage.UploadFile(s.bucket, path, bytes.NewReader(data), storage_go.FileOptions{
		ContentType: &contentType,
		Upsert:      &upsert, // Overwrite if exists
	})
	if err != nil {
		return fmt.Errorf("Supabase upload failed: %w", err)
	}

	return nil
}

// generateStoragePath creates a unique storage path for the photo
func (s *PhotoService) generateStoragePath(userID, filename string) string {
	// Extract extension (default to .jpg)
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".jpg"
	}

	// Generate unique filename: {userID}/{timestamp}_{uuid}{ext}
	timestamp := time.Now().Unix()
	uniqueID := uuid.New().String()
	newFilename := fmt.Sprintf("%d_%s%s", timestamp, uniqueID, ext)

	return filepath.Join(userID, newFilename)
}

// getMockURL returns a mock URL for testing/development when Supabase is disabled
func (s *PhotoService) getMockURL(path string) string {
	return fmt.Sprintf("http://localhost:54321/storage/v1/object/public/%s/%s", s.bucket, path)
}

// IsEnabled returns whether Supabase Storage is enabled
func (s *PhotoService) IsEnabled() bool {
	return s.enabled
}
