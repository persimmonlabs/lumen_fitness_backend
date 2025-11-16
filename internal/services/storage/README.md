# Photo Storage Service

A comprehensive photo storage service with automatic compression, validation, and Supabase Storage integration.

## Features

- **Automatic Image Compression**: Intelligently compresses images to target size (default: 1MB) while maintaining quality
- **Format Support**: JPEG, PNG, and HEIC/HEIF detection
- **File Validation**: Magic byte signature checking, size limits, and corruption detection
- **Supabase Storage Integration**: Seamless integration with optional graceful degradation
- **Unique Filenames**: Automatic generation with UUID and timestamps
- **Mock Mode**: Works without Supabase for development/testing

## Installation

```bash
go get github.com/supabase-community/storage-go
go get github.com/google/uuid
go get golang.org/x/image/draw
```

## Usage

### Basic Setup

```go
import (
    "github.com/pradord/lumen_final/backend/internal/services/storage"
    storage_go "github.com/supabase-community/storage-go"
)

// Initialize with Supabase Storage
storageClient := storage_go.NewClient(
    "https://your-project.supabase.co/storage/v1",
    "your-anon-key",
    nil,
)

photoService := storage.NewPhotoService(storage.PhotoServiceConfig{
    Storage: storageClient,
    Bucket:  "nutrition-photos",
    MaxSize: 10 * 1024 * 1024, // 10MB
})

// Or use mock mode (no Supabase)
photoService := storage.NewPhotoService(storage.PhotoServiceConfig{
    Bucket:  "nutrition-photos",
    MaxSize: 10 * 1024 * 1024,
})
```

### Upload Photo

```go
ctx := context.Background()
file, _ := os.Open("photo.jpg")
defer file.Close()

url, err := photoService.Upload(ctx, "user123", file, "photo.jpg")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Photo URL:", url)
```

### Get Photo URL

```go
url, err := photoService.GetURL(ctx, "user123/1234567890_uuid.jpg")
if err != nil {
    log.Fatal(err)
}
```

### Delete Photo

```go
err := photoService.Delete(ctx, "user123/1234567890_uuid.jpg")
if err != nil {
    log.Fatal(err)
}
```

### Validate Photo

```go
file, _ := os.Open("photo.jpg")
defer file.Close()

err := photoService.Validate(file)
if err != nil {
    fmt.Println("Invalid photo:", err)
}
```

## Compression Strategy

The service uses a multi-step compression approach:

1. **Decode**: Support JPEG, PNG (HEIC detection for future support)
2. **Resize**: If image exceeds 2000px on longest side, resize proportionally
3. **Encode**: Re-encode as JPEG with quality 85
4. **Optimize**: If still > 1MB, reduce quality to 75
5. **Fallback**: If still > 1MB, resize to 1500px maximum

### Compression Options

```go
opts := storage.CompressionOptions{
    MaxWidth:      2000,    // Maximum width in pixels
    MaxHeight:     2000,    // Maximum height in pixels
    Quality:       85,      // Initial JPEG quality (1-100)
    TargetSize:    1048576, // Target size in bytes (1MB)
    MinQuality:    75,      // Minimum acceptable quality
    FallbackWidth: 1500,    // Fallback max dimension
}

compressed, err := storage.CompressImage(reader, opts)
```

## Validation

### File Signature Validation

The service validates files by their magic bytes (file signature), not just extension:

- **JPEG**: `FF D8 FF`
- **PNG**: `89 50 4E 47 0D 0A 1A 0A`
- **HEIC/HEIF**: `ftyp` box with compatible brands

### Size Validation

- Default maximum: 10MB (configurable)
- Checks actual file size before processing

### Dimension Validation

Optional validation for image dimensions:

```go
err := storage.ValidateImageDimensions(width, height, 5000, 5000)
```

## Storage Path Format

Photos are stored with unique paths:

```
{userID}/{timestamp}_{uuid}{extension}

Example: user123/1731702345_a1b2c3d4-e5f6-7890-abcd-ef1234567890.jpg
```

## Configuration

### PhotoServiceConfig

```go
type PhotoServiceConfig struct {
    Storage *storage_go.Client // Supabase Storage client (nil for mock mode)
    Bucket  string              // Storage bucket name
    MaxSize int64               // Maximum file size in bytes (default: 10MB)
}
```

## Testing

Run tests:

```bash
go test ./internal/services/storage/... -v
```

The service includes comprehensive tests:

- Photo upload with compression
- File validation (format, size, corruption)
- Mock mode functionality
- Image compression and resizing
- File signature detection

## Error Handling

The service provides detailed error messages:

```go
url, err := photoService.Upload(ctx, userID, photo, filename)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "validation failed"):
        // Handle validation error
    case strings.Contains(err.Error(), "compression failed"):
        // Handle compression error
    case strings.Contains(err.Error(), "upload failed"):
        // Handle upload error
    }
}
```

## Mock Mode

When Supabase Storage is not configured, the service operates in mock mode:

- Returns mock URLs: `http://localhost:54321/storage/v1/object/public/{bucket}/{path}`
- No actual storage operations
- Perfect for development and testing

## File Structure

```
backend/internal/services/storage/
├── photo.go              # Main PhotoService implementation
├── compression.go        # Image compression utilities
├── validation.go         # File validation functions
├── test_helpers.go       # Test helper functions
├── photo_test.go         # PhotoService tests
├── compression_test.go   # Compression tests
├── validation_test.go    # Validation tests
└── README.md            # This file
```

## Dependencies

- `github.com/supabase-community/storage-go` - Supabase Storage client
- `github.com/google/uuid` - UUID generation
- `golang.org/x/image/draw` - High-quality image resizing

## Future Enhancements

- HEIC/HEIF decoding support (requires additional library)
- Automatic EXIF orientation correction
- Support for additional image formats (WebP, AVIF)
- Progressive JPEG encoding
- Custom watermarking
- Thumbnail generation
- Batch upload support

## License

This service is part of the Lumen nutrition tracking application.
