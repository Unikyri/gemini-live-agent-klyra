package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// MockCourseRepository for testing
type MockCourseRepository struct {
	courses         map[string]*domain.Course
	createFn        func(ctx context.Context, course *domain.Course) error
	findByIDFn      func(ctx context.Context, courseID string) (*domain.Course, error)
	findAllByUserFn func(ctx context.Context, userID string) ([]domain.Course, error)
}

func NewMockCourseRepository() *MockCourseRepository {
	return &MockCourseRepository{
		courses: make(map[string]*domain.Course),
	}
}

func (m *MockCourseRepository) Create(ctx context.Context, course *domain.Course) error {
	if m.createFn != nil {
		if err := m.createFn(ctx, course); err != nil {
			return err
		}
	}
	if course.ID == uuid.Nil {
		course.ID = uuid.New()
	}
	m.courses[course.ID.String()] = course
	return nil
}

func (m *MockCourseRepository) FindByID(ctx context.Context, courseID string) (*domain.Course, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, courseID)
	}
	if course, ok := m.courses[courseID]; ok {
		return course, nil
	}
	return nil, nil
}

func (m *MockCourseRepository) FindAllByUser(ctx context.Context, userID string) ([]domain.Course, error) {
	if m.findAllByUserFn != nil {
		return m.findAllByUserFn(ctx, userID)
	}
	var courses []domain.Course
	userUUID, _ := uuid.Parse(userID)
	for _, course := range m.courses {
		if course.UserID == userUUID && course.DeletedAt == nil {
			courses = append(courses, *course)
		}
	}
	return courses, nil
}

func (m *MockCourseRepository) UpdateAvatarStatus(ctx context.Context, courseID, status, avatarURL string) error {
	if course, ok := m.courses[courseID]; ok {
		course.AvatarStatus = status
		if avatarURL != "" {
			course.AvatarModelURL = avatarURL
		}
		return nil
	}
	return errors.New("course not found")
}

// MockStorageService for testing
type MockStorageService struct {
	uploadFileFn func(ctx context.Context, userID, objectName string, data []byte, mimeType string) (string, error)
}

func NewMockStorageService() *MockStorageService {
	return &MockStorageService{
		uploadFileFn: func(ctx context.Context, userID, objectName string, data []byte, mimeType string) (string, error) {
			return "/static/" + objectName, nil
		},
	}
}

func (m *MockStorageService) UploadFile(ctx context.Context, userID, objectName string, data []byte, mimeType string) (string, error) {
	if m.uploadFileFn != nil {
		return m.uploadFileFn(ctx, userID, objectName, data, mimeType)
	}
	return "/static/" + objectName, nil
}

// MockAvatarGenerator for testing
type MockAvatarGenerator struct {
	generateAvatarFn func(ctx context.Context, referenceImageURL string) ([]byte, string, error)
}

func NewMockAvatarGenerator() *MockAvatarGenerator {
	return &MockAvatarGenerator{
		generateAvatarFn: func(ctx context.Context, referenceImageURL string) ([]byte, string, error) {
			return []byte("fake_avatar_data"), "image/png", nil
		},
	}
}

func (m *MockAvatarGenerator) GenerateAvatar(ctx context.Context, referenceImageURL string) ([]byte, string, error) {
	if m.generateAvatarFn != nil {
		return m.generateAvatarFn(ctx, referenceImageURL)
	}
	return []byte(""), "", nil
}

// MockTopicRepository for testing
type MockTopicRepository struct {
	topics map[string]*domain.Topic
}

func NewMockTopicRepository() *MockTopicRepository {
	return &MockTopicRepository{
		topics: make(map[string]*domain.Topic),
	}
}

func (m *MockTopicRepository) Create(ctx context.Context, topic *domain.Topic) error {
	if topic.ID == uuid.Nil {
		topic.ID = uuid.New()
	}
	m.topics[topic.ID.String()] = topic
	return nil
}

func (m *MockTopicRepository) FindByCourse(ctx context.Context, courseID string) ([]domain.Topic, error) {
	var topics []domain.Topic
	for _, t := range m.topics {
		if t.CourseID.String() == courseID && t.DeletedAt == nil {
			topics = append(topics, *t)
		}
	}
	return topics, nil
}

// --- Course Use Case Tests ---

func TestCourseUseCase_CreateCourse_NoImage(t *testing.T) {
	courseRepo := NewMockCourseRepository()
	topicRepo := NewMockTopicRepository()
	storageService := NewMockStorageService()
	avatarGen := NewMockAvatarGenerator()

	uc := NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)

	input := CreateCourseInput{
		UserID:         uuid.New().String(),
		Name:           "Math 101",
		EducationLevel: "high school",
	}

	course, err := uc.CreateCourse(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateCourse failed: %v", err)
	}

	if course == nil {
		t.Fatal("expected Course, got nil")
	}

	if course.Name != "Math 101" {
		t.Errorf("expected name 'Math 101', got '%s'", course.Name)
	}

	if course.AvatarStatus != "pending" {
		t.Errorf("expected AvatarStatus 'pending', got '%s'", course.AvatarStatus)
	}
}

func TestCourseUseCase_CreateCourse_WithImage(t *testing.T) {
	courseRepo := NewMockCourseRepository()
	topicRepo := NewMockTopicRepository()
	storageService := NewMockStorageService()
	avatarGen := NewMockAvatarGenerator()

	uc := NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)

	input := CreateCourseInput{
		UserID:             uuid.New().String(),
		Name:               "Math 101",
		EducationLevel:     "high school",
		ReferenceImageData: []byte("fake_image_data"),
		ReferenceImageType: "image/jpeg",
	}

	course, err := uc.CreateCourse(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateCourse failed: %v", err)
	}

	if course.ReferenceImageURL == "" {
		t.Error("expected ReferenceImageURL to be set")
	}
}

func TestCourseUseCase_CreateCourse_InvalidUserID(t *testing.T) {
	courseRepo := NewMockCourseRepository()
	topicRepo := NewMockTopicRepository()
	storageService := NewMockStorageService()
	avatarGen := NewMockAvatarGenerator()

	uc := NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)

	input := CreateCourseInput{
		UserID:         "not_a_uuid",
		Name:           "Math 101",
		EducationLevel: "high school",
	}

	_, err := uc.CreateCourse(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestCourseUseCase_GetCoursesByUser(t *testing.T) {
	courseRepo := NewMockCourseRepository()
	topicRepo := NewMockTopicRepository()
	storageService := NewMockStorageService()
	avatarGen := NewMockAvatarGenerator()

	uc := NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)

	userID := uuid.New().String()

	// Create 2 courses for user
	input1 := CreateCourseInput{
		UserID:         userID,
		Name:           "Math 101",
		EducationLevel: "high school",
	}
	input2 := CreateCourseInput{
		UserID:         userID,
		Name:           "Physics 101",
		EducationLevel: "high school",
	}

	uc.CreateCourse(context.Background(), input1)
	uc.CreateCourse(context.Background(), input2)

	courses, err := uc.GetCoursesByUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetCoursesByUser failed: %v", err)
	}

	if len(courses) != 2 {
		t.Errorf("expected 2 courses, got %d", len(courses))
	}
}

func TestCourseUseCase_GetCourseByID_Ownership_Valid(t *testing.T) {
	courseRepo := NewMockCourseRepository()
	topicRepo := NewMockTopicRepository()
	storageService := NewMockStorageService()
	avatarGen := NewMockAvatarGenerator()

	uc := NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)

	userID := uuid.New().String()
	input := CreateCourseInput{
		UserID:         userID,
		Name:           "Math 101",
		EducationLevel: "high school",
	}

	createdCourse, _ := uc.CreateCourse(context.Background(), input)
	courseID := createdCourse.ID.String()

	// Fetch as owner
	course, err := uc.GetCourseByID(context.Background(), courseID, userID)
	if err != nil {
		t.Fatalf("GetCourseByID failed: %v", err)
	}

	if course == nil {
		t.Fatal("expected Course, got nil")
	}

	if course.Name != "Math 101" {
		t.Errorf("expected name 'Math 101', got '%s'", course.Name)
	}
}

func TestCourseUseCase_GetCourseByID_Ownership_Denied(t *testing.T) {
	courseRepo := NewMockCourseRepository()
	topicRepo := NewMockTopicRepository()
	storageService := NewMockStorageService()
	avatarGen := NewMockAvatarGenerator()

	uc := NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)

	userID1 := uuid.New().String()
	userID2 := uuid.New().String()

	input := CreateCourseInput{
		UserID:         userID1,
		Name:           "Math 101",
		EducationLevel: "high school",
	}

	createdCourse, _ := uc.CreateCourse(context.Background(), input)
	courseID := createdCourse.ID.String()

	// Try to fetch as different user
	course, err := uc.GetCourseByID(context.Background(), courseID, userID2)
	if !errors.Is(err, ErrCourseForbidden) {
		t.Fatalf("expected ErrCourseForbidden, got: %v", err)
	}

	if course != nil {
		t.Fatal("expected nil Course when ownership check fails, got course")
	}
}

func TestCourseUseCase_CreateCourse_StorageUploadError(t *testing.T) {
	courseRepo := NewMockCourseRepository()
	topicRepo := NewMockTopicRepository()
	storageService := NewMockStorageService()
	storageService.uploadFileFn = func(ctx context.Context, userID, objectName string, data []byte, mimeType string) (string, error) {
		return "", errors.New("storage down")
	}
	avatarGen := NewMockAvatarGenerator()

	uc := NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)

	input := CreateCourseInput{
		UserID:             uuid.New().String(),
		Name:               "Math 101",
		EducationLevel:     "high school",
		ReferenceImageData: []byte("fake_data"),
		ReferenceImageType: "image/jpeg",
	}

	// Should still succeed even if storage fails (course created, image upload warning logged)
	course, err := uc.CreateCourse(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateCourse should succeed even if image upload fails, got: %v", err)
	}

	if course.ReferenceImageURL != "" {
		t.Error("expected ReferenceImageURL to be empty when upload fails")
	}
}

func TestCourseUseCase_AddTopic_Success(t *testing.T) {
	courseRepo := NewMockCourseRepository()
	topicRepo := NewMockTopicRepository()
	storageService := NewMockStorageService()
	avatarGen := NewMockAvatarGenerator()

	uc := NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)

	userID := uuid.New().String()
	createdCourse, err := uc.CreateCourse(context.Background(), CreateCourseInput{
		UserID:         userID,
		Name:           "Calculus",
		EducationLevel: "university",
	})
	if err != nil {
		t.Fatalf("CreateCourse failed: %v", err)
	}

	topic, err := uc.AddTopic(context.Background(), createdCourse.ID.String(), userID, "Limits")
	if err != nil {
		t.Fatalf("AddTopic failed: %v", err)
	}

	if topic == nil {
		t.Fatal("expected topic, got nil")
	}

	if topic.Title != "Limits" {
		t.Errorf("expected topic title 'Limits', got '%s'", topic.Title)
	}

	if topic.CourseID != createdCourse.ID {
		t.Errorf("expected course ID %s, got %s", createdCourse.ID, topic.CourseID)
	}
}

func TestCourseUseCase_AddTopic_ForbiddenWhenNotOwner(t *testing.T) {
	courseRepo := NewMockCourseRepository()
	topicRepo := NewMockTopicRepository()
	storageService := NewMockStorageService()
	avatarGen := NewMockAvatarGenerator()

	uc := NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)

	ownerID := uuid.New().String()
	otherUserID := uuid.New().String()

	createdCourse, err := uc.CreateCourse(context.Background(), CreateCourseInput{
		UserID:         ownerID,
		Name:           "Physics",
		EducationLevel: "high school",
	})
	if err != nil {
		t.Fatalf("CreateCourse failed: %v", err)
	}

	topic, err := uc.AddTopic(context.Background(), createdCourse.ID.String(), otherUserID, "Kinematics")
	if !errors.Is(err, ErrCourseForbidden) {
		t.Fatalf("expected ErrCourseForbidden, got: %v", err)
	}

	if topic != nil {
		t.Fatal("expected nil topic when user is not owner")
	}
}
