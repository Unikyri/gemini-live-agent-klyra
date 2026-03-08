package http

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
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
// Optional middlewares (e.g. rate limiters) can be passed as extra arguments.
//
// Unified endpoint (Phase 5B+):
//   - POST /auth/login (Strategy Pattern, supports: google, guest)
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	rg.POST("/auth/login", append(middlewares, h.SignIn)...)
}

// signInRequest is the unified authentication request payload.
// Provider selects the strategy; payload fields are provider-specific.
type signInRequest struct {
	Provider string `json:"provider" binding:"required"`
	IDToken  string `json:"id_token,omitempty"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
}

// SignIn handles POST /api/v1/auth/login
// Example payloads:
// - {"provider":"google","id_token":"..."}
// - {"provider":"guest","email":"guest@example.com","name":"Guest User"}
func (h *AuthHandler) SignIn(c *gin.Context) {
	var req signInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}

	credentials := domain.AuthCredentials{}
	switch req.Provider {
	case "google":
		credentials["id_token"] = req.IDToken
	case "guest":
		credentials["email"] = req.Email
		credentials["name"] = req.Name
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported auth provider"})
		return
	}

	result, err := h.authUseCase.Login(c.Request.Context(), req.Provider, credentials)
	if err != nil {
		log.Printf("[Auth] SignIn failed (provider=%s): %v", req.Provider, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"user":          result.User,
		"provider":      result.Provider,
	})
}
