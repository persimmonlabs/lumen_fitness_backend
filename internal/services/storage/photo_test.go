package storage

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"strings"
	"testing"
)

// Helper function to create a real JPEG image for testing
func createRealJPEG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a gradient
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

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

// Helper function to create a large JPEG for compression testing
func createLargeJPEG() []byte {
	return createRealJPEG(3000, 2000)
}

func TestPhotoService_Upload(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		photo       []byte
		filename    string
		maxSize     int64
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid JPEG upload",
			userID:   "user123",
			photo:    createRealJPEG(800, 600),
			filename: "test.jpg",
			maxSize:  10 * 1024 * 1024,
			wantErr:  false,
		},
		{
			name:     "large JPEG with compression",
			userID:   "user123",
			photo:    createLargeJPEG(),
			filename: "large.jpg",
			maxSize:  10 * 1024 * 1024,
			wantErr:  false,
		},
		{
			name:        "file too large",
			userID:      "user123",
			photo:       createRealJPEG(800, 600),
			filename:    "test.jpg",
			maxSize:     100, // Very small limit
			wantErr:     true,
			errContains: "exceeds maximum",
		},
		{
			name:        "invalid file format",
			userID:      "user123",
			photo:       []byte("not an image"),
			filename:    "test.txt",
			maxSize:     10 * 1024 * 1024,
			wantErr:     true,
			errContains: "unsupported file format",
		},
		{
			name:     "empty filename",
			userID:   "user123",
			photo:    createRealJPEG(800, 600),
			filename: "",
			maxSize:  10 * 1024 * 1024,
			wantErr:  false, // Should generate filename
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create service in disabled mode (no Supabase)
			service := NewPhotoService(PhotoServiceConfig{
				Storage: nil, // Disabled mode
				Bucket:  "nutrition-photos",
				MaxSize: tt.maxSize,
			})

			ctx := context.Background()
			url, err := service.Upload(ctx, tt.userID, bytes.NewReader(tt.photo), tt.filename)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Upload() expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Upload() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Upload() unexpected error = %v", err)
				return
			}

			if url == "" {
				t.Errorf("Upload() returned empty URL")
			}

			// Verify URL format
			if !strings.Contains(url, tt.userID) {
				t.Errorf("Upload() URL should contain userID, got %v", url)
			}
		})
	}
}

func TestPhotoService_Validate(t *testing.T) {
	tests := []struct {
		name        string
		photo       []byte
		maxSize     int64
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid JPEG",
			photo:   createRealJPEG(800, 600),
			maxSize: 10 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "valid PNG",
			photo:   CreateTestPNG(),
			maxSize: 10 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:        "invalid format",
			photo:       []byte("not an image"),
			maxSize:     10 * 1024 * 1024,
			wantErr:     true,
			errContains: "unsupported file format",
		},
		{
			name:        "empty file",
			photo:       []byte{},
			maxSize:     10 * 1024 * 1024,
			wantErr:     true,
			errContains: "empty file",
		},
		{
			name:        "file too large",
			photo:       createRealJPEG(800, 600),
			maxSize:     100,
			wantErr:     true,
			errContains: "exceeds maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPhotoService(PhotoServiceConfig{
				Bucket:  "nutrition-photos",
				MaxSize: tt.maxSize,
			})

			err := service.Validate(bytes.NewReader(tt.photo))

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Validate() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestPhotoService_GetURL(t *testing.T) {
	service := NewPhotoService(PhotoServiceConfig{
		Bucket:  "nutrition-photos",
		MaxSize: 10 * 1024 * 1024,
	})

	ctx := context.Background()
	path := "user123/photo.jpg"

	url, err := service.GetURL(ctx, path)
	if err != nil {
		t.Errorf("GetURL() unexpected error = %v", err)
	}

	if !strings.Contains(url, path) {
		t.Errorf("GetURL() = %v, want URL containing %v", url, path)
	}

	if !strings.Contains(url, "nutrition-photos") {
		t.Errorf("GetURL() = %v, want URL containing bucket name", url)
	}
}

func TestPhotoService_Delete(t *testing.T) {
	service := NewPhotoService(PhotoServiceConfig{
		Bucket:  "nutrition-photos",
		MaxSize: 10 * 1024 * 1024,
	})

	ctx := context.Background()
	path := "user123/photo.jpg"

	// Should not error in disabled mode
	err := service.Delete(ctx, path)
	if err != nil {
		t.Errorf("Delete() unexpected error = %v", err)
	}
}

func TestPhotoService_IsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		storage interface{}
		bucket  string
		want    bool
	}{
		{
			name:    "disabled - no storage",
			storage: nil,
			bucket:  "test",
			want:    false,
		},
		{
			name:    "disabled - no bucket",
			storage: &struct{}{},
			bucket:  "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPhotoService(PhotoServiceConfig{
				Storage: nil,
				Bucket:  tt.bucket,
			})

			if got := service.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPhotoService_generateStoragePath(t *testing.T) {
	service := NewPhotoService(PhotoServiceConfig{
		Bucket:  "nutrition-photos",
		MaxSize: 10 * 1024 * 1024,
	})

	tests := []struct {
		name     string
		userID   string
		filename string
	}{
		{
			name:     "with extension",
			userID:   "user123",
			filename: "photo.jpg",
		},
		{
			name:     "without extension",
			userID:   "user456",
			filename: "photo",
		},
		{
			name:     "png extension",
			userID:   "user789",
			filename: "image.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := service.generateStoragePath(tt.userID, tt.filename)

			// Should start with userID
			if !strings.HasPrefix(path, tt.userID+"\\") && !strings.HasPrefix(path, tt.userID+"/") {
				t.Errorf("generateStoragePath() = %v, want path starting with %v", path, tt.userID)
			}

			// Should contain timestamp
			if !strings.Contains(path, "_") {
				t.Errorf("generateStoragePath() = %v, want path containing timestamp separator", path)
			}

			// Should have extension
			if !strings.HasSuffix(path, ".jpg") && !strings.HasSuffix(path, ".png") {
				t.Errorf("generateStoragePath() = %v, want path ending with image extension", path)
			}
		})
	}
}

func TestPhotoService_getMockURL(t *testing.T) {
	service := NewPhotoService(PhotoServiceConfig{
		Bucket:  "nutrition-photos",
		MaxSize: 10 * 1024 * 1024,
	})

	path := "user123/photo.jpg"
	url := service.getMockURL(path)

	expectedParts := []string{
		"http://localhost:54321",
		"storage/v1/object/public",
		"nutrition-photos",
		"user123",
	}

	for _, part := range expectedParts {
		if !strings.Contains(url, part) {
			t.Errorf("getMockURL() = %v, want URL containing %v", url, part)
		}
	}
}

func TestPhotoService_DefaultMaxSize(t *testing.T) {
	service := NewPhotoService(PhotoServiceConfig{
		Bucket: "nutrition-photos",
		// MaxSize not specified
	})

	expectedMaxSize := int64(10 * 1024 * 1024) // 10MB
	if service.maxSize != expectedMaxSize {
		t.Errorf("NewPhotoService() maxSize = %v, want %v", service.maxSize, expectedMaxSize)
	}
}
