package domain

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MaterialChunk represents a text segment extracted from a Material,
// ready to be stored as a vector embedding for RAG retrieval.
type MaterialChunk struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MaterialID uuid.UUID `json:"material_id" gorm:"type:uuid;not null;index"`
	TopicID    uuid.UUID `json:"topic_id" gorm:"type:uuid;not null;index"`
	ChunkIndex int       `json:"chunk_index" gorm:"default:0"`
	Content    string    `json:"content" gorm:"not null"`
	// Embedding is stored as a pgvector vector(768) column.
	// 768 dimensions matches Vertex AI text-embedding-004.
	Embedding PgVector  `json:"embedding,omitempty" gorm:"type:vector(768)"`
	CreatedAt time.Time `json:"created_at"`
}

// PgVector is a custom slice type that GORM serialises as a pgvector literal.
// Format required by pgvector: '[0.1,0.2,...,0.768]'
type PgVector []float32

// Value implements driver.Valuer so GORM can write a pgvector literal.
func (v PgVector) Value() (driver.Value, error) {
	if v == nil {
		return nil, nil
	}
	return PgVectorToLiteral([]float32(v)), nil
}

// Scan implements sql.Scanner so GORM can read back a pgvector literal.
func (v *PgVector) Scan(value interface{}) error {
	if value == nil {
		*v = nil
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("pgVector: expected string, got %T", value)
	}
	str = strings.TrimPrefix(str, "[")
	str = strings.TrimSuffix(str, "]")
	parts := strings.Split(str, ",")
	floats := make([]float32, len(parts))
	for i, p := range parts {
		var f float64
		if _, err := fmt.Sscanf(strings.TrimSpace(p), "%f", &f); err != nil {
			return fmt.Errorf("pgVector: parse error at index %d: %w", i, err)
		}
		floats[i] = float32(f)
	}
	*v = floats
	return nil
}

// PgVectorToLiteral converts a float32 slice to a pgvector string literal '[a,b,c,...]'
func PgVectorToLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// RAGResult bundles a retrieved chunk with its similarity score.
type RAGResult struct {
	Chunk      MaterialChunk
	Similarity float64
}
