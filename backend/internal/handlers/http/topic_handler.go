package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

// TopicHandler exposes readiness, summary, and CRUD endpoints for topics.
type TopicHandler struct {
	topicUseCase  *usecases.TopicUseCase
	courseUseCase *usecases.CourseUseCase
}

// NewTopicHandler creates a TopicHandler.
func NewTopicHandler(topicUseCase *usecases.TopicUseCase) *TopicHandler {
	return &TopicHandler{topicUseCase: topicUseCase}
}

// NewTopicHandlerWithCourseUseCase creates a TopicHandler with course use case for Update/Delete topic.
func NewTopicHandlerWithCourseUseCase(topicUseCase *usecases.TopicUseCase, courseUseCase *usecases.CourseUseCase) *TopicHandler {
	return &TopicHandler{topicUseCase: topicUseCase, courseUseCase: courseUseCase}
}

// RegisterRoutes attaches topic readiness, summary and CRUD routes to protected router.
func (h *TopicHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/courses/:course_id/topics/:topic_id/readiness", h.GetReadiness)
	rg.GET("/courses/:course_id/topics/:topic_id/summary", h.GetSummary)
	rg.PATCH("/courses/:course_id/topics/:topic_id", h.UpdateTopic)
	rg.DELETE("/courses/:course_id/topics/:topic_id", h.DeleteTopic)
}

// GetReadiness returns strict readiness state for tutor entry.
func (h *TopicHandler) GetReadiness(c *gin.Context) {
	if _, exists := c.Get("user_id"); !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	topicID := c.Param("topic_id")
	readiness, err := h.topicUseCase.CheckReadiness(c.Request.Context(), topicID)
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
}

// GetSummary returns cached-or-regenerated markdown summary for the topic.
func (h *TopicHandler) GetSummary(c *gin.Context) {
	if _, exists := c.Get("user_id"); !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	topicID := c.Param("topic_id")
	summary, err := h.topicUseCase.GenerateSummary(c.Request.Context(), topicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate topic summary"})
		return
	}
	if summary == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "topic not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"topic_id":     topicID,
		"summary":      summary.SummaryMarkdown,
		"material_ids": summary.MaterialIDs,
		"from_cache":   summary.FromCache,
	})
}

// UpdateTopic handles PATCH /api/v1/courses/:course_id/topics/:topic_id
func (h *TopicHandler) UpdateTopic(c *gin.Context) {
	if h.courseUseCase == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "not configured"})
		return
	}
	userID, _ := c.Get("user_id")
	courseID := c.Param("course_id")
	topicID := c.Param("topic_id")

	var body struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	topic, err := h.courseUseCase.UpdateTopic(c.Request.Context(), courseID, topicID, userID.(string), usecases.UpdateTopicInput{Title: body.Title})
	if errors.Is(err, usecases.ErrCourseForbidden) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update topic"})
		return
	}
	if topic == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "topic not found"})
		return
	}
	c.JSON(http.StatusOK, topic)
}

// DeleteTopic handles DELETE /api/v1/courses/:course_id/topics/:topic_id
func (h *TopicHandler) DeleteTopic(c *gin.Context) {
	if h.courseUseCase == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "not configured"})
		return
	}
	userID, _ := c.Get("user_id")
	courseID := c.Param("course_id")
	topicID := c.Param("topic_id")

	err := h.courseUseCase.DeleteTopic(c.Request.Context(), courseID, topicID, userID.(string))
	if errors.Is(err, usecases.ErrCourseForbidden) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete topic"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "topic deleted"})
}
