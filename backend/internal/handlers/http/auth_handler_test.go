package httphandlers

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

// Test mocks
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

// Test helper: create a test AuthHandler with mocks
func setupAuthHandler(authUseCase *usecases.AuthUseCase) *AuthHandler {
	return NewAuthHandler(authUseCase)
}

// Test case: GoogleSignIn with valid token
func TestAuthHandler_GoogleSignIn_Success(t *testing.T) {
	userRepo := &testAuthMockUserRepository{
		usersByID:    make(map[string]*domain.User),
		usersByEmail: make(map[string]*domain.User),
	}
	tokenSvc := &testAuthMockTokenService{}
	googleVerifier := &testAuthMockGoogleTokenVerifier{}

	authUseCase := usecases.NewAuthUseCase(userRepo, tokenSvc, googleVerifier)
	handler := setupAuthHandler(authUseCase)

	// Setup Gin test context
	router := gin.New()
	router.POST("/auth/google", handler.GoogleSignIn)

	// Prepare request
	reqBody := map[string]string{
		"id_token": "valid_token",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var respBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &respBody)

	if _, ok := respBody["access_token"]; !ok {
		t.Error("expected access_token in response")
	}

	if _, ok := respBody["refresh_token"]; !ok {
		t.Error("expected refresh_token in response")
	}

	if _, ok := respBody["user"]; !ok {
		t.Error("expected user in response")
	}
}

// Test case: GoogleSignIn missing id_token
func TestAuthHandler_GoogleSignIn_MissingIDToken(t *testing.T) {
	userRepo := &testAuthMockUserRepository{
		usersByID:    make(map[string]*domain.User),
		usersByEmail: make(map[string]*domain.User),
	}
	tokenSvc := &testAuthMockTokenService{}
	googleVerifier := &testAuthMockGoogleTokenVerifier{}

	authUseCase := usecases.NewAuthUseCase(userRepo, tokenSvc, googleVerifier)
	handler := setupAuthHandler(authUseCase)

	router := gin.New()
	router.POST("/auth/google", handler.GoogleSignIn)

	// Request without id_token
	reqBody := map[string]string{}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test case: GoogleSignIn with invalid token
func TestAuthHandler_GoogleSignIn_InvalidToken(t *testing.T) {
	userRepo := &testAuthMockUserRepository{
		usersByID:    make(map[string]*domain.User),
		usersByEmail: make(map[string]*domain.User),
	}
	tokenSvc := &testAuthMockTokenService{}
	googleVerifier := &testAuthMockGoogleTokenVerifier{}

	authUseCase := usecases.NewAuthUseCase(userRepo, tokenSvc, googleVerifier)
	handler := setupAuthHandler(authUseCase)

	router := gin.New()
	router.POST("/auth/google", handler.GoogleSignIn)

	reqBody := map[string]string{
		"id_token": "invalid",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// Test case: GoogleSignIn returns user data
func TestAuthHandler_GoogleSignIn_ReturnsUserData(t *testing.T) {
	userRepo := &testAuthMockUserRepository{
		usersByID:    make(map[string]*domain.User),
		usersByEmail: make(map[string]*domain.User),
	}
	tokenSvc := &testAuthMockTokenService{}
	googleVerifier := &testAuthMockGoogleTokenVerifier{}

	authUseCase := usecases.NewAuthUseCase(userRepo, tokenSvc, googleVerifier)
	handler := setupAuthHandler(authUseCase)

	router := gin.New()
	router.POST("/auth/google", handler.GoogleSignIn)

	reqBody := map[string]string{
		"id_token": "valid_token",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var respBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &respBody)

	user, ok := respBody["user"].(map[string]interface{})
	if !ok {
		t.Fatal("expected user object in response")
	}

	if email, ok := user["email"].(string); !ok || email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%v'", user["email"])
	}
}