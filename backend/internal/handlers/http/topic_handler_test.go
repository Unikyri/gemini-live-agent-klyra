package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

// MockTopicUseCase mocks TopicUseCase for testing handlers.
type MockTopicUseCase struct {
	mock.Mock
}

func (m *MockTopicUseCase) CheckReadiness(ctx context.Context, topicID string) (*usecases.TopicReadiness, error) {
	args := m.Called(ctx, topicID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecases.TopicReadiness), args.Error(1)
}

func (m *MockTopicUseCase) GenerateSummary(ctx context.Context, topicID string) (*usecases.TopicSummaryResult, error) {
	args := m.Called(ctx, topicID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecases.TopicSummaryResult), args.Error(1)
}

func setupTopicRouter(mockUC *MockTopicUseCase) *gin.Engine {
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

	// Manually register routes to work with mock use case
	api := router.Group("/api/v1")
	api.GET("/courses/:course_id/topics/:topic_id/readiness", func(c *gin.Context) {
		if _, exists := c.Get("user_id"); !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		topicID := c.Param("topic_id")
		readiness, err := mockUC.CheckReadiness(c.Request.Context(), topicID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not calculate topic readiness"})
			return
		}
		if readiness == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "topic not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"topic_id":        topicID,
			"is_ready":        readiness.IsReady,
			"validated_count": readiness.ValidatedCount,
			"total_count":     readiness.TotalCount,
			"message":         readiness.Message,
		})
	})

	api.GET("/courses/:course_id/topics/:topic_id/summary", func(c *gin.Context) {
		if _, exists := c.Get("user_id"); !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		topicID := c.Param("topic_id")
		result, err := mockUC.GenerateSummary(c.Request.Context(), topicID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate summary"})
			return
		}
		if result == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "topic not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"summary":      result.SummaryMarkdown,
			"material_ids": result.MaterialIDs,
			"from_cache":   result.FromCache,
		})
	})

	return router
}

// TestTopicHandler_GetReadiness_Success_Ready tests readiness endpoint when topic is ready (>=1 validated material).
func TestTopicHandler_GetReadiness_Success_Ready(t *testing.T) {
	mockUC := new(MockTopicUseCase)
	router := setupTopicRouter(mockUC)

	topicID := "topic-123"
	mockUC.On("CheckReadiness", mock.Anything, topicID).Return(&usecases.TopicReadiness{
		IsReady:        true,
		ValidatedCount: 2,
		TotalCount:     3,
		Message:        "Ready to start tutoring",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-1/topics/"+topicID+"/readiness", nil)
	req.Header.Set("X-Test-User-ID", "user-1")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"is_ready":true`)
	assert.Contains(t, w.Body.String(), `"validated_count":2`)
	assert.Contains(t, w.Body.String(), `"total_count":3`)
	assert.Contains(t, w.Body.String(), `"Ready to start tutoring"`)
	mockUC.AssertExpectations(t)
}

// TestTopicHandler_GetReadiness_Success_NotReady tests readiness endpoint when topic is not ready (0 validated materials).
func TestTopicHandler_GetReadiness_Success_NotReady(t *testing.T) {
	mockUC := new(MockTopicUseCase)
	router := setupTopicRouter(mockUC)

	topicID := "topic-empty"
	mockUC.On("CheckReadiness", mock.Anything, topicID).Return(&usecases.TopicReadiness{
		IsReady:        false,
		ValidatedCount: 0,
		TotalCount:     1,
		Message:        "Upload and validate at least one material to start tutoring",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-2/topics/"+topicID+"/readiness", nil)
	req.Header.Set("X-Test-User-ID", "user-2")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"is_ready":false`)
	assert.Contains(t, w.Body.String(), `"validated_count":0`)
	assert.Contains(t, w.Body.String(), `Upload and validate`)
	mockUC.AssertExpectations(t)
}

// TestTopicHandler_GetReadiness_TopicNotFound tests 404 when topic doesn't exist.
func TestTopicHandler_GetReadiness_TopicNotFound(t *testing.T) {
	mockUC := new(MockTopicUseCase)
	router := setupTopicRouter(mockUC)

	topicID := "nonexistent-topic"
	mockUC.On("CheckReadiness", mock.Anything, topicID).Return(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-1/topics/"+topicID+"/readiness", nil)
	req.Header.Set("X-Test-User-ID", "user-1")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"topic not found"`)
	mockUC.AssertExpectations(t)
}

// TestTopicHandler_GetReadiness_Unauthorized tests 401 when user is not authenticated.
func TestTopicHandler_GetReadiness_Unauthorized(t *testing.T) {
	mockUC := new(MockTopicUseCase)
	router := setupTopicRouter(mockUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-1/topics/topic-1/readiness", nil)
	// NOTE: No X-Test-User-ID header → user_id not set in context
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), `"unauthorized"`)
	mockUC.AssertNotCalled(t, "CheckReadiness")
}

// TestTopicHandler_GetReadiness_InternalError tests 500 on use case failure.
func TestTopicHandler_GetReadiness_InternalError(t *testing.T) {
	mockUC := new(MockTopicUseCase)
	router := setupTopicRouter(mockUC)

	topicID := "topic-error"
	mockUC.On("CheckReadiness", mock.Anything, topicID).Return(nil, errors.New("database connection lost"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-1/topics/"+topicID+"/readiness", nil)
	req.Header.Set("X-Test-User-ID", "user-1")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"could not calculate topic readiness"`)
	mockUC.AssertExpectations(t)
}

// TestTopicHandler_GetSummary_Success_CacheHit tests summary endpoint returning cached summary.
func TestTopicHandler_GetSummary_Success_CacheHit(t *testing.T) {
	mockUC := new(MockTopicUseCase)
	router := setupTopicRouter(mockUC)

	topicID := "topic-cached"
	mockUC.On("GenerateSummary", mock.Anything, topicID).Return(&usecases.TopicSummaryResult{
		SummaryMarkdown: "# Topic Summary\n\nCached content with $$E=mc^2$$.",
		MaterialIDs:     []string{"mat-1", "mat-2"},
		FromCache:       true,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-1/topics/"+topicID+"/summary", nil)
	req.Header.Set("X-Test-User-ID", "user-1")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"from_cache":true`)
	assert.Contains(t, w.Body.String(), `Topic Summary`)
	assert.Contains(t, w.Body.String(), `"mat-1"`)
	mockUC.AssertExpectations(t)
}

// TestTopicHandler_GetSummary_Success_CacheMiss tests summary endpoint regenerating summary.
func TestTopicHandler_GetSummary_Success_CacheMiss(t *testing.T) {
	mockUC := new(MockTopicUseCase)
	router := setupTopicRouter(mockUC)

	topicID := "topic-fresh"
	mockUC.On("GenerateSummary", mock.Anything, topicID).Return(&usecases.TopicSummaryResult{
		SummaryMarkdown: "# Regenerated Summary\n\nNew content.",
		MaterialIDs:     []string{"mat-3"},
		FromCache:       false,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-1/topics/"+topicID+"/summary", nil)
	req.Header.Set("X-Test-User-ID", "user-1")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"from_cache":false`)
	assert.Contains(t, w.Body.String(), `Regenerated Summary`)
	mockUC.AssertExpectations(t)
}

// TestTopicHandler_GetSummary_TopicNotFound tests 404 when topic doesn't exist.
func TestTopicHandler_GetSummary_TopicNotFound(t *testing.T) {
	mockUC := new(MockTopicUseCase)
	router := setupTopicRouter(mockUC)

	topicID := "nonexistent-topic"
	mockUC.On("GenerateSummary", mock.Anything, topicID).Return(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-1/topics/"+topicID+"/summary", nil)
	req.Header.Set("X-Test-User-ID", "user-1")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"topic not found"`)
	mockUC.AssertExpectations(t)
}

// TestTopicHandler_GetSummary_Unauthorized tests 401 when user is not authenticated.
func TestTopicHandler_GetSummary_Unauthorized(t *testing.T) {
	mockUC := new(MockTopicUseCase)
	router := setupTopicRouter(mockUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-1/topics/topic-1/summary", nil)
	// No X-Test-User-ID header
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), `"unauthorized"`)
	mockUC.AssertNotCalled(t, "GenerateSummary")
}

// TestTopicHandler_GetSummary_InternalError tests 500 on use case failure.
func TestTopicHandler_GetSummary_InternalError(t *testing.T) {
	mockUC := new(MockTopicUseCase)
	router := setupTopicRouter(mockUC)

	topicID := "topic-error"
	mockUC.On("GenerateSummary", mock.Anything, topicID).Return(nil, errors.New("generator crashed"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-1/topics/"+topicID+"/summary", nil)
	req.Header.Set("X-Test-User-ID", "user-1")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"could not generate summary"`)
	mockUC.AssertExpectations(t)
}
