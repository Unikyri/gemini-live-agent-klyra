-- Migration: 000006_add_interpretation_and_corrections.down.sql

DROP TABLE IF EXISTS material_corrections;

ALTER TABLE materials
    DROP COLUMN IF EXISTS interpretation_json;

-- Revert status constraint to original set (pending, processing, validated, rejected)
ALTER TABLE materials
    DROP CONSTRAINT IF EXISTS materials_status_check;

ALTER TABLE materials
    ADD CONSTRAINT materials_status_check
        CHECK (status IN ('pending', 'processing', 'validated', 'rejected'));

