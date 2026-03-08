# PostgreSQL pgvector: Configuración & Performance Tuning

## ?? Visión General

La migration **000004** implementa un sistema de almacenamiento vectorial en PostgreSQL para búsqueda semántica de chunks mediante cosine distance.

### Dimensionalidad de Embeddings
- **768 dimensiones**: Vertex AI text-embedding-004 (actual)
- **1536 dimensiones**: OpenAI text-embedding-3-large (alternativa)
- **384 dimensiones**: Gemini Embedding API (más rápido)

## ??? Estructura de la Tabla

\\\sql
CREATE TABLE material_chunks (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    material_id  UUID NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
    topic_id     UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    chunk_index  INTEGER NOT NULL DEFAULT 0,
    content      TEXT NOT NULL,
    embedding    vector(768),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_chunk_material_idx UNIQUE (material_id, chunk_index)
);
\\\

### Decisiones de Diseño
- **CASCADE delete**: Limpia chunks automaticamente cuando se borra el material
- **Denormalizado topic_id**: Acceso rápido sin JOINs (common query pattern)
- **chunk_index**: Preserva orden del documento original (necesario para coherencia)
- **Unique constraint**: Previene chunks duplicados del mismo material

## ?? Estrategia de Índices

### IVFFlat Index (Recomendado para MVP)
\\\sql
CREATE INDEX idx_chunks_embedding_ivfflat
  ON material_chunks USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);  -- lists = sqrt(total_rows)
\\\

**Características:**
- Build time: ~50ms para 10k chunks
- Query time: <10ms
- Recall: ~95%
- Memory overhead: ~10%

### HNSW Index (Alternativa: Mayor Precisión)
\\\sql
CREATE INDEX idx_chunks_embedding_hnsw
  ON material_chunks USING hnsw (embedding vector_cosine_ops)
  WITH (m = 16, ef_construction = 64);
\\\

Para producción cuando necesites recall >98%

## ?? Query de Búsqueda de Similitud

\\\sql
-- KNN search: top-10 chunks más similares
SELECT 
  id, material_id, content,
  1 - (embedding <=> '\'::vector) as similarity
FROM material_chunks
WHERE material_id = '\'::uuid
ORDER BY embedding <=> '\'::vector
LIMIT 10;
\\\

**Operadores de pgvector:**
- <=> : Cosine distance (0=identical, 1=opposite)
- <@> : L2 distance
- <#> : Inner product distance

## ? Validación Post-Migration

\\\ash
# Verificar extension instalada
psql -d klyra_db -c "SELECT * FROM pg_extension WHERE extname='vector';"

# Verificar tabla
psql -d klyra_db -c "\\d+ material_chunks;"

# Verificar índices
psql -d klyra_db -c "SELECT indexname FROM pg_indexes WHERE tablename='material_chunks';"

# Test KNN query (con EXPLAIN)
psql -d klyra_db -c "
  EXPLAIN ANALYZE
  SELECT id, 1 - (embedding <=> '[0.1,0.2,...]'::vector) as sim 
  FROM material_chunks 
  ORDER BY embedding <=> '[0.1,0.2,...]'::vector 
  LIMIT 10;
"
\\\

## ?? Performance Esperada

| # Chunks | IVFFlat Query | HNSW Query | Recall |
|---|---|---|---|
| 1k | 2ms | 1ms | 95-99% |
| 10k | 5ms | 2ms | 93-98% |
| 100k | 8ms | 4ms | 90-97% |

## ??? Reversibilidad

La migration es completamente reversible:

**Para rollback:**
\\\sql
-- 000004_add_pgvector_and_chunks.down.sql
DROP TABLE IF EXISTS material_chunks;
DROP EXTENSION IF EXISTS vector;
\\\

---
**Versión**: 1.0  
**Fecha**: 2026-03-08  
**Autor**: DBA Engineer
