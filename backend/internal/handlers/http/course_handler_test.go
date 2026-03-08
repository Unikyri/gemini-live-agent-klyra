package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

type MockCourseUseCase struct {
	mock.Mock
}

func (m *MockCourseUseCase) CreateCourse(courseReq *usecases.CreateCourseRequest) (*domain.Course, error) {
	args := m.Called(courseReq)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Course), args.Error(1)
}

func (m *MockCourseUseCase) GetCoursesByUser(userID uuid.UUID) ([]*domain.Course, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Course), args.Error(1)
}

func (m *MockCourseUseCase) GetCourseByID(id uuid.UUID) (*domain.Course, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Course), args.Error(1)
}

func setupCourseRouter(mockUC *MockCourseUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewCourseHandler(mockUC)
	router.POST("/courses", handler.CreateCourse)
	router.GET("/courses/:id", handler.GetCourseByID)
	router.GET("/users/:userId/courses", handler.GetCoursesByUser)
	return router
}

func TestCreateCourse_Success(t *testing.T) {
	mockUC := new(MockCourseUseCase)
	userID := uuid.New()
	courseID := uuid.New()

	expectedCourse := &domain.Course{
		ID:          courseID,
		Title:       "Advanced Go",
		Description: "Master Go programming",
		OwnerID:     userID,
	}

	mockUC.On("CreateCourse", mock.MatchedBy(func(req *usecases.CreateCourseRequest) bool {
		return req.Title == "Advanced Go" && req.OwnerID == userID
	})).Return(expectedCourse, nil)

	router := setupCourseRouter(mockUC)

	body := map[string]interface{}{
		"title":       "Advanced Go",
		"description": "Master Go programming",
		"owner_id":    userID.String(),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/courses", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var responseBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, expectedCourse.Title, responseBody["title"])
	mockUC.AssertExpectations(t)
}

func TestCreateCourse_InvalidRequest(t *testing.T) {
	mockUC := new(MockCourseUseCase)
	router := setupCourseRouter(mockUC)

	body := map[string]interface{}{
		"description": "Missing title",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/courses", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetCourseByID_Success(t *testing.T) {
	mockUC := new(MockCourseUseCase)
	courseID := uuid.New()
	userID := uuid.New()

	expectedCourse := &domain.Course{
		ID:          courseID,
		Title:       "Go Basics",
		Description: "Learn Go",
		OwnerID:     userID,
	}

	mockUC.On("GetCourseByID", courseID).Return(expectedCourse, nil)

	router := setupCourseRouter(mockUC)
	req := httptest.NewRequest("GET", "/courses/"+courseID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, expectedCourse.Title, responseBody["title"])
	mockUC.AssertExpectations(t)
}

func TestGetCoursesByUser_Success(t *testing.T) {
	mockUC := new(MockCourseUseCase)
	userID := uuid.New()

	expectedCourses := []*domain.Course{
		{
			ID:      uuid.New(),
			Title:   "Course 1",
			OwnerID: userID,
		},
		{
			ID:      uuid.New(),
			Title:   "Course 2",
			OwnerID: userID,
		},
	}

	mockUC.On("GetCoursesByUser", userID).Return(expectedCourses, nil)

	router := setupCourseRouter(mockUC)
	req := httptest.NewRequest("GET", "/users/"+userID.String()+"/courses", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseBody []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, 2, len(responseBody))
	mockUC.AssertExpectations(t)
}
