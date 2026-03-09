package repositories

import (
	"context"
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

// GetTopicsByCourse retrieves all topics for a course, ordered by sequence.
func (r *TopicRepository) GetTopicsByCourse(courseID uuid.UUID) ([]*domain.Topic, error) {
	var topics []*domain.Topic
	result := r.db.
		Where("course_id = ?", courseID).
		Order("sequence ASC").
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
