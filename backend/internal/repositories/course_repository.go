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
	db            *gorm.DB
	imageService  ImageService
}

// NewCourseRepository creates a new course repository instance.
func NewCourseRepository(db *gorm.DB, imageService ImageService) *CourseRepository {
	return &CourseRepository{
		db:           db,
		imageService: imageService,
	}
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

// GetCoursesByUser retrieves all courses owned by a user.
func (r *CourseRepository) GetCoursesByUser(userID uuid.UUID) ([]*domain.Course, error) {
	var courses []*domain.Course
	result := r.db.
		Where("owner_id = ?", userID).
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
	result := r.db.Model(&domain.Course{}).Where("owner_id = ?", userID).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count courses: %w", result.Error)
	}
	return count, nil
}
