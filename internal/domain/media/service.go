package media

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

const (
	MaxFileSize      = 10 * 1024 * 1024 // 10MB
	MaxImageWidth    = 1920
	ThumbnailWidth   = 400
	WebPQuality      = 85
	JPEGQuality      = 90
)

var (
	AllowedExtensions = []string{".jpg", ".jpeg", ".png", ".heic"}
	AllowedMIMETypes  = []string{"image/jpeg", "image/png", "image/heic"}
)

// Service handles media processing and storage operations
type Service struct {
	repo Repository
}

// NewService creates a new media service
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// UploadResult contains the URLs of uploaded media
type UploadResult struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// ProcessAndUpload validates, optimizes, and uploads an image file
func (s *Service) ProcessAndUpload(ctx context.Context, file io.Reader, filename string, contentType string, fileSize int64) (*UploadResult, error) {
	// Validate file size
	if err := s.validateFileSize(fileSize); err != nil {
		return nil, err
	}

	// Validate file type
	if err := s.validateFileType(filename, contentType); err != nil {
		return nil, err
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Optimize main image
	optimizedImg := s.resizeImage(img, MaxImageWidth)
	optimizedData, err := s.convertToJPEG(optimizedImg, JPEGQuality)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to JPEG: %w", err)
	}

	// Generate thumbnail
	thumbnail := s.resizeImage(img, ThumbnailWidth)
	thumbnailData, err := s.convertToJPEG(thumbnail, JPEGQuality)
	if err != nil {
		return nil, fmt.Errorf("failed to convert thumbnail to JPEG: %w", err)
	}

	// Generate JPEG filenames
	baseFilename := strings.TrimSuffix(filename, filepath.Ext(filename))
	jpegFilename := baseFilename + ".jpg"
	thumbnailFilename := baseFilename + "_thumb.jpg"

	// Upload main image
	url, err := s.repo.Upload(ctx, bytes.NewReader(optimizedData), jpegFilename, "image/jpeg")
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	// Upload thumbnail
	thumbnailURL, err := s.repo.Upload(ctx, bytes.NewReader(thumbnailData), thumbnailFilename, "image/jpeg")
	if err != nil {
		// Try to clean up main image if thumbnail upload fails
		_ = s.repo.Delete(ctx, jpegFilename)
		return nil, fmt.Errorf("failed to upload thumbnail: %w", err)
	}

	// Log original format for debugging
	_ = format // Can be used for metrics/logging

	return &UploadResult{
		URL:          url,
		ThumbnailURL: thumbnailURL,
	}, nil
}

// validateFileSize checks if the file size is within limits
func (s *Service) validateFileSize(size int64) error {
	if size <= 0 {
		return fmt.Errorf("file is empty")
	}
	if size > MaxFileSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", MaxFileSize)
	}
	return nil
}

// validateFileType checks if the file type is allowed
func (s *Service) validateFileType(filename, contentType string) error {
	ext := strings.ToLower(filepath.Ext(filename))

	// Check extension
	isValidExt := false
	for _, allowed := range AllowedExtensions {
		if ext == allowed {
			isValidExt = true
			break
		}
	}
	if !isValidExt {
		return fmt.Errorf("invalid file extension: %s. Allowed: %v", ext, AllowedExtensions)
	}

	// Check MIME type
	isValidMIME := false
	for _, allowed := range AllowedMIMETypes {
		if contentType == allowed {
			isValidMIME = true
			break
		}
	}
	if !isValidMIME {
		return fmt.Errorf("invalid content type: %s. Allowed: %v", contentType, AllowedMIMETypes)
	}

	return nil
}

// resizeImage resizes an image to fit within maxWidth while maintaining aspect ratio
func (s *Service) resizeImage(img image.Image, maxWidth int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()

	// If image is already smaller than maxWidth, return as is
	if width <= maxWidth {
		return img
	}

	// Resize maintaining aspect ratio
	return imaging.Resize(img, maxWidth, 0, imaging.Lanczos)
}

// convertToWebP converts an image to WebP format
// NOTE: WebP encoding requires CGO and libwebp. For Windows compatibility,
// we currently use JPEG format. To enable WebP:
// 1. Install libwebp: https://developers.google.com/speed/webp/download
// 2. Enable CGO: set CGO_ENABLED=1
// 3. Install a C compiler (MinGW-w64 or Visual Studio)
// 4. Use github.com/kolesa-team/go-webp or github.com/chai2010/webp
func (s *Service) convertToWebP(img image.Image, quality int) ([]byte, error) {
	// Fallback to JPEG for now
	return s.convertToJPEG(img, quality)
}

// convertToJPEG converts an image to JPEG format (fallback if WebP fails)
func (s *Service) convertToJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer

	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return buf.Bytes(), nil
}

// convertToPNG converts an image to PNG format
func (s *Service) convertToPNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer

	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}
