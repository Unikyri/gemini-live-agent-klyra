-- Migration: 000001_create_users_table.down.sql
-- Reverses the users table creation. Safe to run to rollback the migration.

DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP TABLE IF EXISTS users;
