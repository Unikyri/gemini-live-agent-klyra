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
	strategies     map[string]ports.AuthStrategy
}

// NewAuthUseCase creates and returns an AuthUseCase instance.
// Dependencies are injected from outside (Dependency Injection).
func NewAuthUseCase(
	userRepo ports.UserRepository,
	tokenSvc ports.TokenService,
	googleVerifier ports.GoogleTokenVerifier,
	strategyMaps ...map[string]ports.AuthStrategy,
) *AuthUseCase {
	strategies := map[string]ports.AuthStrategy{}
	if len(strategyMaps) > 0 && strategyMaps[0] != nil {
		for key, strategy := range strategyMaps[0] {
			if strategy != nil {
				strategies[key] = strategy
			}
		}
	} else {
		// Default strategies keep backward compatibility with existing wiring.
		strategies["google"] = &googleDefaultStrategy{verifier: googleVerifier}
		strategies["guest"] = &guestDefaultStrategy{}
	}

	return &AuthUseCase{
		userRepo:       userRepo,
		tokenSvc:       tokenSvc,
		googleVerifier: googleVerifier,
		strategies:     strategies,
	}
}

// Login authenticates a user using a provider strategy, then executes the common
// find-or-create + token issuance flow.
func (uc *AuthUseCase) Login(ctx context.Context, provider string, credentials domain.AuthCredentials) (*domain.AuthResult, error) {
	strategy, ok := uc.strategies[provider]
	if !ok || strategy == nil {
		return nil, errors.New("unsupported auth provider")
	}

	validated, err := strategy.Authenticate(ctx, credentials)
	if err != nil {
		return nil, err
	}
	if validated == nil || validated.User == nil || validated.User.Email == "" {
		return nil, errors.New("invalid authentication payload")
	}

	// Common flow: find/create user in DB.
	user, err := uc.userRepo.FindByEmail(ctx, validated.User.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		log.Printf("[Auth] New user signing in: %s — creating record.", validated.User.Email)
		user = &domain.User{
			Email:           validated.User.Email,
			Name:            validated.User.Name,
			ProfileImageURL: validated.User.ProfileImageURL,
		}
		if createErr := uc.userRepo.Create(ctx, user); createErr != nil {
			return nil, createErr
		}
	}

	accessToken, err := uc.tokenSvc.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := uc.tokenSvc.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	return &domain.AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
		Provider:     validated.Provider,
	}, nil
}

// GoogleSignIn validates a Google ID Token, then finds or creates the user in our system.
// This implements the "find or create" (upsert-like) pattern common in OAuth2 flows.
func (uc *AuthUseCase) GoogleSignIn(ctx context.Context, googleIDToken string) (*domain.AuthResult, error) {
	return uc.Login(ctx, "google", domain.AuthCredentials{
		"id_token": googleIDToken,
	})
}

// MockSignIn handles development/guest login without Google validation.
// SECURITY WARNING: This is for LOCAL DEVELOPMENT ONLY.
// In production, this endpoint should be disabled or protected by strict IP whitelisting.
func (uc *AuthUseCase) MockSignIn(ctx context.Context, email, name string) (*domain.AuthResult, error) {
	return uc.Login(ctx, "guest", domain.AuthCredentials{
		"email": email,
		"name":  name,
	})
}

// googleDefaultStrategy preserves compatibility when strategies are not injected.
type googleDefaultStrategy struct {
	verifier ports.GoogleTokenVerifier
}

func (s *googleDefaultStrategy) Authenticate(ctx context.Context, credentials domain.AuthCredentials) (*domain.AuthResult, error) {
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
		User:     &domain.User{Email: email, Name: name, ProfileImageURL: picture},
		Provider: "google",
	}, nil
}

// guestDefaultStrategy preserves compatibility when strategies are not injected.
type guestDefaultStrategy struct{}

func (s *guestDefaultStrategy) Authenticate(ctx context.Context, credentials domain.AuthCredentials) (*domain.AuthResult, error) {
	_ = ctx
	email := credentials.GetString("email")
	name := credentials.GetString("name")
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
