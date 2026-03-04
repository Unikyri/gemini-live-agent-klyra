package repositories

import (
	"context"
	"fmt"
	"log"
)

// LocalStorageService is a development stub for ports.StorageService.
// It logs the upload request but does NOT actually upload anything.
// Replace with CloudStorageService for production.
type LocalStorageService struct {
	BaseURL string // base URL for simulating returned file URLs locally
}

// NewLocalStorageService creates a stub storage service for local development.
func NewLocalStorageService() *LocalStorageService {
	return &LocalStorageService{BaseURL: "http://localhost:8080/static"}
}

// UploadFile simulates a file upload and returns a placeholder URL.
// TODO (US3 / DevOps): Replace with google.golang.org/api/storage to upload to GCS.
func (s *LocalStorageService) UploadFile(ctx context.Context, bucket, objectName string, data []byte, contentType string) (string, error) {
	log.Printf("[LocalStorage] Simulating upload: bucket=%s object=%s size=%d bytes type=%s",
		bucket, objectName, len(data), contentType)

	if len(data) == 0 {
		return "", fmt.Errorf("upload failed: empty file")
	}

	// Returns a predictable URL for development testing.
	url := fmt.Sprintf("%s/%s", s.BaseURL, objectName)
	log.Printf("[LocalStorage] Simulated URL: %s", url)
	return url, nil
}
