package domain

import (
	"time"

	"github.com/google/uuid"
)

// Course represents a learning course created by a student.
// Each course has a dedicated Avatar and an isolated AI Agent context.
type Course struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID         uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"` // owner
	Name           string    `json:"name" gorm:"not null"`
	EducationLevel string    `json:"education_level"` // e.g. "university", "high_school"
	// AvatarModelURL points to the transparent-background PNG in Cloud Storage.
	// Starts empty; filled asynchronously after Imagen generation completes.
	AvatarModelURL string `json:"avatar_model_url"`
	// AvatarStatus tracks the async generation process:
	// "pending" → "generating" → "ready" | "failed"
	AvatarStatus string `json:"avatar_status" gorm:"default:pending"`
	// ReferenceImageURL is the original image uploaded by the student for avatar generation.
	ReferenceImageURL string     `json:"reference_image_url"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty" gorm:"index"`

	// GORM associations — loaded on demand, not always included in API responses.
	Topics []Topic `json:"topics,omitempty" gorm:"foreignKey:CourseID"`
}

// Topic represents a sequential learning unit within a Course.
type Topic struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CourseID   uuid.UUID `json:"course_id" gorm:"type:uuid;not null;index"`
	Title      string    `json:"title" gorm:"not null"`
	OrderIndex int       `json:"order_index" gorm:"default:0"` // for sequencing
	// ConsolidatedContext stores the validated, processed content of all materials.
	ConsolidatedContext string     `json:"consolidated_context,omitempty"`
	SummaryMarkdown     string     `json:"summary_markdown,omitempty"`
	SummarySourceHash   string     `json:"summary_source_hash,omitempty" gorm:"index"`
	SummaryMaterialIDs  string     `json:"summary_material_ids,omitempty"`
	SummaryUpdatedAt    *time.Time `json:"summary_updated_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty" gorm:"index"`

	// Materials is the one-to-many relation used by GORM preload in course queries.
	Materials []Material `json:"materials,omitempty" gorm:"foreignKey:TopicID"`
}

// TopicSummaryCache represents the persisted summary state for a topic.
type TopicSummaryCache struct {
	TopicID            uuid.UUID
	SummaryMarkdown    string
	SummarySourceHash  string
	SummaryMaterialIDs []string
	SummaryUpdatedAt   *time.Time
}
