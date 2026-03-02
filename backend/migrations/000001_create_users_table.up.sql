-- Migration: 000001_create_users_table.up.sql
-- Creates the initial users table with UUID primary key and JSONB for the
-- learning profile (Memory Bank). Uses gen_random_uuid() PostgreSQL function.
-- Run: psql -d klyra_dev -f 000001_create_users_table.up.sql

CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- Required for gen_random_uuid()

CREATE TABLE IF NOT EXISTS users (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email            VARCHAR(255) NOT NULL UNIQUE,
    name             VARCHAR(255),
    profile_image_url TEXT,
    -- learning_profile stores the Memory Bank as flexible JSON.
    -- Example: {"prefers_visual": true, "preferred_style": "historical"}
    learning_profile JSONB        DEFAULT '{}',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ  -- NULL = active record (soft delete)
);

-- Index for soft-delete queries — filters out deleted records efficiently.
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
-- Index on email for fast lookups during sign-in.
CREATE INDEX idx_users_email ON users (email) WHERE deleted_at IS NULL;
