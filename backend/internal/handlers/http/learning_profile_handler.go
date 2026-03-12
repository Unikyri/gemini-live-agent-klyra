package http

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

type LearningProfileHandler struct {
	uc *usecases.LearningProfileUseCase
}

func NewLearningProfileHandler(uc *usecases.LearningProfileUseCase) *LearningProfileHandler {
	return &LearningProfileHandler{uc: uc}
}

func (h *LearningProfileHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/users/me/learning-profile", h.GetMyProfile)
	rg.POST("/users/me/learning-profile/update", h.UpdateMyProfile)
}

func (h *LearningProfileHandler) GetMyProfile(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(string)

	profile, err := h.uc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load learning profile"})
		return
	}
	if profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"learning_profile": profile})
}

type updateLearningProfileRequest struct {
	RecentMessages []string `json:"recent_messages"`
}

func (h *LearningProfileHandler) UpdateMyProfile(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(string)

	// Feature flag: FF_LEARNING_PROFILE=false disables LLM/updates; we still
	// accept the request to avoid breaking clients, but no-op with a message.
	if strings.EqualFold(os.Getenv("FF_LEARNING_PROFILE"), "true") == false {
		c.JSON(http.StatusAccepted, gin.H{"status": "skipped", "message": "learning profile disabled"})
		return
	}

	var req updateLearningProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	profile, err := h.uc.UpdateProfile(c.Request.Context(), userID, req.RecentMessages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update learning profile"})
		return
	}
	if profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"learning_profile": profile})
}

