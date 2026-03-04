package repositories

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// GCSStorageService implements ports.StorageService using Google Cloud Storage.
// This replaces the LocalStorageService stub used in development.
//
// Authentication follows the same ADC pattern as VertexImagenService:
//   - Development: GOOGLE_APPLICATION_CREDENTIALS env var pointing to SA JSON.
//   - Production (Cloud Run): Attached Service Account — no extra config needed.
type GCSStorageService struct {
	bucketName string
}

// NewGCSStorageService creates a GCS-backed storage service.
// GCS_BUCKET_NAME must be set in the environment.
func NewGCSStorageService() *GCSStorageService {
	bucket := os.Getenv("GCS_BUCKET_NAME")
	if bucket == "" {
		bucket = os.Getenv("GCP_PROJECT_ID") + "-klyra-assets"
	}
	return &GCSStorageService{bucketName: bucket}
}

// UploadFile uploads raw bytes to GCS and returns the public URL of the object.
// SECURITY: The bucket should have uniform bucket-level access control.
// Avatar images are uploaded as publicly readable (for Flutter to display without auth).
func (s *GCSStorageService) UploadFile(ctx context.Context, bucket, objectName string, data []byte, contentType string) (string, error) {
	// Use the configured bucket if none is provided per-call.
	if bucket == "" {
		bucket = s.bucketName
	}

	// The GCS client automatically uses GOOGLE_APPLICATION_CREDENTIALS / ADC.
	client, err := storage.NewClient(ctx, option.WithScopes("https://www.googleapis.com/auth/cloud-platform"))
	if err != nil {
		return "", fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	obj := client.Bucket(bucket).Object(objectName)
	writer := obj.NewWriter(ctx)
	writer.ContentType = contentType
	// Make avatar images publicly readable so Flutter can display them without auth tokens.
	writer.PredefinedACL = "publicRead"

	_, err = io.Copy(writer, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to write to GCS object %s: %w", objectName, err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to finalize GCS upload: %w", err)
	}

	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, objectName)
	return publicURL, nil
}
