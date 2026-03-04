package usecases

import (
	"context"
	"errors"
	"log"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
)

// AuthUseCase handles authentication-related business logic.
// It only depends on interfaces (ports), never on concrete implementations.
type AuthUseCase struct {
	userRepo       ports.UserRepository
	tokenSvc       ports.TokenService
	googleVerifier ports.GoogleTokenVerifier
}

// NewAuthUseCase creates and returns an AuthUseCase instance.
// Dependencies are injected from outside (Dependency Injection).
func NewAuthUseCase(
	userRepo ports.UserRepository,
	tokenSvc ports.TokenService,
	googleVerifier ports.GoogleTokenVerifier,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:       userRepo,
		tokenSvc:       tokenSvc,
		googleVerifier: googleVerifier,
	}
}

// AuthResult holds the tokens returned to the client after successful login.
type AuthResult struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         *domain.User `json:"user"`
}

// GoogleSignIn validates a Google ID Token, then finds or creates the user in our system.
// This implements the "find or create" (upsert-like) pattern common in OAuth2 flows.
func (uc *AuthUseCase) GoogleSignIn(ctx context.Context, googleIDToken string) (*AuthResult, error) {
	// Step 1: Verify the token with Google's servers.
	email, name, picture, err := uc.googleVerifier.Verify(ctx, googleIDToken)
	if err != nil {
		return nil, errors.New("invalid Google ID token")
	}

	// Step 2: Find or create the user in our database.
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// User not found — create a new one. This is a first-time sign-in.
		log.Printf("New user signing in: %s. Creating record.", email)
		user = &domain.User{
			Email:           email,
			Name:            name,
			ProfileImageURL: picture,
		}
		if createErr := uc.userRepo.Create(ctx, user); createErr != nil {
			return nil, createErr
		}
	}

	// Step 3: Generate our own short-lived Access Token and Refresh Token.
	accessToken, err := uc.tokenSvc.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := uc.tokenSvc.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}
