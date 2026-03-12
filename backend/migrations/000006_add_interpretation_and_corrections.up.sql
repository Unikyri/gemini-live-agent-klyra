-- Migration: 000006_add_interpretation_and_corrections.up.sql
-- Adds interpretation JSON storage and material corrections (overrides).

-- 1) Add interpretation_json to materials.
ALTER TABLE materials
    ADD COLUMN IF NOT EXISTS interpretation_json JSONB;

-- 2) Extend materials.status check constraint to include 'interpreted'.
-- The original migration did not name the constraint explicitly, but PostgreSQL
-- typically generates `<table>_<column>_check`.
ALTER TABLE materials
    DROP CONSTRAINT IF EXISTS materials_status_check;

ALTER TABLE materials
    ADD CONSTRAINT materials_status_check
        CHECK (status IN ('pending', 'processing', 'validated', 'interpreted', 'rejected'));

-- 3) Create material_corrections table.
CREATE TABLE IF NOT EXISTS material_corrections (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    material_id    UUID NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
    chunk_id UUID NULL REFERENCES material_chunks(id) ON DELETE SET NULL,
    block_index    INT  NOT NULL,
    original_text  TEXT NOT NULL,
    corrected_text TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_material_corrections_material_block UNIQUE (material_id, block_index)
);

CREATE INDEX IF NOT EXISTS idx_material_corrections_material_id
    ON material_corrections(material_id);

CREATE INDEX IF NOT EXISTS idx_material_corrections_chunk_id
    ON material_corrections(chunk_id);

