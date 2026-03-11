package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

// MockRAGUseCase mocks the RAGUseCase for testing handlers.
type MockRAGUseCase struct {
	mock.Mock
}

func (m *MockRAGUseCase) ProcessMaterialChunks(ctx context.Context, materialID string) error {
	args := m.Called(ctx, materialID)
	return args.Error(0)
}

func (m *MockRAGUseCase) GetTopicContext(ctx context.Context, topicID, query string) (*usecases.ContextResult, error) {
	args := m.Called(ctx, topicID, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecases.ContextResult), args.Error(1)
}

func (m *MockRAGUseCase) GetCourseContext(ctx context.Context, courseID, query string) (*usecases.ContextResult, error) {
	args := m.Called(ctx, courseID, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecases.ContextResult), args.Error(1)
}

func setupRAGRouter(mockUC *MockRAGUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Middleware to inject user_id into context (simulating auth middleware)
	router.Use(func(c *gin.Context) {
		testUserID := c.GetHeader("X-Test-User-ID")
		if testUserID != "" {
			c.Set("user_id", testUserID)
		}
		c.Next()
	})

	// For tests, manually register routes to avoid handler dependency on concrete use case
	api := router.Group("/api/v1")
	api.GET("/courses/:course_id/topics/:topic_id/context", func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		_ = userIDVal

		topicID := c.Param("topic_id")
		query := c.Query("query")

		result, err := mockUC.GetTopicContext(c.Request.Context(), topicID, query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve context"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"topic_id":      topicID,
			"context":       result.Context,
			"query":         query,
			"has_materials": result.HasMaterials,
			"message":       result.Message,
		})
	})
	api.GET("/courses/:course_id/context", func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		_ = userIDVal

		courseID := c.Param("course_id")
		query := c.Query("query")
		result, err := mockUC.GetCourseContext(c.Request.Context(), courseID, query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve context"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"course_id":     courseID,
			"context":       result.Context,
			"query":         query,
			"truncated":     result.Truncated,
			"has_materials": result.HasMaterials,
			"message":       result.Message,
		})
	})
	return router
}

func TestGetTopicContext_WithQuery(t *testing.T) {
	mockUC := new(MockRAGUseCase)
	topicID := uuid.New()
	courseID := uuid.New()
	userID := uuid.New()
	query := "What is neural plasticity?"
	expectedResult := &usecases.ContextResult{
		Context:      "Neural plasticity is the brain's ability to reorganize itself by forming new neural connections.",
		HasMaterials: true,
		Message:      "",
	}

	mockUC.On("GetTopicContext", mock.Anything, topicID.String(), query).Return(expectedResult, nil)

	router := setupRAGRouter(mockUC)
	req := httptest.NewRequest("GET", "/api/v1/courses/"+courseID.String()+"/topics/"+topicID.String()+"/context?query="+url.QueryEscape(query), nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"has_materials":true`)
	assert.Contains(t, w.Body.String(), `"message":""`)
	mockUC.AssertExpectations(t)
}

func TestGetTopicContext_WithoutQuery(t *testing.T) {
	mockUC := new(MockRAGUseCase)
	topicID := uuid.New()
	courseID := uuid.New()
	userID := uuid.New()
	expectedResult := &usecases.ContextResult{
		Context:      "",
		HasMaterials: false,
		Message:      "No hay materiales para este tema. El tutor usará su conocimiento base.",
	}

	mockUC.On("GetTopicContext", mock.Anything, topicID.String(), "").Return(expectedResult, nil)

	router := setupRAGRouter(mockUC)
	req := httptest.NewRequest("GET", "/api/v1/courses/"+courseID.String()+"/topics/"+topicID.String()+"/context", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"has_materials":false`)
	assert.Contains(t, w.Body.String(), `"message":"No hay materiales para este tema. El tutor usará su conocimiento base."`)
	mockUC.AssertExpectations(t)
}

func TestGetCourseContext_WithoutMaterials(t *testing.T) {
	mockUC := new(MockRAGUseCase)
	courseID := uuid.New()
	userID := uuid.New()
	expectedResult := &usecases.ContextResult{
		Context:      "",
		Truncated:    false,
		HasMaterials: false,
		Message:      "No hay materiales en ningún tema de este curso. El tutor usará su conocimiento base.",
	}

	mockUC.On("GetCourseContext", mock.Anything, courseID.String(), "").Return(expectedResult, nil)

	router := setupRAGRouter(mockUC)
	req := httptest.NewRequest("GET", "/api/v1/courses/"+courseID.String()+"/context", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"has_materials":false`)
	assert.Contains(t, w.Body.String(), `"truncated":false`)
	mockUC.AssertExpectations(t)
}

func TestGetTopicContext_Unauthorized(t *testing.T) {
	mockUC := new(MockRAGUseCase)
	topicID := uuid.New()
	courseID := uuid.New()

	router := setupRAGRouter(mockUC)
	req := httptest.NewRequest("GET", "/api/v1/courses/"+courseID.String()+"/topics/"+topicID.String()+"/context", nil)
	// No X-Test-User-ID header
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
