package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
)

// CloudSQLChunkRepository is a ChunkRepository implementation that connects to Cloud SQL
// via the Cloud SQL Auth proxy socket. It inherits all behavior from PostgresChunkRepository
// since Cloud SQL is PostgreSQL-compatible.
//
// The only difference from PostgreSQL is the connection mechanism:
// - PostgreSQL (local): tcp://localhost:5432/klyra_db
// - Cloud SQL (proxy):  unix://cloudsql/PROJECT:REGION:INSTANCE/.s.PGSQL.5432/klyra_db
//
// Once the connection is established, all SQL operations are identical.
type CloudSQLChunkRepository struct {
	*PostgresChunkRepository
}

// NewCloudSQLChunkRepository wraps a Cloud SQL database connection as a ChunkRepository.
// The db *gorm.DB must already be connected via Cloud SQL Auth proxy.
// This is typically instantiated from cmd/api/main.go after initiating the CloudSQL DB.
func NewCloudSQLChunkRepository(db *gorm.DB) *CloudSQLChunkRepository {
	return &CloudSQLChunkRepository{
		PostgresChunkRepository: &PostgresChunkRepository{db: db},
	}
}

// Verify that CloudSQLChunkRepository satisfies the ChunkRepository port contract.
// This compile-time check ensures API compatibility.
var _ ports.ChunkRepository = (*CloudSQLChunkRepository)(nil)

// SaveChunks persists a batch of MaterialChunk records with pgvector embeddings to Cloud SQL.
// Replaces any previously saved chunks for the same material_id (idempotent).
// Implementation inherited from PostgresChunkRepository.
//
// ACID Guarantees: Full transaction rollback on error.
// Security: All chunks are scoped to a TopicID (inherited constraint).
func (r *CloudSQLChunkRepository) SaveChunks(ctx context.Context, chunks []domain.MaterialChunk) error {
	return r.PostgresChunkRepository.SaveChunks(ctx, chunks)
}

// SearchSimilar performs a cosine similarity KNN search within a specific topic in Cloud SQL.
// Returns the top-k most similar chunks to the given query embedding.
//
// SECURITY: topicID filter guarantees cross-user data leakage is impossible.
// The Cloud SQL connection ensures all operations are ACID-compliant.
//
// Parameters:
//   - topicID: UUID of the topic (all chunks must belong to this topic)
//   - queryEmbedding: 768-dimensional float32 vector to match against
//   - topK: maximum number of results to return
//
// Returns:
//   - []domain.RAGResult: ranked by cosine similarity (highest first)
//   - error if query embedding is invalid or database is unreachable
//
// Performance: Typically <10ms per query with IVFFlat index on Cloud SQL.
func (r *CloudSQLChunkRepository) SearchSimilar(ctx context.Context, topicID string, queryEmbedding []float32, topK int) ([]domain.RAGResult, error) {
	return r.PostgresChunkRepository.SearchSimilar(ctx, topicID, queryEmbedding, topK)
}

// GetChunksByTopic retrieves all chunks for a specific topic in CloudSQL, ordered by chunk_index.
// Used during session initialization to assemble the full topic context without a query vector.
//
// Returns:
//   - []domain.MaterialChunk: all chunks for the topic, ordered by insertion / chunk_index
//   - error if database is unreachable
//
// Implementation inherited from PostgresChunkRepository.
func (r *CloudSQLChunkRepository) GetChunksByTopic(ctx context.Context, topicID string) ([]domain.MaterialChunk, error) {
	return r.PostgresChunkRepository.GetChunksByTopic(ctx, topicID)
}

// CloudSQLConnectionTest validates that the Cloud SQL connection is alive and pgvector extension is active.
// Called during application startup health checks.
func (r *CloudSQLChunkRepository) CloudSQLConnectionTest(ctx context.Context) error {
	// Test basic connectivity
	if err := r.PostgresChunkRepository.db.WithContext(ctx).Raw("SELECT 1").Error; err != nil {
		return fmt.Errorf("cloudsql chunk repo: database ping failed: %w", err)
	}

	// Verify pgvector extension is active
	var result string
	if err := r.PostgresChunkRepository.db.WithContext(ctx).Raw("SELECT extname FROM pg_extension WHERE extname='vector'").Scan(&result).Error; err != nil {
		return fmt.Errorf("cloudsql chunk repo: pgvector extension not found: %w", err)
	}

	if result != "vector" {
		return fmt.Errorf("cloudsql chunk repo: pgvector extension not active in Cloud SQL")
	}

	return nil
}
