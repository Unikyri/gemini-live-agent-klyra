package repositories

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

// VertexImagenService generates avatar images using Google's Imagen model on Vertex AI.
// It implements ports.AvatarGenerator.
//
// Authentication strategy:
//   - Development: Set GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json in .env
//     The Google SDK automatically picks up this env var — no code change needed.
//   - Production (Cloud Run): Attach the Service Account to the Cloud Run Service directly.
//     No JSON file is needed in production. This is the most secure approach.
type VertexImagenService struct {
	projectID string
	region    string
	modelID   string // e.g. "imagen-3.0-generate-001"
}

// NewVertexImagenService creates an Imagen service using the GCP env variables.
func NewVertexImagenService() *VertexImagenService {
	return &VertexImagenService{
		projectID: os.Getenv("GCP_PROJECT_ID"),
		region:    getEnvOrDefault("GCP_REGION", "us-central1"),
		modelID:   getEnvOrDefault("IMAGEN_MODEL_ID", "imagen-3.0-generate-001"),
	}
}

// GenerateAvatar calls Vertex AI Imagen to produce a transparent-background 2D avatar PNG.
// It satisfies the ports.AvatarGenerator interface (returns []byte, string, error).
func (s *VertexImagenService) GenerateAvatar(ctx context.Context, referenceStyle string) ([]byte, string, error) {
	prompt := buildAvatarPrompt(referenceStyle)
	log.Printf("[Imagen] Generating avatar — model: %s, project: %s", s.modelID, s.projectID)

	// Regional API endpoint for Vertex AI.
	endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", s.region)

	// The Google SDK automatically uses GOOGLE_APPLICATION_CREDENTIALS when set.
	// In production Cloud Run, the SDK uses the attached Service Account instead.
	client, err := aiplatform.NewPredictionClient(ctx, option.WithEndpoint(endpoint))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create Vertex AI prediction client: %w", err)
	}
	defer client.Close()

	instance, err := structpb.NewValue(map[string]interface{}{
		"prompt": prompt,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to build Imagen request instance: %w", err)
	}

	params, err := structpb.NewValue(map[string]interface{}{
		"sampleCount":    1,
		"aspectRatio":    "1:1",
		"outputMimeType": "image/png",
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to build Imagen request params: %w", err)
	}

	modelEndpoint := fmt.Sprintf(
		"projects/%s/locations/%s/publishers/google/models/%s",
		s.projectID, s.region, s.modelID,
	)

	resp, err := client.Predict(ctx, &aiplatformpb.PredictRequest{
		Endpoint:   modelEndpoint,
		Instances:  []*structpb.Value{instance},
		Parameters: params,
	})
	if err != nil {
		return nil, "", fmt.Errorf("Imagen Predict API call failed: %w", err)
	}

	if len(resp.Predictions) == 0 {
		return nil, "", fmt.Errorf("Imagen returned empty predictions list")
	}

	// Vertex AI Imagen returns the image as base64 under:
	//   "bytesBase64Encoded" (Imagen 3+) or "imageBytes" (older model versions).
	predMap := resp.Predictions[0].GetStructValue().AsMap()
	b64Str, _ := predMap["bytesBase64Encoded"].(string)
	if b64Str == "" {
		b64Str, _ = predMap["imageBytes"].(string)
	}
	if b64Str == "" {
		raw, _ := json.Marshal(predMap)
		return nil, "", fmt.Errorf("Imagen response is missing image bytes. Response: %s", string(raw))
	}

	imageBytes, err := base64.StdEncoding.DecodeString(b64Str)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode base64 image from Imagen: %w", err)
	}

	log.Printf("[Imagen] Avatar generated successfully (%d bytes)", len(imageBytes))
	return imageBytes, "image/png", nil
}

// buildAvatarPrompt constructs a detailed Imagen prompt for generating tutor avatars.
// The prompt is engineered for PNG output suitable for lip-sync animation overlay:
// fully transparent background, forward-facing, neutral expression.
func buildAvatarPrompt(referenceStyle string) string {
	base := "A professional 2D animated tutor character, fully transparent background (PNG alpha channel), " +
		"facing the camera, neutral expression, clean educational app style, " +
		"high resolution, portrait orientation, isolated figure with no background, " +
		"suitable for overlay on dynamic backgrounds"

	if referenceStyle != "" {
		return fmt.Sprintf("%s, visual style inspired by: %s", base, referenceStyle)
	}
	return base
}

// getEnvOrDefault reads an env variable and falls back to a default.
func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
