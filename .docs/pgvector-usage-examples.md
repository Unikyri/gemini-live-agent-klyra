# pgvector en Klyra — Ejemplos de Uso Real

## ?? Caso 1: RAG Search para Pregunta del Estudiante

```go
// Handler: POST /api/materials/{topicID}/rag-search
func (h *MaterialHandler) RAGSearch(c *gin.Context) {
    topicID := c.Param("topicID")
    query := c.PostForm("query")  // "¿Qué son los neurotransmisores?"
    
    // 1. Generar embedding de la pregunta (llamar a Vertex AI)
    embedding, err := h.embeddingService.Embed(query)
    if err != nil {
        c.JSON(500, gin.H{"error": "embedding failed"})
        return
    }
    
    // 2. Búsqueda KNN en pgvector
    results, err := h.chunkRepo.SimilaritySearch(repositories.SimilaritySearchRequest{
        Embedding:     embedding,          // 768-dimensional vector
        Limit:         5,                  // Top-5 chunks más relevantes
        TopicIDFilter: &topicID,           // Solo de este topic
        MinSimilarity: 0.7,                // Evitar resultados muy bajos
    })
    
    if err != nil {
        c.JSON(500, gin.H{"error": "search failed"})
        return
    }
    
    // 3. Armar respuesta (chunks + similitud)
    var response []gin.H
    for _, result := range results {
        response = append(response, gin.H{
            "content":     result.Chunk.Content,
            "similarity":  fmt.Sprintf("%.1f%%", result.Similarity * 100),
            "material_id": result.Chunk.MaterialID,
            "chunk_index": result.Chunk.ChunkIndex,
        })
    }
    
    c.JSON(200, gin.H{
        "query":   query,
        "results": response,
        "count":   len(results),
    })
}

// Response ejemplo:
// {
//   "query": "¿Qué son los neurotransmisores?",
//   "count": 3,
//   "results": [
//     {
//       "content": "Los neurotransmisores como dopamina, serotonina...",
//       "similarity": "94.3%",
//       "material_id": "550e8400-e29b-41d4-a716-446655440001",
//       "chunk_index": 1
//     },
//     ...
//   ]
// }
```

**Performance esperado**: ~50-100ms total
- Embedding API: ~30-50ms
- pgvector search: ~5-10ms
- Serialization: ~5-10ms

---

## ?? Caso 2: Importación Masiva de Material (Chunking + Embedding)

```go
// Service: Process uploaded PDF
func (s *MaterialService) ProcessUploadedMaterial(ctx context.Context, materialID uuid.UUID) error {
    // 1. Extraer texto del PDF
    text, err := s.textExtractor.ExtractText(materialID)
    if err != nil {
        return err
    }
    
    // 2. Dividir en chunks (256-token windows con overlap)
    chunks := SplitTextIntoChunks(text, 256, 50)
    
    // 3. Generar embeddings en batch (más eficiente)
    embeddings, err := s.embeddingService.BatchEmbed(ctx, chunks)
    // embeddingService internamente hace lotes de 100 para optimize API calls
    
    // 4. Crear MaterialChunk structs
    var dbChunks []*domain.MaterialChunk
    for i, chunk := range chunks {
        dbChunks = append(dbChunks, &domain.MaterialChunk{
            MaterialID: materialID,
            TopicID:    material.TopicID,  // Extraer del material
            ChunkIndex: i,
            Content:    chunk,
            Embedding:  embeddings[i],  // 768-dim vector
        })
    }
    
    // 5. Insertar en batch (transacción única, rollback si falla)
    err = s.chunkRepo.BatchCreateChunks(dbChunks)
    if err != nil {
        return fmt.Errorf("failed to store chunks: %w", err)
    }
    
    // 6. Actualizar material status
    material.Status = domain.MaterialStatusValidated
    material.ExtractedText = text
    return s.materialRepo.Update(material)
}

// Performance esperado para PDF de 50 páginas (~50k tokens):
// - Text extraction: ~1-2s
// - Splitting: ~100ms (CPU-bound)
// - Embedding batch API: ~5-10s (Vertex AI rate limiting)
// - DB insert (500-1000 chunks): ~500ms
// Total: ~6-13s (aceptable para process background)
```

---

## ?? Caso 3: Testing sin APIs

```go
// Test file: chunk_repository_test.go
package repositories

import (
    "testing"
    "github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
    "github.com/stretchr/testify/require"
)

func TestSimilaritySearch_ReturnsSimilarChunksFirst(t *testing.T) {
    // Setup: Use synthetic embeddings (NO API calls)
    fixtures := domain.NewChunkFixtures()
    
    // Create chunks in DB with test embeddings
    chunks := []*domain.MaterialChunk{
        {
            ID:         uuid.New(),
            MaterialID: testMaterialID,
            TopicID:    testTopicID,
            ChunkIndex: 0,
            Content:    "Información sobre sinapsis y neurotransmisores",
            Embedding:  fixtures.NeuroscienceEmbedding1,  // Synthetic
        },
        {
            ID:         uuid.New(),
            MaterialID: testMaterialID,
            TopicID:    testTopicID,
            ChunkIndex: 1,
            Content:    "La dopamina es un neurotransmisor importante",
            Embedding:  fixtures.NeuroscienceEmbedding2,  // Similar a chunk 0
        },
        {
            ID:         uuid.New(),
            MaterialID: testMaterialID,
            TopicID:    testTopicID,
            ChunkIndex: 2,
            Content:    "Python es un lenguaje de programación",
            Embedding:  fixtures.ProgrammingEmbedding1,  // Very different
        },
    }
    
    repo := repositories.NewChunkRepository(testDB)
    
    // Insert test chunks
    for _, chunk := range chunks {
        err := repo.Create(chunk)
        require.NoError(t, err)
    }
    
    // Search with neuroscience query
    results, err := repo.SimilaritySearch(repositories.SimilaritySearchRequest{
        Embedding: fixtures.NeuroscienceEmbedding1,
        Limit:     10,
    })
    
    require.NoError(t, err)
    require.Len(t, results, 3)
    
    // ASSERT: First two results should be neuroscience chunks
    require.Equal(t, chunks[0].ID, results[0].Chunk.ID)  // Exact match
    require.Greater(t, results[0].Similarity, results[1].Similarity)
    require.Greater(t, results[1].Similarity, results[2].Similarity)
    
    // Programming chunk should be last
    require.Equal(t, chunks[2].ID, results[2].Chunk.ID)
}
```

**Ventajas del enfoque**:
- ? NO requiere API calls during testing
- ? Determinístico (mismo seed = mismo resultado)
- ? Rápido (<1ms por test)
- ? CI/CD friendly (sin credenciales API)

---

## ?? Caso 4: Consolidación de Contexto (Para IA)

```go
// Generar contexto consolidado para un tópico completo
func (s *RAGService) GenerateTopicContext(topicID uuid.UUID) (string, error) {
    // 1. Obtener TODOS los chunks de un topic (en orden original)
    chunks, err := s.chunkRepo.GetChunksByTopic(topicID)
    if err != nil {
        return "", err
    }
    
    // 2. Ordenar por material_id + chunk_index
    sort.Slice(chunks, func(i, j int) bool {
        if chunks[i].MaterialID != chunks[j].MaterialID {
            return chunks[i].MaterialID.String() < chunks[j].MaterialID.String()
        }
        return chunks[i].ChunkIndex < chunks[j].ChunkIndex
    })
    
    // 3. Concatenar manteniendo estructura original
    var context strings.Builder
    var currentMaterial uuid.UUID
    var chunkCount int
    
    for _, chunk := range chunks {
        if chunk.MaterialID != currentMaterial {
            currentMaterial = chunk.MaterialID
            context.WriteString(fmt.Sprintf("\n---\nMaterial %s\n---\n", chunk.MaterialID.String()))
        }
        
        context.WriteString(fmt.Sprintf("%s\n\n", chunk.Content))
        chunkCount++
        
        // Limit: 32k tokens (~oclusion limit para Gemini)
        if chunkCount > 500 {
            context.WriteString("\n... (truncado por límite de contexto)\n")
            break
        }
    }
    
    return context.String(), nil
}

// Output ejemplo:
// ---
// Material 550e8400-...
// ---
// Una sinapsis es la conexión entre neuronas...
// Los neurotransmisores como dopamina...
// ... (más chunks en orden original)
```

---

## ?? Comparativa: Diferentes Estrategias

| Estrategia | Casos de Uso | Velocidad | Costo | Calidad |
|---|---|---|---|---|
| **pgvector KNN solo** | "Top-5 chunks para pregunta" | ??? | ?? | ? |
| **pgvector + Full-text** | "Búsqueda multi-criteria" | ?? | ???? | ?? |
| **pgvector + Mini-LM** | "Rerank de chunks" | ? | ?????? | ??? |
| **Solo embeddings (Milvus)** | Alta escala (>1M) | ? | ???????? | ?? |

Para Klyra MVP: **pgvector KNN basta**

---

## ?? Errores Comunes y Soluciones

### Error 1: "Index not being used"
```
Query Plan shows "Seq Scan" instead of "Index Scan"
```
**Solución**: 
```sql
REINDEX INDEX idx_chunks_embedding_ivfflat;
ANALYZE material_chunks;  -- Update statistics
```

### Error 2: "Results seem random"
```
KNN search no retorna chunks similares primero
```
**Solución**: 
- Verificar que embeddings estén normalizados:
  ```go
  sim := domain.CosineSimilarity(embedding1, embedding2)
  // Si sim < 0.3 para chunks del mismo tema, embeddings mal normalizados
  ```

### Error 3: "Timeout en batch insert"
```
BatchCreateChunks timeout con >10k chunks
```
**Solución**: 
- Aumentar batch size: `r.db.CreateInBatches(chunks, 5000)`
- O dividir en múltiples transacciones

---

**Versión**: 1.0  
**Última actualización**: 2026-03-08
