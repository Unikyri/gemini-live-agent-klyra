package usecases

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

type topicRepoMock struct {
	topic *domain.Topic
	cache *domain.TopicSummaryCache
}

func (m *topicRepoMock) Create(ctx context.Context, topic *domain.Topic) error {
	_ = ctx
	m.topic = topic
	return nil
}

func (m *topicRepoMock) FindByID(ctx context.Context, topicID string) (*domain.Topic, error) {
	_ = ctx
	if m.topic != nil && m.topic.ID.String() == topicID {
		return m.topic, nil
	}
	return nil, nil
}

func (m *topicRepoMock) FindByCourse(ctx context.Context, courseID string) ([]domain.Topic, error) {
	_ = ctx
	_ = courseID
	if m.topic == nil {
		return nil, nil
	}
	return []domain.Topic{*m.topic}, nil
}

func (m *topicRepoMock) GetSummaryCache(ctx context.Context, topicID string) (*domain.TopicSummaryCache, error) {
	_ = ctx
	_ = topicID
	return m.cache, nil
}

func (m *topicRepoMock) UpsertSummaryCache(ctx context.Context, cache domain.TopicSummaryCache) error {
	_ = ctx
	m.cache = &cache
	return nil
}

type materialRepoMock struct {
	materials []domain.Material
}

func (m *materialRepoMock) Create(ctx context.Context, material *domain.Material) error {
	_ = ctx
	_ = material
	return nil
}

func (m *materialRepoMock) FindByID(ctx context.Context, id string) (*domain.Material, error) {
	_ = ctx
	_ = id
	return nil, nil
}

func (m *materialRepoMock) FindByTopic(ctx context.Context, topicID string) ([]domain.Material, error) {
	_ = ctx
	_ = topicID
	return m.materials, nil
}

func (m *materialRepoMock) FindValidatedByTopic(ctx context.Context, topicID string) ([]domain.Material, error) {
	_ = ctx
	_ = topicID
	return m.materials, nil
}

func (m *materialRepoMock) CountByTopic(ctx context.Context, topicID string) (int, error) {
	_ = ctx
	_ = topicID
	return len(m.materials), nil
}

func (m *materialRepoMock) CountReadyByTopic(ctx context.Context, topicID string) (int, error) {
	_ = ctx
	_ = topicID
	return len(m.materials), nil
}

func (m *materialRepoMock) UpdateStatus(ctx context.Context, materialID string, status domain.MaterialStatus, extractedText string) error {
	_ = ctx
	_ = materialID
	_ = status
	_ = extractedText
	return nil
}

type summaryGenMock struct{}

func (s *summaryGenMock) Generate(ctx context.Context, topicTitle string, contextText string) (string, error) {
	_ = ctx
	return "## " + topicTitle + "\n\n" + contextText, nil
}

func TestTopicUseCase_GenerateSummary_UsesCache(t *testing.T) {
	topicID := uuid.New()
	now := time.Now()
	mat := domain.Material{ID: uuid.New(), ExtractedText: "contenido", UpdatedAt: now}

	topicRepo := &topicRepoMock{topic: &domain.Topic{ID: topicID, Title: "Algebra"}}
	materialRepo := &materialRepoMock{materials: []domain.Material{mat}}
	uc := NewTopicUseCase(topicRepo, materialRepo, &summaryGenMock{})

	first, err := uc.GenerateSummary(context.Background(), topicID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first == nil || first.FromCache {
		t.Fatalf("expected generated summary on first call")
	}

	second, err := uc.GenerateSummary(context.Background(), topicID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second == nil || !second.FromCache {
		t.Fatalf("expected cache hit on second call")
	}
}

// TestTopicUseCase_GenerateSummary_CacheInvalidationOnContentChange tests cache invalidation when material content changes.
func TestTopicUseCase_GenerateSummary_CacheInvalidationOnContentChange(t *testing.T) {
	topicID := uuid.New()
	matID := uuid.New()
	now := time.Now()
	mat := domain.Material{ID: matID, ExtractedText: "original content", UpdatedAt: now}

	topicRepo := &topicRepoMock{topic: &domain.Topic{ID: topicID, Title: "Physics"}}
	materialRepo := &materialRepoMock{materials: []domain.Material{mat}}
	uc := NewTopicUseCase(topicRepo, materialRepo, &summaryGenMock{})

	first, err := uc.GenerateSummary(context.Background(), topicID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.FromCache {
		t.Fatalf("expected fresh generation on first call")
	}

	// Change content (simulating material update)
	mat.ExtractedText = "modified content"
	mat.UpdatedAt = now.Add(time.Hour)
	materialRepo.materials = []domain.Material{mat}

	second, err := uc.GenerateSummary(context.Background(), topicID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second.FromCache {
		t.Fatalf("expected cache invalidation after material content change")
	}
}

// TestTopicUseCase_GenerateSummary_CacheInvalidationOnMaterialAdded tests cache invalidation when new material added.
func TestTopicUseCase_GenerateSummary_CacheInvalidationOnMaterialAdded(t *testing.T) {
	topicID := uuid.New()
	now := time.Now()
	mat1 := domain.Material{ID: uuid.New(), ExtractedText: "material 1", UpdatedAt: now}

	topicRepo := &topicRepoMock{topic: &domain.Topic{ID: topicID, Title: "Chemistry"}}
	materialRepo := &materialRepoMock{materials: []domain.Material{mat1}}
	uc := NewTopicUseCase(topicRepo, materialRepo, &summaryGenMock{})

	first, err := uc.GenerateSummary(context.Background(), topicID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.FromCache || len(first.MaterialIDs) != 1 {
		t.Fatalf("expected fresh generation with 1 material")
	}

	// Add new material
	mat2 := domain.Material{ID: uuid.New(), ExtractedText: "material 2", UpdatedAt: now}
	materialRepo.materials = []domain.Material{mat1, mat2}

	second, err := uc.GenerateSummary(context.Background(), topicID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second.FromCache {
		t.Fatalf("expected cache invalidation after adding new material")
	}
	if len(second.MaterialIDs) != 2 {
		t.Fatalf("expected 2 materials in summary result")
	}
}

// TestTopicUseCase_CheckReadiness_NotReady_ZeroMaterials tests readiness check when no materials uploaded.
func TestTopicUseCase_CheckReadiness_NotReady_ZeroMaterials(t *testing.T) {
	topicID := uuid.New()
	topicRepo := &topicRepoMock{topic: &domain.Topic{ID: topicID, Title: "Empty Topic"}}
	materialRepo := &materialRepoMock{materials: []domain.Material{}}
	uc := NewTopicUseCase(topicRepo, materialRepo, &summaryGenMock{})

	readiness, err := uc.CheckReadiness(context.Background(), topicID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if readiness.IsReady {
		t.Fatalf("expected not ready when zero materials")
	}
	if readiness.ValidatedCount != 0 || readiness.TotalCount != 0 {
		t.Fatalf("expected count 0, got validated=%d total=%d", readiness.ValidatedCount, readiness.TotalCount)
	}
	if !strings.Contains(readiness.Message, "Upload and validate at least one material") {
		t.Fatalf("expected blocking message, got: %s", readiness.Message)
	}
}

// TestTopicUseCase_CheckReadiness_Ready_OneMaterial tests readiness check when at least one validated material exists.
func TestTopicUseCase_CheckReadiness_Ready_OneMaterial(t *testing.T) {
	topicID := uuid.New()
	mat := domain.Material{ID: uuid.New(), ExtractedText: "validated content"}
	topicRepo := &topicRepoMock{topic: &domain.Topic{ID: topicID, Title: "Ready Topic"}}
	materialRepo := &materialRepoMock{materials: []domain.Material{mat}}
	uc := NewTopicUseCase(topicRepo, materialRepo, &summaryGenMock{})

	readiness, err := uc.CheckReadiness(context.Background(), topicID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !readiness.IsReady {
		t.Fatalf("expected ready when >=1 validated material")
	}
	if readiness.ValidatedCount != 1 || readiness.TotalCount != 1 {
		t.Fatalf("expected count 1, got validated=%d total=%d", readiness.ValidatedCount, readiness.TotalCount)
	}
	if !strings.Contains(readiness.Message, "Ready to start tutoring") {
		t.Fatalf("expected ready message, got: %s", readiness.Message)
	}
}

// TestTopicUseCase_CheckReadiness_TopicNotFound tests nil result when topic doesn't exist.
func TestTopicUseCase_CheckReadiness_TopicNotFound(t *testing.T) {
	topicRepo := &topicRepoMock{topic: nil}
	materialRepo := &materialRepoMock{materials: []domain.Material{}}
	uc := NewTopicUseCase(topicRepo, materialRepo, &summaryGenMock{})

	readiness, err := uc.CheckReadiness(context.Background(), "nonexistent-topic-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if readiness != nil {
		t.Fatalf("expected nil readiness when topic not found")
	}
}

// TestTopicUseCase_GenerateSummary_TopicNotFound tests nil result when topic doesn't exist.
func TestTopicUseCase_GenerateSummary_TopicNotFound(t *testing.T) {
	topicRepo := &topicRepoMock{topic: nil}
	materialRepo := &materialRepoMock{materials: []domain.Material{}}
	uc := NewTopicUseCase(topicRepo, materialRepo, &summaryGenMock{})

	result, err := uc.GenerateSummary(context.Background(), "nonexistent-topic-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result when topic not found")
	}
}
