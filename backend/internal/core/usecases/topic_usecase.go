package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
)

// TopicReadiness holds strict context readiness state for tutoring.
type TopicReadiness struct {
	IsReady        bool
	ValidatedCount int
	TotalCount     int
	Message        string
}

// TopicSummaryResult returns cached-or-regenerated topic summary payload.
type TopicSummaryResult struct {
	SummaryMarkdown string
	MaterialIDs     []string
	FromCache       bool
}

// TopicUseCase encapsulates topic-level readiness and summary operations.
type TopicUseCase struct {
	topicRepo        ports.TopicRepository
	materialRepo     ports.MaterialRepository
	summaryGenerator ports.SummaryGenerator
}

// NewTopicUseCase builds a topic use case with dependencies.
func NewTopicUseCase(
	topicRepo ports.TopicRepository,
	materialRepo ports.MaterialRepository,
	summaryGenerator ports.SummaryGenerator,
) *TopicUseCase {
	return &TopicUseCase{
		topicRepo:        topicRepo,
		materialRepo:     materialRepo,
		summaryGenerator: summaryGenerator,
	}
}

// CheckReadiness enforces strict context requirement: at least one validated material with extracted text.
func (uc *TopicUseCase) CheckReadiness(ctx context.Context, topicID string) (*TopicReadiness, error) {
	topic, err := uc.topicRepo.FindByID(ctx, topicID)
	if err != nil {
		return nil, err
	}
	if topic == nil {
		return nil, nil
	}

	totalCount, err := uc.materialRepo.CountByTopic(ctx, topicID)
	if err != nil {
		return nil, err
	}

	validatedCount, err := uc.materialRepo.CountReadyByTopic(ctx, topicID)
	if err != nil {
		return nil, err
	}

	isReady := validatedCount > 0
	message := "Ready to start tutoring"
	if !isReady {
		message = "Upload and validate at least one material to start tutoring"
	}

	return &TopicReadiness{
		IsReady:        isReady,
		ValidatedCount: validatedCount,
		TotalCount:     totalCount,
		Message:        message,
	}, nil
}

// GenerateSummary returns a persisted summary cache hit when source materials did not change.
func (uc *TopicUseCase) GenerateSummary(ctx context.Context, topicID string) (*TopicSummaryResult, error) {
	topic, err := uc.topicRepo.FindByID(ctx, topicID)
	if err != nil {
		return nil, err
	}
	if topic == nil {
		return nil, nil
	}

	materials, err := uc.materialRepo.FindValidatedByTopic(ctx, topicID)
	if err != nil {
		return nil, err
	}

	sourceHash, ids, consolidated := buildSummarySource(materials)
	cache, err := uc.topicRepo.GetSummaryCache(ctx, topicID)
	if err != nil {
		return nil, err
	}
	if cache != nil && cache.SummaryMarkdown != "" && cache.SummarySourceHash == sourceHash {
		return &TopicSummaryResult{
			SummaryMarkdown: cache.SummaryMarkdown,
			MaterialIDs:     cache.SummaryMaterialIDs,
			FromCache:       true,
		}, nil
	}

	summaryMarkdown, err := uc.summaryGenerator.Generate(ctx, topic.Title, consolidated)
	if err != nil {
		return nil, err
	}

	cachePayload := domain.TopicSummaryCache{
		TopicID:            topic.ID,
		SummaryMarkdown:    summaryMarkdown,
		SummarySourceHash:  sourceHash,
		SummaryMaterialIDs: ids,
	}
	if err := uc.topicRepo.UpsertSummaryCache(ctx, cachePayload); err != nil {
		return nil, err
	}

	return &TopicSummaryResult{
		SummaryMarkdown: summaryMarkdown,
		MaterialIDs:     ids,
		FromCache:       false,
	}, nil
}

func buildSummarySource(materials []domain.Material) (string, []string, string) {
	if len(materials) == 0 {
		sum := sha256.Sum256([]byte(""))
		return hex.EncodeToString(sum[:]), nil, ""
	}

	sort.Slice(materials, func(i, j int) bool {
		return materials[i].ID.String() < materials[j].ID.String()
	})

	ids := make([]string, 0, len(materials))
	builder := strings.Builder{}
	hashBuilder := strings.Builder{}
	for _, material := range materials {
		ids = append(ids, material.ID.String())
		builder.WriteString(material.ExtractedText)
		builder.WriteString("\n\n")

		hashBuilder.WriteString(material.ID.String())
		hashBuilder.WriteString("|")
		hashBuilder.WriteString(material.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"))
		hashBuilder.WriteString("|")
		hashBuilder.WriteString(material.ExtractedText)
		hashBuilder.WriteString("\n")
	}

	sum := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(sum[:]), ids, strings.TrimSpace(builder.String())
}
