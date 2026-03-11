package http

import (
	"errors"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
)

// Size limits per format to keep uploads bounded while allowing lecture audio.
const (
	maxDefaultMaterialSize = 20 << 20 // 20 MB
	maxAudioMaterialSize   = 50 << 20 // 50 MB
)

// allowedMaterialFormats maps MIME types detected from magic bytes to MaterialFormatType.
var allowedMaterialFormats = map[string]domain.MaterialFormatType{
	// PDF
	"application/pdf":   domain.MaterialFormatPDF,
	"application/x-pdf": domain.MaterialFormatPDF,
	// Text
	"text/plain": domain.MaterialFormatTXT,
	// Images
	"image/png":  domain.MaterialFormatPNG,
	"image/jpeg": domain.MaterialFormatJPEG,
	"image/jpg":  domain.MaterialFormatJPEG,
	"image/webp": domain.MaterialFormatWEBP,
	// Audio
	"audio/mpeg":  domain.MaterialFormatAudio,
	"audio/mp3":   domain.MaterialFormatAudio,
	"audio/wav":   domain.MaterialFormatAudio,
	"audio/x-wav": domain.MaterialFormatAudio,
	"audio/mp4":   domain.MaterialFormatAudio,
	"audio/x-m4a": domain.MaterialFormatAudio,
	"audio/m4a":   domain.MaterialFormatAudio,
	// Fallback: application/octet-stream is accepted when the extension is valid.
	"application/octet-stream": domain.MaterialFormatTXT,
}

// allowedExtensions maps file extensions to MaterialFormatType for text/plain disambiguation.
// http.DetectContentType returns "text/plain" for both .txt and .md files.
var allowedExtensions = map[string]domain.MaterialFormatType{
	".pdf":  domain.MaterialFormatPDF,
	".txt":  domain.MaterialFormatTXT,
	".md":   domain.MaterialFormatMD,
	".png":  domain.MaterialFormatPNG,
	".jpg":  domain.MaterialFormatJPG,
	".jpeg": domain.MaterialFormatJPEG,
	".webp": domain.MaterialFormatWEBP,
	".mp3":  domain.MaterialFormatAudio,
	".wav":  domain.MaterialFormatAudio,
	".m4a":  domain.MaterialFormatAudio,
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
	if _, err := uuid.Parse(topicID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course or topic not found"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	// SECURITY: read with the maximum cap first, then enforce format-specific limit.
	limitedReader := io.LimitReader(file, maxAudioMaterialSize+1)
	fileData, err := io.ReadAll(limitedReader)
	if err != nil {
		log.Printf("[Material] Failed to read uploaded file: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read uploaded file"})
		return
	}
	formatType, statusCode, errMsg := validateMaterialFile(header.Filename, fileData)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": errMsg})
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
	if errors.Is(err, usecases.ErrMaterialForbidden) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
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

func validateMaterialFile(filename string, fileData []byte) (domain.MaterialFormatType, int, string) {
	ext := strings.ToLower(filepath.Ext(filename))
	formatType, extOK := allowedExtensions[ext]
	if !extOK {
		return "", http.StatusUnsupportedMediaType, "only PDF, TXT, MD, PNG, JPG, JPEG, WEBP, MP3, WAV and M4A files are accepted"
	}

	sizeLimit := maxDefaultMaterialSize
	if formatType == domain.MaterialFormatAudio {
		sizeLimit = maxAudioMaterialSize
	}
	if int64(len(fileData)) > int64(sizeLimit) {
		if formatType == domain.MaterialFormatAudio {
			return "", http.StatusRequestEntityTooLarge, "audio file exceeds 50 MB limit"
		}
		return "", http.StatusRequestEntityTooLarge, "file exceeds 20 MB limit"
	}

	detectedMIME := http.DetectContentType(fileData)
	if i := strings.IndexByte(detectedMIME, ';'); i >= 0 {
		detectedMIME = strings.TrimSpace(detectedMIME[:i])
	}

	if detectedMIME == "application/octet-stream" {
		log.Printf(
			"[Material] WARN: MIME=application/octet-stream for file=%s ext=%s — accepted by extension",
			filename, ext,
		)
		return formatType, 0, ""
	}
	if formatType == domain.MaterialFormatPDF {
		if detectedMIME != "application/pdf" && detectedMIME != "application/x-pdf" {
			log.Printf(
				"[Material] WARN: MIME mismatch for PDF file=%s ext=%s detected=%s expected=%s action=%s",
				filename, ext, detectedMIME, "application/pdf", "rejected",
			)
			return "", http.StatusUnsupportedMediaType, "file content does not match the .pdf extension"
		}
		return formatType, 0, ""
	}
	if formatType == domain.MaterialFormatTXT || formatType == domain.MaterialFormatMD {
		if detectedMIME != "text/plain" && detectedMIME != "application/octet-stream" {
			log.Printf(
				"[Material] WARN: MIME discrepancy for text file=%s ext=%s detected=%s action=%s",
				filename, ext, detectedMIME, "accepted",
			)
		}
		return formatType, 0, ""
	}
	if _, ok := allowedMaterialFormats[detectedMIME]; !ok {
		log.Printf(
			"[Material] WARN: MIME mismatch for file=%s ext=%s detected=%s action=%s",
			filename, ext, detectedMIME, "rejected",
		)
		return "", http.StatusUnsupportedMediaType, "unsupported file MIME type"
	}

	return formatType, 0, ""
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
	if _, err := uuid.Parse(topicID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course or topic not found"})
		return
	}

	materials, err := h.materialUseCase.GetMaterialsByTopic(c.Request.Context(), courseID, topicID, userID)
	if errors.Is(err, usecases.ErrMaterialForbidden) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
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
