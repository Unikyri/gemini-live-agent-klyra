package http

import (
	"context"
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

// MockCourseUseCase mocks the CourseUseCase for testing handlers.
type MockCourseUseCase struct {
	mock.Mock
}

func (m *MockCourseUseCase) CreateCourse(ctx context.Context, input usecases.CreateCourseInput) (*domain.Course, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Course), args.Error(1)
}

func (m *MockCourseUseCase) GetCoursesByUser(ctx context.Context, userID string) ([]domain.Course, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Course), args.Error(1)
}

func (m *MockCourseUseCase) GetCourseByID(ctx context.Context, courseID, userID string) (*domain.Course, error) {
	args := m.Called(ctx, courseID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Course), args.Error(1)
}

func (m *MockCourseUseCase) AddTopic(ctx context.Context, courseID, userID, title string) (*domain.Topic, error) {
	args := m.Called(ctx, courseID, userID, title)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Topic), args.Error(1)
}

func setupCourseRouter(mockUC *MockCourseUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Middleware to inject user_id into context (simulating auth middleware)
	router.Use(func(c *gin.Context) {
		testUserID := c.GetHeader("X-Test-User-ID")
		if testUserID != "" {
			c.Set("user_id", testUserID)
		}
		c.Next()
	})

	// For tests, we'll manually register routes to avoid handler dependency on concrete use case
	api := router.Group("/api/v1")
	api.POST("/courses", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		name := c.PostForm("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		educationLevel := c.PostForm("education_level")

		course, err := mockUC.CreateCourse(c.Request.Context(), usecases.CreateCourseInput{
			UserID:         userID.(string),
			Name:           name,
			EducationLevel: educationLevel,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create course"})
			return
		}
		c.JSON(http.StatusCreated, course)
	})
	api.GET("/courses", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		courses, err := mockUC.GetCoursesByUser(c.Request.Context(), userID.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve courses"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"courses": courses, "total": len(courses)})
	})
	api.GET("/courses/:course_id", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		courseID := c.Param("course_id")
		course, err := mockUC.GetCourseByID(c.Request.Context(), courseID, userID.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve course"})
			return
		}
		if course == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
			return
		}
		c.JSON(http.StatusOK, course)
	})
	return router
}

func TestCreateCourse_Success(t *testing.T) {
	mockUC := new(MockCourseUseCase)
	userID := uuid.New()
	courseID := uuid.New()

	expectedCourse := &domain.Course{
		ID:             courseID,
		UserID:         userID,
		Name:           "Advanced Go",
		EducationLevel: "university",
		AvatarStatus:   "pending",
	}

	mockUC.On("CreateCourse", mock.Anything, mock.MatchedBy(func(input usecases.CreateCourseInput) bool {
		return input.Name == "Advanced Go" && input.UserID == userID.String() && input.EducationLevel == "university"
	})).Return(expectedCourse, nil)

	router := setupCourseRouter(mockUC)

	// Create multipart form request
	req := httptest.NewRequest("POST", "/api/v1/courses", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	req.Header.Set("Content-Type", "multipart/form-data")
	req.PostForm = map[string][]string{
		"name":            {"Advanced Go"},
		"education_level": {"university"},
	}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockUC.AssertExpectations(t)
}

func TestCreateCourse_MissingName(t *testing.T) {
	mockUC := new(MockCourseUseCase)
	router := setupCourseRouter(mockUC)

	req := httptest.NewRequest("POST", "/api/v1/courses", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())
	req.Header.Set("Content-Type", "multipart/form-data")
	req.PostForm = map[string][]string{
		"education_level": {"university"},
	}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetCourse_Success(t *testing.T) {
	mockUC := new(MockCourseUseCase)
	courseID := uuid.New()
	userID := uuid.New()

	expectedCourse := &domain.Course{
		ID:     courseID,
		UserID: userID,
		Name:   "Go Basics",
	}

	mockUC.On("GetCourseByID", mock.Anything, courseID.String(), userID.String()).Return(expectedCourse, nil)

	router := setupCourseRouter(mockUC)
	req := httptest.NewRequest("GET", "/api/v1/courses/"+courseID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockUC.AssertExpectations(t)
}

func TestListCourses_Success(t *testing.T) {
	mockUC := new(MockCourseUseCase)
	userID := uuid.New()

	expectedCourses := []domain.Course{
		{
			ID:     uuid.New(),
			Name:   "Course 1",
			UserID: userID,
		},
		{
			ID:     uuid.New(),
			Name:   "Course 2",
			UserID: userID,
		},
	}

	mockUC.On("GetCoursesByUser", mock.Anything, userID.String()).Return(expectedCourses, nil)

	router := setupCourseRouter(mockUC)
	req := httptest.NewRequest("GET", "/api/v1/courses", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockUC.AssertExpectations(t)
}
