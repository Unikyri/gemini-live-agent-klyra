package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// CourseRepository handles persistence for educational courses.
type CourseRepository struct {
	db           *gorm.DB
	imageService ImageService
}

// NewCourseRepository creates a new course repository instance.
func NewCourseRepository(db *gorm.DB, imageService ImageService) *CourseRepository {
	return &CourseRepository{
		db:           db,
		imageService: imageService,
	}
}

// NewPostgresCourseRepository is a compatibility constructor used by main wiring.
func NewPostgresCourseRepository(db *gorm.DB) *CourseRepository {
	return NewCourseRepository(db, nil)
}

// Create implements ports.CourseRepository.
func (r *CourseRepository) Create(ctx context.Context, course *domain.Course) error {
	return r.CreateCourse(ctx, course)
}

// FindByID implements ports.CourseRepository.
func (r *CourseRepository) FindByID(ctx context.Context, id string) (*domain.Course, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid course id: %w", err)
	}
	_ = ctx
	return r.GetCourseByID(parsedID)
}

// FindAllByUser implements ports.CourseRepository.
func (r *CourseRepository) FindAllByUser(ctx context.Context, userID string) ([]domain.Course, error) {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	courses, err := r.GetCoursesByUser(parsedUserID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Course, 0, len(courses))
	for _, c := range courses {
		if c != nil {
			out = append(out, *c)
		}
	}
	_ = ctx
	return out, nil
}

// UpdateAvatarStatus implements ports.CourseRepository.
func (r *CourseRepository) UpdateAvatarStatus(ctx context.Context, courseID, status, avatarURL string) error {
	updates := map[string]interface{}{
		"avatar_status": status,
		"updated_at":    time.Now(),
	}
	if avatarURL != "" {
		updates["avatar_model_url"] = avatarURL
	}
	result := r.db.WithContext(ctx).Model(&domain.Course{}).Where("id = ?", courseID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update avatar status: %w", result.Error)
	}
	return nil
}

// CreateCourse persists a new course record with optional thumbnail image.
func (r *CourseRepository) CreateCourse(ctx context.Context, course *domain.Course) error {
	if course.ID == uuid.Nil {
		course.ID = uuid.New()
	}

	course.CreatedAt = time.Now()
	course.UpdatedAt = time.Now()

	result := r.db.WithContext(ctx).Create(course)
	if result.Error != nil {
		return fmt.Errorf("failed to create course: %w", result.Error)
	}

	return nil
}

// GetCourseByID retrieves a course with all its topics and materials.
func (r *CourseRepository) GetCourseByID(courseID uuid.UUID) (*domain.Course, error) {
	var course domain.Course
	result := r.db.
		Preload("Topics", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Materials")
		}).
		Where("id = ?", courseID).
		First(&course)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get course: %w", result.Error)
	}

	return &course, nil
}

// GetCoursesByUser retrieves all courses owned by a user with their topics.
func (r *CourseRepository) GetCoursesByUser(userID uuid.UUID) ([]*domain.Course, error) {
	var courses []*domain.Course
	result := r.db.
		Preload("Topics", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Materials")
		}).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&courses)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get courses for user: %w", result.Error)
	}

	return courses, nil
}

// UpdateCourse updates course metadata (title, description, etc).
func (r *CourseRepository) UpdateCourse(course *domain.Course) error {
	course.UpdatedAt = time.Now()
	result := r.db.Model(course).Updates(course)
	if result.Error != nil {
		return fmt.Errorf("failed to update course: %w", result.Error)
	}
	return nil
}

// DeleteCourse soft-deletes a course (cascade to topics/materials via FK).
func (r *CourseRepository) DeleteCourse(courseID uuid.UUID) error {
	result := r.db.Delete(&domain.Course{}, "id = ?", courseID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete course: %w", result.Error)
	}
	return nil
}

// CountCoursesByUser returns the number of courses owned by a user.
func (r *CourseRepository) CountCoursesByUser(userID uuid.UUID) (int64, error) {
	var count int64
	result := r.db.Model(&domain.Course{}).Where("user_id = ?", userID).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count courses: %w", result.Error)
	}
	return count, nil
}
