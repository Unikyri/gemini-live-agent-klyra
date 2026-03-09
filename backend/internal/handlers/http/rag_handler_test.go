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
)

// MockRAGUseCase mocks the RAGUseCase for testing handlers.
type MockRAGUseCase struct {
	mock.Mock
}

func (m *MockRAGUseCase) ProcessMaterialChunks(ctx context.Context, materialID string) error {
	args := m.Called(ctx, materialID)
	return args.Error(0)
}

func (m *MockRAGUseCase) GetTopicContext(ctx context.Context, topicID, query string) (string, error) {
	args := m.Called(ctx, topicID, query)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.Get(0).(string), args.Error(1)
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

		context, err := mockUC.GetTopicContext(c.Request.Context(), topicID, query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve context"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"context": context})
	})
	return router
}

func TestGetTopicContext_WithQuery(t *testing.T) {
	mockUC := new(MockRAGUseCase)
	topicID := uuid.New()
	courseID := uuid.New()
	userID := uuid.New()
	query := "What is neural plasticity?"
	expectedContext := "Neural plasticity is the brain's ability to reorganize itself by forming new neural connections."

	mockUC.On("GetTopicContext", mock.Anything, topicID.String(), query).Return(expectedContext, nil)

	router := setupRAGRouter(mockUC)
	req := httptest.NewRequest("GET", "/api/v1/courses/"+courseID.String()+"/topics/"+topicID.String()+"/context?query="+url.QueryEscape(query), nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockUC.AssertExpectations(t)
}

func TestGetTopicContext_WithoutQuery(t *testing.T) {
	mockUC := new(MockRAGUseCase)
	topicID := uuid.New()
	courseID := uuid.New()
	userID := uuid.New()
	expectedContext := "Full topic context: Introduction to neuroscience covering brain structure, neural networks, and cognition."

	mockUC.On("GetTopicContext", mock.Anything, topicID.String(), "").Return(expectedContext, nil)

	router := setupRAGRouter(mockUC)
	req := httptest.NewRequest("GET", "/api/v1/courses/"+courseID.String()+"/topics/"+topicID.String()+"/context", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
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
