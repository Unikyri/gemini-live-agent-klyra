-- Migration: 000005_add_topic_summary_cache.down.sql

DROP INDEX IF EXISTS idx_topics_summary_source_hash;

ALTER TABLE topics
    DROP COLUMN IF EXISTS summary_updated_at,
    DROP COLUMN IF EXISTS summary_material_ids,
    DROP COLUMN IF EXISTS summary_source_hash,
    DROP COLUMN IF EXISTS summary_markdown;