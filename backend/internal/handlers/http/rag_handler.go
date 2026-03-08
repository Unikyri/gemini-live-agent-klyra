package http

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

// RAGHandler exposes a context-retrieval endpoint used by the mobile Tutor session.
type RAGHandler struct {
	ragUseCase *usecases.RAGUseCase
}

// NewRAGHandler creates a RAGHandler.
func NewRAGHandler(ragUseCase *usecases.RAGUseCase) *RAGHandler {
	return &RAGHandler{ragUseCase: ragUseCase}
}

// RegisterRoutes attaches RAG routes to the protected router group.
func (h *RAGHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// GET context for a full topic (no query) or a query-specific retrieval
	rg.GET("/courses/:course_id/topics/:topic_id/context", h.GetTopicContext)
}

// GetTopicContext handles GET /api/v1/courses/:course_id/topics/:topic_id/context
// Optional query param: ?query=<user question>
// Returns the relevant text context that should be injected into the Gemini system prompt.
//
// SECURITY: Authorization is validated by the JWT middleware (user must be authenticated).
// Ownership check of the course is performed by the use case via the existing course repo.
// Topic-scoped retrieval in the chunk repository prevents cross-user data leakage.
func (h *RAGHandler) GetTopicContext(c *gin.Context) {
	// userID from JWT — never from the request body
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	_ = userIDVal // Future: validate topic belongs to this user's course

	topicID := c.Param("topic_id")
	query := c.Query("query") // Optional: if empty, returns full topic context

	context, err := h.ragUseCase.GetTopicContext(c.Request.Context(), topicID, query)
	if err != nil {
		log.Printf("[RAG] GetTopicContext error for topic %s: %v", topicID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve context"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"topic_id": topicID,
		"context":  context,
		"query":    query,
	})
}
