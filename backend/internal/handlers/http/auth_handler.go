package httphandlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

// AuthHandler handles HTTP requests for the authentication module.
// It only knows about use cases — never about the database or Google SDK directly.
type AuthHandler struct {
	authUseCase *usecases.AuthUseCase
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authUseCase *usecases.AuthUseCase) *AuthHandler {
	return &AuthHandler{authUseCase: authUseCase}
}

// RegisterRoutes attaches auth routes to the given Gin router group.
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/auth/google", h.GoogleSignIn)
}

// googleSignInRequest is the expected JSON body from the Flutter client.
type googleSignInRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

// GoogleSignIn handles POST /api/v1/auth/google
// Expected body: { "id_token": "<Google ID Token from Flutter>" }
// Returns: { "access_token": "...", "refresh_token": "...", "user": {...} }
func (h *AuthHandler) GoogleSignIn(c *gin.Context) {
	var req googleSignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Returns 400 if id_token is missing — "required" tag enforces this.
		c.JSON(http.StatusBadRequest, gin.H{"error": "id_token is required"})
		return
	}

	result, err := h.authUseCase.GoogleSignIn(c.Request.Context(), req.IDToken)
	if err != nil {
		// SECURITY: We log the real error server-side but never expose it to the client.
		// This prevents information leakage about our internal auth flow.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"user":          result.User,
	})
}
