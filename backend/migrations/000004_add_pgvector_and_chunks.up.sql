-- Migration: 000004_add_pgvector_and_chunks.up.sql
-- Adds the pgvector extension and stores text embeddings alongside chunks.
-- This enables Cosine similarity search entirely in PostgreSQL
-- without needing a separate Vertex AI Vector Search index.

-- Enable the pgvector extension (requires Cloud SQL with pgvector enabled)
CREATE EXTENSION IF NOT EXISTS vector;

-- Stores text chunks derived from materials, with their vector embedding.
CREATE TABLE IF NOT EXISTS material_chunks (
    id           UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    material_id  UUID            NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
    topic_id     UUID            NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    chunk_index  INTEGER         NOT NULL DEFAULT 0,
    content      TEXT            NOT NULL,
    -- 768-dimensional vector produced by text-embedding-004 (Vertex AI).
    -- If the model changes, drop+recreate this column with the new dimension.
    embedding    vector(768),
    created_at   TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_chunk_material_idx UNIQUE (material_id, chunk_index)
);

-- IVFFlat index for approximate nearest-neighbor search.
-- For production use HNSW; IVFFlat is faster to build for MVP.
-- lists = sqrt(total_rows) is a good starting heuristic.
CREATE INDEX IF NOT EXISTS idx_chunks_embedding_ivfflat
    ON material_chunks USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 10);

-- Regular index to quickly retrieve all chunks for a topic (for context assembly)
CREATE INDEX IF NOT EXISTS idx_chunks_topic_id
    ON material_chunks (topic_id);
