package domain

import (
	"time"

	"github.com/google/uuid"
)

// MaterialCorrection stores a user-provided override for a specific interpreted block.
type MaterialCorrection struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MaterialID    uuid.UUID  `json:"material_id" gorm:"type:uuid;not null;index"`
	ChunkID       *uuid.UUID `json:"chunk_id,omitempty" gorm:"type:uuid;index"`
	BlockIndex    int        `json:"block_index" gorm:"not null"`
	OriginalText  string     `json:"original_text" gorm:"not null"`
	CorrectedText string     `json:"corrected_text" gorm:"not null"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

