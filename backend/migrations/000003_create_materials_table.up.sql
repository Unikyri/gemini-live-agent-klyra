-- Migration: 000003_create_materials_table.up.sql
-- Creates the materials table for tracking learning resources uploaded to GCS.

CREATE TABLE IF NOT EXISTS materials (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic_id      UUID        NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    format_type   VARCHAR(20) NOT NULL CHECK (format_type IN ('pdf', 'txt', 'md', 'audio')),
    storage_url   TEXT        NOT NULL,
    extracted_text TEXT,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending', 'processing', 'validated', 'rejected')),
    original_name TEXT        NOT NULL,
    size_bytes    BIGINT      NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

-- Index for fast lookups by topic (primary access pattern in GetMaterialsByTopic).
CREATE INDEX IF NOT EXISTS idx_materials_topic_id ON materials(topic_id) WHERE deleted_at IS NULL;
