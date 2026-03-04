-- Migration: 000002_create_courses_and_topics.down.sql
-- Reverses the courses & topics tables creation.

DROP INDEX IF EXISTS idx_topics_deleted_at;
DROP INDEX IF EXISTS idx_topics_course_id;
DROP TABLE IF EXISTS topics;

DROP INDEX IF EXISTS idx_courses_deleted_at;
DROP INDEX IF EXISTS idx_courses_user_id;
DROP TABLE IF EXISTS courses;
