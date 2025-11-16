package storage

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"
)

// CompressionOptions defines parameters for image compression
type CompressionOptions struct {
	MaxWidth      int // Maximum width in pixels
	MaxHeight     int // Maximum height in pixels
	Quality       int // JPEG quality (1-100)
	TargetSize    int // Target file size in bytes
	MinQuality    int // Minimum acceptable JPEG quality
	FallbackWidth int // Fallback width if target size not met
}

// CompressImage compresses an image according to the specified options
func CompressImage(reader io.Reader, opts CompressionOptions) ([]byte, error) {
	// Decode image
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get image dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Determine if resizing is needed
	needsResize := width > opts.MaxWidth || height > opts.MaxHeight

	var resized image.Image = img
	if needsResize {
		resized = resizeImage(img, opts.MaxWidth, opts.MaxHeight)
	}

	// Encode as JPEG with initial quality
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: opts.Quality}); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	// If still too large, reduce quality
	if buf.Len() > opts.TargetSize && opts.Quality > opts.MinQuality {
		buf.Reset()
		if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: opts.MinQuality}); err != nil {
			return nil, fmt.Errorf("failed to encode image with reduced quality: %w", err)
		}
	}

	// If still too large, resize more aggressively
	if buf.Len() > opts.TargetSize && opts.FallbackWidth > 0 {
		resized = resizeImage(img, opts.FallbackWidth, opts.FallbackWidth)
		buf.Reset()
		if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: opts.MinQuality}); err != nil {
			return nil, fmt.Errorf("failed to encode image with fallback size: %w", err)
		}
	}

	// Handle PNG specially if original was PNG
	if format == "png" {
		// For PNG, we always convert to JPEG for better compression
		// Already done above
	}

	return buf.Bytes(), nil
}

// resizeImage resizes an image to fit within the specified dimensions while maintaining aspect ratio
func resizeImage(img image.Image, maxWidth, maxHeight int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate scaling factor
	scaleX := float64(maxWidth) / float64(width)
	scaleY := float64(maxHeight) / float64(height)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Don't upscale
	if scale >= 1.0 {
		return img
	}

	// Calculate new dimensions
	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	// Create new image
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Use high-quality scaling
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	return dst
}

// DecodeImage decodes an image from a reader, supporting JPEG, PNG, and HEIC
func DecodeImage(reader io.Reader) (image.Image, string, error) {
	// Read first 512 bytes to detect format
	buffer := make([]byte, 512)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, "", fmt.Errorf("failed to read image header: %w", err)
	}

	// Check if HEIC/HEIF (simplified check)
	if isHEIC(buffer[:n]) {
		return nil, "", fmt.Errorf("HEIC/HEIF format not yet supported - please convert to JPEG or PNG")
	}

	// Create a new reader with the buffer prepended
	fullReader := io.MultiReader(bytes.NewReader(buffer[:n]), reader)

	// Decode image
	img, format, err := image.Decode(fullReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	return img, format, nil
}

// isHEIC checks if the buffer contains HEIC/HEIF file signature
func isHEIC(buffer []byte) bool {
	if len(buffer) < 12 {
		return false
	}

	// HEIC files have 'ftyp' at bytes 4-7 and 'heic' or 'mif1' at bytes 8-11
	return bytes.Equal(buffer[4:8], []byte("ftyp")) &&
		(bytes.Equal(buffer[8:12], []byte("heic")) ||
			bytes.Equal(buffer[8:12], []byte("mif1")) ||
			bytes.Equal(buffer[8:12], []byte("heif")))
}

// GetImageDimensions returns the dimensions of an image
func GetImageDimensions(reader io.Reader) (width, height int, err error) {
	config, _, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image config: %w", err)
	}

	return config.Width, config.Height, nil
}

// init registers image formats
func init() {
	// Register standard formats
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("png", "\x89PNG\r\n\x1a\n", png.Decode, png.DecodeConfig)
}
