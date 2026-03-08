# pgvector Klyra — Guía de Referencia Rápida

## ?? Estado Actual

? **Configuración Completada**:
- pgvector extension habilitada en PostgreSQL 15
- Tabla `material_chunks` creada con migration 000004
- Índice IVFFlat configurado para búsquedas rápidas (lists = 10)
- Dimensión: 768 (Vertex AI text-embedding-004)

## ?? Cómo Usar en el Código

### 1. Repository Initialization

```go
package main

import (
"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/infrastructure/repositories"
"gorm.io/gorm"
)

// En tu inicializador de servicios:
chunkRepo := repositories.NewChunkRepository(db) // db es tu instancia GORM
```

### 2. Insert Chunks with Embeddings

```go
chunk := &domain.MaterialChunk{
MaterialID: materialUUID,
TopicID:    topicUUID,
ChunkIndex: 0,
Content:    "Texto del material...",
Embedding:  embeddingVector, // domain.PgVector ([]float32)
}

err := chunkRepo.Create(chunk)
```

### 3. Similarity Search (KNN)

```go
// Buscar los 10 chunks más similares a un embedding
results, err := chunkRepo.SimilaritySearch(repositories.SimilaritySearchRequest{
Embedding: queryEmbedding, // domain.PgVector
Limit:     10,
Offset:    0,
TopicIDFilter: &topicID, // Opcional: filtrar por topic
MinSimilarity: 0.75,      // Opcional: threshold de similitud
})

for _, result := range results {
fmt.Printf("Sim: %.2f%% | Content: %s\n", 
result.Similarity * 100, 
result.Chunk.Content)
}
```

### 4. Get Chunks by Material (Full Context)

```go
// Recuperar TODOS los chunks de un material en orden
chunks, err := chunkRepo.GetChunksByMaterial(materialID)
// chunks están ordenados por chunk_index (preservan orden del documento)
```

## ?? Testing sin APIs

### Setup Test Data

```bash
# 1. Usar embeddings sintéticos en tus tests:
fixtures := domain.NewChunkFixtures()

# 2. Verificar que pgvector está activo:
bash backend/migrations/fixtures/validate-pgvector-setup.sh

# 3. Cargar fixtures SQL (si necesitas datos en DB):
PGPASSWORD=klyra_pass psql -h localhost -p 5433 -U klyra_user -d klyra_db \
  -f backend/migrations/fixtures/seed-pgvector-test-data.sql
```

### Go Test Fixtures

```go
import "github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"

func TestChunkSearch(t *testing.T) {
// Fixtures generan embeddings determinísticos sin APIs
fixtures := domain.NewChunkFixtures()

// Usar en tus tests:
chunk := &domain.MaterialChunk{
Embedding: fixtures.NeuroscienceEmbedding1,
}

// Validar similitud entre embeddings:
sim := domain.CosineSimilarity(
fixtures.NeuroscienceEmbedding1,
fixtures.NeuroscienceEmbedding2,
)
// sim should be high (~0.8-0.9) para embeddings del mismo dominio
}
```

## ?? Performance Esperado

| Métrica | Valor |
|---|---|
| **Query time** | <10ms para 10k chunks |
| **Build time (index)** | ~50ms |
| **Memory overhead** | ~10% del tamaño de datos |
| **Recall (IVFFlat)** | ~95% |

## ?? Troubleshooting

### "pgvector extension not found"
```sql
-- En psql:
CREATE EXTENSION IF NOT EXISTS vector;
```

### Index not being used in queries
```sql
-- Verificar con ANALYZE:
EXPLAIN ANALYZE
SELECT id, 1 - (embedding <=> '[...]'::vector) as sim
FROM material_chunks
ORDER BY embedding <=> '[...]'::vector
LIMIT 10;

-- Debe decir "Index Scan using idx_chunks_embedding_ivfflat"
-- Si dice "Seq Scan", fuerza reindex:
REINDEX INDEX idx_chunks_embedding_ivfflat;
```

### Query returns wrong results
```sql
-- Verificar que embeddings están normalizados (norma L2 = 1)
-- pgvector cosine funciona mejor con vectores normalizados
SELECT id, sqrt(sum(embedding ^ 2)) as norm 
FROM material_chunks 
LIMIT 5;
-- Deben ser values cercanos a 1.0
```

## ?? Escalado Futuro

### Si crece a >100k chunks:
1. Migrar IVFFlat ? HNSW (mayor recall)
   ```sql
   ALTER INDEX idx_chunks_embedding_ivfflat 
   RENAME TO idx_chunks_embedding_ivfflat_old;
   
   CREATE INDEX idx_chunks_embedding_hnsw
   ON material_chunks USING hnsw (embedding vector_cosine_ops)
   WITH (m = 16, ef_construction = 64);
   
   DROP INDEX idx_chunks_embedding_ivfflat_old;
   ```

2. Paritioning de tabla:
   ```sql
   ALTER TABLE material_chunks 
   PARTITION BY RANGE (created_at);
   ```

## ?? Archivos Clave

| Archivo | Propósito |
|---|---|
| `/backend/migrations/000004_add_pgvector_and_chunks.up.sql` | Schema y índices |
| `/backend/internal/infrastructure/repositories/chunk_repository.go` | GORM repository con KNN |
| `/backend/internal/core/domain/chunk.go` | Model MaterialChunk + PgVector |
| `/backend/internal/core/domain/test_fixtures.go` | Test helpers sin APIs |
| `/backend/migrations/fixtures/seed-pgvector-test-data.sql` | Fixture data |
| `/.docs/pgvector-configuration.md` | Documentación técnica completa |

## ? Comandos Útiles

```bash
# Ver stats de uso de índices
psql -d klyra_db -c "SELECT * FROM pg_stat_user_indexes WHERE tablename='material_chunks';"

# Analizar query plan
psql -d klyra_db -c "EXPLAIN ANALYZE SELECT ... FROM material_chunks ORDER BY embedding <=> ... LIMIT 10;"

# Reindex si es necesario
psql -d klyra_db -c "REINDEX INDEX idx_chunks_embedding_ivfflat;"

# Vacío y análisis para optimización
psql -d klyra_db -c "VACUUM ANALYZE material_chunks;"
```

---

**Versión**: 1.0  
**Última actualización**: 2026-03-08  
**Responsable**: DBA Engineer — Klyra Project
