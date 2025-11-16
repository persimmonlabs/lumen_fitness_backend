package media

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"testing"
)

// MockRepository mocks the Repository interface
type MockRepository struct {
	uploadFunc func(ctx context.Context, file io.Reader, filename string, contentType string) (string, error)
	deleteFunc func(ctx context.Context, filename string) error
}

func (m *MockRepository) Upload(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, file, filename, contentType)
	}
	return "https://example.com/" + filename, nil
}

func (m *MockRepository) GetPublicURL(filename string) string {
	return "https://example.com/" + filename
}

func (m *MockRepository) Delete(ctx context.Context, filename string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, filename)
	}
	return nil
}

// createTestImage creates a test image with specified width and height
func createTestImage(width, height int) *bytes.Buffer {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a simple pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			})
		}
	}

	buf := new(bytes.Buffer)
	jpeg.Encode(buf, img, &jpeg.Options{Quality: 90})
	return buf
}

func TestService_ValidateFileSize(t *testing.T) {
	service := NewService(&MockRepository{})

	tests := []struct {
		name        string
		size        int64
		expectError bool
	}{
		{"valid size", 5 * 1024 * 1024, false},
		{"max size", MaxFileSize, false},
		{"over size", MaxFileSize + 1, true},
		{"zero size", 0, true},
		{"negative size", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateFileSize(tt.size)
			if (err != nil) != tt.expectError {
				t.Errorf("validateFileSize() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestService_ValidateFileType(t *testing.T) {
	service := NewService(&MockRepository{})

	tests := []struct {
		name        string
		filename    string
		contentType string
		expectError bool
	}{
		{"valid jpg", "photo.jpg", "image/jpeg", false},
		{"valid jpeg", "photo.jpeg", "image/jpeg", false},
		{"valid png", "photo.png", "image/png", false},
		{"valid heic", "photo.heic", "image/heic", false},
		{"invalid extension", "photo.gif", "image/gif", true},
		{"invalid mime", "photo.jpg", "image/gif", true},
		{"no extension", "photo", "image/jpeg", true},
		{"wrong case jpg", "photo.JPG", "image/jpeg", false}, // Should pass - case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateFileType(tt.filename, tt.contentType)
			if (err != nil) != tt.expectError {
				t.Errorf("validateFileType() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestService_ResizeImage(t *testing.T) {
	service := NewService(&MockRepository{})

	tests := []struct {
		name          string
		originalWidth int
		maxWidth      int
		expectResize  bool
	}{
		{"resize large image", 2500, MaxImageWidth, true},
		{"keep small image", 1000, MaxImageWidth, false},
		{"exact size", MaxImageWidth, MaxImageWidth, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			img := image.NewRGBA(image.Rect(0, 0, tt.originalWidth, 1000))

			resized := service.resizeImage(img, tt.maxWidth)

			newWidth := resized.Bounds().Dx()

			if tt.expectResize {
				if newWidth != tt.maxWidth {
					t.Errorf("expected width %d, got %d", tt.maxWidth, newWidth)
				}
			} else {
				if newWidth != tt.originalWidth {
					t.Errorf("expected width %d (no resize), got %d", tt.originalWidth, newWidth)
				}
			}
		})
	}
}

func TestService_ConvertToWebP(t *testing.T) {
	service := NewService(&MockRepository{})

	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	data, err := service.convertToWebP(img, WebPQuality)
	if err != nil {
		t.Fatalf("convertToWebP() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty WebP data")
	}
}

func TestService_ConvertToJPEG(t *testing.T) {
	service := NewService(&MockRepository{})

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	data, err := service.convertToJPEG(img, JPEGQuality)
	if err != nil {
		t.Fatalf("convertToJPEG() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JPEG data")
	}
}

func TestService_ConvertToPNG(t *testing.T) {
	service := NewService(&MockRepository{})

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	data, err := service.convertToPNG(img)
	if err != nil {
		t.Fatalf("convertToPNG() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty PNG data")
	}
}

func TestService_ProcessAndUpload(t *testing.T) {
	tests := []struct {
		name          string
		setupImage    func() *bytes.Buffer
		filename      string
		contentType   string
		fileSize      int64
		mockUpload    func(ctx context.Context, file io.Reader, filename string, contentType string) (string, error)
		expectError   bool
		errorContains string
	}{
		{
			name: "successful upload with JPEG",
			setupImage: func() *bytes.Buffer {
				return createTestImage(800, 600)
			},
			filename:    "test.jpg",
			contentType: "image/jpeg",
			fileSize:    1024 * 100, // 100KB
			mockUpload: func(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
				return "https://example.com/" + filename, nil
			},
			expectError: false,
		},
		{
			name: "file too large",
			setupImage: func() *bytes.Buffer {
				return createTestImage(800, 600)
			},
			filename:      "test.jpg",
			contentType:   "image/jpeg",
			fileSize:      MaxFileSize + 1,
			expectError:   true,
			errorContains: "file size exceeds",
		},
		{
			name: "invalid file type",
			setupImage: func() *bytes.Buffer {
				return createTestImage(800, 600)
			},
			filename:      "test.gif",
			contentType:   "image/gif",
			fileSize:      1024 * 100,
			expectError:   true,
			errorContains: "invalid file extension",
		},
		{
			name: "successful upload with PNG",
			setupImage: func() *bytes.Buffer {
				img := image.NewRGBA(image.Rect(0, 0, 800, 600))
				buf := new(bytes.Buffer)
				png.Encode(buf, img)
				return buf
			},
			filename:    "test.png",
			contentType: "image/png",
			fileSize:    1024 * 100,
			mockUpload: func(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
				return "https://example.com/" + filename, nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uploadCallCount := 0
			mockRepo := &MockRepository{
				uploadFunc: func(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
					uploadCallCount++
					if tt.mockUpload != nil {
						return tt.mockUpload(ctx, file, filename, contentType)
					}
					return "https://example.com/" + filename, nil
				},
			}

			service := NewService(mockRepo)

			imageData := tt.setupImage()

			result, err := service.ProcessAndUpload(
				context.Background(),
				imageData,
				tt.filename,
				tt.contentType,
				tt.fileSize,
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Fatal("expected non-nil result")
				}
				if result.URL == "" {
					t.Error("expected non-empty URL")
				}
				if result.ThumbnailURL == "" {
					t.Error("expected non-empty thumbnail URL")
				}
				// Should upload both main image and thumbnail
				if uploadCallCount != 2 {
					t.Errorf("expected 2 uploads (image + thumbnail), got %d", uploadCallCount)
				}
			}
		})
	}
}

func TestNewService(t *testing.T) {
	repo := &MockRepository{}
	service := NewService(repo)

	if service == nil {
		t.Fatal("expected non-nil service")
	}

	if service.repo != repo {
		t.Error("repository not set correctly")
	}
}
