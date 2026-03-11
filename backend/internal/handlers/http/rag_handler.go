package http

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

// RAGHandler exposes a context-retrieval endpoint used by the mobile Tutor session.
type RAGHandler struct {
	ragUseCase    *usecases.RAGUseCase
	courseUseCase *usecases.CourseUseCase
}

// NewRAGHandler creates a RAGHandler.
func NewRAGHandler(ragUseCase *usecases.RAGUseCase) *RAGHandler {
	return &RAGHandler{ragUseCase: ragUseCase}
}

// NewRAGHandlerWithCourseUseCase creates a RAGHandler with course use case for ownership validation.
func NewRAGHandlerWithCourseUseCase(ragUseCase *usecases.RAGUseCase, courseUseCase *usecases.CourseUseCase) *RAGHandler {
	return &RAGHandler{ragUseCase: ragUseCase, courseUseCase: courseUseCase}
}

// RegisterRoutes attaches RAG routes to the protected router group.
func (h *RAGHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// GET context for a full topic (no query) or a query-specific retrieval
	rg.GET("/courses/:course_id/topics/:topic_id/context", h.GetTopicContext)
	rg.GET("/courses/:course_id/context", h.GetCourseContext)
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
	userID := userIDVal.(string)

	courseID := c.Param("course_id")
	topicID := c.Param("topic_id")
	query := c.Query("query") // Optional: if empty, returns full topic context

	// Validate course ownership/existence and ensure topic belongs to course and is not deleted.
	if h.courseUseCase != nil {
		course, err := h.courseUseCase.GetCourseByID(c.Request.Context(), courseID, userID)
		if err != nil {
			if err == usecases.ErrCourseForbidden {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
			log.Printf("[RAG] GetTopicContext GetCourseByID error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve context"})
			return
		}
		if course == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
			return
		}

		found := false
		for _, t := range course.Topics {
			if strings.EqualFold(t.ID.String(), topicID) && t.DeletedAt == nil {
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "topic not found"})
			return
		}
	}

	result, err := h.ragUseCase.GetTopicContext(c.Request.Context(), topicID, query)
	if err != nil {
		log.Printf("[RAG] GetTopicContext error for topic %s: %v", topicID, err)
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
}

// GetCourseContext handles GET /api/v1/courses/:course_id/context
// Optional query param: ?query=<text> for similarity search. Without query returns truncated course context.
// Validates course ownership before returning context.
func (h *RAGHandler) GetCourseContext(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(string)
	courseID := c.Param("course_id")
	query := c.Query("query")

	if h.courseUseCase == nil {
		log.Printf("[RAG] GetCourseContext: course use case not set")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve context"})
		return
	}

	course, err := h.courseUseCase.GetCourseByID(c.Request.Context(), courseID, userID)
	if err != nil {
		if err == usecases.ErrCourseForbidden {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		log.Printf("[RAG] GetCourseContext GetCourseByID error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve context"})
		return
	}
	if course == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}

	result, err := h.ragUseCase.GetCourseContext(c.Request.Context(), courseID, query)
	if err != nil {
		log.Printf("[RAG] GetCourseContext error for course %s: %v", courseID, err)
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
}
