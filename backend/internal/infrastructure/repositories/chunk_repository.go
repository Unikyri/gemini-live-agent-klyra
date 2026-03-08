package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// ChunkRepository handles persistence for MaterialChunk records and vector similarity searches.
type ChunkRepository struct {
db *gorm.DB
}

// NewChunkRepository creates a new chunk repository instance.
func NewChunkRepository(db *gorm.DB) *ChunkRepository {
return &ChunkRepository{db: db}
}

// Create persists a new MaterialChunk with its embedding vector.
func (r *ChunkRepository) Create(chunk *domain.MaterialChunk) error {
if chunk.ID == uuid.Nil {
chunk.ID = uuid.New()
}

result := r.db.Create(chunk)
if result.Error != nil {
return fmt.Errorf("failed to create chunk: %w", result.Error)
}
return nil
}

// BatchCreateChunks inserts multiple chunks in a single transaction (batch size 1000).
func (r *ChunkRepository) BatchCreateChunks(chunks []*domain.MaterialChunk) error {
if len(chunks) == 0 {
return nil
}

result := r.db.CreateInBatches(chunks, 1000)
if result.Error != nil {
return fmt.Errorf("failed to batch create chunks: %w", result.Error)
}
return nil
}

// SimilaritySearchRequest bundles parameters for KNN search
type SimilaritySearchRequest struct {
Embedding            domain.PgVector
Limit                int
Offset               int
MaterialIDFilter     *uuid.UUID
TopicIDFilter        *uuid.UUID
MinSimilarity        float64
}

// SimilaritySearchResult contains a chunk and its similarity score
type SimilaritySearchResult struct {
Chunk      domain.MaterialChunk
Similarity float64
}

// SimilaritySearch performs KNN search using pgvector cosine distance.
// Performance: <10ms for 10k+ documents with IVFFlat index
func (r *ChunkRepository) SimilaritySearch(req SimilaritySearchRequest) ([]SimilaritySearchResult, error) {
if req.Limit == 0 {
req.Limit = 10
}

query := r.db.
Select(`
id, material_id, topic_id, chunk_index, content, embedding, created_at,
1 - (embedding <=> ?) as similarity
`, domain.PgVectorToLiteral(req.Embedding)).
Table("material_chunks")

if req.MaterialIDFilter != nil {
query = query.Where("material_id = ?", *req.MaterialIDFilter)
}
if req.TopicIDFilter != nil {
query = query.Where("topic_id = ?", *req.TopicIDFilter)
}
if req.MinSimilarity > 0 {
query = query.Where("1 - (embedding <=> ?) >= ?",
domain.PgVectorToLiteral(req.Embedding), req.MinSimilarity)
}

query = query.
Order(gorm.Expr("embedding <=> ?", domain.PgVectorToLiteral(req.Embedding))).
Limit(req.Limit).
Offset(req.Offset)

var results []SimilaritySearchResult
if err := query.Scan(&results).Error; err != nil {
return nil, fmt.Errorf("similarity search failed: %w", err)
}

return results, nil
}

// GetChunksByMaterial retrieves all chunks for a specific material in order.
func (r *ChunkRepository) GetChunksByMaterial(materialID uuid.UUID) ([]*domain.MaterialChunk, error) {
var chunks []*domain.MaterialChunk
result := r.db.
Where("material_id = ?", materialID).
Order("chunk_index ASC").
Find(&chunks)

if result.Error != nil {
return nil, fmt.Errorf("failed to get chunks for material: %w", result.Error)
}
return chunks, nil
}

// GetChunksByTopic retrieves all chunks for a specific topic.
func (r *ChunkRepository) GetChunksByTopic(topicID uuid.UUID) ([]*domain.MaterialChunk, error) {
var chunks []*domain.MaterialChunk
result := r.db.
Where("topic_id = ?", topicID).
Order("material_id, chunk_index ASC").
Find(&chunks)

if result.Error != nil {
return nil, fmt.Errorf("failed to get chunks for topic: %w", result.Error)
}
return chunks, nil
}

// GetChunkByID retrieves a single chunk by UUID.
func (r *ChunkRepository) GetChunkByID(id uuid.UUID) (*domain.MaterialChunk, error) {
var chunk domain.MaterialChunk
result := r.db.Where("id = ?", id).First(&chunk)
if result.Error != nil {
if result.Error == gorm.ErrRecordNotFound {
return nil, nil
}
return nil, fmt.Errorf("failed to get chunk: %w", result.Error)
}
return &chunk, nil
}

// DeleteChunksByMaterial deletes all chunks for a material (ON DELETE CASCADE also applies).
func (r *ChunkRepository) DeleteChunksByMaterial(materialID uuid.UUID) error {
result := r.db.Delete(&domain.MaterialChunk{}, "material_id = ?", materialID)
if result.Error != nil {
return fmt.Errorf("failed to delete chunks for material: %w", result.Error)
}
return nil
}

// CountChunksByMaterial returns the number of chunks for a material.
func (r *ChunkRepository) CountChunksByMaterial(materialID uuid.UUID) (int64, error) {
var count int64
result := r.db.Model(&domain.MaterialChunk{}).Where("material_id = ?", materialID).Count(&count)
if result.Error != nil {
return 0, fmt.Errorf("failed to count chunks: %w", result.Error)
}
return count, nil
}
