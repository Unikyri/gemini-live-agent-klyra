package repositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// PostgresChunkRepository saves and retrieves MaterialChunk records using pgvector.
type PostgresChunkRepository struct {
	db *gorm.DB
}

// NewPostgresChunkRepository creates a new repository backed by the given DB.
func NewPostgresChunkRepository(db *gorm.DB) *PostgresChunkRepository {
	return &PostgresChunkRepository{db: db}
}

// SaveChunks replaces all chunks for the given material, then bulk-inserts the new ones.
func (r *PostgresChunkRepository) SaveChunks(ctx context.Context, chunks []domain.MaterialChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	materialID := chunks[0].MaterialID.String()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete old chunks for this material (clean-reindex strategy)
		if err := tx.Where("material_id = ?", materialID).Delete(&domain.MaterialChunk{}).Error; err != nil {
			return fmt.Errorf("chunk repo: delete old chunks: %w", err)
		}
		// Batch insert new chunks
		if err := tx.Create(&chunks).Error; err != nil {
			return fmt.Errorf("chunk repo: insert chunks: %w", err)
		}
		return nil
	})
}

// SearchSimilar finds the top-k chunks closest (by cosine similarity) to the query embedding,
// SCOPED to the given topicID.
// SECURITY: The topicID filter guarantees that a user can never retrieve chunks
// from topics they don't own, preventing cross-user RAG leakage.
func (r *PostgresChunkRepository) SearchSimilar(ctx context.Context, topicID string, queryEmbedding []float32, topK int) ([]domain.RAGResult, error) {
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("chunk repo: empty query embedding")
	}

	vectorLiteral := domain.PgVectorToLiteral(queryEmbedding)

	type row struct {
		domain.MaterialChunk
		Similarity float64
	}

	// Note: GORM's Order() accepts a single clause.Expr for raw SQL with args.
	// We use a raw query here for the pgvector <=> operator (not standard SQL).
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT *, 1 - (embedding <=> ?::vector) AS similarity
		FROM material_chunks
		WHERE topic_id = ?
		ORDER BY embedding <=> ?::vector
		LIMIT ?`,
		vectorLiteral, topicID, vectorLiteral, topK,
	).Scan(&rows).Error

	if err != nil {
		return nil, fmt.Errorf("chunk repo: similarity search: %w", err)
	}

	results := make([]domain.RAGResult, len(rows))
	for i, row := range rows {
		results[i] = domain.RAGResult{
			Chunk:      row.MaterialChunk,
			Similarity: row.Similarity,
		}
	}
	return results, nil
}

// GetChunksByTopic retrieves all chunks for a topic ordered by chunk_index.
// Used to assemble the full validated context for a topic (without a query vector).
func (r *PostgresChunkRepository) GetChunksByTopic(ctx context.Context, topicID string) ([]domain.MaterialChunk, error) {
	tid, err := uuid.Parse(topicID)
	if err != nil {
		return nil, fmt.Errorf("chunk repo: invalid topicID: %w", err)
	}

	var chunks []domain.MaterialChunk
	if err := r.db.WithContext(ctx).
		Where("topic_id = ?", tid).
		Order("chunk_index ASC").
		Find(&chunks).Error; err != nil {
		return nil, fmt.Errorf("chunk repo: get by topic: %w", err)
	}
	return chunks, nil
}
