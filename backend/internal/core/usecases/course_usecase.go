package usecases

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
)

// CourseUseCase holds the business rules for the Course Management module.
type CourseUseCase struct {
	courseRepo   ports.CourseRepository
	topicRepo    ports.TopicRepository
	materialRepo ports.MaterialRepository
	chunkRepo    ports.ChunkRepository
	storage      ports.StorageService
	avatarGen    ports.AvatarGenerator
	db         *gorm.DB
}

// NewCourseUseCase creates a new CourseUseCase with injected dependencies.
func NewCourseUseCase(
	courseRepo ports.CourseRepository,
	topicRepo ports.TopicRepository,
	storage ports.StorageService,
	avatarGen ports.AvatarGenerator,
) *CourseUseCase {
	return &CourseUseCase{
		courseRepo: courseRepo,
		topicRepo:  topicRepo,
		storage:    storage,
		avatarGen:  avatarGen,
	}
}

// NewCourseUseCaseWithCascade creates CourseUseCase with materialRepo, chunkRepo and db for cascade delete.
func NewCourseUseCaseWithCascade(
	courseRepo ports.CourseRepository,
	topicRepo ports.TopicRepository,
	materialRepo ports.MaterialRepository,
	chunkRepo ports.ChunkRepository,
	db *gorm.DB,
	storage ports.StorageService,
	avatarGen ports.AvatarGenerator,
) *CourseUseCase {
	return &CourseUseCase{
		courseRepo:   courseRepo,
		topicRepo:    topicRepo,
		materialRepo: materialRepo,
		chunkRepo:    chunkRepo,
		db:           db,
		storage:      storage,
		avatarGen:    avatarGen,
	}
}

// ErrCourseForbidden is returned when a user tries to access another user's course.
var ErrCourseForbidden = errors.New("forbidden course access")
var ErrCourseNotFound = errors.New("course not found")

// CreateCourseInput holds the data required to create a new course.
type CreateCourseInput struct {
	UserID             string
	Name               string
	EducationLevel     string
	ReferenceImageData []byte // raw bytes of the uploaded reference image
	ReferenceImageType string // MIME type, e.g. "image/jpeg"
}

// CreateCourse persists a new Course and triggers async avatar generation.
// The avatar is NOT generated synchronously — it would block the user too long.
// Instead, we mark it as "pending" and kick off a goroutine to generate it.
func (uc *CourseUseCase) CreateCourse(ctx context.Context, input CreateCourseInput) (*domain.Course, error) {
	userUUID, err := uuid.Parse(input.UserID)
	if err != nil {
		return nil, err
	}

	course := &domain.Course{
		UserID:         userUUID,
		Name:           input.Name,
		EducationLevel: input.EducationLevel,
		AvatarStatus:   "pending",
	}

	// If a reference image was provided, upload to Storage first.
	if len(input.ReferenceImageData) > 0 {
		objectName := "reference-images/" + uuid.New().String()
		refURL, uploadErr := uc.storage.UploadFile(ctx, "", objectName, input.ReferenceImageData, input.ReferenceImageType)
		if uploadErr != nil {
			log.Printf("Warning: failed to upload reference image: %v. Proceeding without it.", uploadErr)
		} else {
			course.ReferenceImageURL = refURL
		}
	}

	if err := uc.courseRepo.Create(ctx, course); err != nil {
		return nil, err
	}

	// Trigger avatar generation asynchronously so the API responds immediately.
	// The goroutine updates AvatarStatus to "ready" or "failed" when done.
	// NOTE: In production, replace with a Cloud Tasks / Pub/Sub job for reliability.
	go uc.generateAvatarAsync(course.ID.String(), course.ReferenceImageURL)

	return course, nil
}

// generateAvatarAsync is the background worker that calls Imagen and uploads the result.
// Runs in a separate goroutine — NEVER call synchronously (it would block the HTTP response).
// NOTE: In production, replace with Cloud Tasks or Pub/Sub for retry guarantees.
func (uc *CourseUseCase) generateAvatarAsync(courseID, referenceImageURL string) {
	// WARNING fix: use a bounded context so this goroutine cannot run forever
	// if the Vertex AI or GCS call hangs indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("[Avatar] Starting generation for course %s", courseID)

	// Mark the course as "generating" so the Flutter UI can show a loading indicator.
	if err := uc.courseRepo.UpdateAvatarStatus(ctx, courseID, "generating", ""); err != nil {
		log.Printf("[Avatar] Failed to set status=generating for course %s: %v", courseID, err)
	}

	// Step 1: Call Imagen to generate the transparent avatar PNG.
	// referenceImageURL is used as a style hint — pass it as the style description.
	imageBytes, mimeType, err := uc.avatarGen.GenerateAvatar(ctx, referenceImageURL)
	if err != nil {
		log.Printf("[Avatar] Imagen generation failed for course %s: %v", courseID, err)
		_ = uc.courseRepo.UpdateAvatarStatus(ctx, courseID, "failed", "")
		return
	}

	// Step 2: Upload the generated PNG to Cloud Storage.
	objectName := "avatars/" + courseID + "/avatar.png"
	avatarURL, err := uc.storage.UploadFile(ctx, "", objectName, imageBytes, mimeType)
	if err != nil {
		log.Printf("[Avatar] Upload to storage failed for course %s: %v", courseID, err)
		_ = uc.courseRepo.UpdateAvatarStatus(ctx, courseID, "failed", "")
		return
	}

	// Step 3: Update the course record with the avatar URL and "ready" status.
	if err := uc.courseRepo.UpdateAvatarStatus(ctx, courseID, "ready", avatarURL); err != nil {
		log.Printf("[Avatar] DB update failed for course %s: %v", courseID, err)
		return
	}

	log.Printf("[Avatar] Avatar ready for course %s — URL: %s", courseID, avatarURL)
}

// GetCoursesByUser retrieves all active courses belonging to a user.
// SECURITY: The userID MUST come from the JWT claims, never from the request body.
func (uc *CourseUseCase) GetCoursesByUser(ctx context.Context, userID string) ([]domain.Course, error) {
	return uc.courseRepo.FindAllByUser(ctx, userID)
}

// GetCourseByID retrieves a single Course, validating ownership.
func (uc *CourseUseCase) GetCourseByID(ctx context.Context, courseID, userID string) (*domain.Course, error) {
	course, err := uc.courseRepo.FindByID(ctx, courseID)
	if err != nil || course == nil {
		return nil, err
	}
	// SECURITY: Authorization check — ensure the requester owns this course.
	if course.UserID.String() != userID {
		return nil, ErrCourseForbidden
	}
	return course, nil
}

// AddTopic creates a new Topic under a Course, after validating ownership.
func (uc *CourseUseCase) AddTopic(ctx context.Context, courseID, userID, title string) (*domain.Topic, error) {
	// Validate the user owns the course before adding a topic.
	course, err := uc.GetCourseByID(ctx, courseID, userID)
	if err != nil || course == nil {
		return nil, err
	}

	courseUUID, err := uuid.Parse(courseID)
	if err != nil {
		// WARNING fix: return an explicit error instead of silently using a zero UUID.
		return nil, fmt.Errorf("invalid course ID format: %w", err)
	}
	topic := &domain.Topic{
		CourseID: courseUUID,
		Title:    title,
	}
	if err := uc.topicRepo.Create(ctx, topic); err != nil {
		return nil, err
	}
	return topic, nil
}

// UpdateCourseInput holds optional fields for partial course update.
type UpdateCourseInput struct {
	Name           string
	EducationLevel string
}

// UpdateCourse updates a course (partial update). Validates ownership.
func (uc *CourseUseCase) UpdateCourse(ctx context.Context, courseID, userID string, input UpdateCourseInput) (*domain.Course, error) {
	course, err := uc.GetCourseByID(ctx, courseID, userID)
	if err != nil || course == nil {
		return nil, err
	}
	if input.Name != "" {
		course.Name = input.Name
	}
	if input.EducationLevel != "" {
		course.EducationLevel = input.EducationLevel
	}
	if err := uc.courseRepo.Update(ctx, course); err != nil {
		return nil, err
	}
	return course, nil
}

// DeleteCourse soft-deletes a course and cascades to topics, materials, and chunks in a transaction.
func (uc *CourseUseCase) DeleteCourse(ctx context.Context, courseID, userID string) error {
	course, err := uc.GetCourseByID(ctx, courseID, userID)
	if err != nil || course == nil {
		if err != nil {
			return err
		}
		return ErrCourseNotFound
	}
	if uc.db == nil {
		return fmt.Errorf("cascade delete not configured")
	}
	topics, err := uc.topicRepo.FindByCourseForCascade(ctx, courseID)
	if err != nil {
		return err
	}
	topicIDs := make([]string, 0, len(topics))
	topicUUIDs := make([]uuid.UUID, 0, len(topics))
	for _, t := range topics {
		topicIDs = append(topicIDs, t.ID.String())
		topicUUIDs = append(topicUUIDs, t.ID)
	}
	err = uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)
		if len(topicUUIDs) > 0 {
			if res := tx.Unscoped().Where("topic_id IN ?", topicUUIDs).Delete(&domain.MaterialChunk{}); res.Error != nil {
				return res.Error
			}
			if res := tx.Model(&domain.Material{}).Where("topic_id IN ?", topicUUIDs).Where("deleted_at IS NULL").Update("deleted_at", time.Now()); res.Error != nil {
				return res.Error
			}
			for _, id := range topicIDs {
				if res := tx.Model(&domain.Topic{}).Where("id = ?", id).Update("deleted_at", time.Now()); res.Error != nil {
					return res.Error
				}
			}
		}
		return tx.Model(&domain.Course{}).Where("id = ?", courseID).Update("deleted_at", time.Now()).Error
	})
	return err
}

// UpdateTopicInput holds optional fields for partial topic update.
type UpdateTopicInput struct {
	Title string
}

// UpdateTopic updates a topic (partial update). Validates course ownership and topic belongs to course.
func (uc *CourseUseCase) UpdateTopic(ctx context.Context, courseID, topicID, userID string, input UpdateTopicInput) (*domain.Topic, error) {
	course, err := uc.GetCourseByID(ctx, courseID, userID)
	if err != nil || course == nil {
		return nil, err
	}
	topic, err := uc.topicRepo.FindByID(ctx, topicID)
	if err != nil || topic == nil {
		return nil, err
	}
	if topic.CourseID.String() != courseID {
		return nil, fmt.Errorf("topic does not belong to course")
	}
	if input.Title != "" {
		topic.Title = input.Title
	}
	if err := uc.topicRepo.Update(ctx, topic); err != nil {
		return nil, err
	}
	return topic, nil
}

// DeleteTopic soft-deletes a topic and cascades to materials and chunks in a transaction.
func (uc *CourseUseCase) DeleteTopic(ctx context.Context, courseID, topicID, userID string) error {
	course, err := uc.GetCourseByID(ctx, courseID, userID)
	if err != nil || course == nil {
		return err
	}
	topic, err := uc.topicRepo.FindByID(ctx, topicID)
	if err != nil || topic == nil {
		return err
	}
	if topic.CourseID.String() != courseID {
		return fmt.Errorf("topic does not belong to course")
	}
	topicUUID, _ := uuid.Parse(topicID)
	err = uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)
		if res := tx.Unscoped().Where("topic_id = ?", topicUUID).Delete(&domain.MaterialChunk{}); res.Error != nil {
			return res.Error
		}
		if res := tx.Model(&domain.Material{}).Where("topic_id = ?", topicUUID).Where("deleted_at IS NULL").Update("deleted_at", time.Now()); res.Error != nil {
			return res.Error
		}
		return tx.Model(&domain.Topic{}).Where("id = ?", topicID).Update("deleted_at", time.Now()).Error
	})
	return err
}
