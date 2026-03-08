package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

type MockMaterialUseCase struct {
	mock.Mock
}

func (m *MockMaterialUseCase) CreateMaterial(req *usecases.CreateMaterialRequest) (*domain.Material, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Material), args.Error(1)
}

func (m *MockMaterialUseCase) GetMaterialsByTopic(topicID uuid.UUID) ([]*domain.Material, error) {
	args := m.Called(topicID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Material), args.Error(1)
}

func (m *MockMaterialUseCase) GetMaterialByID(materialID uuid.UUID) (*domain.Material, error) {
	args := m.Called(materialID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Material), args.Error(1)
}

func setupMaterialRouter(mockUC *MockMaterialUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewMaterialHandler(mockUC)
	router.POST("/materials", handler.CreateMaterial)
	router.GET("/materials/:id", handler.GetMaterialByID)
	router.GET("/topics/:topicId/materials", handler.GetMaterialsByTopic)
	return router
}

func TestCreateMaterial_Success(t *testing.T) {
	mockUC := new(MockMaterialUseCase)
	materialID := uuid.New()
	topicID := uuid.New()

	expectedMaterial := &domain.Material{
		ID:       materialID,
		TopicID:  topicID,
		Title:    "Neuroscience Overview",
		Content:  "Chapter 1: The Brain System",
		FileType: "pdf",
	}

	mockUC.On("CreateMaterial", mock.MatchedBy(func(req *usecases.CreateMaterialRequest) bool {
		return req.Title == "Neuroscience Overview"
	})).Return(expectedMaterial, nil)

	router := setupMaterialRouter(mockUC)

	body := map[string]interface{}{
		"title":    "Neuroscience Overview",
		"topic_id": topicID.String(),
		"content":  "Chapter 1: The Brain System",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/materials", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var responseBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, expectedMaterial.Title, responseBody["title"])
	mockUC.AssertExpectations(t)
}

func TestGetMaterialByID_Success(t *testing.T) {
	mockUC := new(MockMaterialUseCase)
	materialID := uuid.New()

	expectedMaterial := &domain.Material{
		ID:      materialID,
		Title:   "Neural Networks",
		Content: "Deep learning fundamentals",
	}

	mockUC.On("GetMaterialByID", materialID).Return(expectedMaterial, nil)

	router := setupMaterialRouter(mockUC)
	req := httptest.NewRequest("GET", "/materials/"+materialID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, expectedMaterial.Title, responseBody["title"])
	mockUC.AssertExpectations(t)
}

func TestGetMaterialsByTopic_Success(t *testing.T) {
	mockUC := new(MockMaterialUseCase)
	topicID := uuid.New()

	expectedMaterials := []*domain.Material{
		{
			ID:       uuid.New(),
			TopicID:  topicID,
			Title:    "Biology Basics",
			FileType: "pdf",
		},
		{
			ID:       uuid.New(),
			TopicID:  topicID,
			Title:    "Advanced Biology",
			FileType: "docx",
		},
	}

	mockUC.On("GetMaterialsByTopic", topicID).Return(expectedMaterials, nil)

	router := setupMaterialRouter(mockUC)
	req := httptest.NewRequest("GET", "/topics/"+topicID.String()+"/materials", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseBody []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, 2, len(responseBody))
	mockUC.AssertExpectations(t)
}
