package httphandlers

import (
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

// maxImageSize limits uploaded reference images to 10 MB.
// This prevents DoS attacks via excessively large uploads.
const maxImageSize = 10 << 20 // 10 MB

// CourseHandler handles HTTP requests for the Course Management module.
type CourseHandler struct {
	courseUseCase *usecases.CourseUseCase
}

// NewCourseHandler creates a CourseHandler.
func NewCourseHandler(courseUseCase *usecases.CourseUseCase) *CourseHandler {
	return &CourseHandler{courseUseCase: courseUseCase}
}

// RegisterRoutes attaches all course routes to the given (protected) router group.
// All routes here require the AuthMiddleware to have already set "user_id" in context.
func (h *CourseHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/courses", h.CreateCourse)
	rg.GET("/courses", h.ListCourses)
	rg.GET("/courses/:course_id", h.GetCourse)
	rg.POST("/courses/:course_id/topics", h.AddTopic)
}

// CreateCourse handles POST /api/v1/courses
// Accepts multipart/form-data with fields: name, education_level, reference_image (file, optional).
func (h *CourseHandler) CreateCourse(c *gin.Context) {
	// SECURITY: user_id is read from the JWT middleware context — NOT from the request body.
	userID, _ := c.Get("user_id")

	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	educationLevel := c.PostForm("education_level")

	// Handle optional reference image upload.
	var imageData []byte
	var imageType string
	file, _, err := c.Request.FormFile("reference_image")
	if err == nil && file != nil {
		defer file.Close()

		// SECURITY (BLOCKER fix): limit reads to maxImageSize to prevent DoS via huge uploads.
		limitedReader := io.LimitReader(file, maxImageSize+1)
		imageData, err = io.ReadAll(limitedReader)
		if err != nil {
			log.Printf("[Course] Failed to read uploaded image: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "could not read uploaded file"})
			return
		}
		if int64(len(imageData)) > maxImageSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "image exceeds 10 MB limit"})
			return
		}

		// SECURITY (WARNING fix): detect the real MIME type from magic bytes — never trust client headers.
		detectedType := http.DetectContentType(imageData)
		allowedTypes := map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true}
		if !allowedTypes[detectedType] {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "only JPEG, PNG, WebP and GIF images are allowed"})
			return
		}
		imageType = detectedType
	}

	course, err := h.courseUseCase.CreateCourse(c.Request.Context(), usecases.CreateCourseInput{
		UserID:             userID.(string),
		Name:               name,
		EducationLevel:     educationLevel,
		ReferenceImageData: imageData,
		ReferenceImageType: imageType,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create course"})
		return
	}

	c.JSON(http.StatusCreated, course)
}

// ListCourses handles GET /api/v1/courses
// Returns all active courses owned by the authenticated user.
func (h *CourseHandler) ListCourses(c *gin.Context) {
	userID, _ := c.Get("user_id")

	courses, err := h.courseUseCase.GetCoursesByUser(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve courses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"courses": courses, "total": len(courses)})
}

// GetCourse handles GET /api/v1/courses/:course_id
// Returns a single course with its topics. Validates ownership before returning.
func (h *CourseHandler) GetCourse(c *gin.Context) {
	userID, _ := c.Get("user_id")
	courseID := c.Param("course_id")

	course, err := h.courseUseCase.GetCourseByID(c.Request.Context(), courseID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve course"})
		return
	}
	if course == nil {
		// Returns 404 whether the course doesn't exist OR doesn't belong to this user.
		// This prevents resource enumeration attacks (SECURITY: Elevation of Privilege).
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}

	c.JSON(http.StatusOK, course)
}

// AddTopic handles POST /api/v1/courses/:course_id/topics
// Body: { "title": "Integral Calculus" }
func (h *CourseHandler) AddTopic(c *gin.Context) {
	userID, _ := c.Get("user_id")
	courseID := c.Param("course_id")

	var body struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}

	topic, err := h.courseUseCase.AddTopic(c.Request.Context(), courseID, userID.(string), body.Title)
	if err != nil || topic == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not add topic"})
		return
	}

	c.JSON(http.StatusCreated, topic)
}
