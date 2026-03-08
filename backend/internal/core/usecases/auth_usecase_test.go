package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// MockUserRepository for testing
type MockUserRepository struct {
	users         map[string]*domain.User
	findByEmailFn func(ctx context.Context, email string) (*domain.User, error)
	createFn      func(ctx context.Context, user *domain.User) error
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*domain.User),
		findByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, nil // not found
		},
		createFn: func(ctx context.Context, user *domain.User) error {
			return nil // success
		},
	}
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	if user, ok := m.users[email]; ok {
		return user, nil
	}
	return nil, nil
}

func (m *MockUserRepository) FindByID(ctx context.Context, userID string) (*domain.User, error) {
	for _, user := range m.users {
		if user.ID.String() == userID {
			return user, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	m.users[user.Email] = user
	return nil
}

// MockTokenService for testing
type MockTokenService struct {
	generateAccessTokenFn  func(user *domain.User) (string, error)
	generateRefreshTokenFn func(userID string) (string, error)
	validateAccessTokenFn  func(token string) (map[string]interface{}, error)
}

func NewMockTokenService() *MockTokenService {
	return &MockTokenService{
		generateAccessTokenFn: func(user *domain.User) (string, error) {
			return "mock_access_token_" + user.ID.String(), nil
		},
		generateRefreshTokenFn: func(userID string) (string, error) {
			return "mock_refresh_token_" + userID, nil
		},
		validateAccessTokenFn: func(token string) (map[string]interface{}, error) {
			if token == "" {
				return nil, errors.New("invalid token")
			}
			return map[string]interface{}{"user_id": "user_123", "exp": float64(9999999999)}, nil
		},
	}
}

func (m *MockTokenService) GenerateAccessToken(user *domain.User) (string, error) {
	if m.generateAccessTokenFn != nil {
		return m.generateAccessTokenFn(user)
	}
	return "", nil
}

func (m *MockTokenService) GenerateRefreshToken(userID string) (string, error) {
	if m.generateRefreshTokenFn != nil {
		return m.generateRefreshTokenFn(userID)
	}
	return "", nil
}

func (m *MockTokenService) ValidateAccessToken(token string) (map[string]interface{}, error) {
	if m.validateAccessTokenFn != nil {
		return m.validateAccessTokenFn(token)
	}
	return map[string]interface{}{}, nil
}

// MockGoogleTokenVerifier for testing
type MockGoogleTokenVerifier struct {
	verifyFn func(ctx context.Context, token string) (email, name, picture string, err error)
}

func NewMockGoogleTokenVerifier() *MockGoogleTokenVerifier {
	return &MockGoogleTokenVerifier{
		verifyFn: func(ctx context.Context, token string) (string, string, string, error) {
			if token == "invalid" {
				return "", "", "", errors.New("invalid token")
			}
			return "test@example.com", "Test User", "https://example.com/pic.jpg", nil
		},
	}
}

func (m *MockGoogleTokenVerifier) Verify(ctx context.Context, token string) (email, name, picture string, err error) {
	if m.verifyFn != nil {
		return m.verifyFn(ctx, token)
	}
	return "", "", "", nil
}

// --- Auth Use Case Tests ---

func TestAuthUseCase_GoogleSignIn_NewUser(t *testing.T) {
	userRepo := NewMockUserRepository()
	tokenSvc := NewMockTokenService()
	googleVerifier := NewMockGoogleTokenVerifier()

	uc := NewAuthUseCase(userRepo, tokenSvc, googleVerifier)

	result, err := uc.GoogleSignIn(context.Background(), "valid_token")
	if err != nil {
		t.Fatalf("GoogleSignIn failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected AuthResult, got nil")
	}

	if result.User.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", result.User.Email)
	}

	if result.AccessToken == "" {
		t.Error("expected AccessToken, got empty string")
	}

	if result.RefreshToken == "" {
		t.Error("expected RefreshToken, got empty string")
	}
}

func TestAuthUseCase_GoogleSignIn_ExistingUser(t *testing.T) {
	existingUser := &domain.User{
		Email: "test@example.com",
		Name:  "Existing User",
	}

	userRepo := NewMockUserRepository()
	userRepo.findByEmailFn = func(ctx context.Context, email string) (*domain.User, error) {
		if email == "test@example.com" {
			return existingUser, nil
		}
		return nil, nil
	}

	tokenSvc := NewMockTokenService()
	googleVerifier := NewMockGoogleTokenVerifier()

	uc := NewAuthUseCase(userRepo, tokenSvc, googleVerifier)

	result, err := uc.GoogleSignIn(context.Background(), "valid_token")
	if err != nil {
		t.Fatalf("GoogleSignIn failed: %v", err)
	}

	if result.User.Name != "Existing User" {
		t.Errorf("expected existing user name, got '%s'", result.User.Name)
	}
}

func TestAuthUseCase_GoogleSignIn_InvalidToken(t *testing.T) {
	userRepo := NewMockUserRepository()
	tokenSvc := NewMockTokenService()
	googleVerifier := NewMockGoogleTokenVerifier()

	uc := NewAuthUseCase(userRepo, tokenSvc, googleVerifier)

	_, err := uc.GoogleSignIn(context.Background(), "invalid")
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}
}

func TestAuthUseCase_GoogleSignIn_TokenGenerationError(t *testing.T) {
	userRepo := NewMockUserRepository()

	tokenSvc := NewMockTokenService()
	tokenSvc.generateAccessTokenFn = func(user *domain.User) (string, error) {
		return "", errors.New("token service down")
	}

	googleVerifier := NewMockGoogleTokenVerifier()

	uc := NewAuthUseCase(userRepo, tokenSvc, googleVerifier)

	_, err := uc.GoogleSignIn(context.Background(), "valid_token")
	if err == nil {
		t.Fatal("expected error when token generation fails, got nil")
	}
}

func TestAuthUseCase_GoogleSignIn_UserRepositoryError(t *testing.T) {
	userRepo := NewMockUserRepository()
	userRepo.findByEmailFn = func(ctx context.Context, email string) (*domain.User, error) {
		return nil, errors.New("database connection failed")
	}

	tokenSvc := NewMockTokenService()
	googleVerifier := NewMockGoogleTokenVerifier()

	uc := NewAuthUseCase(userRepo, tokenSvc, googleVerifier)

	_, err := uc.GoogleSignIn(context.Background(), "valid_token")
	if err == nil {
		t.Fatal("expected error when repository fails, got nil")
	}
}