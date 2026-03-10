package ports

import (
	"context"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// CourseRepository defines persistence operations for Courses.
// Receives context for cancellation and timeout propagation.
type CourseRepository interface {
	Create(ctx context.Context, course *domain.Course) error
	FindByID(ctx context.Context, id string) (*domain.Course, error)
	// FindAllByUser returns all non-deleted courses for a given user.
	FindAllByUser(ctx context.Context, userID string) ([]domain.Course, error)
	// UpdateAvatarStatus updates avatar URL & status after async generation.
	UpdateAvatarStatus(ctx context.Context, courseID, status, avatarURL string) error
}

// TopicRepository defines persistence operations for Topics.
type TopicRepository interface {
	Create(ctx context.Context, topic *domain.Topic) error
	FindByID(ctx context.Context, topicID string) (*domain.Topic, error)
	FindByCourse(ctx context.Context, courseID string) ([]domain.Topic, error)
	GetSummaryCache(ctx context.Context, topicID string) (*domain.TopicSummaryCache, error)
	UpsertSummaryCache(ctx context.Context, cache domain.TopicSummaryCache) error
}

// StorageService defines the contract for file storage (Cloud Storage).
// Decoupled from the use case so it can be swapped (local, GCS, S3).
type StorageService interface {
	// UploadFile uploads a file and returns its public or signed URL.
	UploadFile(ctx context.Context, bucket, objectName string, data []byte, contentType string) (string, error)
}

// AvatarGenerator defines the contract for AI image generation.
// The concrete implementation (VertexImagenService) lives in the repositories layer.
type AvatarGenerator interface {
	// GenerateAvatar produces a transparent-background avatar PNG
	// based on an optional reference style hint.
	GenerateAvatar(ctx context.Context, referenceStyle string) (imageBytes []byte, mimeType string, err error)
}
