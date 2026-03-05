package usecases

import (
	"context"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
)

const (
	// chunkSize is the maximum number of runes per chunk.
	// ~500 tokens ≈ 800 runes for most languages.
	defaultChunkSize = 800
	// chunkOverlap is the number of runes shared between adjacent chunks
	// so that context is not lost at chunk boundaries.
	defaultChunkOverlap = 100
	// maxChunksPerMaterial limits memory usage during embedding.
	maxChunksPerMaterial = 50
	// minChunkLen skips chunks that are too short to be meaningful.
	minChunkLen = 20
)

// RAGUseCase orchestrates the text chunking, embedding, and retrieval pipeline.
type RAGUseCase struct {
	materialRepo ports.MaterialRepository
	chunkRepo    ports.ChunkRepository
	embedder     ports.Embedder
}

// NewRAGUseCase constructs a new RAGUseCase.
func NewRAGUseCase(
	materialRepo ports.MaterialRepository,
	chunkRepo ports.ChunkRepository,
	embedder ports.Embedder,
) *RAGUseCase {
	return &RAGUseCase{
		materialRepo: materialRepo,
		chunkRepo:    chunkRepo,
		embedder:     embedder,
	}
}

// ProcessMaterialChunks splits a validated material's extracted text into chunks,
// generates embeddings for each chunk, and persists them in the chunk store.
// This is called asynchronously after a material transitions to "validated" status.
func (uc *RAGUseCase) ProcessMaterialChunks(ctx context.Context, materialID string) error {
	material, err := uc.materialRepo.FindByID(ctx, materialID)
	if err != nil {
		return fmt.Errorf("rag: find material: %w", err)
	}
	if material == nil {
		return fmt.Errorf("rag: material not found: %s", materialID)
	}
	if material.ExtractedText == "" {
		log.Printf("[RAG] Material %s has no extracted text — skipping embedding.", materialID)
		return nil
	}

	rawChunks := chunkText(material.ExtractedText, defaultChunkSize, defaultChunkOverlap)
	if len(rawChunks) > maxChunksPerMaterial {
		rawChunks = rawChunks[:maxChunksPerMaterial]
	}

	log.Printf("[RAG] Processing %d chunks for material %s", len(rawChunks), materialID)

	mID := material.ID
	tID := material.TopicID
	domainChunks := make([]domain.MaterialChunk, 0, len(rawChunks))

	for i, text := range rawChunks {
		if utf8.RuneCountInString(text) < minChunkLen {
			continue
		}
		embedding, err := uc.embedder.Embed(ctx, text)
		if err != nil {
			// Log but don't fail all chunks because of a single API error.
			log.Printf("[RAG] Failed to embed chunk %d for material %s: %v", i, materialID, err)
			continue
		}
		domainChunks = append(domainChunks, domain.MaterialChunk{
			ID:         uuid.New(),
			MaterialID: mID,
			TopicID:    tID,
			ChunkIndex: i,
			Content:    text,
			Embedding:  domain.PgVector(embedding),
		})
	}

	if err := uc.chunkRepo.SaveChunks(ctx, domainChunks); err != nil {
		return fmt.Errorf("rag: save chunks: %w", err)
	}

	log.Printf("[RAG] Saved %d embedded chunks for material %s", len(domainChunks), materialID)
	return nil
}

// GetTopicContext retrieves the most relevant context for a topic given a user query.
// If query is empty, returns the full concatenated text of all chunks (for session init).
// SECURITY: topicID filters all retrieval — cross-user leakage is impossible.
func (uc *RAGUseCase) GetTopicContext(ctx context.Context, topicID, query string) (string, error) {
	if query == "" {
		// Return full context (used when starting a tutoring session)
		chunks, err := uc.chunkRepo.GetChunksByTopic(ctx, topicID)
		if err != nil {
			return "", fmt.Errorf("rag: get topic context: %w", err)
		}
		parts := make([]string, len(chunks))
		for i, c := range chunks {
			parts[i] = c.Content
		}
		return strings.Join(parts, "\n\n"), nil
	}

	// Embed the query and do similarity search
	queryEmbedding, err := uc.embedder.Embed(ctx, query)
	if err != nil {
		return "", fmt.Errorf("rag: embed query: %w", err)
	}

	results, err := uc.chunkRepo.SearchSimilar(ctx, topicID, queryEmbedding, 5)
	if err != nil {
		return "", fmt.Errorf("rag: similarity search: %w", err)
	}

	var sb strings.Builder
	for _, r := range results {
		sb.WriteString(r.Chunk.Content)
		sb.WriteString("\n\n")
	}
	return sb.String(), nil
}

// -- chunkText splits text into overlapping rune-based windows --

func chunkText(text string, chunkSize, overlap int) []string {
	runes := []rune(text)
	total := len(runes)
	if total == 0 {
		return nil
	}

	var chunks []string
	start := 0
	for start < total {
		end := start + chunkSize
		if end > total {
			end = total
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == total {
			break
		}
		start = end - overlap
	}
	return chunks
}
