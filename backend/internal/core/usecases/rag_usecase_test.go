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