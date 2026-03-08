//go:build integration
// +build integration

package repositories

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/infrastructure/database"
)

func TestSimilaritySearch_ReturnsTopK(t *testing.T) {
	db := mustOpenIntegrationDB(t)
	cleanupTables(t, db)

	repo := NewPostgresChunkRepository(db)
	fixture := seedGraph(t, db)

	chunks := make([]domain.MaterialChunk, 0, 10)
	for i := 0; i < 10; i++ {
		theta := float64(i) * 0.15
		chunks = append(chunks, domain.MaterialChunk{
			ID:         uuid.New(),
			MaterialID: fixture.materialID,
			TopicID:    fixture.topicID,
			ChunkIndex: i,
			Content:    fmt.Sprintf("chunk-%d", i),
			Embedding:  domain.PgVector(unitVectorByAngle(theta)),
		})
	}

	if err := repo.SaveChunks(context.Background(), chunks); err != nil {
		t.Fatalf("save chunks failed: %v", err)
	}

	query := makeEmbedding(768)
	query[0] = 1
	results, err := repo.SearchSimilar(context.Background(), fixture.topicID.String(), query, 5)
	if err != nil {
		t.Fatalf("search similar failed: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("expected top 5 results, got %d", len(results))
	}

	for i := 1; i < len(results); i++ {
		if results[i-1].Similarity < results[i].Similarity {
			t.Fatalf("results are not ordered by similarity desc at position %d", i)
		}
	}
	if results[0].Chunk.ChunkIndex != 0 {
		t.Fatalf("expected most similar chunk index 0, got %d", results[0].Chunk.ChunkIndex)
	}
}

func TestSimilaritySearch_TopicIDScoping(t *testing.T) {
	db := mustOpenIntegrationDB(t)
	cleanupTables(t, db)

	repo := NewPostgresChunkRepository(db)
	fixtureA := seedGraph(t, db)
	fixtureB := seedGraph(t, db)

	queryLike := domain.PgVector(makeEmbedding(768))
	queryLike[0] = 1

	chunksA := []domain.MaterialChunk{
		{ID: uuid.New(), MaterialID: fixtureA.materialID, TopicID: fixtureA.topicID, ChunkIndex: 0, Content: "A-secure", Embedding: queryLike},
	}
	chunksB := []domain.MaterialChunk{
		{ID: uuid.New(), MaterialID: fixtureB.materialID, TopicID: fixtureB.topicID, ChunkIndex: 0, Content: "B-secret-leak", Embedding: queryLike},
	}

	if err := repo.SaveChunks(context.Background(), chunksA); err != nil {
		t.Fatalf("save chunks A failed: %v", err)
	}
	if err := repo.SaveChunks(context.Background(), chunksB); err != nil {
		t.Fatalf("save chunks B failed: %v", err)
	}

	results, err := repo.SearchSimilar(context.Background(), fixtureA.topicID.String(), []float32(queryLike), 10)
	if err != nil {
		t.Fatalf("search similar failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for topic A")
	}
	for _, r := range results {
		if r.Chunk.TopicID != fixtureA.topicID {
			t.Fatalf("security regression: got chunk from different topic: %s", r.Chunk.TopicID)
		}
		if strings.Contains(r.Chunk.Content, "leak") {
			t.Fatal("security regression: leaked chunk content from another topic")
		}
	}
}

func TestSimilaritySearch_EmptyQueryVector(t *testing.T) {
	db := mustOpenIntegrationDB(t)
	cleanupTables(t, db)

	repo := NewPostgresChunkRepository(db)
	_, err := repo.SearchSimilar(context.Background(), uuid.New().String(), nil, 5)
	if err == nil {
		t.Fatal("expected error for empty query embedding")
	}
	if !strings.Contains(err.Error(), "empty query embedding") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetChunksByTopic_ReturnsOrderedByIndex(t *testing.T) {
	db := mustOpenIntegrationDB(t)
	cleanupTables(t, db)

	repo := NewPostgresChunkRepository(db)
	fixture := seedGraph(t, db)

	input := []domain.MaterialChunk{
		{ID: uuid.New(), MaterialID: fixture.materialID, TopicID: fixture.topicID, ChunkIndex: 2, Content: "two", Embedding: domain.PgVector(unitVectorByAngle(0.2))},
		{ID: uuid.New(), MaterialID: fixture.materialID, TopicID: fixture.topicID, ChunkIndex: 0, Content: "zero", Embedding: domain.PgVector(unitVectorByAngle(0.0))},
		{ID: uuid.New(), MaterialID: fixture.materialID, TopicID: fixture.topicID, ChunkIndex: 1, Content: "one", Embedding: domain.PgVector(unitVectorByAngle(0.1))},
	}

	if err := repo.SaveChunks(context.Background(), input); err != nil {
		t.Fatalf("save chunks failed: %v", err)
	}

	got, err := repo.GetChunksByTopic(context.Background(), fixture.topicID.String())
	if err != nil {
		t.Fatalf("get chunks by topic failed: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(got))
	}
	if got[0].ChunkIndex != 0 || got[1].ChunkIndex != 1 || got[2].ChunkIndex != 2 {
		t.Fatalf("expected ordered chunk indexes [0,1,2], got [%d,%d,%d]", got[0].ChunkIndex, got[1].ChunkIndex, got[2].ChunkIndex)
	}
}

func TestSaveChunks_ReplacesOldChunks(t *testing.T) {
	db := mustOpenIntegrationDB(t)
	cleanupTables(t, db)

	repo := NewPostgresChunkRepository(db)
	fixture := seedGraph(t, db)

	first := []domain.MaterialChunk{
		{ID: uuid.New(), MaterialID: fixture.materialID, TopicID: fixture.topicID, ChunkIndex: 0, Content: "old-0", Embedding: domain.PgVector(unitVectorByAngle(0.0))},
		{ID: uuid.New(), MaterialID: fixture.materialID, TopicID: fixture.topicID, ChunkIndex: 1, Content: "old-1", Embedding: domain.PgVector(unitVectorByAngle(0.1))},
		{ID: uuid.New(), MaterialID: fixture.materialID, TopicID: fixture.topicID, ChunkIndex: 2, Content: "old-2", Embedding: domain.PgVector(unitVectorByAngle(0.2))},
	}
	second := []domain.MaterialChunk{
		{ID: uuid.New(), MaterialID: fixture.materialID, TopicID: fixture.topicID, ChunkIndex: 0, Content: "new-0", Embedding: domain.PgVector(unitVectorByAngle(0.0))},
		{ID: uuid.New(), MaterialID: fixture.materialID, TopicID: fixture.topicID, ChunkIndex: 1, Content: "new-1", Embedding: domain.PgVector(unitVectorByAngle(0.1))},
	}

	if err := repo.SaveChunks(context.Background(), first); err != nil {
		t.Fatalf("first save failed: %v", err)
	}
	if err := repo.SaveChunks(context.Background(), second); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	got, err := repo.GetChunksByTopic(context.Background(), fixture.topicID.String())
	if err != nil {
		t.Fatalf("get chunks by topic failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 chunks after replacement, got %d", len(got))
	}
	for _, c := range got {
		if strings.HasPrefix(c.Content, "old-") {
			t.Fatalf("expected old chunks to be replaced, found %q", c.Content)
		}
	}
}

func TestSaveChunks_RollbackOnInsertFailure(t *testing.T) {
	db := mustOpenIntegrationDB(t)
	cleanupTables(t, db)

	repo := NewPostgresChunkRepository(db)
	fixture := seedGraph(t, db)

	stable := []domain.MaterialChunk{
		{ID: uuid.New(), MaterialID: fixture.materialID, TopicID: fixture.topicID, ChunkIndex: 0, Content: "stable-0", Embedding: domain.PgVector(unitVectorByAngle(0.0))},
		{ID: uuid.New(), MaterialID: fixture.materialID, TopicID: fixture.topicID, ChunkIndex: 1, Content: "stable-1", Embedding: domain.PgVector(unitVectorByAngle(0.1))},
	}
	if err := repo.SaveChunks(context.Background(), stable); err != nil {
		t.Fatalf("initial save failed: %v", err)
	}

	duplicateIndex := []domain.MaterialChunk{
		{ID: uuid.New(), MaterialID: fixture.materialID, TopicID: fixture.topicID, ChunkIndex: 0, Content: "dup-a", Embedding: domain.PgVector(unitVectorByAngle(0.2))},
		{ID: uuid.New(), MaterialID: fixture.materialID, TopicID: fixture.topicID, ChunkIndex: 0, Content: "dup-b", Embedding: domain.PgVector(unitVectorByAngle(0.3))},
	}

	err := repo.SaveChunks(context.Background(), duplicateIndex)
	if err == nil {
		t.Fatal("expected unique constraint error due to duplicate chunk_index")
	}

	got, getErr := repo.GetChunksByTopic(context.Background(), fixture.topicID.String())
	if getErr != nil {
		t.Fatalf("get chunks failed: %v", getErr)
	}
	if len(got) != 2 {
		t.Fatalf("expected rollback to keep original 2 chunks, got %d", len(got))
	}
	contents := []string{got[0].Content, got[1].Content}
	sort.Strings(contents)
	if contents[0] != "stable-0" || contents[1] != "stable-1" {
		t.Fatalf("expected original chunks after rollback, got %v", contents)
	}
}

type integrationFixture struct {
	userID     uuid.UUID
	courseID   uuid.UUID
	topicID    uuid.UUID
	materialID uuid.UUID
}

func mustOpenIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	repo, err := database.NewPostgreSQLRepository("localhost", "5433", "klyra_db", "klyra_user", "klyra_pass", "disable")
	if err != nil {
		t.Skipf("skipping integration test, PostgreSQL unavailable: %v", err)
	}
	db := repo.GetDB()
	if pingErr := repo.Ping(); pingErr != nil {
		t.Skipf("skipping integration test, db ping failed: %v", pingErr)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	return db
}

func cleanupTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	err := db.Exec(`
		TRUNCATE TABLE material_chunks, materials, topics, courses, users
		RESTART IDENTITY CASCADE;
	`).Error
	if err != nil {
		t.Fatalf("cleanup tables failed: %v", err)
	}
}

func seedGraph(t *testing.T, db *gorm.DB) integrationFixture {
	t.Helper()

	fx := integrationFixture{
		userID:     uuid.New(),
		courseID:   uuid.New(),
		topicID:    uuid.New(),
		materialID: uuid.New(),
	}

	if err := db.Exec(`
		INSERT INTO users (id, email, name)
		VALUES (?, ?, ?)
	`, fx.userID, fmt.Sprintf("user-%s@example.com", fx.userID.String()[:8]), "integration user").Error; err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	if err := db.Exec(`
		INSERT INTO courses (id, user_id, name, education_level, avatar_status)
		VALUES (?, ?, ?, ?, ?)
	`, fx.courseID, fx.userID, "Integration Course", "university", "pending").Error; err != nil {
		t.Fatalf("insert course failed: %v", err)
	}

	if err := db.Exec(`
		INSERT INTO topics (id, course_id, title, order_index)
		VALUES (?, ?, ?, ?)
	`, fx.topicID, fx.courseID, "Integration Topic", 1).Error; err != nil {
		t.Fatalf("insert topic failed: %v", err)
	}

	if err := db.Exec(`
		INSERT INTO materials (id, topic_id, format_type, storage_url, extracted_text, status, original_name, size_bytes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, fx.materialID, fx.topicID, "txt", "gs://integration/material.txt", "seed text", "validated", "material.txt", 123).Error; err != nil {
		t.Fatalf("insert material failed: %v", err)
	}

	return fx
}

func makeEmbedding(size int) []float32 {
	v := make([]float32, size)
	return v
}

func unitVectorByAngle(theta float64) []float32 {
	v := makeEmbedding(768)
	v[0] = float32(math.Cos(theta))
	v[1] = float32(math.Sin(theta))
	return v
}
