package repositories

import (
	"context"
	"errors"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
)

// GoogleAuthStrategy validates Google credentials and extracts user identity.
type GoogleAuthStrategy struct {
	verifier ports.GoogleTokenVerifier
}

func NewGoogleAuthStrategy(verifier ports.GoogleTokenVerifier) *GoogleAuthStrategy {
	return &GoogleAuthStrategy{verifier: verifier}
}

func (s *GoogleAuthStrategy) Authenticate(ctx context.Context, credentials domain.AuthCredentials) (*domain.AuthResult, error) {
	if s.verifier == nil {
		return nil, errors.New("google verifier is not configured")
	}

	idToken := credentials.GetString("id_token")
	if idToken == "" {
		return nil, errors.New("id_token is required")
	}

	email, name, picture, err := s.verifier.Verify(ctx, idToken)
	if err != nil {
		return nil, errors.New("invalid Google ID token")
	}

	return &domain.AuthResult{
		User: &domain.User{
			Email:           email,
			Name:            name,
			ProfileImageURL: picture,
		},
		Provider: "google",
	}, nil
}
