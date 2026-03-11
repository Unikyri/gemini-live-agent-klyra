package http

import (
	"bytes"
	"context"
	"mime/multipart"
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

// MockMaterialUseCase mocks the MaterialUseCase for testing handlers.
type MockMaterialUseCase struct {
	mock.Mock
}

func (m *MockMaterialUseCase) UploadMaterial(ctx context.Context, input usecases.UploadMaterialInput) (*domain.Material, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Material), args.Error(1)
}

func (m *MockMaterialUseCase) GetMaterialsByTopic(ctx context.Context, courseID, topicID, userID string) ([]domain.Material, error) {
	args := m.Called(ctx, courseID, topicID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Material), args.Error(1)
}

func setupMaterialRouter(mockUC *MockMaterialUseCase) *gin.Engine {
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

	// For tests, manually register routes to avoid handler dependency on concrete use case
	api := router.Group("/api/v1")
	api.POST("/courses/:course_id/topics/:topic_id/materials", func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userID := userIDVal.(string)
		courseID := c.Param("course_id")
		topicID := c.Param("topic_id")

		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
			return
		}
		defer file.Close()

		fileData := make([]byte, header.Size)
		file.Read(fileData)

		material, err := mockUC.UploadMaterial(c.Request.Context(), usecases.UploadMaterialInput{
			UserID:     userID,
			CourseID:   courseID,
			TopicID:    topicID,
			FileName:   header.Filename,
			FileData:   fileData,
			FormatType: domain.MaterialFormatPDF,
			SizeBytes:  header.Size,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not upload material"})
			return
		}
		if material == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "course or topic not found"})
			return
		}
		c.JSON(http.StatusCreated, material)
	})
	api.GET("/courses/:course_id/topics/:topic_id/materials", func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userID := userIDVal.(string)
		courseID := c.Param("course_id")
		topicID := c.Param("topic_id")

		materials, err := mockUC.GetMaterialsByTopic(c.Request.Context(), courseID, topicID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve materials"})
			return
		}
		if materials == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "course or topic not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"materials": materials, "total": len(materials)})
	})
	return router
}

func TestUploadMaterial_Success(t *testing.T) {
	mockUC := new(MockMaterialUseCase)
	materialID := uuid.New()
	topicID := uuid.New()
	courseID := uuid.New()
	userID := uuid.New()

	expectedMaterial := &domain.Material{
		ID:           materialID,
		TopicID:      topicID,
		FormatType:   domain.MaterialFormatPDF,
		StorageURL:   "gs://bucket/materials/test.pdf",
		OriginalName: "test.pdf",
		Status:       domain.MaterialStatusPending,
	}

	mockUC.On("UploadMaterial", mock.Anything, mock.MatchedBy(func(input usecases.UploadMaterialInput) bool {
		return input.UserID == userID.String() &&
			input.CourseID == courseID.String() &&
			input.TopicID == topicID.String() &&
			input.FormatType == domain.MaterialFormatPDF
	})).Return(expectedMaterial, nil)

	router := setupMaterialRouter(mockUC)

	// Create multipart form with file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.pdf")
	part.Write([]byte("%PDF-1.4 test content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/courses/"+courseID.String()+"/topics/"+topicID.String()+"/materials", body)
	req.Header.Set("X-Test-User-ID", userID.String())
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockUC.AssertExpectations(t)
}

func TestUploadMaterial_MissingFile(t *testing.T) {
	mockUC := new(MockMaterialUseCase)
	courseID := uuid.New()
	topicID := uuid.New()
	userID := uuid.New()
	router := setupMaterialRouter(mockUC)

	req := httptest.NewRequest("POST", "/api/v1/courses/"+courseID.String()+"/topics/"+topicID.String()+"/materials", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	req.Header.Set("Content-Type", "multipart/form-data")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListMaterials_Success(t *testing.T) {
	mockUC := new(MockMaterialUseCase)
	topicID := uuid.New()
	courseID := uuid.New()
	userID := uuid.New()

	expectedMaterials := []domain.Material{
		{
			ID:         uuid.New(),
			TopicID:    topicID,
			FormatType: domain.MaterialFormatPDF,
		},
		{
			ID:         uuid.New(),
			TopicID:    topicID,
			FormatType: domain.MaterialFormatTXT,
		},
	}

	mockUC.On("GetMaterialsByTopic", mock.Anything, courseID.String(), topicID.String(), userID.String()).Return(expectedMaterials, nil)

	router := setupMaterialRouter(mockUC)
	req := httptest.NewRequest("GET", "/api/v1/courses/"+courseID.String()+"/topics/"+topicID.String()+"/materials", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockUC.AssertExpectations(t)
}

func TestValidateMaterialFile_ExtensionFirstRules(t *testing.T) {
	t.Run("allowed map includes legacy MIME variants", func(t *testing.T) {
		assert.Equal(t, domain.MaterialFormatPDF, allowedMaterialFormats["application/x-pdf"])
		assert.Equal(t, domain.MaterialFormatJPEG, allowedMaterialFormats["image/jpg"])
	})

	testCases := []struct {
		name         string
		filename     string
		fileData     []byte
		wantStatus   int
		wantFormat   domain.MaterialFormatType
		wantErr      string
	}{
		{
			name:       "pdf canonical mime accepted",
			filename:   "ok.pdf",
			fileData:   []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\n"),
			wantStatus: 0,
			wantFormat: domain.MaterialFormatPDF,
		},
		{
			name:       "pdf octet-stream accepted by extension",
			filename:   "fallback.pdf",
			fileData:   []byte{0x00, 0x01, 0x02, 0x03},
			wantStatus: 0,
			wantFormat: domain.MaterialFormatPDF,
		},
		{
			name:       "jpg accepted",
			filename:   "photo.jpg",
			fileData:   []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46},
			wantStatus: 0,
			wantFormat: domain.MaterialFormatJPG,
		},
		{
			name:       "unsupported extension returns 415",
			filename:   "paper.docx",
			fileData:   []byte("docx bytes"),
			wantStatus: http.StatusUnsupportedMediaType,
			wantErr:    "only PDF, TXT, MD, PNG, JPG, JPEG, WEBP, MP3, WAV and M4A files are accepted",
		},
		{
			name:       "pdf with image bytes returns 415",
			filename:   "fake.pdf",
			fileData:   []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46},
			wantStatus: http.StatusUnsupportedMediaType,
			wantErr:    "file content does not match the .pdf extension",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotFormat, gotStatus, gotErr := validateMaterialFile(tc.filename, tc.fileData)
			assert.Equal(t, tc.wantStatus, gotStatus)
			assert.Equal(t, tc.wantErr, gotErr)
			if tc.wantStatus == 0 {
				assert.Equal(t, tc.wantFormat, gotFormat)
			}
		})
	}
}
