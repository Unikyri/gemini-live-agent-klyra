package domain

import (
	"crypto/md5"
	"math"
)

// FakeSimilarityEmbedding generates a deterministic embedding for testing without API calls
// Uses MD5 hash of seed string to generate consistent float values
// Vector is L2-normalized to unit length (required for cosine distance)
func FakeSimilarityEmbedding(seed string, dimensions int) PgVector {
hash := md5.Sum([]byte(seed))

// Convert hash bytes to float stream
embedding := make(PgVector, dimensions)
for i := 0; i < dimensions; i++ {
// Use hash bytes cyclically + index number for variation
hashByte := float32(hash[i%len(hash)])
embedding[i] = float32(math.Cos(float64(hashByte)*float64(i+1))) * 0.5
}

// L2 normalize to unit vector
return normalize(embedding)
}

// normalize returns a vector with unit length (L2 norm = 1)
func normalize(v PgVector) PgVector {
var sum float32 = 0
for _, val := range v {
sum += val * val
}
magnitude := float32(math.Sqrt(float64(sum)))

if magnitude == 0 {
return v
}

result := make(PgVector, len(v))
for i, val := range v {
result[i] = val / magnitude
}
return result
}

// CosineSimilarity computes cosine distance between two vectors
// Returns value in [0, 1] where 1 = identical, 0 = orthogonal, -1 = opposite
func CosineSimilarity(a, b PgVector) float64 {
if len(a) != len(b) || len(a) == 0 {
return 0
}

var dotProduct, normA, normB float64
for i := range a {
dotProduct += float64(a[i]) * float64(b[i])
normA += float64(a[i]) * float64(a[i])
normB += float64(b[i]) * float64(b[i])
}

if normA == 0 || normB == 0 {
return 0
}

return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// TestFixtures for Chunk repository tests
type ChunkFixtures struct {
NeuroscienceEmbedding1 PgVector // sinapsis
NeuroscienceEmbedding2 PgVector // neurotransmisores
ProgrammingEmbedding1  PgVector // Python
ProgrammingEmbedding2  PgVector // JavaScript
}

// NewChunkFixtures creates deterministic test embeddings
func NewChunkFixtures() ChunkFixtures {
return ChunkFixtures{
NeuroscienceEmbedding1: FakeSimilarityEmbedding("neuroscience_sinapsis", 768),
NeuroscienceEmbedding2: FakeSimilarityEmbedding("neuroscience_neurotransmitters", 768),
ProgrammingEmbedding1:  FakeSimilarityEmbedding("programming_python", 768),
ProgrammingEmbedding2:  FakeSimilarityEmbedding("programming_javascript", 768),
}
}

// ExpectedSimilarity returns approximate expected cosine distance
// Helps validate that index is working (should return similar vectors together)
func (f ChunkFixtures) ExpectedSimilarity(a, b PgVector) float64 {
return CosineSimilarity(a, b)
}

// Example usage in tests:
/*
func TestChunkRepository_SimilaritySearch(t *testing.T) {
fixtures := NewChunkFixtures()

// Query with neuroscience embedding
// Should return chunks 1,2 (neuroscience) before chunk 3,4 (programming)
results, err := repo.SimilaritySearch(SimilaritySearchRequest{
Embedding: fixtures.NeuroscienceEmbedding1,
Limit:     10,
})

require.NoError(t, err)
require.Len(t, results, 4) // assuming 4 chunks in DB

// Validate ordering: neuroscience chunks should be first
sim01 := CosineSimilarity(fixtures.NeuroscienceEmbedding1, results[0].Chunk.Embedding)
sim02 := CosineSimilarity(fixtures.NeuroscienceEmbedding1, results[1].Chunk.Embedding)
sim03 := CosineSimilarity(fixtures.NeuroscienceEmbedding1, results[2].Chunk.Embedding)

require.Greater(t, sim01, sim02) // first should be more similar
require.Greater(t, sim02, sim03) // second should still be closer
}
*/