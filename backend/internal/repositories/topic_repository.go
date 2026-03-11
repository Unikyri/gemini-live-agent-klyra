package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// TopicRepository handles persistence for course topics and their organization.
type TopicRepository struct {
	db *gorm.DB
}

// NewTopicRepository creates a new topic repository instance.
func NewTopicRepository(db *gorm.DB) *TopicRepository {
	return &TopicRepository{db: db}
}

// NewPostgresTopicRepository is a compatibility constructor used by main wiring.
func NewPostgresTopicRepository(db *gorm.DB) *TopicRepository {
	return NewTopicRepository(db)
}

// Create implements ports.TopicRepository.
func (r *TopicRepository) Create(ctx context.Context, topic *domain.Topic) error {
	_ = ctx
	return r.CreateTopic(topic)
}

// FindByID implements ports.TopicRepository.
func (r *TopicRepository) FindByID(ctx context.Context, topicID string) (*domain.Topic, error) {
	parsedTopicID, err := uuid.Parse(topicID)
	if err != nil {
		return nil, fmt.Errorf("invalid topic id: %w", err)
	}
	var topic domain.Topic
	result := r.db.WithContext(ctx).Where("id = ?", parsedTopicID).First(&topic)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find topic: %w", result.Error)
	}
	return &topic, nil
}

// FindByCourse implements ports.TopicRepository.
func (r *TopicRepository) FindByCourse(ctx context.Context, courseID string) ([]domain.Topic, error) {
	parsedCourseID, err := uuid.Parse(courseID)
	if err != nil {
		return nil, fmt.Errorf("invalid course id: %w", err)
	}
	topics, err := r.GetTopicsByCourse(parsedCourseID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Topic, 0, len(topics))
	for _, t := range topics {
		if t != nil {
			out = append(out, *t)
		}
	}
	_ = ctx
	return out, nil
}

// Update implements ports.TopicRepository.
func (r *TopicRepository) Update(ctx context.Context, topic *domain.Topic) error {
	topic.UpdatedAt = time.Now()
	result := r.db.WithContext(ctx).Model(topic).Updates(topic)
	if result.Error != nil {
		return fmt.Errorf("failed to update topic: %w", result.Error)
	}
	return nil
}

// SoftDelete marks the topic as deleted (sets deleted_at).
func (r *TopicRepository) SoftDelete(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&domain.Topic{}).Where("id = ?", id).Update("deleted_at", now)
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete topic: %w", result.Error)
	}
	return nil
}

// FindByCourseForCascade returns all topics for a course including soft-deleted (for cascade delete).
func (r *TopicRepository) FindByCourseForCascade(ctx context.Context, courseID string) ([]domain.Topic, error) {
	parsedCourseID, err := uuid.Parse(courseID)
	if err != nil {
		return nil, fmt.Errorf("invalid course id: %w", err)
	}
	var topics []domain.Topic
	err = r.db.WithContext(ctx).Unscoped().Where("course_id = ?", parsedCourseID).Order("order_index ASC").Find(&topics).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find topics for cascade: %w", err)
	}
	return topics, nil
}

// GetSummaryCache implements ports.TopicRepository.
func (r *TopicRepository) GetSummaryCache(ctx context.Context, topicID string) (*domain.TopicSummaryCache, error) {
	topic, err := r.FindByID(ctx, topicID)
	if err != nil || topic == nil {
		return nil, err
	}

	ids := make([]string, 0)
	if topic.SummaryMaterialIDs != "" {
		if err := json.Unmarshal([]byte(topic.SummaryMaterialIDs), &ids); err != nil {
			return nil, fmt.Errorf("failed to decode summary material IDs: %w", err)
		}
	}

	return &domain.TopicSummaryCache{
		TopicID:            topic.ID,
		SummaryMarkdown:    topic.SummaryMarkdown,
		SummarySourceHash:  topic.SummarySourceHash,
		SummaryMaterialIDs: ids,
		SummaryUpdatedAt:   topic.SummaryUpdatedAt,
	}, nil
}

// UpsertSummaryCache implements ports.TopicRepository.
func (r *TopicRepository) UpsertSummaryCache(ctx context.Context, cache domain.TopicSummaryCache) error {
	idsJSON, err := json.Marshal(cache.SummaryMaterialIDs)
	if err != nil {
		return fmt.Errorf("failed to encode summary material IDs: %w", err)
	}

	now := time.Now()
	updates := map[string]interface{}{
		"summary_markdown":     cache.SummaryMarkdown,
		"summary_source_hash":  cache.SummarySourceHash,
		"summary_material_ids": string(idsJSON),
		"summary_updated_at":   now,
		"updated_at":           now,
	}

	result := r.db.WithContext(ctx).Model(&domain.Topic{}).Where("id = ?", cache.TopicID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to upsert summary cache: %w", result.Error)
	}

	return nil
}

// CreateTopic persists a new topic under a course.
func (r *TopicRepository) CreateTopic(topic *domain.Topic) error {
	if topic.ID == uuid.Nil {
		topic.ID = uuid.New()
	}

	topic.CreatedAt = time.Now()
	topic.UpdatedAt = time.Now()

	result := r.db.Create(topic)
	if result.Error != nil {
		return fmt.Errorf("failed to create topic: %w", result.Error)
	}

	return nil
}

// GetTopicByID retrieves a topic with all its materials.
func (r *TopicRepository) GetTopicByID(topicID uuid.UUID) (*domain.Topic, error) {
	var topic domain.Topic
	result := r.db.
		Preload("Materials").
		Where("id = ?", topicID).
		First(&topic)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get topic: %w", result.Error)
	}

	return &topic, nil
}

// GetTopicsByCourse retrieves all topics for a course, ordered by order_index.
func (r *TopicRepository) GetTopicsByCourse(courseID uuid.UUID) ([]*domain.Topic, error) {
	var topics []*domain.Topic
	result := r.db.
		Where("course_id = ?", courseID).
		Order("order_index ASC").
		Find(&topics)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get topics for course: %w", result.Error)
	}

	return topics, nil
}

// UpdateTopic updates topic metadata (title, description, sequence).
func (r *TopicRepository) UpdateTopic(topic *domain.Topic) error {
	topic.UpdatedAt = time.Now()
	result := r.db.Model(topic).Updates(topic)
	if result.Error != nil {
		return fmt.Errorf("failed to update topic: %w", result.Error)
	}
	return nil
}

// DeleteTopic soft-deletes a topic (cascade to materials and chunks via FK).
func (r *TopicRepository) DeleteTopic(topicID uuid.UUID) error {
	result := r.db.Delete(&domain.Topic{}, "id = ?", topicID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete topic: %w", result.Error)
	}
	return nil
}

// CountTopicsByCourse returns the number of topics in a course.
func (r *TopicRepository) CountTopicsByCourse(courseID uuid.UUID) (int64, error) {
	var count int64
	result := r.db.Model(&domain.Topic{}).Where("course_id = ?", courseID).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count topics: %w", result.Error)
	}
	return count, nil
}
