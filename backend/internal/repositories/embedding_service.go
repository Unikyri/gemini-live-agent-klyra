package repositories

import (
	"context"
	"fmt"
	"math"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// EmbeddingServiceImpl generates text embeddings using Vertex AI.
type EmbeddingServiceImpl struct {
	// Would contain Vertex AI client configuration
	dimension int
}

// NewEmbeddingService creates a new embedding service instance.
func NewEmbeddingService(dimension int) *EmbeddingServiceImpl {
	return &EmbeddingServiceImpl{
		dimension: dimension,
	}
}

// GenerateEmbedding returns a vector for a text query
func (s *EmbeddingServiceImpl) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text provided")
	}

	// TODO: Call Vertex AI embeddings API
	// For now, return a placeholder vector
	embedding := make([]float64, s.dimension)
	// Initialize with deterministic pattern for testing
	for i := range embedding {
		embedding[i] = 0.1 * float64(len(text)) / float64(i+1)
	}

	return embedding, nil
}

// BatchGenerateEmbeddings returns vectors for multiple texts (batch processing for efficiency)
func (s *EmbeddingServiceImpl) BatchGenerateEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("empty texts provided")
	}

	embeddings := make([][]float64, len(texts))
	for i, text := range texts {
		emb, err := s.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings[i] = emb
	}

	return embeddings, nil
}

// CosineSimilarity computes similarity between two vectors [0, 1]
func CosineSimilarity(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d vs %d", len(a), len(b))
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, nil
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)), nil
}

// PgVectorFromEmbedding converts a float64 slice to pgvector domain type
func PgVectorFromEmbedding(embedding []float64) domain.PgVector {
	vector := make([]float32, len(embedding))
	for i, v := range embedding {
		vector[i] = float32(v)
	}
	return domain.PgVector(vector)
}
