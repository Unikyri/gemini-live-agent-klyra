package httphandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

// Test mocks for course handlers
type testCourseMockCourseRepository struct {
	mu      sync.Mutex
	courses map[string]*domain.Course
}

func (m *testCourseMockCourseRepository) Create(ctx context.Context, course *domain.Course) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if course.ID == uuid.Nil {
		course.ID = uuid.New()
	}
	m.courses[course.ID.String()] = course
	return nil
}

func (m *testCourseMockCourseRepository) FindByID(ctx context.Context, id string) (*domain.Course, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	course, ok := m.courses[id]
	if !ok {
		return nil, nil
	}
	return course, nil
}

func (m *testCourseMockCourseRepository) FindAllByUser(ctx context.Context, userID string) ([]domain.Course, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	courses := make([]domain.Course, 0)
	for _, c := range m.courses {
		if c.UserID == uid && c.DeletedAt == nil {
			courses = append(courses, *c)
		}
	}
	return courses, nil
}

func (m *testCourseMockCourseRepository) UpdateAvatarStatus(ctx context.Context, courseID, status, avatarURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	course, ok := m.courses[courseID]
	if !ok {
		return errors.New("course not found")
	}
	course.AvatarStatus = status
	if avatarURL != "" {
		course.AvatarModelURL = avatarURL
	}
	return nil
}

type testCourseMockTopicRepository struct {
	mu     sync.Mutex
	topics map[string]*domain.Topic
}

func (m *testCourseMockTopicRepository) Create(ctx context.Context, topic *domain.Topic) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if topic.ID == uuid.Nil {
		topic.ID = uuid.New()
	}
	m.topics[topic.ID.String()] = topic
	return nil
}

func (m *testCourseMockTopicRepository) FindByCourse(ctx context.Context, courseID string) ([]domain.Topic, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	results := make([]domain.Topic, 0)
	for _, t := range m.topics {
		if t.CourseID.String() == courseID && t.DeletedAt == nil {
			results = append(results, *t)
		}
	}
	return results, nil
}

type testCourseMockStorageService struct{}

func (m *testCourseMockStorageService) UploadFile(ctx context.Context, bucket, objectName string, data []byte, contentType string) (string, error) {
	return "/static/" + objectName, nil
}

type testCourseMockAvatarGenerator struct{}

func (m *testCourseMockAvatarGenerator) GenerateAvatar(ctx context.Context, referenceStyle string) ([]byte, string, error) {
	return []byte("fake"), "image/png", nil
}

// Test helper: create test CourseHandler with mocks
func setupCourseHandler(courseUseCase *usecases.CourseUseCase) *CourseHandler {
	return NewCourseHandler(courseUseCase)
}

// Test helper: add user_id to context
func withUserContext(router *gin.Engine, userID string) {
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
}

// Test case: CreateCourse success
func TestCourseHandler_CreateCourse_Success(t *testing.T) {
	courseRepo := &testCourseMockCourseRepository{courses: make(map[string]*domain.Course)}
	topicRepo := &testCourseMockTopicRepository{topics: make(map[string]*domain.Topic)}
	storageService := &testCourseMockStorageService{}
	avatarGen := &testCourseMockAvatarGenerator{}

	courseUseCase := usecases.NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)
	handler := setupCourseHandler(courseUseCase)

	router := gin.New()
	userID := uuid.New().String()
	withUserContext(router, userID)
	router.POST("/courses", handler.CreateCourse)

	// Prepare form request
	req := httptest.NewRequest(http.MethodPost, "/courses", nil)
	req.PostForm = map[string][]string{
		"name":            {"Math 101"},
		"education_level": {"high school"},
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Request = req
	handler.CreateCourse(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

// Test case: CreateCourse missing name
func TestCourseHandler_CreateCourse_MissingName(t *testing.T) {
	courseRepo := &testCourseMockCourseRepository{courses: make(map[string]*domain.Course)}
	topicRepo := &testCourseMockTopicRepository{topics: make(map[string]*domain.Topic)}
	storageService := &testCourseMockStorageService{}
	avatarGen := &testCourseMockAvatarGenerator{}

	courseUseCase := usecases.NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)
	handler := setupCourseHandler(courseUseCase)

	userID := uuid.New().String()

	req := httptest.NewRequest(http.MethodPost, "/courses", nil)
	req.PostForm = map[string][]string{
		"education_level": {"high school"},
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Request = req
	handler.CreateCourse(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test case: ListCourses
func TestCourseHandler_ListCourses(t *testing.T) {
	courseRepo := &testCourseMockCourseRepository{courses: make(map[string]*domain.Course)}
	topicRepo := &testCourseMockTopicRepository{topics: make(map[string]*domain.Topic)}
	storageService := &testCourseMockStorageService{}
	avatarGen := &testCourseMockAvatarGenerator{}

	courseUseCase := usecases.NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)
	handler := setupCourseHandler(courseUseCase)

	userID := uuid.New().String()

	// Create 2 courses
	input1 := usecases.CreateCourseInput{
		UserID:         userID,
		Name:           "Math 101",
		EducationLevel: "high school",
	}
	input2 := usecases.CreateCourseInput{
		UserID:         userID,
		Name:           "Physics 101",
		EducationLevel: "high school",
	}
	courseUseCase.CreateCourse(nil, input1)
	courseUseCase.CreateCourse(nil, input2)

	// List courses
	req := httptest.NewRequest(http.MethodGet, "/courses", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Request = req
	handler.ListCourses(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var respBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &respBody)

	if total, ok := respBody["total"].(float64); !ok || int(total) != 2 {
		t.Errorf("expected 2 courses, got %v", respBody["total"])
	}
}

// Test case: GetCourse with valid ownership
func TestCourseHandler_GetCourse_Success(t *testing.T) {
	courseRepo := &testCourseMockCourseRepository{courses: make(map[string]*domain.Course)}
	topicRepo := &testCourseMockTopicRepository{topics: make(map[string]*domain.Topic)}
	storageService := &testCourseMockStorageService{}
	avatarGen := &testCourseMockAvatarGenerator{}

	courseUseCase := usecases.NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)
	handler := setupCourseHandler(courseUseCase)

	userID := uuid.New().String()

	input := usecases.CreateCourseInput{
		UserID:         userID,
		Name:           "Math 101",
		EducationLevel: "high school",
	}
	createdCourse, _ := courseUseCase.CreateCourse(nil, input)
	courseID := createdCourse.ID.String()

	req := httptest.NewRequest(http.MethodGet, "/courses/"+courseID, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Params = []gin.Param{{Key: "course_id", Value: courseID}}
	c.Request = req
	handler.GetCourse(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Test case: GetCourse without ownership (must return 403)
func TestCourseHandler_GetCourse_Ownership_Denied(t *testing.T) {
	courseRepo := &testCourseMockCourseRepository{courses: make(map[string]*domain.Course)}
	topicRepo := &testCourseMockTopicRepository{topics: make(map[string]*domain.Topic)}
	storageService := &testCourseMockStorageService{}
	avatarGen := &testCourseMockAvatarGenerator{}

	courseUseCase := usecases.NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)
	handler := setupCourseHandler(courseUseCase)

	userID1 := uuid.New().String()
	userID2 := uuid.New().String()

	input := usecases.CreateCourseInput{
		UserID:         userID1,
		Name:           "Math 101",
		EducationLevel: "high school",
	}
	createdCourse, _ := courseUseCase.CreateCourse(nil, input)
	courseID := createdCourse.ID.String()

	// Try to access with different user
	req := httptest.NewRequest(http.MethodGet, "/courses/"+courseID, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID2)
	c.Params = []gin.Param{{Key: "course_id", Value: courseID}}
	c.Request = req
	handler.GetCourse(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// Test case: AddTopic
func TestCourseHandler_AddTopic_Success(t *testing.T) {
	courseRepo := &testCourseMockCourseRepository{courses: make(map[string]*domain.Course)}
	topicRepo := &testCourseMockTopicRepository{topics: make(map[string]*domain.Topic)}
	storageService := &testCourseMockStorageService{}
	avatarGen := &testCourseMockAvatarGenerator{}

	courseUseCase := usecases.NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)
	handler := setupCourseHandler(courseUseCase)

	userID := uuid.New().String()

	input := usecases.CreateCourseInput{
		UserID:         userID,
		Name:           "Math 101",
		EducationLevel: "high school",
	}
	createdCourse, _ := courseUseCase.CreateCourse(nil, input)
	courseID := createdCourse.ID.String()

	// Add topic
	topicReqBody := map[string]string{
		"title": "Calculus",
	}
	bodyBytes, _ := json.Marshal(topicReqBody)

	req := httptest.NewRequest(http.MethodPost, "/courses/"+courseID+"/topics", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Params = []gin.Param{{Key: "course_id", Value: courseID}}
	c.Request = req
	handler.AddTopic(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	topics, err := topicRepo.FindByCourse(context.Background(), courseID)
	if err != nil {
		t.Fatalf("unexpected error reading topics: %v", err)
	}
	if len(topics) != 1 || topics[0].Title != "Calculus" {
		t.Fatalf("expected persisted topic 'Calculus', got %+v", topics)
	}
}

// Test case: AddTopic missing title
func TestCourseHandler_AddTopic_MissingTitle(t *testing.T) {
	courseRepo := &testCourseMockCourseRepository{courses: make(map[string]*domain.Course)}
	topicRepo := &testCourseMockTopicRepository{topics: make(map[string]*domain.Topic)}
	storageService := &testCourseMockStorageService{}
	avatarGen := &testCourseMockAvatarGenerator{}

	courseUseCase := usecases.NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)
	handler := setupCourseHandler(courseUseCase)

	userID := uuid.New().String()
	courseID := uuid.New().String()

	// Empty body
	topicReqBody := map[string]string{}
	bodyBytes, _ := json.Marshal(topicReqBody)

	req := httptest.NewRequest(http.MethodPost, "/courses/"+courseID+"/topics", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Params = []gin.Param{{Key: "course_id", Value: courseID}}
	c.Request = req
	handler.AddTopic(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test case: AddTopic denied for non-owner (must return 403)
func TestCourseHandler_AddTopic_Ownership_Denied(t *testing.T) {
	courseRepo := &testCourseMockCourseRepository{courses: make(map[string]*domain.Course)}
	topicRepo := &testCourseMockTopicRepository{topics: make(map[string]*domain.Topic)}
	storageService := &testCourseMockStorageService{}
	avatarGen := &testCourseMockAvatarGenerator{}

	courseUseCase := usecases.NewCourseUseCase(courseRepo, topicRepo, storageService, avatarGen)
	handler := setupCourseHandler(courseUseCase)

	ownerID := uuid.New().String()
	otherUserID := uuid.New().String()

	createdCourse, _ := courseUseCase.CreateCourse(context.Background(), usecases.CreateCourseInput{
		UserID:         ownerID,
		Name:           "Math 101",
		EducationLevel: "high school",
	})
	courseID := createdCourse.ID.String()

	bodyBytes, _ := json.Marshal(map[string]string{"title": "Trigonometry"})
	req := httptest.NewRequest(http.MethodPost, "/courses/"+courseID+"/topics", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", otherUserID)
	c.Params = []gin.Param{{Key: "course_id", Value: courseID}}
	c.Request = req
	handler.AddTopic(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}
