-- Migration: 000002_create_courses_and_topics.up.sql
-- Creates the courses and topics tables with UUID PKs, soft delete, and indexes.

CREATE TABLE IF NOT EXISTS courses (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                 VARCHAR(255) NOT NULL,
    education_level      VARCHAR(100),
    avatar_model_url     TEXT,
    -- avatar_status tracks async Imagen generation: pending → generating → ready | failed
    avatar_status        VARCHAR(50)  NOT NULL DEFAULT 'pending',
    reference_image_url  TEXT,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

-- Fast lookup of courses by user (most common query)
CREATE INDEX idx_courses_user_id     ON courses (user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_courses_deleted_at  ON courses (deleted_at);

CREATE TABLE IF NOT EXISTS topics (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id            UUID         NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title                VARCHAR(255) NOT NULL,
    order_index          INT          NOT NULL DEFAULT 0,
    -- consolidated_context stores the validated, AI-processed material for this topic.
    consolidated_context TEXT,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

CREATE INDEX idx_topics_course_id   ON topics (course_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_topics_deleted_at  ON topics (deleted_at);
