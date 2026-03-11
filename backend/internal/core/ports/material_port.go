package ports

import (
	"context"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// MaterialRepository defines persistence operations for Materials.
type MaterialRepository interface {
	// Create persists a new material record and sets its generated UUID.
	Create(ctx context.Context, material *domain.Material) error
	// FindByID retrieves a material by its ID.
	// Returns (nil, nil) if not found — use case must treat this as 404.
	FindByID(ctx context.Context, id string) (*domain.Material, error)
	// FindByTopic retrieves all non-deleted materials for a specific topic.
	FindByTopic(ctx context.Context, topicID string) ([]domain.Material, error)
	// FindValidatedByTopic retrieves validated materials with extracted text.
	FindValidatedByTopic(ctx context.Context, topicID string) ([]domain.Material, error)
	// CountByTopic returns total materials linked to a topic.
	CountByTopic(ctx context.Context, topicID string) (int, error)
	// CountReadyByTopic returns validated materials with non-empty extracted text.
	CountReadyByTopic(ctx context.Context, topicID string) (int, error)
	// UpdateStatus updates the processing status and optionally the extracted text.
	UpdateStatus(ctx context.Context, materialID string, status domain.MaterialStatus, extractedText string) error
	// SoftDeleteByTopicIDs marks materials as deleted for the given topic IDs.
	SoftDeleteByTopicIDs(ctx context.Context, topicIDs []string) error
}

// TextExtractor defines the contract for extracting plain text from a raw file.
// The concrete implementation lives in the repositories layer.
type TextExtractor interface {
	// Extract returns the plain text content from a file's raw bytes.
	// formatType specifies the file format (txt, md, pdf).
	// Returns an empty string if extraction is not supported for the format.
	Extract(ctx context.Context, data []byte, formatType domain.MaterialFormatType) (string, error)
	// ExtractFromImage runs OCR over image bytes and returns text with confidence.
	ExtractFromImage(ctx context.Context, imageData []byte) (text string, confidence float64, err error)
	// ExtractFromAudio runs speech-to-text over audio bytes.
	ExtractFromAudio(ctx context.Context, audioData []byte, mimeType string) (transcript string, err error)
}
