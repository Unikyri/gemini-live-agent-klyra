package repositories

import (
	"context"
	"errors"
	"strings"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// GuestAuthStrategy validates guest credentials for development login.
type GuestAuthStrategy struct{}

func NewGuestAuthStrategy() *GuestAuthStrategy {
	return &GuestAuthStrategy{}
}

func (s *GuestAuthStrategy) Authenticate(ctx context.Context, credentials domain.AuthCredentials) (*domain.AuthResult, error) {
	_ = ctx

	email := strings.TrimSpace(credentials.GetString("email"))
	name := strings.TrimSpace(credentials.GetString("name"))
	if email == "" || name == "" {
		return nil, errors.New("email and name are required")
	}

	return &domain.AuthResult{
		User: &domain.User{
			Email:           email,
			Name:            name,
			ProfileImageURL: "https://via.placeholder.com/150?text=" + name,
		},
		Provider: "guest",
	}, nil
}
