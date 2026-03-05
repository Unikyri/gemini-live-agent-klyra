package repositories

import (
	"context"
	"fmt"
	"log"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

// VertexEmbeddingService generates text embeddings using Vertex AI text-embedding-004.
type VertexEmbeddingService struct {
	projectID string
	location  string
	modelID   string
	apiKey    string // Service Account credentials path (GOOGLE_APPLICATION_CREDENTIALS)
}

// NewVertexEmbeddingService creates a new embedding service.
// projectID: your GCP project ID.
// location: region, e.g., "us-central1".
// modelID: e.g., "text-embedding-004".
func NewVertexEmbeddingService(projectID, location, modelID, credentialsFile string) *VertexEmbeddingService {
	return &VertexEmbeddingService{
		projectID: projectID,
		location:  location,
		modelID:   modelID,
		apiKey:    credentialsFile,
	}
}

// Embed sends a text string to the Vertex AI Embeddings endpoint and returns a float32 vector.
// Implements ports.Embedder.
func (s *VertexEmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", s.location)

	client, err := aiplatform.NewPredictionClient(ctx,
		option.WithEndpoint(endpoint),
		option.WithCredentialsFile(s.apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("embedding: failed to create prediction client: %w", err)
	}
	defer func() {
		if cerr := client.Close(); cerr != nil {
			log.Printf("[Embedding] Warning: failed to close client: %v", cerr)
		}
	}()

	// Build the prediction request payload.
	// text-embedding-004 expects {"content": "<text>"} per instance.
	instance, err := structpb.NewValue(map[string]interface{}{
		"content": text,
	})
	if err != nil {
		return nil, fmt.Errorf("embedding: failed to build instance: %w", err)
	}

	modelResource := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s",
		s.projectID, s.location, s.modelID)

	resp, err := client.Predict(ctx, &aiplatformpb.PredictRequest{
		Endpoint:  modelResource,
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
