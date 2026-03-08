//go:build integration

package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// TestCloudSQLChunkRepository_ConnectionTest validates that the CloudSQL adapter
// properly detects database availability and pgvector extension status.
// This test REQUIRES a running PostgreSQL with pgvector extension.
func TestCloudSQLChunkRepository_ConnectionTest(t *testing.T) {
	// Setup: Use the same test database infrastructure as PostgreSQL tests
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewCloudSQLChunkRepository(db)

	// Should succeed when DB is healthy
	if err := repo.CloudSQLConnectionTest(context.Background()); err != nil {
		t.Fatalf("CloudSQL connection test failed: %v", err)
	}
}

// TestCloudSQLChunkRepository_SaveAndSearchChunks verifies that the CloudSQL adapter
// correctly delegates all chunk operations to the underlying PostgreSQL implementation.
func TestCloudSQLChunkRepository_SaveAndSearchChunks(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewCloudSQLChunkRepository(db)
	ctx := context.Background()

	// Create test data
	materialID := uuid.New()
	topicID := uuid.New()

	chunks := []domain.MaterialChunk{
		{
			ID:         uuid.New(),
			MaterialID: materialID,
			TopicID:    topicID,
			ChunkIndex: 0,
			Content:    "Machine learning fundamentals",
			Embedding:  domain.PgVector([]float32{0.1, 0.2, 0.3, 0.4}),
		},
		{
			ID:         uuid.New(),
			MaterialID: materialID,
			TopicID:    topicID,
			ChunkIndex: 1,
			Content:    "Neural networks and deep learning",
			Embedding:  domain.PgVector([]float32{0.2, 0.3, 0.4, 0.5}),
		},
	}

	// Save chunks via CloudSQL repo
	if err := repo.SaveChunks(ctx, chunks); err != nil {
		t.Fatalf("SaveChunks failed: %v", err)
	}

	// Retrieve by topic
	retrieved, err := repo.GetChunksByTopic(ctx, topicID.String())
	if err != nil {
		t.Fatalf("GetChunksByTopic failed: %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(retrieved))
	}
}

// TestCloudSQLChunkRepository_TopicIDScoping verifies that the CloudSQL adapter
// enforces topicID boundaries, preventing cross-topic chunk leakage.
func TestCloudSQLChunkRepository_TopicIDScoping(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewCloudSQLChunkRepository(db)
	ctx := context.Background()

	// Create two separate topics with chunks
	topic1ID := uuid.New()
	topic2ID := uuid.New()
	materialID1 := uuid.New()
	materialID2 := uuid.New()

	chunks1 := []domain.MaterialChunk{
		{
			ID:         uuid.New(),
			MaterialID: materialID1,
			TopicID:    topic1ID,
			ChunkIndex: 0,
			Content:    "Topic 1 content",
			Embedding:  domain.PgVector([]float32{0.1, 0.2, 0.3}),
		},
	}

	chunks2 := []domain.MaterialChunk{
		{
			ID:         uuid.New(),
			MaterialID: materialID2,
			TopicID:    topic2ID,
			ChunkIndex: 0,
			Content:    "Topic 2 content",
			Embedding:  domain.PgVector([]float32{0.4, 0.5, 0.6}),
		},
	}

	if err := repo.SaveChunks(ctx, chunks1); err != nil {
		t.Fatalf("SaveChunks for topic1 failed: %v", err)
	}

	if err := repo.SaveChunks(ctx, chunks2); err != nil {
		t.Fatalf("SaveChunks for topic2 failed: %v", err)
	}

	// Verify topic1 only returns its own chunks
	topic1Chunks, err := repo.GetChunksByTopic(ctx, topic1ID.String())
	if err != nil {
		t.Fatalf("GetChunksByTopic for topic1 failed: %v", err)
	}

	if len(topic1Chunks) != 1 || topic1Chunks[0].TopicID != topic1ID {
		t.Errorf("topic1 should only see its own chunks, got %d chunks from other topics", len(topic1Chunks))
	}

	// Verify topic2 only returns its own chunks
	topic2Chunks, err := repo.GetChunksByTopic(ctx, topic2ID.String())
	if err != nil {
		t.Fatalf("GetChunksByTopic for topic2 failed: %v", err)
	}

	if len(topic2Chunks) != 1 || topic2Chunks[0].TopicID != topic2ID {
		t.Errorf("topic2 should only see its own chunks, got %d chunks from other topics", len(topic2Chunks))
	}
}

// TestCloudSQLChunkRepository_SimilaritySearchViaCloudSQL verifies that the CloudSQL adapter
// correctly delegates similarity search to pgvector.
func TestCloudSQLChunkRepository_SimilaritySearchViaCloudSQL(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewCloudSQLChunkRepository(db)
	ctx := context.Background()

	topicID := uuid.New()
	materialID := uuid.New()

	// Create test chunks with meaningful embeddings
	chunks := []domain.MaterialChunk{
		{
			ID:         uuid.New(),
			MaterialID: materialID,
			TopicID:    topicID,
			ChunkIndex: 0,
			Content:    "Apple is a fruit",
			Embedding:  domain.PgVector(domain.CreateTestEmbedding("apple", 768)),
		},
		{
			ID:         uuid.New(),
			MaterialID: materialID,
			TopicID:    topicID,
			ChunkIndex: 1,
			Content:    "Banana is also a fruit",
			Embedding:  domain.PgVector(domain.CreateTestEmbedding("banana", 768)),
		},
	}

	if err := repo.SaveChunks(ctx, chunks); err != nil {
		t.Fatalf("SaveChunks failed: %v", err)
	}

	// Search for similar chunks using an "apple" embedding
	queryEmbedding := domain.CreateTestEmbedding("apple", 768)
	results, err := repo.SearchSimilar(ctx, topicID.String(), queryEmbedding, 10)
	if err != nil {
		t.Fatalf("SearchSimilar failed: %v", err)
	}

	if len(results) < 1 {
		t.Errorf("expected at least 1 result, got %d", len(results))
	}

	// Highest similarity result should be the "apple" chunk itself
	if results[0].Chunk.Content != "Apple is a fruit" {
		t.Errorf("expected Apple chunk first, got %s", results[0].Chunk.Content)
	}
}

// setupTestDB initializes a connection to the test PostgreSQL database.
// This is shared infrastructure with PostgreSQL chunk repository tests.
func setupTestDB(t *testing.T) *gorm.DB {
	// In a real integration test, this would connect to a test PostgreSQL instance.
	// For now, we use the same setup as the PostgreSQL tests.
	dsn := "user=klyra_user password=klyra_pass host=localhost port=5432 dbname=klyra_db sslmode=disable"
	db, err := gorm.Open("postgres", dsn)
	if err != nil {
		t.Skipf("skipping integration test: PostgreSQL not available: %v", err)
	}

	// Ensure migrations are run
	if err := db.AutoMigrate(&domain.MaterialChunk{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

// teardownTestDB cleans up test data and closes the database connection.
func teardownTestDB(t *testing.T, db *gorm.DB) {
	// Clean up test chunks
	if err := db.Exec("DELETE FROM material_chunks WHERE created_at > now() - interval '1 minute'").Error; err != nil {
		t.Logf("warning: failed to clean up test chunks: %v", err)
	}

	// Close connection
	sqlDB, err := db.DB()
	if err != nil {
		t.Logf("warning: failed to get database connection: %v", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		t.Logf("warning: failed to close database connection: %v", err)
	}
}
