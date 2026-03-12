package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

type testLPUserRepo struct {
	user *domain.User
}

func (r *testLPUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	_ = ctx
	_ = email
	return nil, nil
}
func (r *testLPUserRepo) Create(ctx context.Context, user *domain.User) error {
	_ = ctx
	r.user = user
	return nil
}
func (r *testLPUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	_ = ctx
	if r.user != nil && r.user.ID.String() == id {
		return r.user, nil
	}
	return nil, nil
}
func (r *testLPUserRepo) UpdateLearningProfile(ctx context.Context, id string, profile map[string]interface{}) error {
	_ = ctx
	u, _ := r.FindByID(context.Background(), id)
	if u == nil {
		return assert.AnError
	}
	u.LearningProfile = profile
	return nil
}

func setupLPTestRouter(handler *LearningProfileHandler, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)
	return router
}

func TestLearningProfile_Update_SkippedWhenFlagOff(t *testing.T) {
	t.Setenv("FF_LEARNING_PROFILE", "false")

	userRepo := &testLPUserRepo{
		user: &domain.User{
			ID:             uuid.New(),
			Email:          "a@b.com",
			LearningProfile: map[string]interface{}{},
		},
	}
	uc := usecases.NewLearningProfileUseCase(userRepo)
	h := NewLearningProfileHandler(uc)
	router := setupLPTestRouter(h, userRepo.user.ID.String())

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/me/learning-profile/update",
		bytes.NewBufferString(`{"recent_messages":["hola"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, "skipped", body["status"])
}

func TestLearningProfile_Update_UpdatesWhenFlagOn(t *testing.T) {
	t.Setenv("FF_LEARNING_PROFILE", "true")
	defer os.Unsetenv("FF_LEARNING_PROFILE")

	userRepo := &testLPUserRepo{
		user: &domain.User{
			ID:             uuid.New(),
			Email:          "a@b.com",
			LearningProfile: map[string]interface{}{},
		},
	}
	uc := usecases.NewLearningProfileUseCase(userRepo)
	h := NewLearningProfileHandler(uc)
	router := setupLPTestRouter(h, userRepo.user.ID.String())

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/me/learning-profile/update",
		bytes.NewBufferString(`{"recent_messages":["paso a paso","integral"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, userRepo.user.LearningProfile)
}

func TestLearningProfile_Get_ReturnsProfile(t *testing.T) {
	t.Setenv("FF_LEARNING_PROFILE", "true")

	userRepo := &testLPUserRepo{
		user: &domain.User{
			ID:    uuid.New(),
			Email: "a@b.com",
			LearningProfile: map[string]interface{}{
				"total_sessions": 2,
			},
		},
	}
	uc := usecases.NewLearningProfileUseCase(userRepo)
	h := NewLearningProfileHandler(uc)
	router := setupLPTestRouter(h, userRepo.user.ID.String())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/learning-profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "learning_profile")
}

