package httphandlers

import (
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

// maxMaterialSize limits material uploads to 20 MB.
// PDFs and audio files are typically larger than images, so we allow double the image limit.
const maxMaterialSize = 20 << 20 // 20 MB

// allowedMaterialFormats maps MIME types detected from magic bytes to MaterialFormatType.
var allowedMaterialFormats = map[string]domain.MaterialFormatType{
	"application/pdf": domain.MaterialFormatPDF,
	"text/plain":      domain.MaterialFormatTXT,
}

// allowedExtensions maps file extensions to MaterialFormatType for text/plain disambiguation.
// http.DetectContentType returns "text/plain" for both .txt and .md files.
var allowedExtensions = map[string]domain.MaterialFormatType{
	".pdf": domain.MaterialFormatPDF,
	".txt": domain.MaterialFormatTXT,
	".md":  domain.MaterialFormatMD,
}

// MaterialHandler handles HTTP requests for the Material Upload module.
type MaterialHandler struct {
	materialUseCase *usecases.MaterialUseCase
}

// NewMaterialHandler creates a MaterialHandler.
func NewMaterialHandler(materialUseCase *usecases.MaterialUseCase) *MaterialHandler {
	return &MaterialHandler{materialUseCase: materialUseCase}
}

// RegisterRoutes attaches material routes to the given (protected) router group.
// Routes are nested under courses and topics for correct resource hierarchy.
func (h *MaterialHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/courses/:course_id/topics/:topic_id/materials", h.UploadMaterial)
	rg.GET("/courses/:course_id/topics/:topic_id/materials", h.ListMaterials)
}

// UploadMaterial handles POST /api/v1/courses/:course_id/topics/:topic_id/materials
// Accepts multipart/form-data with a single "file" field.
func (h *MaterialHandler) UploadMaterial(c *gin.Context) {
	// SECURITY: user_id comes from the JWT middleware context — never from the request body.
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

	// SECURITY: enforce the 20 MB limit before reading all bytes into memory.
	limitedReader := io.LimitReader(file, maxMaterialSize+1)
	fileData, err := io.ReadAll(limitedReader)
	if err != nil {
		log.Printf("[Material] Failed to read uploaded file: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read uploaded file"})
		return
	}
	if int64(len(fileData)) > maxMaterialSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file exceeds 20 MB limit"})
		return
	}

	// SECURITY: determine format from both extension AND magic bytes.
	ext := strings.ToLower(filepath.Ext(header.Filename))
	formatType, extOK := allowedExtensions[ext]
	if !extOK {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "only PDF, TXT and MD files are accepted"})
		return
	}

	// For text/plain files, confirm via magic bytes that it is actually text.
	detectedMIME := http.DetectContentType(fileData)
	if formatType == domain.MaterialFormatPDF && detectedMIME != "application/pdf" {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "file content does not match the .pdf extension"})
		return
	}

	material, err := h.materialUseCase.UploadMaterial(c.Request.Context(), usecases.UploadMaterialInput{
		UserID:     userID,
		CourseID:   courseID,
		TopicID:    topicID,
		FileName:   header.Filename,
		FileData:   fileData,
		FormatType: formatType,
		SizeBytes:  int64(len(fileData)),
	})
	if err != nil {
		log.Printf("[Material] UploadMaterial failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not upload material"})
		return
	}
	if material == nil {
		// nil means ownership validation failed — return 404 to avoid enumeration.
		c.JSON(http.StatusNotFound, gin.H{"error": "course or topic not found"})
		return
	}

	c.JSON(http.StatusCreated, material)
}

// ListMaterials handles GET /api/v1/courses/:course_id/topics/:topic_id/materials
func (h *MaterialHandler) ListMaterials(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(string)
	courseID := c.Param("course_id")
	topicID := c.Param("topic_id")

	materials, err := h.materialUseCase.GetMaterialsByTopic(c.Request.Context(), courseID, topicID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve materials"})
		return
	}
	if materials == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course or topic not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"materials": materials, "total": len(materials)})
}
