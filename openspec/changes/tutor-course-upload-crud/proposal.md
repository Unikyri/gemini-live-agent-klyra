# Proposal: Tutor por Curso, Upload de Material y CRUD Completo

## Intent

Klyra permite a estudiantes crear cursos, agregar topics y subir material que alimenta un tutor IA. Actualmente hay tres deficiencias críticas que bloquean la experiencia completa:

1. **Contexto del tutor sin estrategia de carga eficiente**: El tutor necesita poder operar a nivel de curso (una sesión por curso), pero cargar todo el contexto de todos los topics al iniciar la sesión excede fácilmente los límites de tokens de Gemini y degrada la calidad de las respuestas. Se requiere una estrategia de **contexto bajo demanda**: el backend tiene disponible tanto el contexto por topic como por curso, pero la sesión del tutor solo inyecta el contexto del topic que el estudiante solicita en cada momento, cargándolo dinámicamente cuando el usuario indica "quiero hablar del tema X".
2. **Upload roto**: El botón de subir material existe en la UI pero falla silenciosamente (HTTP 400) porque el cliente Dart fuerza un `contentType` que impide a Dio generar el boundary correcto del multipart. Sin upload funcional, el pipeline RAG completo queda inutilizable.
3. **Sin operaciones de edición/borrado**: No existen endpoints PATCH/DELETE para courses ni topics en backend, ni controles en la UI móvil para editar o eliminar. El estudiante queda atrapado con datos que no puede corregir.

Resolver estos tres bloques en conjunto es necesario porque el bloque 2 (upload) es prerequisito para que el bloque 1 (tutor por curso) tenga datos con los que trabajar, y el bloque 3 (CRUD) complementa la gestión integral que un usuario espera.

### Rol del material subido

El material que sube el estudiante (PDFs, documentos) sirve como **guía o roadmap del curso**: define la estructura, los temas y los puntos de referencia. **No** es la base de conocimiento profundo del tutor. El tutor de Klyra amplía y enriquece las explicaciones utilizando la API de IA (Gemini) como fuente principal de conocimiento, usando el material subido como contexto estructural y de referencia para orientar sus respuestas, no como única fuente de verdad.

## Scope

### In Scope

- **Bloque 1 – Tutor por curso con contexto bajo demanda**:
  - La sesión de tutoría se abre a nivel de curso (una sesión por curso), pero **no** se inyecta todo el contexto al iniciar.
  - **Contexto por topic on-demand**: cuando el estudiante indica "quiero hablar del tema X" o "siguiente tema", el cliente solicita `GET /api/v1/topics/:topic_id/context` (endpoint existente) y lo inyecta en la conversación o actualiza el system prompt con ese contexto.
  - **Contexto de curso completo (fallback resumido)**: nuevo endpoint `GET /api/v1/courses/:course_id/context` en `RAGHandler` que devuelve un resumen/selección de los chunks más relevantes de todo el curso (truncamiento inteligente para no exceder el límite de tokens). Se usa solo cuando el estudiante pide explícitamente hablar del curso completo.
  - Nuevo método `GetCourseContext` en `RAGUseCase` con lógica de selección/resumen de chunks.
  - Opcional: agregar columna `course_id` a `material_chunks` para joins directos (alternativa recomendada: join a través de `topics.course_id`).
  - Actualizar mobile: `TutorSessionScreen` inicia sesión a nivel de curso sin inyectar contexto masivo; `GeminiLiveService` expone método para cargar/actualizar contexto dinámicamente por topic o por curso resumido.

- **Bloque 2 – Fix upload de material**:
  - Eliminar `options: Options(contentType: 'multipart/form-data')` en `material_remote_datasource.dart` para que Dio genere automáticamente el header con boundary.
  - Verificar que el flujo completo funcione: upload → procesamiento → extracción de texto → embedding.

- **Bloque 3 – CRUD completo de courses y topics**:
  - Backend: `PATCH /courses/:course_id`, `DELETE /courses/:course_id`, `PATCH /courses/:course_id/topics/:topic_id`, `DELETE /courses/:course_id/topics/:topic_id`.
  - Ports/interfaces: agregar `Update` y `Delete` a `CourseRepository` y `TopicRepository`.
  - Use cases: lógica de actualización parcial y borrado (soft delete con validación de ownership).
  - Borrado en cascada: al borrar un course se deben soft-delete sus topics, materials y chunks asociados.
  - Mobile: `PopupMenuButton` o menú contextual en las cards de course y topic con opciones "Editar" y "Eliminar".
  - Mobile: diálogos de confirmación antes de borrar.
  - Mobile: formularios de edición (nombre de curso, nivel educativo, título de topic).

### Out of Scope

- Reordenamiento drag-and-drop de topics (futuro).
- Búsqueda semántica cross-curso (el contexto se limita a un curso a la vez).
- Migración a HNSW para el índice vectorial (optimización futura, IVFFlat suficiente para MVP).
- Edición/borrado de materials individuales (puede abordarse en un cambio posterior).
- Cambios en el flujo de autenticación o permisos más allá de la validación de ownership existente.

### Nota futura: Voz/Audio del asistente

Cuando se implemente la funcionalidad de voz/audio en el asistente, se establecerá que el asistente **pregunte activamente al estudiante** de qué tema quiere hablar (por ejemplo al iniciar la sesión o durante la conversación). Esto garantiza que se cargue el contexto dedicado al topic que el estudiante elija, manteniendo la estrategia de contexto bajo demanda también en el modo de voz. Este comportamiento no se implementa en este cambio, pero el diseño de contexto on-demand está preparado para soportarlo.

## Approach

### Bloque 1 – Tutor por curso con contexto bajo demanda

La estrategia se basa en dos niveles de carga de contexto, ambos disponibles desde el backend:

**1. Contexto por topic (on-demand, caso principal):**
El endpoint existente `GET /api/v1/topics/:topic_id/context` ya devuelve los chunks de un topic individual. Este es el mecanismo principal: cuando el estudiante pide hablar de un tema concreto, el cliente invoca este endpoint y el contexto se inyecta en la conversación o se actualiza el system prompt. No se carga contexto al iniciar la sesión; el system prompt inicial solo contiene las instrucciones del tutor y la estructura del curso (lista de topics disponibles).

**2. Contexto de curso resumido (fallback explícito):**
Para el caso en que el estudiante quiera hablar del curso completo, se agrega un método `GetChunksByCourse(ctx, courseID)` al `ChunkRepository` que ejecuta un JOIN a través de la relación existente `material_chunks.topic_id → topics.course_id`:

```sql
SELECT mc.* FROM material_chunks mc
JOIN topics t ON mc.topic_id = t.id
WHERE t.course_id = $1 AND t.deleted_at IS NULL
ORDER BY t.order_index, mc.chunk_index
```

En `RAGUseCase` se crea `GetCourseContext(ctx, courseID, query)` que, a diferencia de la carga por topic, aplica **selección y resumen** de los chunks más relevantes (por similarity search o truncamiento a los N chunks más importantes por topic) para no exceder el límite de tokens. El `RAGHandler` expone el nuevo endpoint `GET /courses/:course_id/context`.

**En mobile**, `TutorSessionScreen` abre la sesión a nivel de curso pero sin inyectar contexto masivo. `GeminiLiveService` expone un método para actualizar el contexto dinámicamente: al detectar que el estudiante quiere cambiar de tema (mediante análisis del mensaje o selección explícita en UI), se solicita el contexto del topic correspondiente y se inyecta. El material subido se utiliza como referencia estructural (guía/roadmap); las explicaciones profundas las genera el tutor usando las capacidades de Gemini, no limitándose al contenido literal del material.

### Bloque 2 – Fix upload de material

Es un fix de una línea en `material_remote_datasource.dart` línea 59. Al enviar un `FormData`, Dio detecta automáticamente que debe usar `multipart/form-data` y genera el boundary correcto. Forzar el header manualmente elimina el boundary del header `Content-Type`, causando que el backend (Gin) no pueda parsear el multipart y devuelva 400.

Se elimina el parámetro `options: Options(contentType: 'multipart/form-data')` del `_dio.post(...)`. No se requieren cambios en backend.

### Bloque 3 – CRUD completo

**Backend (Go/Gin)**:
- Se extienden las interfaces `CourseRepository` y `TopicRepository` con métodos `Update(ctx, entity)` y `SoftDelete(ctx, id)`.
- Se agrega lógica de use case: validación de ownership, actualización parcial (solo campos no vacíos), y soft delete con cascada.
- Se registran nuevas rutas `PATCH` y `DELETE` en `CourseHandler` y `TopicHandler`.
- El borrado de un course ejecuta soft delete en cascada: course → topics → materials → chunks (aprovechando `ON DELETE CASCADE` ya definido para hard delete, pero para soft delete se necesita lógica explícita en el use case).

**Mobile (Flutter/Riverpod)**:
- Se agregan métodos `updateCourse`, `deleteCourse`, `updateTopic`, `deleteTopic` en los datasources y repositorios.
- Se añaden `PopupMenuButton` en las cards de cursos y topics con opciones "Editar nombre" y "Eliminar".
- Diálogo de confirmación antes de borrar con mensaje claro sobre la acción irreversible.
- Tras editar/borrar, se invalida el provider de cursos para refrescar la lista.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/internal/core/ports/rag_port.go` | Modified | Agregar `GetChunksByCourse` al `ChunkRepository` interface |
| `backend/internal/core/usecases/rag_usecase.go` | Modified | Nuevo método `GetCourseContext` |
| `backend/internal/handlers/http/rag_handler.go` | Modified | Nuevo endpoint `GET /courses/:course_id/context` |
| `backend/internal/repositories/chunk_repository.go` | Modified | Implementar `GetChunksByCourse` |
| `backend/internal/infrastructure/repositories/chunk_repository.go` | Modified | Implementar `GetChunksByCourse` (CloudSQL) |
| `backend/internal/core/ports/course_port.go` | Modified | Agregar `Update`, `SoftDelete` a ambos repos |
| `backend/internal/core/usecases/course_usecase.go` | Modified | Métodos `UpdateCourse`, `DeleteCourse`, `UpdateTopic`, `DeleteTopic` |
| `backend/internal/handlers/http/course_handler.go` | Modified | Rutas PATCH/DELETE para courses y topics |
| `backend/internal/handlers/http/topic_handler.go` | Modified | Rutas PATCH/DELETE para topics (o consolidar en course_handler) |
| `backend/internal/repositories/course_repository.go` | Modified | Implementar Update y SoftDelete |
| `backend/internal/repositories/topic_repository.go` | Modified | Implementar Update y SoftDelete |
| `mobile/lib/features/course/data/material_remote_datasource.dart` | Modified | Eliminar contentType forzado (fix upload) |
| `mobile/lib/features/course/data/course_remote_datasource.dart` | Modified | Agregar métodos update/delete para courses y topics |
| `mobile/lib/features/course/presentation/course_controller.dart` | Modified | Exponer acciones de editar/borrar |
| `mobile/lib/features/tutor/presentation/tutor_session_screen.dart` | Modified | Iniciar sesión a nivel de curso sin inyectar contexto masivo; soportar carga dinámica de contexto por topic |
| `mobile/lib/features/tutor/data/gemini_live_service.dart` | Modified | Exponer método para actualizar system prompt dinámicamente con contexto de topic o curso resumido |
| `mobile/lib/features/course/presentation/screens/*` | Modified | Agregar PopupMenuButton y diálogos de edición/borrado |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Contexto de curso completo demasiado grande para el system prompt de Gemini (token limit) | Medium | El enfoque on-demand por topic lo mitiga por defecto; para el caso "curso completo" se aplica truncamiento/resumen inteligente seleccionando los N chunks más relevantes por similarity search |
| Detección imprecisa de cambio de tema por parte del estudiante (falsos positivos/negativos en análisis de mensaje) | Medium | En MVP usar selección explícita de topic en UI además del análisis de mensaje; iterar la heurística de detección en versiones futuras |
| Latencia perceptible al cargar contexto on-demand durante la conversación | Low | El endpoint de contexto por topic ya existe y es rápido (pocos chunks por topic); cachear en memoria el contexto ya cargado en la sesión para evitar llamadas repetidas |
| Soft delete en cascada deja datos huérfanos si falla a mitad de la transacción | Low | Envolver el borrado en una transacción de base de datos; en caso de error, rollback completo |
| El fix de upload funciona en Android pero no en Web (comportamiento diferente de Dio por plataforma) | Low | El datasource ya maneja ambos paths (bytes vs file path); verificar en ambas plataformas después del fix |
| La migración de agregar `course_id` a chunks (si se elige esa ruta) requiere backfill de datos existentes | Low | Se recomienda la ruta de JOIN para evitar migración; si se elige la columna, hacerla nullable y backfill con script |
| Conflictos de concurrencia: usuario borra un topic mientras otro proceso está generando embeddings | Low | Verificar existencia del topic antes de guardar chunks; el soft delete no rompe FK constraints |
| El estudiante asume que el tutor "sabe todo" del curso sin seleccionar tema, generando respuestas sin contexto | Medium | El system prompt inicial incluye la lista de topics y orienta al estudiante a elegir un tema; en futuro con voz, el asistente preguntará proactivamente |

## Rollback Plan

- **Bloque 1**: El endpoint existente `GET .../topics/:topic_id/context` no se modifica. El nuevo `GET /courses/:course_id/context` es aditivo. La lógica de carga dinámica en mobile se puede desactivar con un feature flag `useOnDemandContext = true/false`; si se desactiva, se vuelve al comportamiento de cargar todo el contexto al inicio (comportamiento anterior). El system prompt inicial sin contexto masivo es un cambio seguro de revertir.
- **Bloque 2**: Si el fix causa regresión, restaurar la línea `options: Options(contentType: 'multipart/form-data')` (un revert de una línea). Improbable dado que el fix sigue las mejores prácticas de Dio.
- **Bloque 3**: Las nuevas rutas PATCH/DELETE son aditivas y no modifican las existentes (POST/GET). Si hay issues, se pueden deshabilitar desregistrando las rutas en el handler. En mobile, los botones de editar/borrar se pueden ocultar con un feature flag sin afectar la funcionalidad existente.

## Dependencies

- **Bloque 2 debe completarse antes que Bloque 1**: Sin upload funcional no se pueden probar los chunks generados ni el contexto de curso.
- **Bloque 3 es independiente** de los otros dos y puede desarrollarse en paralelo.
- **Infraestructura existente es suficiente**: pgvector, Cloud SQL, GCS y el pipeline de embedding ya están operativos. No se requieren dependencias externas nuevas.
- **No se requieren nuevas migraciones SQL** si se usa la estrategia de JOIN para el contexto de curso (recomendado).

## Success Criteria

- [ ] La sesión de tutoría se abre a nivel de curso sin inyectar contexto masivo en el system prompt inicial (el prompt solo contiene instrucciones del tutor y la lista de topics).
- [ ] Al indicar el estudiante que quiere hablar de un topic específico, el contexto de ese topic se carga dinámicamente y se inyecta en la conversación (verificable viendo que el system prompt se actualiza con los chunks del topic solicitado).
- [ ] Al solicitar el estudiante hablar del curso completo, se carga un resumen/selección de chunks que no excede el límite de tokens de Gemini.
- [ ] El material subido funciona como guía estructural: el tutor referencia la estructura del curso pero amplía las explicaciones usando las capacidades de Gemini, no limitándose al contenido literal.
- [ ] El upload de material desde la app móvil completa exitosamente y el material aparece en estado "validated" con texto extraído y chunks generados.
- [ ] Se pueden editar el nombre de un curso y el título de un topic desde la app móvil, y los cambios persisten tras refrescar.
- [ ] Se puede eliminar un curso desde la app móvil y sus topics, materials y chunks asociados quedan soft-deleted.
- [ ] Se puede eliminar un topic individual desde la app móvil con confirmación previa.
- [ ] Todos los endpoints nuevos validan ownership (un usuario no puede editar/borrar recursos de otro).
