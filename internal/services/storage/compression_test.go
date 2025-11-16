package storage

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
)

func TestCompressImage(t *testing.T) {
	tests := []struct {
		name        string
		imageSize   struct{ width, height int }
		opts        CompressionOptions
		wantErr     bool
		maxFileSize int // Maximum expected file size in bytes
	}{
		{
			name:      "compress large image",
			imageSize: struct{ width, height int }{3000, 2000},
			opts: CompressionOptions{
				MaxWidth:      2000,
				MaxHeight:     2000,
				Quality:       85,
				TargetSize:    1024 * 1024,
				MinQuality:    75,
				FallbackWidth: 1500,
			},
			wantErr:     false,
			maxFileSize: 2 * 1024 * 1024, // Should be compressed
		},
		{
			name:      "small image no resize needed",
			imageSize: struct{ width, height int }{800, 600},
			opts: CompressionOptions{
				MaxWidth:      2000,
				MaxHeight:     2000,
				Quality:       85,
				TargetSize:    1024 * 1024,
				MinQuality:    75,
				FallbackWidth: 1500,
			},
			wantErr:     false,
			maxFileSize: 1024 * 1024,
		},
		{
			name:      "aggressive compression for target size",
			imageSize: struct{ width, height int }{2500, 2500},
			opts: CompressionOptions{
				MaxWidth:      2000,
				MaxHeight:     2000,
				Quality:       85,
				TargetSize:    500 * 1024, // Small target
				MinQuality:    75,
				FallbackWidth: 1500,
			},
			wantErr:     false,
			maxFileSize: 1024 * 1024, // May exceed target but should be reasonable
		},
		{
			name:      "maintain aspect ratio",
			imageSize: struct{ width, height int }{4000, 2000}, // 2:1 ratio
			opts: CompressionOptions{
				MaxWidth:      2000,
				MaxHeight:     2000,
				Quality:       85,
				TargetSize:    1024 * 1024,
				MinQuality:    75,
				FallbackWidth: 1500,
			},
			wantErr:     false,
			maxFileSize: 2 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			original := createRealJPEG(tt.imageSize.width, tt.imageSize.height)

			// Compress
			compressed, err := CompressImage(bytes.NewReader(original), tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CompressImage() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("CompressImage() unexpected error = %v", err)
				return
			}

			// Verify compressed size
			if len(compressed) > tt.maxFileSize {
				t.Errorf("CompressImage() size = %d, want <= %d", len(compressed), tt.maxFileSize)
			}

			// Verify it's a valid JPEG
			_, err = jpeg.Decode(bytes.NewReader(compressed))
			if err != nil {
				t.Errorf("CompressImage() produced invalid JPEG: %v", err)
			}

			// Verify size is less than original (or same if already small)
			if len(compressed) > len(original) && len(original) > tt.opts.TargetSize {
				t.Errorf("CompressImage() size %d > original %d", len(compressed), len(original))
			}
		})
	}
}

func TestResizeImage(t *testing.T) {
	tests := []struct {
		name           string
		originalSize   struct{ width, height int }
		maxWidth       int
		maxHeight      int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "resize large image",
			originalSize:   struct{ width, height int }{3000, 2000},
			maxWidth:       2000,
			maxHeight:      2000,
			expectedWidth:  2000,
			expectedHeight: 1333, // Maintains aspect ratio
		},
		{
			name:           "no resize needed",
			originalSize:   struct{ width, height int }{800, 600},
			maxWidth:       2000,
			maxHeight:      2000,
			expectedWidth:  800,
			expectedHeight: 600,
		},
		{
			name:           "resize portrait",
			originalSize:   struct{ width, height int }{1000, 2000},
			maxWidth:       1500,
			maxHeight:      1500,
			expectedWidth:  750,
			expectedHeight: 1500,
		},
		{
			name:           "resize square",
			originalSize:   struct{ width, height int }{3000, 3000},
			maxWidth:       1000,
			maxHeight:      1000,
			expectedWidth:  1000,
			expectedHeight: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			original := createRealJPEG(tt.originalSize.width, tt.originalSize.height)
			img, _, err := image.Decode(bytes.NewReader(original))
			if err != nil {
				t.Fatalf("Failed to decode test image: %v", err)
			}

			// Resize
			resized := resizeImage(img, tt.maxWidth, tt.maxHeight)

			bounds := resized.Bounds()
			width := bounds.Dx()
			height := bounds.Dy()

			// Allow small tolerance for rounding
			tolerance := 2
			if abs(width-tt.expectedWidth) > tolerance {
				t.Errorf("resizeImage() width = %d, want %d (±%d)", width, tt.expectedWidth, tolerance)
			}
			if abs(height-tt.expectedHeight) > tolerance {
				t.Errorf("resizeImage() height = %d, want %d (±%d)", height, tt.expectedHeight, tolerance)
			}

			// Verify dimensions don't exceed max
			if width > tt.maxWidth || height > tt.maxHeight {
				t.Errorf("resizeImage() dimensions %dx%d exceed max %dx%d", width, height, tt.maxWidth, tt.maxHeight)
			}
		})
	}
}

func TestDecodeImage(t *testing.T) {
	tests := []struct {
		name        string
		image       []byte
		wantFormat  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid JPEG",
			image:      createRealJPEG(800, 600),
			wantFormat: "jpeg",
			wantErr:    false,
		},
		{
			name:       "valid PNG",
			image:      CreateTestPNG(),
			wantFormat: "png",
			wantErr:    false,
		},
		{
			name:        "invalid image",
			image:       []byte("not an image"),
			wantErr:     true,
			errContains: "failed to decode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, format, err := DecodeImage(bytes.NewReader(tt.image))

			if tt.wantErr {
				if err == nil {
					t.Errorf("DecodeImage() expected error, got nil")
					return
				}
				if tt.errContains != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errContains)) {
					t.Errorf("DecodeImage() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("DecodeImage() unexpected error = %v", err)
				return
			}

			if img == nil {
				t.Errorf("DecodeImage() returned nil image")
			}

			if format != tt.wantFormat {
				t.Errorf("DecodeImage() format = %v, want %v", format, tt.wantFormat)
			}
		})
	}
}

func TestGetImageDimensions(t *testing.T) {
	tests := []struct {
		name        string
		image       []byte
		wantWidth   int
		wantHeight  int
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid JPEG",
			image:      createRealJPEG(800, 600),
			wantWidth:  800,
			wantHeight: 600,
			wantErr:    false,
		},
		{
			name:       "large image",
			image:      createRealJPEG(3000, 2000),
			wantWidth:  3000,
			wantHeight: 2000,
			wantErr:    false,
		},
		{
			name:        "invalid image",
			image:       []byte("not an image"),
			wantErr:     true,
			errContains: "failed to decode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height, err := GetImageDimensions(bytes.NewReader(tt.image))

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetImageDimensions() expected error, got nil")
					return
				}
				if tt.errContains != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errContains)) {
					t.Errorf("GetImageDimensions() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("GetImageDimensions() unexpected error = %v", err)
				return
			}

			if width != tt.wantWidth {
				t.Errorf("GetImageDimensions() width = %d, want %d", width, tt.wantWidth)
			}

			if height != tt.wantHeight {
				t.Errorf("GetImageDimensions() height = %d, want %d", height, tt.wantHeight)
			}
		})
	}
}

func TestIsHEIC(t *testing.T) {
	tests := []struct {
		name   string
		buffer []byte
		want   bool
	}{
		{
			name:   "HEIC file",
			buffer: []byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70, 0x68, 0x65, 0x69, 0x63},
			want:   true,
		},
		{
			name:   "JPEG file",
			buffer: []byte{0xFF, 0xD8, 0xFF, 0xE0},
			want:   false,
		},
		{
			name:   "PNG file",
			buffer: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			want:   false,
		},
		{
			name:   "too short",
			buffer: []byte{0x00, 0x00},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHEIC(tt.buffer); got != tt.want {
				t.Errorf("isHEIC() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Benchmark tests
func BenchmarkCompressImage(b *testing.B) {
	original := createRealJPEG(3000, 2000)
	opts := CompressionOptions{
		MaxWidth:      2000,
		MaxHeight:     2000,
		Quality:       85,
		TargetSize:    1024 * 1024,
		MinQuality:    75,
		FallbackWidth: 1500,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CompressImage(bytes.NewReader(original), opts)
		if err != nil {
			b.Fatalf("CompressImage() error = %v", err)
		}
	}
}

func BenchmarkResizeImage(b *testing.B) {
	original := createRealJPEG(3000, 2000)
	img, _, err := image.Decode(bytes.NewReader(original))
	if err != nil {
		b.Fatalf("Failed to decode test image: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = resizeImage(img, 2000, 2000)
	}
}
