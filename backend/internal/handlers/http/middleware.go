package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
)

// AuthMiddleware validates the Bearer JWT token on protected routes.
// This is the enforcement layer for authorization — if the token is invalid
// or missing, the request is rejected before reaching any business logic.
func AuthMiddleware(tokenSvc ports.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed Authorization header"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := tokenSvc.ValidateAccessToken(tokenString)
		if err != nil {
			// SECURITY: Never expose the specific error (e.g., "expired") to prevent
			// attackers from learning about our token structure.
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Store the user_id in the context for downstream handlers to use.
		// This is the only source of truth for the authenticated user's identity.
		c.Set("user_id", claims["sub"])
		c.Set("user_email", claims["email"])
		c.Next()
	}
}
