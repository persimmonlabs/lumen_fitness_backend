package storage

import (
	"bytes"
	"fmt"
)

// Supported image formats with their magic bytes (file signatures)
var (
	// JPEG signatures
	jpegSignatures = [][]byte{
		{0xFF, 0xD8, 0xFF},
	}

	// PNG signature
	pngSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	// HEIC/HEIF signatures (simplified)
	heicSignatures = [][]byte{
		// 'ftyp' followed by 'heic', 'heif', 'mif1', etc.
		{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70, 0x68, 0x65, 0x69, 0x63}, // heic
		{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70, 0x68, 0x65, 0x69, 0x63}, // heic variant
		{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70, 0x6D, 0x69, 0x66, 0x31}, // mif1
	}
)

// ValidateImageSignature checks if the data has a valid image file signature
func ValidateImageSignature(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty file")
	}

	// Check JPEG
	if isJPEG(data) {
		return nil
	}

	// Check PNG
	if isPNG(data) {
		return nil
	}

	// Check HEIC/HEIF
	if isHEICSignature(data) {
		return nil
	}

	return fmt.Errorf("unsupported file format - only JPEG, PNG, and HEIC/HEIF are accepted")
}

// isJPEG checks if data starts with JPEG signature
func isJPEG(data []byte) bool {
	for _, sig := range jpegSignatures {
		if len(data) >= len(sig) && bytes.Equal(data[:len(sig)], sig) {
			return true
		}
	}
	return false
}

// isPNG checks if data starts with PNG signature
func isPNG(data []byte) bool {
	return len(data) >= len(pngSignature) && bytes.Equal(data[:len(pngSignature)], pngSignature)
}

// isHEICSignature checks if data has HEIC/HEIF signature
func isHEICSignature(data []byte) bool {
	// HEIC files need at least 12 bytes
	if len(data) < 12 {
		return false
	}

	// Check for 'ftyp' box (bytes 4-7)
	if !bytes.Equal(data[4:8], []byte("ftyp")) {
		return false
	}

	// Check for compatible brands at bytes 8-11
	compatibleBrands := [][]byte{
		[]byte("heic"),
		[]byte("heif"),
		[]byte("mif1"),
		[]byte("msf1"),
		[]byte("hevc"),
		[]byte("hevx"),
		[]byte("heix"),
	}

	brand := data[8:12]
	for _, cb := range compatibleBrands {
		if bytes.Equal(brand, cb) {
			return true
		}
	}

	return false
}

// ValidateImageSize checks if the file size is within acceptable limits
func ValidateImageSize(size int64, maxSize int64) error {
	if size <= 0 {
		return fmt.Errorf("invalid file size: %d", size)
	}

	if size > maxSize {
		return fmt.Errorf("file size %d bytes exceeds maximum allowed size %d bytes", size, maxSize)
	}

	return nil
}

// ValidateImageDimensions checks if image dimensions are within acceptable limits
func ValidateImageDimensions(width, height, maxWidth, maxHeight int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid image dimensions: %dx%d", width, height)
	}

	if width > maxWidth || height > maxHeight {
		return fmt.Errorf("image dimensions %dx%d exceed maximum %dx%d", width, height, maxWidth, maxHeight)
	}

	return nil
}

// GetFileExtensionFromSignature returns the file extension based on file signature
func GetFileExtensionFromSignature(data []byte) (string, error) {
	if isJPEG(data) {
		return ".jpg", nil
	}

	if isPNG(data) {
		return ".png", nil
	}

	if isHEICSignature(data) {
		return ".heic", nil
	}

	return "", fmt.Errorf("unknown file signature")
}
