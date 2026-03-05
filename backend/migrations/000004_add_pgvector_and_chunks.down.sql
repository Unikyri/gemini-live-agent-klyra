-- Migration: 000004_add_pgvector_and_chunks.down.sql
DROP TABLE IF EXISTS material_chunks;
DROP EXTENSION IF EXISTS vector;
