package repositories

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorageService is a development stub for ports.StorageService.
// It logs the upload request but does NOT actually upload anything.
// Replace with CloudStorageService for production.
type LocalStorageService struct {
	BaseURL  string // base URL for returned file URLs locally
	BasePath string // root path for local files
}

// NewLocalStorageService creates a stub storage service for local development.
func NewLocalStorageService(basePath string) *LocalStorageService {
	if strings.TrimSpace(basePath) == "" {
		basePath = "./storage"
	}
	return &LocalStorageService{
		BaseURL:  "http://localhost:8080/static",
		BasePath: basePath,
	}
}

// UploadFile writes the file to local disk and returns a predictable static URL.
func (s *LocalStorageService) UploadFile(ctx context.Context, bucket, objectName string, data []byte, contentType string) (string, error) {
	_ = ctx
	_ = contentType

	if len(data) == 0 {
		return "", fmt.Errorf("upload failed: empty file")
	}
	if strings.TrimSpace(objectName) == "" {
		return "", fmt.Errorf("upload failed: objectName is required")
	}

	cleanObject := filepath.Clean(objectName)
	cleanObject = strings.TrimPrefix(cleanObject, string(filepath.Separator))
	fullPath := filepath.Join(s.BasePath, cleanObject)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", fmt.Errorf("upload failed: cannot create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("upload failed: cannot write file: %w", err)
	}

	publicObject := strings.ReplaceAll(cleanObject, "\\", "/")
	url := fmt.Sprintf("%s/%s", strings.TrimRight(s.BaseURL, "/"), publicObject)
	return url, nil
}
