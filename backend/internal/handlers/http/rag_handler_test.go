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

type MockRAGUseCase struct {
	mock.Mock
}

func (m *MockRAGUseCase) ProcessMaterialChunks(req *usecases.ProcessMaterialChunksRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockRAGUseCase) GetTopicContext(query string, topicID uuid.UUID, limit int) ([]*domain.MaterialChunk, error) {
	args := m.Called(query, topicID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.MaterialChunk), args.Error(1)
}

func setupRAGRouter(mockUC *MockRAGUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewRAGHandler(mockUC)
	router.POST("/topics/:topicId/process", handler.ProcessMaterial)
	router.POST("/topics/:topicId/query", handler.QueryContext)
	return router
}

func TestProcessMaterial_Success(t *testing.T) {
	mockUC := new(MockRAGUseCase)
	topicID := uuid.New()
	materialID := uuid.New()

	mockUC.On("ProcessMaterialChunks", mock.MatchedBy(func(req *usecases.ProcessMaterialChunksRequest) bool {
		return req.MaterialID == materialID && req.TopicID == topicID
	})).Return(nil)

	router := setupRAGRouter(mockUC)

	body := map[string]interface{}{
		"material_id": materialID.String(),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/topics/"+topicID.String()+"/process", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockUC.AssertExpectations(t)
}

func TestQueryContext_Success(t *testing.T) {
	mockUC := new(MockRAGUseCase)
	topicID := uuid.New()
	query := "What is neural plasticity?"

	expectedChunks := []*domain.MaterialChunk{
		{
			ID:       uuid.New(),
			TopicID:  topicID,
			Content:  "Neural plasticity is the brain's ability to reorganize itself...",
			Index:    0,
		},
	}

	mockUC.On("GetTopicContext", query, topicID, mock.MatchedBy(func(limit int) bool {
		return limit > 0
	})).Return(expectedChunks, nil)

	router := setupRAGRouter(mockUC)

	body := map[string]interface{}{
		"query": query,
		"limit": 5,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/topics/"+topicID.String()+"/query", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	contextData, ok := responseBody["context"]
	assert.True(t, ok, "Response should contain context field")
	assert.NotNil(t, contextData)
	mockUC.AssertExpectations(t)
}
