package repositories

import (
	"context"
	"fmt"
	"log"
	"sync"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

// VertexEmbeddingService generates text embeddings using Vertex AI text-embedding-004.
// The PredictionClient is created once and reused across all Embed() calls for efficiency.
type VertexEmbeddingService struct {
	projectID     string
	location      string
	modelResource string // fully-qualified model path (cached)
	client        *aiplatform.PredictionClient
	once          sync.Once // ensures the client is closed only once
}

// NewVertexEmbeddingService creates a new embedding service and eagerly initialises
// the Vertex AI PredictionClient so it can be reused across all Embed() calls.
// modelID: e.g., "text-embedding-004".
// credentialsFile: path to the service account JSON key (GOOGLE_APPLICATION_CREDENTIALS).
func NewVertexEmbeddingService(projectID, location, modelID, credentialsFile string) (*VertexEmbeddingService, error) {
	endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)

	client, err := aiplatform.NewPredictionClient(context.Background(),
		option.WithEndpoint(endpoint),
		option.WithCredentialsFile(credentialsFile),
	)
	if err != nil {
		return nil, fmt.Errorf("embedding: failed to create prediction client: %w", err)
	}

	modelResource := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s",
		projectID, location, modelID)

	return &VertexEmbeddingService{
		projectID:     projectID,
		location:      location,
		modelResource: modelResource,
		client:        client,
	}, nil
}

// Embed sends a text string to the Vertex AI Embeddings endpoint and returns a float32 vector.
// Implements ports.Embedder.
// SECURITY: text is passed directly as a struct field — no shell injection risk.
func (s *VertexEmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	// Build the prediction request payload.
	// text-embedding-004 expects {"content": "<text>"} per instance.
	instance, err := structpb.NewValue(map[string]interface{}{
		"content": text,
	})
	if err != nil {
		return nil, fmt.Errorf("embedding: failed to build instance: %w", err)
	}

	resp, err := s.client.Predict(ctx, &aiplatformpb.PredictRequest{
		Endpoint:  s.modelResource,
		Instances: []*structpb.Value{instance},
	})
	if err != nil {
		return nil, fmt.Errorf("embedding: predict failed: %w", err)
	}

	if len(resp.Predictions) == 0 {
		return nil, fmt.Errorf("embedding: no predictions returned")
	}

	// Extract the embedding values from the response.
	embeddingMap := resp.Predictions[0].GetStructValue().GetFields()
	valuesField, ok := embeddingMap["embeddings"]
	if !ok {
		return nil, fmt.Errorf("embedding: 'embeddings' field missing in response")
	}
	valsList := valuesField.GetStructValue().GetFields()["values"].GetListValue().GetValues()

	embedding := make([]float32, len(valsList))
	for i, v := range valsList {
		embedding[i] = float32(v.GetNumberValue())
	}

	return embedding, nil
}

// Close shuts down the Vertex AI client. Call this when the server shuts down.
func (s *VertexEmbeddingService) Close() {
	s.once.Do(func() {
		if err := s.client.Close(); err != nil {
			log.Printf("[Embedding] Warning: failed to close client: %v", err)
		}
	})
}
