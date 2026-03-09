package repositories

import (
	"context"
	"mime/multipart"

	"github.com/google/uuid"
)

// StorageService abstracts file storage operations (GCS, S3, local).
type StorageService interface {
	// UploadFile persists a file and returns its storage URL.
	UploadFile(ctx context.Context, materialID uuid.UUID, file interface{}, filename string) (string, error)
	// DeleteFile removes a file from storage.
	DeleteFile(ctx context.Context, storagePath string) error
	// DownloadFile retrieves file contents.
	DownloadFile(ctx context.Context, storagePath string) ([]byte, error)
}

// TextExtractor abstracts text extraction from various file formats (PDF, DOCX, TXT).
type TextExtractor interface {
	// ExtractText returns text content from a file.
	ExtractText(ctx context.Context, file interface{}, contentType string) (string, error)
}

// ImageService abstracts image processing and storage (thumbnails, optimization).
type ImageService interface {
	// UploadImage persists an image and returns its storage URL.
	UploadImage(ctx context.Context, courseID uuid.UUID, imageFile *multipart.FileHeader) (string, error)
	// DeleteImage removes an image from storage.
	DeleteImage(ctx context.Context, imagePath string) error
}

// EmbeddingService abstracts vector embedding generation (Vertex AI, OpenAI, local).
type EmbeddingService interface {
	// GenerateEmbedding returns a vector for a text query.
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
	// BatchGenerateEmbeddings returns vectors for multiple texts.
	BatchGenerateEmbeddings(ctx context.Context, texts []string) ([][]float64, error)
}

// JWTService abstracts token generation and validation (access + refresh tokens).
type JWTService interface {
	// GenerateTokens creates access and refresh tokens for a user.
	GenerateTokens(userID string) (accessToken, refreshToken string, err error)
	// ValidateAccessToken verifies an access token and returns the user ID.
	ValidateAccessToken(token string) (userID string, err error)
	// ValidateRefreshToken verifies a refresh token and returns the user ID.
	ValidateRefreshToken(token string) (userID string, err error)
	// RefreshAccessToken generates a new access token from a refresh token.
	RefreshAccessToken(refreshToken string) (newAccessToken string, err error)
}

// GoogleIDVerifier abstracts Google OAuth token verification (legacy local contract).
// Kept with a distinct name to avoid clashing with the concrete GoogleVerifier type.
type GoogleIDVerifier interface {
	// VerifyIDToken validates a Google OAuth ID token and extracts user info.
	VerifyIDToken(ctx context.Context, idToken string) (googleID, email, name string, err error)
}
