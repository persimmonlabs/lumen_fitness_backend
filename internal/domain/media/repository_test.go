package media

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	storage "github.com/supabase-community/storage-go"
)

// MockStorageClient mocks the Supabase Storage client for testing
type MockStorageClient struct {
	uploadFunc func(bucket, path string, reader io.Reader) error
	removeFunc func(bucket string, paths []string) error
}

func (m *MockStorageClient) UploadFile(bucket, path string, reader io.Reader) (interface{}, error) {
	if m.uploadFunc != nil {
		return nil, m.uploadFunc(bucket, path, reader)
	}
	return nil, nil
}

func (m *MockStorageClient) RemoveFile(bucket string, paths []string) (interface{}, error) {
	if m.removeFunc != nil {
		return nil, m.removeFunc(bucket, paths)
	}
	return nil, nil
}

// CreateBucket implements storage client interface
func (m *MockStorageClient) CreateBucket(bucketName string, options storage.BucketOptions) (interface{}, error) {
	return nil, nil
}

// GetBucket implements storage client interface
func (m *MockStorageClient) GetBucket(bucketName string) (interface{}, error) {
	return nil, nil
}

// ListBuckets implements storage client interface
func (m *MockStorageClient) ListBuckets() (interface{}, error) {
	return nil, nil
}

// UpdateBucket implements storage client interface
func (m *MockStorageClient) UpdateBucket(bucketName string, options storage.BucketOptions) (interface{}, error) {
	return nil, nil
}

// EmptyBucket implements storage client interface
func (m *MockStorageClient) EmptyBucket(bucketName string) (interface{}, error) {
	return nil, nil
}

// DeleteBucket implements storage client interface
func (m *MockStorageClient) DeleteBucket(bucketName string) (interface{}, error) {
	return nil, nil
}

// MoveFile implements storage client interface
func (m *MockStorageClient) MoveFile(bucketName string, fromPath string, toPath string) (interface{}, error) {
	return nil, nil
}

// CopyFile implements storage client interface
func (m *MockStorageClient) CopyFile(bucketName string, fromPath string, toPath string) (interface{}, error) {
	return nil, nil
}

// ListFiles implements storage client interface
func (m *MockStorageClient) ListFiles(bucketName string, path string, options storage.FileSearchOptions) (interface{}, error) {
	return nil, nil
}

// GetPublicUrl implements storage client interface
func (m *MockStorageClient) GetPublicUrl(bucketName string, path string) (interface{}, error) {
	return nil, nil
}

// GetFile implements storage client interface
func (m *MockStorageClient) GetFile(bucketName string, path string) (interface{}, error) {
	return nil, nil
}

// DownloadFile implements storage client interface
func (m *MockStorageClient) DownloadFile(bucketName string, path string) (interface{}, error) {
	return nil, nil
}

func TestStorageRepository_Upload(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		content       string
		contentType   string
		uploadFunc    func(bucket, path string, reader io.Reader) error
		expectError   bool
		errorContains string
	}{
		{
			name:        "successful upload",
			filename:    "test.jpg",
			content:     "fake image content",
			contentType: "image/jpeg",
			uploadFunc:  func(bucket, path string, reader io.Reader) error { return nil },
			expectError: false,
		},
		{
			name:        "upload failure",
			filename:    "test.jpg",
			content:     "fake image content",
			contentType: "image/jpeg",
			uploadFunc: func(bucket, path string, reader io.Reader) error {
				return errors.New("storage error")
			},
			expectError:   true,
			errorContains: "failed to upload file to storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockStorageClient{
				uploadFunc: tt.uploadFunc,
			}

			repo := NewStorageRepository(mockClient, "test-bucket", "https://example.supabase.co")

			fileReader := bytes.NewReader([]byte(tt.content))
			url, err := repo.Upload(context.Background(), fileReader, tt.filename, tt.contentType)

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
				if url == "" {
					t.Error("expected non-empty URL")
				}
			}
		})
	}
}

func TestStorageRepository_GetPublicURL(t *testing.T) {
	repo := NewStorageRepository(&MockStorageClient{}, "test-bucket", "https://example.supabase.co")

	url := repo.GetPublicURL("test-file.jpg")

	expectedURL := "https://example.supabase.co/storage/v1/object/public/test-bucket/test-file.jpg"
	if url != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, url)
	}
}

func TestStorageRepository_Delete(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		removeFunc    func(bucket string, paths []string) error
		expectError   bool
		errorContains string
	}{
		{
			name:     "successful delete",
			filename: "test.jpg",
			removeFunc: func(bucket string, paths []string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:     "delete failure",
			filename: "test.jpg",
			removeFunc: func(bucket string, paths []string) error {
				return errors.New("storage error")
			},
			expectError:   true,
			errorContains: "failed to delete file from storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockStorageClient{
				removeFunc: tt.removeFunc,
			}

			repo := NewStorageRepository(mockClient, "test-bucket", "https://example.supabase.co")

			err := repo.Delete(context.Background(), tt.filename)

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
			}
		})
	}
}

func TestNewStorageRepository(t *testing.T) {
	client := &MockStorageClient{}
	bucketName := "test-bucket"
	publicURL := "https://example.supabase.co"

	repo := NewStorageRepository(client, bucketName, publicURL)

	if repo == nil {
		t.Fatal("expected non-nil repository")
	}

	if repo.bucketName != bucketName {
		t.Errorf("expected bucket name %q, got %q", bucketName, repo.bucketName)
	}

	if repo.publicURL != publicURL {
		t.Errorf("expected public URL %q, got %q", publicURL, repo.publicURL)
	}
}
