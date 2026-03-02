package ports

import (
	"context"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// UserRepository defines the contract for user persistence.
// The use cases know ONLY this interface, never the concrete DB implementation.
// This is the cornerstone of Clean Architecture: the domain dictates the contract.
type UserRepository interface {
	// FindByEmail retrieves a user by their email address.
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	// Create persists a new user to the store.
	Create(ctx context.Context, user *domain.User) error
	// FindByID retrieves a user by their UUID.
	FindByID(ctx context.Context, id string) (*domain.User, error)
}

// TokenService defines the contract for JWT operations.
type TokenService interface {
	// GenerateAccessToken creates a short-lived JWT for a given user.
	GenerateAccessToken(user *domain.User) (string, error)
	// GenerateRefreshToken creates a long-lived opaque refresh token.
	GenerateRefreshToken(userID string) (string, error)
	// ValidateAccessToken verifies a JWT and returns the claims.
	ValidateAccessToken(tokenString string) (map[string]interface{}, error)
}

// GoogleTokenVerifier defines the contract for verifying Google ID Tokens.
type GoogleTokenVerifier interface {
	// Verify validates a Google ID Token and returns the user's email and name.
	Verify(ctx context.Context, idToken string) (email, name, picture string, err error)
}
