package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

type testAuthMockUserRepository struct {
	mu           sync.Mutex
	usersByID    map[string]*domain.User
	usersByEmail map[string]*domain.User
}

func (m *testAuthMockUserRepository) Create(ctx context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	m.usersByID[user.ID.String()] = user
	m.usersByEmail[user.Email] = user
	return nil
}

func (m *testAuthMockUserRepository) FindByID(ctx context.Context, userID string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if u, ok := m.usersByID[userID]; ok {
		return u, nil
	}
	return nil, nil
}

func (m *testAuthMockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if u, ok := m.usersByEmail[email]; ok {
		return u, nil
	}
	return nil, nil
}

func (m *testAuthMockUserRepository) UpdateLearningProfile(ctx context.Context, id string, profile map[string]interface{}) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.usersByID[id]
	if !ok || u == nil {
		return errors.New("not found")
	}
	u.LearningProfile = profile
	return nil
}

type testAuthMockTokenService struct{}

func (m *testAuthMockTokenService) GenerateAccessToken(user *domain.User) (string, error) {
	if user == nil || user.ID == uuid.Nil {
		return "", errors.New("invalid user")
	}
	return "access_token_" + user.ID.String(), nil
}

func (m *testAuthMockTokenService) GenerateRefreshToken(userID string) (string, error) {
	if userID == "" {
		return "", errors.New("invalid user id")
	}
	return "refresh_token_" + userID, nil
}

func (m *testAuthMockTokenService) ValidateAccessToken(tokenString string) (map[string]interface{}, error) {
	if tokenString == "invalid" {
		return nil, errors.New("invalid token")
	}
	return map[string]interface{}{"user_id": "user_id_fake"}, nil
}

type testAuthMockGoogleTokenVerifier struct {
	shouldFail bool
}

func (m *testAuthMockGoogleTokenVerifier) Verify(ctx context.Context, idToken string) (string, string, string, error) {
	if m.shouldFail || idToken == "invalid" {
		return "", "", "", errors.New("invalid token")
	}
	return "test@example.com", "Test User", "https://example.com/avatar.png", nil
}

func setupAuthHandler(authUseCase *usecases.AuthUseCase) *AuthHandler {
	return NewAuthHandler(authUseCase)
}

func TestAuthHandler_SignIn_Google_Success(t *testing.T) {
	userRepo := &testAuthMockUserRepository{
		usersByID:    make(map[string]*domain.User),
		usersByEmail: make(map[string]*domain.User),
	}
	tokenSvc := &testAuthMockTokenService{}
	googleVerifier := &testAuthMockGoogleTokenVerifier{}

	authUseCase := usecases.NewAuthUseCase(userRepo, tokenSvc, googleVerifier)
	handler := setupAuthHandler(authUseCase)

	router := gin.New()
	router.POST("/auth/login", handler.SignIn)

	reqBody := map[string]string{
		"provider": "google",
		"id_token": "valid_token",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthHandler_SignIn_Guest_Success(t *testing.T) {
	userRepo := &testAuthMockUserRepository{
		usersByID:    make(map[string]*domain.User),
		usersByEmail: make(map[string]*domain.User),
	}
	tokenSvc := &testAuthMockTokenService{}
	googleVerifier := &testAuthMockGoogleTokenVerifier{}

	authUseCase := usecases.NewAuthUseCase(userRepo, tokenSvc, googleVerifier)
	handler := setupAuthHandler(authUseCase)

	router := gin.New()
	router.POST("/auth/login", handler.SignIn)

	reqBody := map[string]string{
		"provider": "guest",
		"email":    "guest@example.com",
		"name":     "Guest User",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
