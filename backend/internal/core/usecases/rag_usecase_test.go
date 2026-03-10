package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

func TestGetTopicContext_QueryEmpty(t *testing.T) {
	ctx := context.Background()
	materialRepo := NewMockMaterialRepository()
	chunkRepo := NewMockChunkRepository()
	embedder := NewMockEmbedder()
	uc := NewRAGUseCase(materialRepo, chunkRepo, embedder)

	topicID := uuid.New()
	materialID := uuid.New()
	chunkRepo.chunksByTopic[topicID.String()] = []domain.MaterialChunk{
		{ID: uuid.New(), MaterialID: materialID, TopicID: topicID, ChunkIndex: 0, Content: "contexto A"},
		{ID: uuid.New(), MaterialID: materialID, TopicID: topicID, ChunkIndex: 1, Content: "contexto B"},
	}

	ctxText, err := uc.GetTopicContext(ctx, topicID.String(), "")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ctxText != "contexto A\n\ncontexto B" {
		t.Fatalf("unexpected context: %q", ctxText)
	}
}

func TestGetTopicContext_QueryProvided(t *testing.T) {
	ctx := context.Background()
	materialRepo := NewMockMaterialRepository()
	chunkRepo := NewMockChunkRepository()
	embedder := NewMockEmbedder()
	uc := NewRAGUseCase(materialRepo, chunkRepo, embedder)

	topicID := uuid.New()
	SeedTopicWithThreeMaterialsFiveChunksEach(topicID, chunkRepo)

	ctxText, err := uc.GetTopicContext(ctx, topicID.String(), "algebra")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ctxText == "" {
		t.Fatal("expected non-empty context")
	}
	if !strings.Contains(ctxText, "algebra") {
		t.Fatalf("expected similar chunks in context, got: %q", ctxText)
	}

	parts := strings.Split(ctxText, "\n\n")
	nonEmpty := 0
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			nonEmpty++
		}
	}
	if nonEmpty != 5 {
		t.Fatalf("expected top-k context with 5 chunks, got %d", nonEmpty)
	}
}

func TestGetTopicContext_NoChunksFound(t *testing.T) {
	ctx := context.Background()
	materialRepo := NewMockMaterialRepository()
	chunkRepo := NewMockChunkRepository()
	embedder := NewMockEmbedder()
	uc := NewRAGUseCase(materialRepo, chunkRepo, embedder)

	ctxText, err := uc.GetTopicContext(ctx, uuid.New().String(), "consulta")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ctxText != "" {
		t.Fatalf("expected empty context, got %q", ctxText)
	}
}

func TestGetTopicContext_EmbedderTimeout(t *testing.T) {
	ctx := context.Background()
	materialRepo := NewMockMaterialRepository()
	chunkRepo := NewMockChunkRepository()
	embedder := NewMockEmbedder()
	embedder.returnErr = context.DeadlineExceeded
	uc := NewRAGUseCase(materialRepo, chunkRepo, embedder)

	_, err := uc.GetTopicContext(ctx, uuid.New().String(), "consulta")
	if err == nil {
		t.Fatal("expected error for embedding timeout")
	}
	if !strings.Contains(err.Error(), "rag: embed query") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTopicContext_RepositoryErrorsPropagate(t *testing.T) {
	ctx := context.Background()
	materialRepo := NewMockMaterialRepository()
	chunkRepo := NewMockChunkRepository()
	chunkRepo.getByTopicErr = errors.New("db unavailable")
	embedder := NewMockEmbedder()
	uc := NewRAGUseCase(materialRepo, chunkRepo, embedder)

	_, err := uc.GetTopicContext(ctx, uuid.New().String(), "")
	if err == nil {
		t.Fatal("expected repository error")
	}
	if !strings.Contains(err.Error(), "rag: get topic context") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChunkText_DefaultSize(t *testing.T) {
	text := strings.Repeat("x", defaultChunkSize+300)
	chunks := chunkText(text, defaultChunkSize, defaultChunkOverlap)

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if len([]rune(chunks[0])) != defaultChunkSize {
		t.Fatalf("expected first chunk size %d, got %d", defaultChunkSize, len([]rune(chunks[0])))
	}
}

func TestChunkText_Overlap(t *testing.T) {
	text := strings.Repeat("0123456789", 120)
	chunks := chunkText(text, defaultChunkSize, defaultChunkOverlap)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	firstTail := []rune(chunks[0])[len([]rune(chunks[0]))-defaultChunkOverlap:]
	secondHead := []rune(chunks[1])[:defaultChunkOverlap]
	if string(firstTail) != string(secondHead) {
		t.Fatal("expected chunk overlap to match")
	}
}

func TestChunkText_SmallText(t *testing.T) {
	text := strings.Repeat("hola", 10)
	chunks := chunkText(text, defaultChunkSize, defaultChunkOverlap)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Fatalf("expected chunk to equal original text")
	}
}

type MockMaterialRepositoryForRAG struct{}

func NewMockMaterialRepository() *MockMaterialRepositoryForRAG {
	return &MockMaterialRepositoryForRAG{}
}

func (m *MockMaterialRepositoryForRAG) Create(ctx context.Context, material *domain.Material) error {
	_ = ctx
	_ = material
	return nil
}

func (m *MockMaterialRepositoryForRAG) FindByID(ctx context.Context, id string) (*domain.Material, error) {
	_ = ctx
	_ = id
	return nil, nil
}

func (m *MockMaterialRepositoryForRAG) FindByTopic(ctx context.Context, topicID string) ([]domain.Material, error) {
	_ = ctx
	_ = topicID
	return nil, nil
}

func (m *MockMaterialRepositoryForRAG) FindValidatedByTopic(ctx context.Context, topicID string) ([]domain.Material, error) {
	_ = ctx
	_ = topicID
	return nil, nil
}

func (m *MockMaterialRepositoryForRAG) CountByTopic(ctx context.Context, topicID string) (int, error) {
	_ = ctx
	_ = topicID
	return 0, nil
}

func (m *MockMaterialRepositoryForRAG) CountReadyByTopic(ctx context.Context, topicID string) (int, error) {
	_ = ctx
	_ = topicID
	return 0, nil
}

func (m *MockMaterialRepositoryForRAG) UpdateStatus(ctx context.Context, materialID string, status domain.MaterialStatus, extractedText string) error {
	_ = ctx
	_ = materialID
	_ = status
	_ = extractedText
	return nil
}

type MockChunkRepository struct {
	chunksByTopic map[string][]domain.MaterialChunk
	searchByTopic map[string][]domain.RAGResult
	getByTopicErr error
}

func NewMockChunkRepository() *MockChunkRepository {
	return &MockChunkRepository{
		chunksByTopic: map[string][]domain.MaterialChunk{},
		searchByTopic: map[string][]domain.RAGResult{},
	}
}

func (m *MockChunkRepository) SaveChunks(ctx context.Context, chunks []domain.MaterialChunk) error {
	_ = ctx
	if len(chunks) == 0 {
		return nil
	}
	topicID := chunks[0].TopicID.String()
	m.chunksByTopic[topicID] = chunks
	return nil
}

func (m *MockChunkRepository) SearchSimilar(ctx context.Context, topicID string, queryEmbedding []float32, topK int) ([]domain.RAGResult, error) {
	_ = ctx
	_ = queryEmbedding
	results := m.searchByTopic[topicID]
	if len(results) == 0 {
		chunks := m.chunksByTopic[topicID]
		for _, c := range chunks {
			results = append(results, domain.RAGResult{Chunk: c, Similarity: 0.9})
		}
	}
	if topK > 0 && len(results) > topK {
		return results[:topK], nil
	}
	return results, nil
}

func (m *MockChunkRepository) GetChunksByTopic(ctx context.Context, topicID string) ([]domain.MaterialChunk, error) {
	_ = ctx
	if m.getByTopicErr != nil {
		return nil, m.getByTopicErr
	}
	return m.chunksByTopic[topicID], nil
}

type MockEmbedder struct {
	returnErr error
}

func NewMockEmbedder() *MockEmbedder {
	return &MockEmbedder{}
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func SeedTopicWithThreeMaterialsFiveChunksEach(topicID uuid.UUID, chunkRepo *MockChunkRepository) {
	chunks := make([]domain.MaterialChunk, 0, 5)
	for i := 0; i < 5; i++ {
		chunks = append(chunks, domain.MaterialChunk{
			ID:         uuid.New(),
			MaterialID: uuid.New(),
			TopicID:    topicID,
			ChunkIndex: i,
			Content:    "algebra chunk " + string(rune('A'+i)),
		})
	}
	chunkRepo.chunksByTopic[topicID.String()] = chunks
}
