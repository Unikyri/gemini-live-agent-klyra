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
	avatarGen  ports.AvatarGenerator
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
	// New context — the HTTP request context will be cancelled by the time this runs.
	ctx := context.Background()
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
