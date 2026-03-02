package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a student registered in Klyra.
// It uses UUID as primary key to prevent enumeration attacks.
type User struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email           string     `json:"email" gorm:"uniqueIndex;not null"`
	Name            string     `json:"name"`
	ProfileImageURL string     `json:"profile_image_url"`
	// LearningProfile stores the user's learning preferences (Memory Bank).
	// Using JSONB for flexible, schema-less storage at the DB level.
	LearningProfile  map[string]interface{} `json:"learning_profile,omitempty" gorm:"type:jsonb;serializer:json"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	DeletedAt        *time.Time             `json:"deleted_at,omitempty" gorm:"index"` // Soft delete
}
