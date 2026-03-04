package repositories

import (
	"context"
	"errors"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"gorm.io/gorm"
)

// PostgresCourseRepository is the GORM implementation of ports.CourseRepository.
type PostgresCourseRepository struct {
	db *gorm.DB
}

// NewPostgresCourseRepository creates a new repository instance.
func NewPostgresCourseRepository(db *gorm.DB) *PostgresCourseRepository {
	return &PostgresCourseRepository{db: db}
}

func (r *PostgresCourseRepository) Create(ctx context.Context, course *domain.Course) error {
	return r.db.WithContext(ctx).Create(course).Error
}

func (r *PostgresCourseRepository) FindByID(ctx context.Context, id string) (*domain.Course, error) {
	var course domain.Course
	result := r.db.WithContext(ctx).
		Preload("Topics").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&course)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &course, result.Error
}

func (r *PostgresCourseRepository) FindAllByUser(ctx context.Context, userID string) ([]domain.Course, error) {
	var courses []domain.Course
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").
		Find(&courses)
	return courses, result.Error
}

func (r *PostgresCourseRepository) UpdateAvatarStatus(ctx context.Context, courseID, status, avatarURL string) error {
	updates := map[string]interface{}{
		"avatar_status": status,
	}
	if avatarURL != "" {
		updates["avatar_model_url"] = avatarURL
	}
	return r.db.WithContext(ctx).
		Model(&domain.Course{}).
		Where("id = ?", courseID).
		Updates(updates).Error
}

// PostgresTopicRepository is the GORM implementation of ports.TopicRepository.
type PostgresTopicRepository struct {
	db *gorm.DB
}

// NewPostgresTopicRepository creates a new topic repository.
func NewPostgresTopicRepository(db *gorm.DB) *PostgresTopicRepository {
	return &PostgresTopicRepository{db: db}
}

func (r *PostgresTopicRepository) Create(ctx context.Context, topic *domain.Topic) error {
	return r.db.WithContext(ctx).Create(topic).Error
}

func (r *PostgresTopicRepository) FindByCourse(ctx context.Context, courseID string) ([]domain.Topic, error) {
	var topics []domain.Topic
	result := r.db.WithContext(ctx).
		Where("course_id = ? AND deleted_at IS NULL", courseID).
		Order("order_index ASC, created_at ASC").
		Find(&topics)
	return topics, result.Error
}
