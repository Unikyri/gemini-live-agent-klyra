package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// MaterialFormatType enumerates the accepted file formats for learning materials.
type MaterialFormatType string

const (
	MaterialFormatPDF   MaterialFormatType = "pdf"
	MaterialFormatTXT   MaterialFormatType = "txt"
	MaterialFormatMD    MaterialFormatType = "md"
	MaterialFormatPNG   MaterialFormatType = "png"
	MaterialFormatJPG   MaterialFormatType = "jpg"
	MaterialFormatJPEG  MaterialFormatType = "jpeg"
	MaterialFormatWEBP  MaterialFormatType = "webp"
	MaterialFormatAudio MaterialFormatType = "audio"
)

// MaterialStatus tracks the processing lifecycle of an uploaded material.
// pending → processing → validated | rejected
type MaterialStatus string

const (
	MaterialStatusPending    MaterialStatus = "pending"
	MaterialStatusProcessing MaterialStatus = "processing"
	MaterialStatusValidated  MaterialStatus = "validated"
	MaterialStatusInterpreted MaterialStatus = "interpreted"
	MaterialStatusRejected   MaterialStatus = "rejected"
)

// Material represents a learning document uploaded by the student for a specific Topic.
// All text extraction and embedding generation happens asynchronously after upload.
type Material struct {
	ID         uuid.UUID          `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TopicID    uuid.UUID          `json:"topic_id" gorm:"type:uuid;not null;index"` // parent topic
	FormatType MaterialFormatType `json:"format_type" gorm:"not null"`
	// StorageURL is the GCS URL where the raw file is stored.
	StorageURL string `json:"storage_url" gorm:"not null"`
	// ExtractedText holds the plain-text content used for RAG embedding.
	// Populated asynchronously; empty while status is "pending" or "processing".
	ExtractedText string         `json:"extracted_text,omitempty"`
	// InterpretationJSON stores structured multimodal interpretation (blocks, LaTeX, transcripts).
	// Populated asynchronously when Gemini interpretation is enabled.
	InterpretationJSON datatypes.JSON `json:"interpretation_json,omitempty" gorm:"type:jsonb"`
	Status        MaterialStatus `json:"status" gorm:"default:pending"`
	OriginalName  string         `json:"original_name"` // original filename from the upload
	SizeBytes     int64          `json:"size_bytes"`    // file size in bytes
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     *time.Time     `json:"deleted_at,omitempty" gorm:"index"`
}
