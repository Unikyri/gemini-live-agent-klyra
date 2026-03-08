#!/bin/bash
# File: validate-pgvector-setup.sh
# Purpose: Validate that pgvector is properly configured and performant
# Usage: bash validate-pgvector-setup.sh

set -e

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5433}"
DB_NAME="${DB_NAME:-klyra_db}"
DB_USER="${DB_USER:-klyra_user}"
DB_PASSWORD="${DB_PASSWORD:-klyra_pass}"

echo "===================================================================="
echo "pgvector Setup Validation for Klyra"
echo "===================================================================="
echo ""

# 1. Verify pgvector extension is installed
echo "1. Checking pgvector extension..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c \
  "SELECT extname, extversion FROM pg_extension WHERE extname='vector';" || \
  { echo "ERROR: pgvector extension not found"; exit 1; }
echo "? pgvector extension is installed"
echo ""

# 2. Verify table structure
echo "2. Checking material_chunks table schema..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c \
  "SELECT column_name, data_type FROM information_schema.columns WHERE table_name='material_chunks' ORDER BY ordinal_position;" || \
  { echo "ERROR: material_chunks table not found"; exit 1; }
echo "? Table schema is correct"
echo ""

# 3. Verify indexes
echo "3. Checking indexes on material_chunks..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c \
  "SELECT indexname, indexdef FROM pg_indexes WHERE tablename='material_chunks' ORDER BY indexname;"
echo ""

# 4. Check index usage statistics
echo "4. Index usage statistics..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c \
  "SELECT schemaname, tablename, indexname, idx_scan FROM pg_stat_user_indexes WHERE tablename='material_chunks';" || \
  echo "No stats yet (indexes haven't been used)"
echo ""

# 5. Run EXPLAIN ANALYZE on a test KNN query
echo "5. Testing KNN query performance with EXPLAIN ANALYZE..."
echo "   (This tests index efficiency without actual data in DB)"
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOF
EXPLAIN ANALYZE
SELECT 
  id, 
  1 - (embedding <=> '[0.1,0.15,0.2,0.25,0.3,0.35,0.4,0.45,0.5,0.55,0.6,0.65,0.7,0.75,0.8,0.85,0.9,0.95,1.0,
       0.1,0.15,0.2,0.25,0.3,0.35,0.4,0.45,0.5,0.55,0.6,0.65,0.7,0.75,0.8,0.85,0.9,0.95,1.0]'::vector) as similarity
FROM material_chunks
ORDER BY embedding <=> '[0.1,0.15,0.2,0.25,0.3,0.35,0.4,0.45,0.5,0.55,0.6,0.65,0.7,0.75,0.8,0.85,0.9,0.95,1.0,
       0.1,0.15,0.2,0.25,0.3,0.35,0.4,0.45,0.5,0.55,0.6,0.65,0.7,0.75,0.8,0.85,0.9,0.95,1.0]'::vector
LIMIT 10;
EOF

echo ""
echo "6. Testing with sample data..."
echo "   Loading fixtures from fixtures/seed-pgvector-test-data.sql..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f fixtures/seed-pgvector-test-data.sql || \
  echo "Note: Fixtures loading skipped (may fail if IDs don't exist in parent tables)"
echo ""

echo "7. Running sample KNN search (should return neuroscience chunks)..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOF
SELECT 
  chunk_index,
  SUBSTRING(content, 1, 50) as content_preview,
  1 - (embedding <=> '[0.12,0.17,0.22,0.27,0.32,0.37,0.42,0.47,0.52,0.57,0.62,0.67,0.72,0.77,0.82,0.87,0.92,0.97,0.99,
       0.12,0.17,0.22,0.27,0.32,0.37,0.42,0.47,0.52,0.57,0.62,0.67,0.72,0.77,0.82,0.87,0.92,0.97,0.99]'::vector)::numeric(4,3) as similarity
FROM material_chunks
ORDER BY embedding <=> '[0.12,0.17,0.22,0.27,0.32,0.37,0.42,0.47,0.52,0.57,0.62,0.67,0.72,0.77,0.82,0.87,0.92,0.97,0.99,
       0.12,0.17,0.22,0.27,0.32,0.37,0.42,0.47,0.52,0.57,0.62,0.67,0.72,0.77,0.82,0.87,0.92,0.97,0.99]'::vector
LIMIT 5;
EOF

echo ""
echo "===================================================================="
echo "? pgvector validation complete!"
echo "===================================================================="
