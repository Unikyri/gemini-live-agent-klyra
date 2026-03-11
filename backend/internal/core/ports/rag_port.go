package ports

import (
	"context"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// Embedder generates vector representations for text using an AI model.
// The concrete implementation calls Vertex AI text-embedding-004.
type Embedder interface {
	// Embed generates a float32 embedding vector for the given text.
	// Returns an error if the API call fails.
	Embed(ctx context.Context, text string) ([]float32, error)
}

// ChunkRepository manages persistence of MaterialChunks (the pgvector store).
type ChunkRepository interface {
	// SaveChunks persists a batch of chunks (with embeddings) for a material.
	// Replaces any previously saved chunks for the same material_id.
	SaveChunks(ctx context.Context, chunks []domain.MaterialChunk) error

	// SearchSimilar performs a cosine similarity search within a specific topic.
	// Returns the top-k most similar chunks to the given query embedding.
	// SECURITY: topicID scoping ensures cross-user data leakage is impossible.
	SearchSimilar(ctx context.Context, topicID string, queryEmbedding []float32, topK int) ([]domain.RAGResult, error)

	// GetChunksByTopic retrieves all chunks for a topic (for full context build).
	GetChunksByTopic(ctx context.Context, topicID string) ([]domain.MaterialChunk, error)

	// GetChunksByCourse retrieves all chunks for a course (JOIN topics), only non-deleted topics.
	GetChunksByCourse(ctx context.Context, courseID string) ([]domain.MaterialChunk, error)
	// SearchSimilarByCourse performs similarity search across all chunks of a course.
	SearchSimilarByCourse(ctx context.Context, courseID string, queryEmbedding []float32, topK int) ([]domain.RAGResult, error)
	// HardDeleteByTopicIDs deletes all chunks for the given topic IDs (cascade; chunks are derived data).
	HardDeleteByTopicIDs(ctx context.Context, topicIDs []string) error
}
