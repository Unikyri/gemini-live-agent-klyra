package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

// TopicHandler exposes readiness and summary endpoints for tutoring flow gating.
type TopicHandler struct {
	topicUseCase *usecases.TopicUseCase
}

// NewTopicHandler creates a TopicHandler.
func NewTopicHandler(topicUseCase *usecases.TopicUseCase) *TopicHandler {
	return &TopicHandler{topicUseCase: topicUseCase}
}

// RegisterRoutes attaches topic readiness and summary routes to protected router.
func (h *TopicHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/courses/:course_id/topics/:topic_id/readiness", h.GetReadiness)
	rg.GET("/courses/:course_id/topics/:topic_id/summary", h.GetSummary)
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
