-- Migration: 000005_add_topic_summary_cache.up.sql
-- Persists generated summary cache metadata at topic level.

ALTER TABLE topics
    ADD COLUMN IF NOT EXISTS summary_markdown TEXT,
    ADD COLUMN IF NOT EXISTS summary_source_hash VARCHAR(128),
    ADD COLUMN IF NOT EXISTS summary_material_ids TEXT,
    ADD COLUMN IF NOT EXISTS summary_updated_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_topics_summary_source_hash
    ON topics (summary_source_hash)
    WHERE deleted_at IS NULL;