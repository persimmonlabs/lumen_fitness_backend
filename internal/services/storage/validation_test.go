package storage

import (
	"strings"
	"testing"
)

func TestValidateImageSignature(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid JPEG",
			data:    createRealJPEG(100, 100),
			wantErr: false,
		},
		{
			name:    "valid PNG",
			data:    CreateTestPNG(),
			wantErr: false,
		},
		{
			name: "valid HEIC",
			data: []byte{
				0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70, // ftyp box
				0x68, 0x65, 0x69, 0x63, // heic brand
			},
			wantErr: false,
		},
		{
			name:        "invalid format - text",
			data:        []byte("This is plain text"),
			wantErr:     true,
			errContains: "unsupported file format",
		},
		{
			name:        "empty file",
			data:        []byte{},
			wantErr:     true,
			errContains: "empty file",
		},
		{
			name:        "invalid format - random bytes",
			data:        []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
			wantErr:     true,
			errContains: "unsupported file format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageSignature(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateImageSignature() expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateImageSignature() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateImageSignature() unexpected error = %v", err)
			}
		})
	}
}

func TestIsJPEG(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "valid JPEG",
			data: createRealJPEG(100, 100),
			want: true,
		},
		{
			name: "JPEG signature only",
			data: []byte{0xFF, 0xD8, 0xFF},
			want: true,
		},
		{
			name: "PNG file",
			data: CreateTestPNG(),
			want: false,
		},
		{
			name: "random bytes",
			data: []byte{0x00, 0x01, 0x02},
			want: false,
		},
		{
			name: "empty",
			data: []byte{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isJPEG(tt.data); got != tt.want {
				t.Errorf("isJPEG() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPNG(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "valid PNG",
			data: CreateTestPNG(),
			want: true,
		},
		{
			name: "PNG signature",
			data: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			want: true,
		},
		{
			name: "JPEG file",
			data: createRealJPEG(100, 100),
			want: false,
		},
		{
			name: "random bytes",
			data: []byte{0x00, 0x01, 0x02},
			want: false,
		},
		{
			name: "partial signature",
			data: []byte{0x89, 0x50, 0x4E},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPNG(tt.data); got != tt.want {
				t.Errorf("isPNG() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsHEICSignature(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "valid HEIC",
			data: []byte{
				0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70,
				0x68, 0x65, 0x69, 0x63,
			},
			want: true,
		},
		{
			name: "valid HEIF",
			data: []byte{
				0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70,
				0x68, 0x65, 0x69, 0x66,
			},
			want: true,
		},
		{
			name: "valid MIF1",
			data: []byte{
				0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70,
				0x6D, 0x69, 0x66, 0x31,
			},
			want: true,
		},
		{
			name: "JPEG file",
			data: createRealJPEG(100, 100),
			want: false,
		},
		{
			name: "too short",
			data: []byte{0x00, 0x00, 0x00},
			want: false,
		},
		{
			name: "wrong ftyp",
			data: []byte{
				0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70,
				0x69, 0x73, 0x6F, 0x6D, // isom brand (MP4)
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHEICSignature(tt.data); got != tt.want {
				t.Errorf("isHEICSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateImageSize(t *testing.T) {
	tests := []struct {
		name        string
		size        int64
		maxSize     int64
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid size",
			size:    1024 * 1024, // 1MB
			maxSize: 10 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "exactly at limit",
			size:    10 * 1024 * 1024,
			maxSize: 10 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:        "exceeds limit",
			size:        11 * 1024 * 1024,
			maxSize:     10 * 1024 * 1024,
			wantErr:     true,
			errContains: "exceeds maximum",
		},
		{
			name:        "zero size",
			size:        0,
			maxSize:     10 * 1024 * 1024,
			wantErr:     true,
			errContains: "invalid file size",
		},
		{
			name:        "negative size",
			size:        -1,
			maxSize:     10 * 1024 * 1024,
			wantErr:     true,
			errContains: "invalid file size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageSize(tt.size, tt.maxSize)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateImageSize() expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateImageSize() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateImageSize() unexpected error = %v", err)
			}
		})
	}
}

func TestValidateImageDimensions(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		height      int
		maxWidth    int
		maxHeight   int
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid dimensions",
			width:     800,
			height:    600,
			maxWidth:  2000,
			maxHeight: 2000,
			wantErr:   false,
		},
		{
			name:      "exactly at limit",
			width:     2000,
			height:    2000,
			maxWidth:  2000,
			maxHeight: 2000,
			wantErr:   false,
		},
		{
			name:        "width exceeds limit",
			width:       2001,
			height:      1000,
			maxWidth:    2000,
			maxHeight:   2000,
			wantErr:     true,
			errContains: "exceed maximum",
		},
		{
			name:        "height exceeds limit",
			width:       1000,
			height:      2001,
			maxWidth:    2000,
			maxHeight:   2000,
			wantErr:     true,
			errContains: "exceed maximum",
		},
		{
			name:        "zero width",
			width:       0,
			height:      1000,
			maxWidth:    2000,
			maxHeight:   2000,
			wantErr:     true,
			errContains: "invalid image dimensions",
		},
		{
			name:        "negative dimensions",
			width:       -1,
			height:      1000,
			maxWidth:    2000,
			maxHeight:   2000,
			wantErr:     true,
			errContains: "invalid image dimensions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageDimensions(tt.width, tt.height, tt.maxWidth, tt.maxHeight)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateImageDimensions() expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateImageDimensions() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateImageDimensions() unexpected error = %v", err)
			}
		})
	}
}

func TestGetFileExtensionFromSignature(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    string
		wantErr bool
	}{
		{
			name:    "JPEG file",
			data:    createRealJPEG(100, 100),
			want:    ".jpg",
			wantErr: false,
		},
		{
			name:    "PNG file",
			data:    CreateTestPNG(),
			want:    ".png",
			wantErr: false,
		},
		{
			name: "HEIC file",
			data: []byte{
				0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70,
				0x68, 0x65, 0x69, 0x63,
			},
			want:    ".heic",
			wantErr: false,
		},
		{
			name:    "unknown file",
			data:    []byte("random data"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFileExtensionFromSignature(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetFileExtensionFromSignature() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetFileExtensionFromSignature() unexpected error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("GetFileExtensionFromSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Benchmark tests
func BenchmarkValidateImageSignature(b *testing.B) {
	data := createRealJPEG(100, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateImageSignature(data)
	}
}

func BenchmarkIsJPEG(b *testing.B) {
	data := createRealJPEG(100, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isJPEG(data)
	}
}

func BenchmarkIsPNG(b *testing.B) {
	data := CreateTestPNG()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isPNG(data)
	}
}

func BenchmarkIsHEICSignature(b *testing.B) {
	data := []byte{
		0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70,
		0x68, 0x65, 0x69, 0x63,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isHEICSignature(data)
	}
}
