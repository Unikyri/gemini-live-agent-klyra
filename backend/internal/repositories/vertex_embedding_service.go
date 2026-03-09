package repositories

import "context"

// VertexEmbeddingService is a lightweight adapter that satisfies ports.Embedder.
// For now it delegates to the deterministic local embedding service.
type VertexEmbeddingService struct {
	fallback *EmbeddingServiceImpl
}

// NewVertexEmbeddingService creates an embedder compatible with the current composition root.
// Parameters are kept for compatibility with planned Vertex AI integration.
func NewVertexEmbeddingService(projectID, location, modelID, credentialsPath string) (*VertexEmbeddingService, error) {
	_ = projectID
	_ = location
	_ = modelID
	_ = credentialsPath
	return &VertexEmbeddingService{fallback: NewEmbeddingService(768)}, nil
}

// Embed generates an embedding vector for a text query.
func (s *VertexEmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	emb64, err := s.fallback.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}
	emb32 := make([]float32, len(emb64))
	for i, v := range emb64 {
		emb32[i] = float32(v)
	}
	return emb32, nil
}

// Close is a no-op for the local fallback implementation.
func (s *VertexEmbeddingService) Close() error {
	return nil
}
