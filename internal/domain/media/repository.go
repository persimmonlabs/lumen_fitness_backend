package media

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// StorageClient defines the interface for storage operations
type StorageClient interface {
	UploadFile(bucket, path string, reader io.Reader) (interface{}, error)
	RemoveFile(bucket string, paths []string) (interface{}, error)
}

// Repository handles media storage operations
type Repository interface {
	Upload(ctx context.Context, file io.Reader, filename string, contentType string) (string, error)
	GetPublicURL(filename string) string
	Delete(ctx context.Context, filename string) error
}

// StorageRepository implements Repository using Supabase Storage
type StorageRepository struct {
	client     StorageClient
	bucketName string
	publicURL  string // Base public URL for the bucket
}

// NewStorageRepository creates a new Supabase storage repository
func NewStorageRepository(client StorageClient, bucketName string, publicURL string) *StorageRepository {
	return &StorageRepository{
		client:     client,
		bucketName: bucketName,
		publicURL:  publicURL,
	}
}

// Upload uploads a file to Supabase Storage and returns the public URL
func (r *StorageRepository) Upload(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file content: %w", err)
	}

	// Generate unique filename with timestamp and UUID
	timestamp := time.Now().Unix()
	uniqueID := uuid.New().String()
	uniqueFilename := fmt.Sprintf("%d_%s_%s", timestamp, uniqueID, filename)

	// Upload to Supabase Storage
	_, err = r.client.UploadFile(
		r.bucketName,
		uniqueFilename,
		bytes.NewReader(content),
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to storage: %w", err)
	}

	// Get public URL
	url := r.GetPublicURL(uniqueFilename)
	return url, nil
}

// GetPublicURL generates a public URL for a file in storage
func (r *StorageRepository) GetPublicURL(filename string) string {
	// Format: {baseURL}/storage/v1/object/public/{bucket}/{filename}
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", r.publicURL, r.bucketName, filename)
}

// Delete removes a file from Supabase Storage
func (r *StorageRepository) Delete(ctx context.Context, filename string) error {
	_, err := r.client.RemoveFile(r.bucketName, []string{filename})
	if err != nil {
		return fmt.Errorf("failed to delete file from storage: %w", err)
	}
	return nil
}
