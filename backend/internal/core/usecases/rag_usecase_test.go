package usecases

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

type MockMaterialRepository struct {
	materials map[string]*domain.Material
	findByID  func(ctx context.Context, id string) (*domain.Material, error)
}

func NewMockMaterialRepository() *MockMaterialRepository {
	return &MockMaterialRepository{materials: make(map[string]*domain.Material)}
}

func (m *MockMaterialRepository) Create(ctx context.Context, material *domain.Material) error {
	if material.ID == uuid.Nil {
		material.ID = uuid.New()
	}
	m.materials[material.ID.String()] = material
	return nil
}

func (m *MockMaterialRepository) FindByID(ctx context.Context, id string) (*domain.Material, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	if mat, ok := m.materials[id]; ok {
		return mat, nil
	}
	return nil, nil
}

func (m *MockMaterialRepository) FindByTopic(ctx context.Context, topicID string) ([]domain.Material, error) {
	var out []domain.Material
	for _, mat := range m.materials {
		if mat.TopicID.String() == topicID {
			out = append(out, *mat)
		}
	}
	return out, nil
}

func (m *MockMaterialRepository) UpdateStatus(ctx context.Context, materialID string, status domain.MaterialStatus, extractedText string) error {
	mat, ok := m.materials[materialID]
	if !ok {
		return fmt.Errorf("material not found")
	}
	mat.Status = status
	if extractedText != "" {
		mat.ExtractedText = extractedText
	}
	return nil
}

type MockChunkRepository struct {
	chunksByMaterial map[string][]domain.MaterialChunk
	chunksByTopic    map[string][]domain.MaterialChunk
	saveCalls        int

	saveErr       error
	searchErr     error
	getByTopicErr error
}

func NewMockChunkRepository() *MockChunkRepository {
	return &MockChunkRepository{
		chunksByMaterial: make(map[string][]domain.MaterialChunk),
		chunksByTopic:    make(map[string][]domain.MaterialChunk),
	}
}

func (m *MockChunkRepository) SaveChunks(ctx context.Context, chunks []domain.MaterialChunk) error {
	m.saveCalls++
	if m.saveErr != nil {
		return m.saveErr
	}
	if len(chunks) == 0 {
		return nil
	}

	materialID := chunks[0].MaterialID.String()
	topicID := chunks[0].TopicID.String()

	m.chunksByMaterial[materialID] = append([]domain.MaterialChunk(nil), chunks...)

	filtered := make([]domain.MaterialChunk, 0, len(m.chunksByTopic[topicID]))
	for _, c := range m.chunksByTopic[topicID] {
		if c.MaterialID.String() != materialID {
			filtered = append(filtered, c)
		}
	}
	m.chunksByTopic[topicID] = append(filtered, chunks...)
	sort.Slice(m.chunksByTopic[topicID], func(i, j int) bool {
		if m.chunksByTopic[topicID][i].ChunkIndex == m.chunksByTopic[topicID][j].ChunkIndex {
			return m.chunksByTopic[topicID][i].MaterialID.String() < m.chunksByTopic[topicID][j].MaterialID.String()
		}
		return m.chunksByTopic[topicID][i].ChunkIndex < m.chunksByTopic[topicID][j].ChunkIndex
	})

	return nil
}

func (m *MockChunkRepository) SearchSimilar(ctx context.Context, topicID string, queryEmbedding []float32, topK int) ([]domain.RAGResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("empty query embedding")
	}

	chunks := m.chunksByTopic[topicID]
	results := make([]domain.RAGResult, 0, len(chunks))
	for _, c := range chunks {
		sim := cosineSimilarity([]float32(c.Embedding), queryEmbedding)
		results = append(results, domain.RAGResult{Chunk: c, Similarity: sim})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})
	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func (m *MockChunkRepository) GetChunksByTopic(ctx context.Context, topicID string) ([]domain.MaterialChunk, error) {
	if m.getByTopicErr != nil {
		return nil, m.getByTopicErr
	}
	return append([]domain.MaterialChunk(nil), m.chunksByTopic[topicID]...), nil
}

type MockEmbedder struct {
	callCount      int
	failEvery      int
	failText       map[string]error
	returnErr      error
	vectorSize     int
	lastEmbedded   []string
	forceQueryText string
}

func NewMockEmbedder() *MockEmbedder {
	return &MockEmbedder{
		failText:   make(map[string]error),
		vectorSize: 16,
	}
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	m.callCount++
	m.lastEmbedded = append(m.lastEmbedded, text)

	if m.returnErr != nil {
		return nil, m.returnErr
	}
	if err, ok := m.failText[text]; ok {
		return nil, err
	}
	if m.failEvery > 0 && m.callCount%m.failEvery == 0 {
		return nil, fmt.Errorf("embedder transient error")
	}

	return deterministicVector(text, m.vectorSize), nil
}

func CreateTestMaterial(topicID uuid.UUID, text string) *domain.Material {
	return &domain.Material{
		ID:            uuid.New(),
		TopicID:       topicID,
		FormatType:    domain.MaterialFormatTXT,
		StorageURL:    "gs://test/material.txt",
		ExtractedText: text,
		Status:        domain.MaterialStatusValidated,
		OriginalName:  "material.txt",
		SizeBytes:     int64(len(text)),
	}
}

func AssertChunksCreated(t *testing.T, repo *MockChunkRepository, materialID uuid.UUID, expected int) {
	t.Helper()
	got := len(repo.chunksByMaterial[materialID.String()])
	if got != expected {
		t.Fatalf("expected %d chunks, got %d", expected, got)
	}
}

func deterministicVector(text string, size int) []float32 {
	if size <= 0 {
		size = 16
	}
	v := make([]float32, size)
	h := fnv.New64a()
	_, _ = h.Write([]byte(text))
	seed := h.Sum64()
	for i := 0; i < size; i++ {
		part := float32((seed>>uint((i%8)*8))&0xFF) / 255.0
		v[i] = part
	}
	return v
}

func cosineSimilarity(a, b []float32) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		aa := float64(a[i])
		bb := float64(b[i])
		dot += aa * bb
		na += aa * aa
		nb += bb * bb
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func buildTextForChunks(numChunks int) string {
	if numChunks <= 0 {
		return ""
	}
	segment := strings.Repeat("A", defaultChunkSize-defaultChunkOverlap)
	return strings.Repeat(segment, numChunks+1)
}

func SeedTopicWithThreeMaterialsFiveChunksEach(topicID uuid.UUID, repo *MockChunkRepository) {
	for m := 0; m < 3; m++ {
		materialID := uuid.New()
		for i := 0; i < 5; i++ {
			content := fmt.Sprintf("material-%d chunk-%d algebra vectors matrices", m, i)
			repo.chunksByTopic[topicID.String()] = append(repo.chunksByTopic[topicID.String()], domain.MaterialChunk{
				ID:         uuid.New(),
				MaterialID: materialID,
				TopicID:    topicID,
				ChunkIndex: i,
				Content:    content,
				Embedding:  domain.PgVector(deterministicVector(content, 16)),
			})
		}
	}
}

func TestProcessMaterialChunks_ValidInput(t *testing.T) {
	ctx := context.Background()
	materialRepo := NewMockMaterialRepository()
	chunkRepo := NewMockChunkRepository()
	embedder := NewMockEmbedder()
	uc := NewRAGUseCase(materialRepo, chunkRepo, embedder)

	topicID := uuid.New()
	material := CreateTestMaterial(topicID, buildTextForChunks(3))
	materialRepo.materials[material.ID.String()] = material

	err := uc.ProcessMaterialChunks(ctx, material.ID.String())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if embedder.callCount == 0 {
		t.Fatal("expected embedder to be called")
	}
	if chunkRepo.saveCalls != 1 {
		t.Fatalf("expected SaveChunks to be called once, got %d", chunkRepo.saveCalls)
	}
	AssertChunksCreated(t, chunkRepo, material.ID, embedder.callCount)
}

func TestProcessMaterialChunks_EmptyText(t *testing.T) {
	ctx := context.Background()
	materialRepo := NewMockMaterialRepository()
	chunkRepo := NewMockChunkRepository()
	embedder := NewMockEmbedder()
	uc := NewRAGUseCase(materialRepo, chunkRepo, embedder)

	material := CreateTestMaterial(uuid.New(), "")
	materialRepo.materials[material.ID.String()] = material

	err := uc.ProcessMaterialChunks(ctx, material.ID.String())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if embedder.callCount != 0 {
		t.Fatalf("expected no embed calls, got %d", embedder.callCount)
	}
	if chunkRepo.saveCalls != 0 {
		t.Fatalf("expected SaveChunks not to be called, got %d", chunkRepo.saveCalls)
	}
}

func TestProcessMaterialChunks_LargeText(t *testing.T) {
	ctx := context.Background()
	materialRepo := NewMockMaterialRepository()
	chunkRepo := NewMockChunkRepository()
	embedder := NewMockEmbedder()
	uc := NewRAGUseCase(materialRepo, chunkRepo, embedder)

	largeText := strings.Repeat("B", 45000)
	material := CreateTestMaterial(uuid.New(), largeText)
	materialRepo.materials[material.ID.String()] = material

	err := uc.ProcessMaterialChunks(ctx, material.ID.String())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	created := len(chunkRepo.chunksByMaterial[material.ID.String()])
	if created != maxChunksPerMaterial {
		t.Fatalf("expected %d chunks (max limit), got %d", maxChunksPerMaterial, created)
	}
}

func TestProcessMaterialChunks_EmbedderFails(t *testing.T) {
	ctx := context.Background()
	materialRepo := NewMockMaterialRepository()
	chunkRepo := NewMockChunkRepository()
	embedder := NewMockEmbedder()
	embedder.failEvery = 2
	uc := NewRAGUseCase(materialRepo, chunkRepo, embedder)

	material := CreateTestMaterial(uuid.New(), buildTextForChunks(5))
	materialRepo.materials[material.ID.String()] = material

	err := uc.ProcessMaterialChunks(ctx, material.ID.String())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	totalCalls := embedder.callCount
	saved := len(chunkRepo.chunksByMaterial[material.ID.String()])
	if saved == totalCalls {
		t.Fatalf("expected partial save when embedder fails, saved=%d calls=%d", saved, totalCalls)
	}
	if saved == 0 {
		t.Fatal("expected some chunks to be saved despite embed failures")
	}
}

func TestProcessMaterialChunks_SaveFails(t *testing.T) {
	ctx := context.Background()
	materialRepo := NewMockMaterialRepository()
	chunkRepo := NewMockChunkRepository()
	chunkRepo.saveErr = errors.New("db write failed")
	embedder := NewMockEmbedder()
	uc := NewRAGUseCase(materialRepo, chunkRepo, embedder)

	material := CreateTestMaterial(uuid.New(), buildTextForChunks(3))
	materialRepo.materials[material.ID.String()] = material

	err := uc.ProcessMaterialChunks(ctx, material.ID.String())
	if err == nil {
		t.Fatal("expected error when SaveChunks fails")
	}
	if !strings.Contains(err.Error(), "rag: save chunks") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunkRepo.chunksByMaterial[material.ID.String()]) != 0 {
		t.Fatal("expected no chunks persisted on save failure")
	}
}

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
