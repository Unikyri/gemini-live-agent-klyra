package httphandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/repositories"
)

type testMaterialMockCourseRepository struct {
	mu      sync.Mutex
	courses map[string]*domain.Course
}

func (m *testMaterialMockCourseRepository) Create(ctx context.Context, course *domain.Course) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if course.ID == uuid.Nil {
		course.ID = uuid.New()
	}
	m.courses[course.ID.String()] = course
	return nil
}

func (m *testMaterialMockCourseRepository) FindByID(ctx context.Context, id string) (*domain.Course, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.courses[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (m *testMaterialMockCourseRepository) FindAllByUser(ctx context.Context, userID string) ([]domain.Course, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Course, 0)
	for _, c := range m.courses {
		if c.UserID == uid && c.DeletedAt == nil {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (m *testMaterialMockCourseRepository) UpdateAvatarStatus(ctx context.Context, courseID, status, avatarURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.courses[courseID]
	if !ok {
		return nil
	}
	c.AvatarStatus = status
	if avatarURL != "" {
		c.AvatarModelURL = avatarURL
	}
	return nil
}

type testMaterialMockTopicRepository struct {
	mu     sync.Mutex
	topics map[string]*domain.Topic
}

func (m *testMaterialMockTopicRepository) Create(ctx context.Context, topic *domain.Topic) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if topic.ID == uuid.Nil {
		topic.ID = uuid.New()
	}
	m.topics[topic.ID.String()] = topic
	return nil
}

func (m *testMaterialMockTopicRepository) FindByCourse(ctx context.Context, courseID string) ([]domain.Topic, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Topic, 0)
	for _, t := range m.topics {
		if t.CourseID.String() == courseID && t.DeletedAt == nil {
			out = append(out, *t)
		}
	}
	return out, nil
}

type testMaterialMockMaterialRepository struct {
	mu        sync.Mutex
	materials map[string]*domain.Material
}

func (m *testMaterialMockMaterialRepository) Create(ctx context.Context, material *domain.Material) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if material.ID == uuid.Nil {
		material.ID = uuid.New()
	}
	m.materials[material.ID.String()] = material
	return nil
}

func (m *testMaterialMockMaterialRepository) FindByID(ctx context.Context, id string) (*domain.Material, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mat, ok := m.materials[id]
	if !ok {
		return nil, nil
	}
	return mat, nil
}

func (m *testMaterialMockMaterialRepository) FindByTopic(ctx context.Context, topicID string) ([]domain.Material, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Material, 0)
	for _, mat := range m.materials {
		if mat.TopicID.String() == topicID && mat.DeletedAt == nil {
			out = append(out, *mat)
		}
	}
	return out, nil
}

func (m *testMaterialMockMaterialRepository) UpdateStatus(ctx context.Context, materialID string, status domain.MaterialStatus, extractedText string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if mat, ok := m.materials[materialID]; ok {
		mat.Status = status
		mat.ExtractedText = extractedText
	}
	return nil
}

type testMaterialMockExtractor struct{}

func (m *testMaterialMockExtractor) Extract(ctx context.Context, data []byte, formatType domain.MaterialFormatType) (string, error) {
	return string(data), nil
}

func setupMaterialHandler(materialUseCase *usecases.MaterialUseCase) *MaterialHandler {
	return NewMaterialHandler(materialUseCase)
}

func buildMultipartFromBytes(t *testing.T, fieldName, fileName string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("failed to create multipart part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write multipart content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func buildMultipartFromPath(t *testing.T, fieldName, filePath string) (*bytes.Buffer, string) {
	t.Helper()
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		t.Fatalf("failed to create multipart part: %v", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		t.Fatalf("failed to copy file into multipart: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func newMaterialTestDeps(t *testing.T) (*MaterialHandler, *testMaterialMockMaterialRepository, *domain.Course, *domain.Topic, string, string, string) {
	t.Helper()

	courseRepo := &testMaterialMockCourseRepository{courses: make(map[string]*domain.Course)}
	topicRepo := &testMaterialMockTopicRepository{topics: make(map[string]*domain.Topic)}
	materialRepo := &testMaterialMockMaterialRepository{materials: make(map[string]*domain.Material)}
	storagePath := t.TempDir()
	storage := repositories.NewLocalStorageService(storagePath)
	extractor := &testMaterialMockExtractor{}

	uc := usecases.NewMaterialUseCase(materialRepo, topicRepo, courseRepo, storage, extractor)
	handler := setupMaterialHandler(uc)

	ownerID := uuid.New().String()
	ownerUUID, _ := uuid.Parse(ownerID)
	course := &domain.Course{ID: uuid.New(), UserID: ownerUUID, Name: "Math 101", EducationLevel: "high school", AvatarStatus: "pending"}
	_ = courseRepo.Create(context.Background(), course)

	topic := &domain.Topic{ID: uuid.New(), CourseID: course.ID, Title: "Algebra"}
	_ = topicRepo.Create(context.Background(), topic)

	return handler, materialRepo, course, topic, ownerID, storagePath, uuid.New().String()
}

func TestMaterialHandler_UploadMaterial_WebBytes_PersistsToLocalStorage(t *testing.T) {
	handler, materialRepo, course, topic, ownerID, storagePath, _ := newMaterialTestDeps(t)

	body, contentType := buildMultipartFromBytes(t, "file", "notes.txt", []byte("hello from web bytes"))
	req := httptest.NewRequest(http.MethodPost, "/courses/"+course.ID.String()+"/topics/"+topic.ID.String()+"/materials", body)
	req.Header.Set("Content-Type", contentType)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", ownerID)
	c.Params = []gin.Param{{Key: "course_id", Value: course.ID.String()}, {Key: "topic_id", Value: topic.ID.String()}}
	c.Request = req

	handler.UploadMaterial(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	materials, err := materialRepo.FindByTopic(context.Background(), topic.ID.String())
	if err != nil {
		t.Fatalf("failed to query materials: %v", err)
	}
	if len(materials) != 1 {
		t.Fatalf("expected 1 material, got %d", len(materials))
	}

	parts := strings.SplitN(materials[0].StorageURL, "/static/", 2)
	if len(parts) != 2 {
		t.Fatalf("unexpected storage URL: %s", materials[0].StorageURL)
	}
	storedPath := filepath.Join(storagePath, filepath.FromSlash(parts[1]))
	storedBytes, readErr := os.ReadFile(storedPath)
	if readErr != nil {
		t.Fatalf("expected persisted file at %s: %v", storedPath, readErr)
	}
	if string(storedBytes) != "hello from web bytes" {
		t.Fatalf("unexpected persisted content: %q", string(storedBytes))
	}
}

func TestMaterialHandler_UploadMaterial_FilePathFlow_PersistsToLocalStorage(t *testing.T) {
	handler, materialRepo, course, topic, ownerID, storagePath, _ := newMaterialTestDeps(t)

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "guide.md")
	if err := os.WriteFile(sourcePath, []byte("# markdown from local path"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	body, contentType := buildMultipartFromPath(t, "file", sourcePath)
	req := httptest.NewRequest(http.MethodPost, "/courses/"+course.ID.String()+"/topics/"+topic.ID.String()+"/materials", body)
	req.Header.Set("Content-Type", contentType)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", ownerID)
	c.Params = []gin.Param{{Key: "course_id", Value: course.ID.String()}, {Key: "topic_id", Value: topic.ID.String()}}
	c.Request = req

	handler.UploadMaterial(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	materials, err := materialRepo.FindByTopic(context.Background(), topic.ID.String())
	if err != nil {
		t.Fatalf("failed to query materials: %v", err)
	}
	if len(materials) != 1 {
		t.Fatalf("expected 1 material, got %d", len(materials))
	}

	parts := strings.SplitN(materials[0].StorageURL, "/static/", 2)
	if len(parts) != 2 {
		t.Fatalf("unexpected storage URL: %s", materials[0].StorageURL)
	}
	storedPath := filepath.Join(storagePath, filepath.FromSlash(parts[1]))
	storedBytes, readErr := os.ReadFile(storedPath)
	if readErr != nil {
		t.Fatalf("expected persisted file at %s: %v", storedPath, readErr)
	}
	if string(storedBytes) != "# markdown from local path" {
		t.Fatalf("unexpected persisted content: %q", string(storedBytes))
	}
}

func TestMaterialHandler_UploadMaterial_OwnershipDenied_Returns403(t *testing.T) {
	handler, _, course, topic, _, _, otherUserID := newMaterialTestDeps(t)

	body, contentType := buildMultipartFromBytes(t, "file", "notes.txt", []byte("secret"))
	req := httptest.NewRequest(http.MethodPost, "/courses/"+course.ID.String()+"/topics/"+topic.ID.String()+"/materials", body)
	req.Header.Set("Content-Type", contentType)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", otherUserID)
	c.Params = []gin.Param{{Key: "course_id", Value: course.ID.String()}, {Key: "topic_id", Value: topic.ID.String()}}
	c.Request = req

	handler.UploadMaterial(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestMaterialHandler_ListMaterials_OwnershipDenied_Returns403(t *testing.T) {
	handler, _, course, topic, _, _, otherUserID := newMaterialTestDeps(t)

	req := httptest.NewRequest(http.MethodGet, "/courses/"+course.ID.String()+"/topics/"+topic.ID.String()+"/materials", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", otherUserID)
	c.Params = []gin.Param{{Key: "course_id", Value: course.ID.String()}, {Key: "topic_id", Value: topic.ID.String()}}
	c.Request = req

	handler.ListMaterials(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestMaterialHandler_ListMaterials_Success(t *testing.T) {
	handler, _, course, topic, ownerID, _, _ := newMaterialTestDeps(t)

	body, contentType := buildMultipartFromBytes(t, "file", "notes.txt", []byte("hello list"))
	uploadReq := httptest.NewRequest(http.MethodPost, "/courses/"+course.ID.String()+"/topics/"+topic.ID.String()+"/materials", body)
	uploadReq.Header.Set("Content-Type", contentType)
	uploadW := httptest.NewRecorder()
	uploadCtx, _ := gin.CreateTestContext(uploadW)
	uploadCtx.Set("user_id", ownerID)
	uploadCtx.Params = []gin.Param{{Key: "course_id", Value: course.ID.String()}, {Key: "topic_id", Value: topic.ID.String()}}
	uploadCtx.Request = uploadReq
	handler.UploadMaterial(uploadCtx)
	if uploadW.Code != http.StatusCreated {
		t.Fatalf("expected upload status %d, got %d", http.StatusCreated, uploadW.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/courses/"+course.ID.String()+"/topics/"+topic.ID.String()+"/materials", nil)
	listW := httptest.NewRecorder()
	listCtx, _ := gin.CreateTestContext(listW)
	listCtx.Set("user_id", ownerID)
	listCtx.Params = []gin.Param{{Key: "course_id", Value: course.ID.String()}, {Key: "topic_id", Value: topic.ID.String()}}
	listCtx.Request = listReq
	handler.ListMaterials(listCtx)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, listW.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(listW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if total, ok := resp["total"].(float64); !ok || int(total) != 1 {
		t.Fatalf("expected total=1, got %v", resp["total"])
	}
}
