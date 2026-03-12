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
	// maxChunksPerTopic limits chunks per topic when building course context (no query).
	maxChunksPerTopic = 3
	// maxCourseChunks is the total chunk limit for course context (no query).
	maxCourseChunks = 30
)

// ContextResult separates the retrieved context from additional metadata used by clients.
type ContextResult struct {
	Context      string
	Truncated    bool
	HasMaterials bool
	Message      string
}

// RAGUseCase orchestrates the text chunking, embedding, and retrieval pipeline.
type RAGUseCase struct {
	materialRepo ports.MaterialRepository
	chunkRepo    ports.ChunkRepository
	topicRepo    ports.TopicRepository
	correctionRepo ports.CorrectionRepository
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

// NewRAGUseCaseWithTopicRepo constructs RAGUseCase with topic repo (for GetCourseContext).
func NewRAGUseCaseWithTopicRepo(
	materialRepo ports.MaterialRepository,
	chunkRepo ports.ChunkRepository,
	topicRepo ports.TopicRepository,
	embedder ports.Embedder,
) *RAGUseCase {
	return &RAGUseCase{
		materialRepo: materialRepo,
		chunkRepo:    chunkRepo,
		topicRepo:    topicRepo,
		embedder:     embedder,
	}
}

// NewRAGUseCaseWithCorrections wires a correction repository for override merge.
func NewRAGUseCaseWithCorrections(
	materialRepo ports.MaterialRepository,
	chunkRepo ports.ChunkRepository,
	topicRepo ports.TopicRepository,
	correctionRepo ports.CorrectionRepository,
	embedder ports.Embedder,
) *RAGUseCase {
	return &RAGUseCase{
		materialRepo:    materialRepo,
		chunkRepo:       chunkRepo,
		topicRepo:       topicRepo,
		correctionRepo:  correctionRepo,
		embedder:        embedder,
	}
}

func (uc *RAGUseCase) applyCorrectionsByChunkIDs(ctx context.Context, chunks []domain.MaterialChunk) ([]domain.MaterialChunk, error) {
	if uc.correctionRepo == nil || len(chunks) == 0 {
		return chunks, nil
	}
	ids := make([]string, 0, len(chunks))
	for _, c := range chunks {
		ids = append(ids, c.ID.String())
	}
	corrections, err := uc.correctionRepo.FindByChunkIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(corrections) == 0 {
		return chunks, nil
	}
	byChunk := map[string]domain.MaterialCorrection{}
	for _, corr := range corrections {
		if corr.ChunkID == nil {
			continue
		}
		byChunk[corr.ChunkID.String()] = corr
	}
	out := make([]domain.MaterialChunk, len(chunks))
	copy(out, chunks)
	for i := range out {
		if corr, ok := byChunk[out[i].ID.String()]; ok && corr.CorrectedText != "" {
			out[i].Content = corr.CorrectedText
		}
	}
	return out, nil
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
		domainChunks = append(domainChunks, domain.MaterialChunk{
			ID:         uuid.New(),
			MaterialID: mID,
			TopicID:    tID,
			ChunkIndex: i,
			Content:    text,
		})

		// Only add embeddings when the embedder is configured.
		if uc.embedder != nil {
			embedding, err := uc.embedder.Embed(ctx, text)
			if err != nil {
				// Log but don't fail all chunks because of a single API error.
				log.Printf("[RAG] Failed to embed chunk %d for material %s: %v", i, materialID, err)
			} else {
				domainChunks[len(domainChunks)-1].Embedding = domain.PgVector(embedding)
			}
		}
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
func (uc *RAGUseCase) GetTopicContext(ctx context.Context, topicID, query string) (*ContextResult, error) {
	if query == "" {
		// Return full context (used when starting a tutoring session)
		chunks, err := uc.chunkRepo.GetChunksByTopic(ctx, topicID)
		if err != nil {
			return nil, fmt.Errorf("rag: get topic context: %w", err)
		}
		chunks, err = uc.applyCorrectionsByChunkIDs(ctx, chunks)
		if err != nil {
			return nil, fmt.Errorf("rag: apply corrections: %w", err)
		}
		if len(chunks) == 0 {
			return &ContextResult{
				Context:      "",
				Truncated:    false,
				HasMaterials: false,
				Message:      "No hay materiales para este tema. El tutor usará su conocimiento base.",
			}, nil
		}
		parts := make([]string, len(chunks))
		for i, c := range chunks {
			parts[i] = c.Content
		}
		return &ContextResult{
			Context:      strings.Join(parts, "\n\n"),
			Truncated:    false,
			HasMaterials: true,
			Message:      "",
		}, nil
	}

	// In local/dev environments embeddings may be disabled. Fallback to full topic context.
	if uc.embedder == nil {
		chunks, err := uc.chunkRepo.GetChunksByTopic(ctx, topicID)
		if err != nil {
			return nil, fmt.Errorf("rag: get topic context: %w", err)
		}
		chunks, err = uc.applyCorrectionsByChunkIDs(ctx, chunks)
		if err != nil {
			return nil, fmt.Errorf("rag: apply corrections: %w", err)
		}
		if len(chunks) == 0 {
			return &ContextResult{
				Context:      "",
				Truncated:    false,
				HasMaterials: false,
				Message:      "No hay materiales para este tema. El tutor usará su conocimiento base.",
			}, nil
		}
		parts := make([]string, len(chunks))
		for i, c := range chunks {
			parts[i] = c.Content
		}
		return &ContextResult{
			Context:      strings.Join(parts, "\n\n"),
			Truncated:    false,
			HasMaterials: true,
			Message:      "",
		}, nil
	}

	// Embed the query and do similarity search
	queryEmbedding, err := uc.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("rag: embed query: %w", err)
	}

	results, err := uc.chunkRepo.SearchSimilar(ctx, topicID, queryEmbedding, 5)
	if err != nil {
		return nil, fmt.Errorf("rag: similarity search: %w", err)
	}

	if len(results) == 0 {
		return &ContextResult{
			Context:      "",
			Truncated:    false,
			HasMaterials: false,
			Message:      "No hay materiales para este tema. El tutor usará su conocimiento base.",
		}, nil
	}

	chunks := make([]domain.MaterialChunk, 0, len(results))
	for _, r := range results {
		chunks = append(chunks, r.Chunk)
	}
	chunks, err = uc.applyCorrectionsByChunkIDs(ctx, chunks)
	if err != nil {
		return nil, fmt.Errorf("rag: apply corrections: %w", err)
	}
	var sb strings.Builder
	for _, c := range chunks {
		sb.WriteString(c.Content)
		sb.WriteString("\n\n")
	}
	return &ContextResult{
		Context:      strings.TrimSuffix(sb.String(), "\n\n"),
		Truncated:    false,
		HasMaterials: true,
		Message:      "",
	}, nil
}

// GetCourseContext returns context for the whole course: either truncated top-N per topic (no query)
// or similarity search across course (with query). Result is grouped by topic with "### <title>" headers.
// Returns a ContextResult with context, truncated flag and metadata.
func (uc *RAGUseCase) GetCourseContext(ctx context.Context, courseID, query string) (*ContextResult, error) {
	if uc.topicRepo == nil {
		return nil, fmt.Errorf("rag: topic repo required for GetCourseContext")
	}

	topics, err := uc.topicRepo.FindByCourse(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("rag: find topics: %w", err)
	}
	topicTitles := make(map[string]string)
	for _, t := range topics {
		topicTitles[t.ID.String()] = t.Title
	}

	// If embeddings are disabled, fallback to the non-query path.
	if query != "" && uc.embedder != nil {
		queryEmbedding, err := uc.embedder.Embed(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("rag: embed query: %w", err)
		}
		results, err := uc.chunkRepo.SearchSimilarByCourse(ctx, courseID, queryEmbedding, 10)
		if err != nil {
			return nil, fmt.Errorf("rag: search similar by course: %w", err)
		}
		if len(results) == 0 {
			return &ContextResult{
				Context:      "",
				Truncated:    false,
				HasMaterials: false,
				Message:      "No hay materiales en ningún tema de este curso. El tutor usará su conocimiento base.",
			}, nil
		}
		// Apply corrections (override chunk content) if configured.
		candidates := make([]domain.MaterialChunk, 0, len(results))
		for _, r := range results {
			candidates = append(candidates, r.Chunk)
		}
		candidates, err = uc.applyCorrectionsByChunkIDs(ctx, candidates)
		if err != nil {
			return nil, fmt.Errorf("rag: apply corrections: %w", err)
		}
		for i := range results {
			results[i].Chunk = candidates[i]
		}
		return &ContextResult{
			Context:      buildCourseContextFromResults(results, topicTitles),
			Truncated:    true,
			HasMaterials: true,
			Message:      "",
		}, nil
	}

	chunks, err := uc.chunkRepo.GetChunksByCourse(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("rag: get chunks by course: %w", err)
	}
	chunks, err = uc.applyCorrectionsByChunkIDs(ctx, chunks)
	if err != nil {
		return nil, fmt.Errorf("rag: apply corrections: %w", err)
	}
	if len(chunks) == 0 {
		return &ContextResult{
			Context:      "",
			Truncated:    false,
			HasMaterials: false,
			Message:      "No hay materiales en ningún tema de este curso. El tutor usará su conocimiento base.",
		}, nil
	}

	// Group by topic_id, take first maxChunksPerTopic per topic, cap total at maxCourseChunks.
	byTopic := make(map[string][]domain.MaterialChunk)
	for _, c := range chunks {
		tid := c.TopicID.String()
		byTopic[tid] = append(byTopic[tid], c)
	}

	var selected []domain.MaterialChunk
	perTopic := maxChunksPerTopic
	remaining := maxCourseChunks
	for _, tid := range orderedTopicIDs(chunks) {
		list := byTopic[tid]
		n := perTopic
		if n > len(list) {
			n = len(list)
		}
		if n > remaining {
			n = remaining
		}
		if n > 0 {
			selected = append(selected, list[:n]...)
			remaining -= n
			if remaining <= 0 {
				break
			}
		}
	}
	truncated := len(selected) < len(chunks)
	return &ContextResult{
		Context:      buildCourseContextFromChunks(selected, topicTitles),
		Truncated:    truncated,
		HasMaterials: true,
		Message:      "",
	}, nil
}

func orderedTopicIDs(chunks []domain.MaterialChunk) []string {
	seen := make(map[string]struct{})
	var order []string
	for _, c := range chunks {
		tid := c.TopicID.String()
		if _, ok := seen[tid]; !ok {
			seen[tid] = struct{}{}
			order = append(order, tid)
		}
	}
	return order
}

func buildCourseContextFromChunks(chunks []domain.MaterialChunk, topicTitles map[string]string) string {
	var sb strings.Builder
	var lastTopicID string
	for _, c := range chunks {
		tid := c.TopicID.String()
		if tid != lastTopicID {
			if lastTopicID != "" {
				sb.WriteString("\n\n")
			}
			title := topicTitles[tid]
			if title == "" {
				title = tid
			}
			sb.WriteString("### ")
			sb.WriteString(title)
			sb.WriteString("\n\n")
			lastTopicID = tid
		}
		sb.WriteString(c.Content)
		sb.WriteString("\n\n")
	}
	return strings.TrimSuffix(sb.String(), "\n\n")
}

func buildCourseContextFromResults(results []domain.RAGResult, topicTitles map[string]string) string {
	var sb strings.Builder
	var lastTopicID string
	for _, r := range results {
		tid := r.Chunk.TopicID.String()
		if tid != lastTopicID {
			if lastTopicID != "" {
				sb.WriteString("\n\n")
			}
			title := topicTitles[tid]
			if title == "" {
				title = tid
			}
			sb.WriteString("### ")
			sb.WriteString(title)
			sb.WriteString("\n\n")
			lastTopicID = tid
		}
		sb.WriteString(r.Chunk.Content)
		sb.WriteString("\n\n")
	}
	return strings.TrimSuffix(sb.String(), "\n\n")
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
