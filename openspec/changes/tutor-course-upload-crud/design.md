# Design: Tutor por Curso, Upload de Material y CRUD Completo

## Enfoque Técnico

Este cambio aborda tres bloques funcionales sobre la arquitectura existente de Clean Architecture (ports → usecases → handlers/repositories) en Go/Gin/GORM para el backend y Flutter/Riverpod/Freezed para mobile. La estrategia general es:

1. **Bloque 1 (Tutor por curso)**: Extender el pipeline RAG existente con un nuevo método `GetChunksByCourse` en `ChunkRepository` y `GetCourseContext` en `RAGUseCase`, exponiendo un endpoint adicional en `RAGHandler`. En mobile, refactorizar `TutorSessionScreen` para abrir a nivel de curso y cargar contexto por topic bajo demanda.
2. **Bloque 2 (Fix upload)**: Eliminar una línea en `material_remote_datasource.dart` que fuerza el `Content-Type` e impide a Dio generar el boundary del multipart.
3. **Bloque 3 (CRUD)**: Promover los métodos `Update`/`SoftDelete` ya existentes en las implementaciones concretas de los repositorios al nivel de las interfaces (ports), agregar lógica de cascada en el use case, registrar rutas PATCH/DELETE, y crear la UI correspondiente en mobile.

---

## Decisiones de Arquitectura

### Decisión 1: JOIN a través de `topics.course_id` para contexto de curso (sin agregar columna `course_id` a `material_chunks`)

**Elección**: Usar un JOIN `material_chunks → topics` filtrando por `topics.course_id` para obtener chunks a nivel de curso.

**Alternativas consideradas**:
- Agregar columna `course_id` directamente a `material_chunks` para JOIN directo (requiere migración + backfill de datos existentes).

**Razonamiento**: La tabla `material_chunks` ya tiene `topic_id` con FK a `topics`, y `topics` tiene `course_id`. El JOIN es directo, no requiere migración de esquema, y el índice existente `idx_chunks_topic_id` + `idx_topics_course_id` lo hace eficiente. Para el volumen MVP (decenas de topics con ~50 chunks cada uno), la performance es más que suficiente. Si en el futuro se necesita optimizar, se puede agregar la columna como cambio incremental.

---

### Decisión 2: Estrategia de truncamiento para contexto de curso completo — Top-N chunks por topic ordenados por `chunk_index`

**Elección**: Para `GetCourseContext`, seleccionar hasta `maxChunksPerTopic` (3) chunks por topic, ordenados por `chunk_index` ASC (los primeros chunks tienden a contener la introducción/resumen del material). Límite total: `maxCourseChunks` = 30 chunks. Si se pasa un `query`, usar similarity search global sobre todos los chunks del curso.

**Alternativas consideradas**:
- Similarity search siempre (requiere query obligatorio; no sirve para fallback genérico).
- Enviar todos los chunks truncando por tokens (impredecible; puede cortar a mitad de tema).
- Resumir con un LLM antes de enviar (añade latencia y costo; complejidad prematura para MVP).

**Razonamiento**: Los primeros chunks de cada material contienen típicamente el contexto introductorio más valioso. Limitar a 3 por topic y 30 en total mantiene el contexto dentro de ~15k tokens (30 × 500 tokens/chunk), bien por debajo del límite de Gemini (~1M tokens, pero se busca calidad sobre cantidad). Cuando hay query, la similarity search global sobre el curso da resultados más relevantes. Esta estrategia es simple, predecible y no requiere llamadas adicionales a APIs de IA.

---

### Decisión 3: Contexto bajo demanda en mobile — Selección explícita de topic en UI (MVP), sin análisis automático de mensajes

**Elección**: En `TutorSessionScreen`, mostrar un selector (dropdown/chips) con los topics del curso. Al seleccionar un topic, el cliente solicita `GET /courses/:course_id/topics/:topic_id/context` e inyecta el contexto como un mensaje de texto en la sesión WebSocket (tool_call o client_content). El system prompt inicial NO incluye chunks, solo la lista de topics y las instrucciones del tutor.

**Alternativas consideradas**:
- Análisis automático de los mensajes del estudiante para detectar cambio de tema (complejo, propenso a falsos positivos, requiere NLP adicional en cliente).
- Reconectar el WebSocket con un nuevo system_instruction cada vez que se cambia de topic (pierde historial de conversación).

**Razonamiento**: La API de Gemini Live (BidiGenerateContent) no permite modificar `system_instruction` una vez establecida la conexión. Sin embargo, sí acepta mensajes `client_content` que el modelo incorpora como contexto de la conversación. Inyectar el contexto del topic como un mensaje de texto con formato especial (`[CONTEXTO DEL TEMA: ...]`) es el mecanismo más simple y efectivo. La selección explícita en UI es más confiable que la detección automática para MVP; en futuras iteraciones con voz, el asistente preguntará proactivamente.

---

### Decisión 4: Soft delete en cascada con lógica explícita en el use case (no solo FKs)

**Elección**: Implementar la cascada de soft delete explícitamente en `CourseUseCase.DeleteCourse` dentro de una transacción GORM: actualizar `deleted_at` en course → topics → materials → chunks. No confiar en `ON DELETE CASCADE` de las FKs (que solo aplica a hard delete).

**Alternativas consideradas**:
- Trigger de base de datos para soft delete en cascada (acoplamiento a PostgreSQL, difícil de testear).
- Hard delete en cascada confiando en las FKs (pierdes datos para posible restauración).

**Razonamiento**: GORM ya soporta soft delete con `deleted_at` en todos los modelos. `ON DELETE CASCADE` de las FKs solo ejecuta hard delete. Para soft delete en cascada se necesita lógica explícita. Envolver todo en una transacción GORM (`db.Transaction(...)`) garantiza atomicidad: si falla algún paso, se hace rollback completo. Este patrón es el estándar en proyectos Go/GORM con soft delete.

---

### Decisión 5: Agregar `Update` y `SoftDelete` a las interfaces de ports (promover métodos existentes)

**Elección**: Extender `CourseRepository` y `TopicRepository` en `ports/course_port.go` con `Update(ctx, entity)` y `SoftDelete(ctx, id)`. Las implementaciones concretas en `repositories/` ya tienen `UpdateCourse`, `DeleteCourse`, `UpdateTopic`, `DeleteTopic`; solo se necesita adaptar las firmas para satisfacer la interfaz (agregar `context.Context` y usar `string` para IDs).

**Alternativas consideradas**:
- Crear interfaces separadas para operaciones de escritura (over-engineering para MVP).
- Llamar directamente a los métodos concretos desde el use case (rompe Clean Architecture).

**Razonamiento**: Seguir la convención existente del proyecto: los ports definen la interfaz, las implementaciones concretas la satisfacen. Los métodos ya existen en las implementaciones, solo falta exponerlos a través de la interfaz. Esto mantiene la coherencia arquitectónica.

---

## Flujo de Datos

### Bloque 1: Tutor por curso con contexto bajo demanda

#### Flujo: Inicio de sesión de tutoría a nivel de curso

```
Mobile (TutorSessionScreen)
  │
  ├─ 1. Abrir sesión con courseId (sin topicId fijo)
  │    Obtener lista de topics del curso (ya disponible en state)
  │
  ├─ 2. Conectar WebSocket a Gemini Live
  │    System prompt: instrucciones tutor + lista de topics (sin chunks)
  │
  ├─ 3. Usuario selecciona topic en UI (chip/dropdown)
  │    │
  │    ├─ GET /api/v1/courses/:course_id/topics/:topic_id/context
  │    │   │
  │    │   ├─ RAGHandler.GetTopicContext()
  │    │   │   └─ RAGUseCase.GetTopicContext(topicID, "")
  │    │   │       └─ ChunkRepository.GetChunksByTopic(topicID)
  │    │   │           → Retorna chunks concatenados
  │    │   │
  │    │   └─ Response: { context: "...", topic_id: "..." }
  │    │
  │    └─ 4. Inyectar contexto como client_content en WebSocket
  │         { clientContent: { parts: [{ text: "[CONTEXTO: ...]" }] } }
  │
  └─ 5. (Opcional) Usuario pide "hablar del curso completo"
       │
       ├─ GET /api/v1/courses/:course_id/context  ← NUEVO
       │   │
       │   ├─ RAGHandler.GetCourseContext()  ← NUEVO
       │   │   └─ RAGUseCase.GetCourseContext(courseID, query?)
       │   │       └─ ChunkRepository.GetChunksByCourse(courseID)  ← NUEVO
       │   │           → SQL: SELECT mc.* FROM material_chunks mc
       │   │                  JOIN topics t ON mc.topic_id = t.id
       │   │                  WHERE t.course_id = $1
       │   │                    AND t.deleted_at IS NULL
       │   │                  ORDER BY t.order_index, mc.chunk_index
       │   │           → Truncamiento: top-3 por topic, max 30 total
       │   │
       │   └─ Response: { context: "...", course_id: "..." }
       │
       └─ Inyectar contexto resumido como client_content en WebSocket
```

#### Flujo: GetCourseContext internamente

```
RAGUseCase.GetCourseContext(ctx, courseID, query)
  │
  ├─ Si query == "" (fallback genérico):
  │   ├─ chunkRepo.GetChunksByCourse(ctx, courseID)
  │   │   → Retorna []MaterialChunk ordenados por topic + chunk_index
  │   ├─ Agrupar por topic_id
  │   ├─ De cada grupo, tomar los primeros 3 (maxChunksPerTopic)
  │   ├─ Total máximo: 30 (maxCourseChunks)
  │   └─ Concatenar con separadores: "## Topic: {title}\n{content}\n\n"
  │
  └─ Si query != "" (similarity search global):
      ├─ embedder.Embed(ctx, query) → queryEmbedding
      ├─ chunkRepo.SearchSimilarByCourse(ctx, courseID, queryEmbedding, 10)
      │   → SQL con JOIN a topics filtrado por course_id
      └─ Concatenar resultados
```

### Bloque 2: Fix upload

```
material_remote_datasource.dart
  │
  ├─ ANTES: _dio.post(..., options: Options(contentType: 'multipart/form-data'))
  │   → Dio sobrescribe Content-Type sin boundary → backend rechaza (400)
  │
  └─ DESPUÉS: _dio.post(..., data: formData)   // sin options
      → Dio detecta FormData, genera Content-Type con boundary automáticamente
      → Backend (Gin) parsea multipart correctamente
```

### Bloque 3: CRUD - Soft delete en cascada

```
CourseHandler.DeleteCourse(c)
  │
  ├─ Extraer user_id de JWT, course_id de URL
  │
  └─ CourseUseCase.DeleteCourse(ctx, courseID, userID)
      │
      ├─ Validar ownership: courseRepo.FindByID → check UserID == userID
      │
      └─ db.Transaction(func(tx *gorm.DB) error {
            ├─ tx.Model(&MaterialChunk{}).Where(subquery topics+materials).Update("deleted_at", now)
            ├─ tx.Model(&Material{}).Where(subquery topics).Update("deleted_at", now)
            ├─ tx.Model(&Topic{}).Where("course_id = ?", courseID).Update("deleted_at", now)
            └─ tx.Model(&Course{}).Where("id = ?", courseID).Update("deleted_at", now)
          })
```

---

## Cambios en Archivos

| Archivo | Acción | Descripción |
|---------|--------|-------------|
| `backend/internal/core/ports/rag_port.go` | Modificar | Agregar `GetChunksByCourse(ctx, courseID) ([]MaterialChunk, error)` y `SearchSimilarByCourse(ctx, courseID, queryEmbedding, topK) ([]RAGResult, error)` a `ChunkRepository` |
| `backend/internal/core/ports/course_port.go` | Modificar | Agregar `Update(ctx, course) error` y `SoftDelete(ctx, id) error` a `CourseRepository`; agregar `Update(ctx, topic) error` y `SoftDelete(ctx, id) error` a `TopicRepository` |
| `backend/internal/core/usecases/rag_usecase.go` | Modificar | Agregar método `GetCourseContext(ctx, courseID, query) (string, error)` con lógica de truncamiento y constantes `maxChunksPerTopic=3`, `maxCourseChunks=30`. Agregar dependencia de `topicRepo` al constructor |
| `backend/internal/core/usecases/course_usecase.go` | Modificar | Agregar métodos `UpdateCourse(ctx, courseID, userID, input)`, `DeleteCourse(ctx, courseID, userID)`, `UpdateTopic(ctx, courseID, topicID, userID, title)`, `DeleteTopic(ctx, courseID, topicID, userID)`. Agregar dependencias `materialRepo`, `chunkRepo`, `db` al constructor para la cascada transaccional |
| `backend/internal/handlers/http/rag_handler.go` | Modificar | Agregar ruta `GET /courses/:course_id/context` y handler `GetCourseContext`. Agregar validación de ownership (inyectar `courseUseCase` como dependencia) |
| `backend/internal/handlers/http/course_handler.go` | Modificar | Agregar rutas `PATCH /courses/:course_id` y `DELETE /courses/:course_id`; handlers `UpdateCourse` y `DeleteCourse` |
| `backend/internal/handlers/http/topic_handler.go` | Modificar | Agregar rutas `PATCH /courses/:course_id/topics/:topic_id` y `DELETE /courses/:course_id/topics/:topic_id`; handlers `UpdateTopic` y `DeleteTopic`. Inyectar `courseUseCase` para usar sus métodos de update/delete de topics |
| `backend/internal/repositories/chunk_repository.go` | Modificar | Implementar `GetChunksByCourse(ctx, courseID)` con JOIN a `topics`, y `SearchSimilarByCourse(ctx, courseID, queryEmbedding, topK)` con similarity search filtrada por curso |
| `backend/internal/repositories/course_repository.go` | Modificar | Adaptar `UpdateCourse` y `DeleteCourse` existentes para satisfacer la interfaz de ports (agregar `context.Context`, recibir `string` para IDs). Agregar método `SoftDeleteCascade(ctx, tx, courseID)` para la cascada transaccional |
| `backend/internal/repositories/topic_repository.go` | Modificar | Adaptar `UpdateTopic` y `DeleteTopic` existentes para satisfacer la interfaz de ports. Agregar `SoftDeleteCascade(ctx, tx, topicID)` |
| `backend/cmd/api/main.go` | Modificar | Actualizar wiring: pasar nuevas dependencias a `CourseUseCase` (materialRepo, chunkRepo, db), pasar `courseUseCase` a `RAGHandler` y `TopicHandler` |
| `mobile/lib/features/course/data/material_remote_datasource.dart` | Modificar | Eliminar `options: Options(contentType: 'multipart/form-data')` en línea 59 |
| `mobile/lib/features/course/data/course_remote_datasource.dart` | Modificar | Agregar métodos `updateCourse(courseId, name, educationLevel)`, `deleteCourse(courseId)`, `updateTopic(courseId, topicId, title)`, `deleteTopic(courseId, topicId)`, `fetchCourseContext(courseId)` |
| `mobile/lib/features/course/data/course_repository.dart` | Modificar | Exponer nuevos métodos del datasource: `updateCourse`, `deleteCourse`, `updateTopic`, `deleteTopic`, `fetchCourseContext` |
| `mobile/lib/features/course/presentation/course_controller.dart` | Modificar | Agregar métodos `updateCourse(courseId, name, educationLevel)`, `deleteCourse(courseId)`, `updateTopic(courseId, topicId, title)`, `deleteTopic(courseId, topicId)` con invalidación de estado |
| `mobile/lib/features/course/presentation/course_dashboard_screen.dart` | Modificar | Agregar `PopupMenuButton` a `_CourseCard` con opciones "Editar" y "Eliminar" |
| `mobile/lib/features/course/presentation/course_detail_screen.dart` | Modificar | Agregar `PopupMenuButton` a `_TopicSection` con opciones "Editar" y "Eliminar". Agregar menú de edición de curso en el AppBar |
| `mobile/lib/features/tutor/presentation/tutor_session_screen.dart` | Modificar | Refactorizar para recibir solo `courseId` (topic como parámetro opcional). Agregar selector de topics (chips). Agregar botón "Curso completo" |
| `mobile/lib/features/tutor/presentation/tutor_session_controller.dart` | Modificar | Refactorizar `startSession(courseId)` sin topicId obligatorio. Agregar `loadTopicContext(courseId, topicId)` y `loadCourseContext(courseId)` para carga dinámica. Agregar estado `currentTopicId` y `loadedTopics` (set de topics ya cargados) |
| `mobile/lib/features/tutor/data/gemini_live_service.dart` | Modificar | Agregar método `sendContextUpdate(contextText)` que envía un mensaje `client_content` al WebSocket. Refactorizar `_buildSystemPrompt` para no incluir chunks, solo instrucciones y lista de topics |

---

## Interfaces / Contratos

### Backend: Extensión de `ChunkRepository` (rag_port.go)

```go
type ChunkRepository interface {
    // ... métodos existentes ...
    
    GetChunksByCourse(ctx context.Context, courseID string) ([]domain.MaterialChunk, error)
    SearchSimilarByCourse(ctx context.Context, courseID string, queryEmbedding []float32, topK int) ([]domain.RAGResult, error)
}
```

### Backend: Extensión de `CourseRepository` y `TopicRepository` (course_port.go)

```go
type CourseRepository interface {
    // ... métodos existentes ...
    
    Update(ctx context.Context, course *domain.Course) error
    SoftDelete(ctx context.Context, id string) error
}

type TopicRepository interface {
    // ... métodos existentes ...
    
    Update(ctx context.Context, topic *domain.Topic) error
    SoftDelete(ctx context.Context, id string) error
}
```

### Backend: Nuevos métodos en `RAGUseCase`

```go
const (
    maxChunksPerTopic = 3
    maxCourseChunks   = 30
)

func (uc *RAGUseCase) GetCourseContext(ctx context.Context, courseID, query string) (string, error)
```

### Backend: Nuevos métodos en `CourseUseCase`

```go
type UpdateCourseInput struct {
    Name           string
    EducationLevel string
}

type UpdateTopicInput struct {
    Title string
}

func (uc *CourseUseCase) UpdateCourse(ctx context.Context, courseID, userID string, input UpdateCourseInput) (*domain.Course, error)
func (uc *CourseUseCase) DeleteCourse(ctx context.Context, courseID, userID string) error
func (uc *CourseUseCase) UpdateTopic(ctx context.Context, courseID, topicID, userID string, input UpdateTopicInput) (*domain.Topic, error)
func (uc *CourseUseCase) DeleteTopic(ctx context.Context, courseID, topicID, userID string) error
```

### Backend: Nuevas rutas HTTP

```
PATCH  /api/v1/courses/:course_id                              → CourseHandler.UpdateCourse
DELETE /api/v1/courses/:course_id                              → CourseHandler.DeleteCourse
PATCH  /api/v1/courses/:course_id/topics/:topic_id             → TopicHandler.UpdateTopic
DELETE /api/v1/courses/:course_id/topics/:topic_id             → TopicHandler.DeleteTopic
GET    /api/v1/courses/:course_id/context                      → RAGHandler.GetCourseContext
```

### Backend: Cuerpos de request/response

```json
// PATCH /courses/:course_id
// Request:
{ "name": "Nuevo nombre", "education_level": "university" }
// Response: 200 + Course JSON completo

// DELETE /courses/:course_id
// Response: 200 + { "message": "course deleted" }

// PATCH /courses/:course_id/topics/:topic_id
// Request:
{ "title": "Nuevo título del topic" }
// Response: 200 + Topic JSON completo

// DELETE /courses/:course_id/topics/:topic_id
// Response: 200 + { "message": "topic deleted" }

// GET /courses/:course_id/context?query=opcional
// Response:
{ "course_id": "...", "context": "texto concatenado...", "query": "..." }
```

### Mobile: Extensión de `GeminiLiveService`

```dart
void sendContextUpdate(String contextText) {
  if (_channel == null) return;
  final message = {
    'clientContent': {
      'parts': [
        {'text': '[CONTEXTO DEL TEMA SELECCIONADO]\n$contextText'}
      ],
      'turnComplete': true,
    }
  };
  _channel!.sink.add(jsonEncode(message));
}

String _buildSystemPrompt(List<String> topicTitles) {
  final topicList = topicTitles.asMap().entries
      .map((e) => '${e.key + 1}. ${e.value}')
      .join('\n');
  return '''You are Klyra, an enthusiastic, patient, and encouraging AI tutor.
Your goal is to help the student understand their course material.

Available topics in this course:
$topicList

When the student selects a topic, you will receive the relevant context.
Until then, you can discuss the course structure and help the student choose a topic.
Ask questions to check understanding. Celebrate correct answers and gently correct mistakes.''';
}
```

### Mobile: Extensión de `TutorSessionState`

```dart
class TutorSessionState {
  final SessionState sessionState;
  final String transcript;
  final String? error;
  final bool isMicrophoneActive;
  final String? currentTopicId;
  final Set<String> loadedTopicIds;
  final bool isLoadingContext;
  // ...
}
```

---

## Estrategia de Testing

| Capa | Qué Testear | Enfoque |
|------|-------------|---------|
| Unit (Backend) | `RAGUseCase.GetCourseContext` con mock de `ChunkRepository` — verificar truncamiento (max 3/topic, max 30 total), concatenación, y branch con/sin query | Tabla de tests en `rag_usecase_test.go`, siguiendo el patrón existente con `test_helpers.go` |
| Unit (Backend) | `CourseUseCase.DeleteCourse` — verificar cascada (se llaman SoftDelete en orden), ownership check, error de transacción produce rollback | Mock de repos en `course_usecase_test.go` |
| Unit (Backend) | `CourseUseCase.UpdateCourse` / `UpdateTopic` — verificar actualización parcial (solo campos no vacíos), ownership check | Mock de repos |
| Integration (Backend) | `ChunkRepository.GetChunksByCourse` — verificar que el JOIN devuelve solo chunks de topics no-deleted del curso correcto | Test con DB real o testcontainers, patrón existente en `*_test.go` |
| Integration (Backend) | `ChunkRepository.SearchSimilarByCourse` — verificar que similarity search filtra por curso | Test con pgvector habilitado |
| Unit (Backend) | Handlers PATCH/DELETE — verificar status codes (200, 400, 403, 404), binding de body, extracción de params | `httptest` siguiendo `course_handler_test.go` |
| Unit (Mobile) | Fix upload — verificar que `uploadMaterial` NO pasa `options` con contentType | Test unitario del datasource con Dio mock |
| Integration (Mobile) | Flujo completo de upload → procesamiento → chunks generados | Test E2E manual contra backend local |
| Unit (Mobile) | `TutorSessionController.loadTopicContext` — verificar que llama al endpoint correcto y actualiza state | Test con mock de Dio y GeminiLiveService |
| Manual (E2E) | Sesión de tutoría: abrir curso → seleccionar topic → verificar que contexto se inyecta → cambiar topic → verificar que nuevo contexto se agrega | Dispositivo físico o emulador |

---

## Migración / Rollout

No se requieren migraciones de base de datos. Todas las tablas y columnas necesarias ya existen:
- `material_chunks.topic_id` con FK a `topics` (para el JOIN).
- `deleted_at` en `courses`, `topics`, `materials` (para soft delete).
- `material_chunks` no tiene `deleted_at` pero los chunks se filtran indirectamente por `topics.deleted_at IS NULL` en el JOIN.

**Nota sobre chunks y soft delete**: La tabla `material_chunks` no tiene columna `deleted_at`. Para el soft delete en cascada, las opciones son:
1. Agregar `deleted_at` a `material_chunks` con una migración (preferido para consistencia).
2. Hacer hard delete de chunks cuando se borra en cascada (los chunks son derivados y regenerables).

**Recomendación**: Usar hard delete para chunks en la cascada (`DELETE FROM material_chunks WHERE topic_id IN (...)`) ya que son datos derivados (se regeneran desde el `extracted_text` del material). Esto evita agregar migración y mantiene la tabla limpia para las búsquedas vectoriales (no hay que filtrar por `deleted_at IS NULL` en cada query de similarity).

**Orden de deploy**: Backend primero (las nuevas rutas son aditivas). Mobile después. El fix de upload (Bloque 2) puede desplegarse inmediatamente sin dependencia del backend.

---

## Consideraciones de Tokens y Límites

### System Prompt Inicial (sin chunks)
- Instrucciones del tutor: ~100 tokens
- Lista de topics (asumiendo ~10 topics): ~50 tokens
- **Total system prompt**: ~150 tokens ✓

### Contexto por Topic (on-demand)
- Máximo 50 chunks por material × 500 tokens/chunk = 25,000 tokens por material
- En la práctica, 1-3 materiales por topic → ~25,000-75,000 tokens
- Inyectado como `client_content`, no como system_instruction
- **Riesgo**: Topics con muchos materiales grandes. **Mitigación**: El límite existente de `maxChunksPerMaterial = 50` ya controla esto.

### Contexto de Curso Completo (fallback)
- 3 chunks/topic × 10 topics = 30 chunks × ~500 tokens = ~15,000 tokens
- **Bien dentro de límites** de Gemini (1M tokens para input)

---

## Preguntas Abiertas

- [x] ~~¿Agregar `deleted_at` a `material_chunks`?~~ → No. Hard delete de chunks en cascada (son datos derivados regenerables).
- [ ] ¿La API de Gemini Live acepta `client_content` con `turnComplete: true` sin que el modelo esté esperando un turno del usuario? Verificar en la documentación. Si no, la alternativa es enviar como `realtimeInput` con mime type `text/plain` (pero esto no está documentado). **Plan B**: Desconectar y reconectar con nuevo system prompt que incluya el contexto (pierde historial pero es funcional).
- [ ] ¿El selector de topics en `TutorSessionScreen` debe permitir seleccionar múltiples topics a la vez o solo uno? **Recomendación**: Solo uno a la vez para MVP, con opción "Curso completo" como alternativa.

---

## Riesgos

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|-------------|---------|------------|
| `client_content` en Gemini Live WebSocket no funcione como mecanismo de inyección de contexto dinámico | Media | Alto | Probar primero. Plan B: reconectar WebSocket con nuevo system_instruction (pierde historial). Plan C: acumular contexto en el system prompt y reconectar solo al cambiar de tema |
| Soft delete de chunks no se filtra correctamente al hacer similarity search (chunks de topics borrados aparecen en resultados) | Baja | Medio | Usar hard delete para chunks en cascada. Además, el JOIN por `topics.deleted_at IS NULL` en `GetChunksByCourse` ya excluye topics borrados |
| Latencia al cargar contexto on-demand percibida por el usuario durante la conversación de voz | Baja | Medio | El endpoint de contexto por topic es rápido (<100ms). Mostrar indicador de carga en UI. Cachear contextos ya cargados en el controller (set `loadedTopicIds`) para no repetir llamadas |
