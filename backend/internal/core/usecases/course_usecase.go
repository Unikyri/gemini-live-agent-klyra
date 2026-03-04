package usecases

import (
	"context"
	"log"

	"github.com/google/uuid"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
)

// CourseUseCase holds the business rules for the Course Management module.
type CourseUseCase struct {
	courseRepo ports.CourseRepository
	topicRepo  ports.TopicRepository
	storage    ports.StorageService
}

// NewCourseUseCase creates a new CourseUseCase with injected dependencies.
func NewCourseUseCase(
	courseRepo ports.CourseRepository,
	topicRepo ports.TopicRepository,
	storage ports.StorageService,
) *CourseUseCase {
	return &CourseUseCase{
		courseRepo: courseRepo,
		topicRepo:  topicRepo,
		storage:    storage,
	}
}

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

// generateAvatarAsync is the background worker that calls Imagen for avatar generation.
// It runs in a separate goroutine — never call this synchronously.
func (uc *CourseUseCase) generateAvatarAsync(courseID, referenceImageURL string) {
	// We create a new background context since the HTTP request context will be cancelled.
	ctx := context.Background()
	log.Printf("Starting avatar generation for course %s", courseID)

	// TODO (US3): Replace this log with actual Gemini Imagen API call.
	// When implemented, it will:
	// 1. Call Imagen with a prompt based on referenceImageURL.
	// 2. Receive a transparent-background PNG.
	// 3. Upload the result to Storage.
	// 4. Call uc.courseRepo.UpdateAvatarStatus(ctx, courseID, "ready", avatarURL).
	log.Printf("Avatar generation placeholder for course %s — will be implemented in US3.", courseID)
	_ = uc.courseRepo.UpdateAvatarStatus(ctx, courseID, "pending", "")
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
		return nil, nil // return nil to trigger 404, not 403 (to avoid resource enumeration)
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

	courseUUID, _ := uuid.Parse(courseID)
	topic := &domain.Topic{
		CourseID: courseUUID,
		Title:    title,
	}
	if err := uc.topicRepo.Create(ctx, topic); err != nil {
		return nil, err
	}
	return topic, nil
}
