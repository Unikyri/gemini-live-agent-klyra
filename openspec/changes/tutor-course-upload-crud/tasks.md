# Tasks: Tutor por Curso, Upload de Material y CRUD Completo

> Orden de ejecución: **Bloque 2 → Bloque 1 → Bloque 3** (Bloque 3 puede ejecutarse en paralelo con Bloque 1).
> Bloque 2 es prerequisito de Bloque 1 porque sin upload funcional no hay chunks para probar el contexto.

---

## Estado

- **Archivado**: 2026-03-10
- **Verificación**: pendiente (tareas manuales sin completar: B2-T2, E2E-T1, E2E-T2)

---

## Bloque 2 — Fix Upload de Material (prerequisito)

- [x] **B2-T1: Eliminar `contentType` forzado en `material_remote_datasource.dart`**
  Archivo: `mobile/lib/features/course/data/material_remote_datasource.dart` (~línea 59).
  Eliminar el parámetro `options: Options(contentType: 'multipart/form-data')` de la llamada `_dio.post(...)` para que Dio genere el boundary automáticamente.
  Verificación: el código no contiene `Options(contentType:` en la llamada de upload; el POST envía `Content-Type: multipart/form-data; boundary=...` correctamente.

- [ ] **B2-T2: Verificar flujo completo upload → processing → validated**
  Subir un PDF desde la app móvil (Android) y confirmar respuesta 201 con `status: "pending"`. Hacer polling de `GET .../materials` hasta que `status == "validated"` y `extracted_text` no esté vacío. Confirmar que se generaron chunks verificando `GET .../context` del topic.
  Verificación: criterios B2-AC1, B2-AC2, B2-AC3.
  **Nota:** Verificación manual por el usuario; no hay pipeline E2E automático en el proyecto.

---

## Bloque 1 — Tutor por Curso con Contexto Bajo Demanda

### Fase 1.1: Backend — Repositorio de chunks a nivel de curso

- [x] **B1-T1: Agregar `GetChunksByCourse` y `SearchSimilarByCourse` al port `ChunkRepository`**
  Archivo: `backend/internal/core/ports/rag_port.go`.
  Agregar a la interfaz `ChunkRepository`:
  - `GetChunksByCourse(ctx context.Context, courseID string) ([]domain.MaterialChunk, error)`
  - `SearchSimilarByCourse(ctx context.Context, courseID string, queryEmbedding []float32, topK int) ([]domain.RAGResult, error)`
  Verificación: el código compila sin errores.

- [x] **B1-T2: Implementar `GetChunksByCourse` en el repositorio concreto**
  Archivo: `backend/internal/repositories/chunk_repository.go`.
  SQL: `SELECT mc.* FROM material_chunks mc JOIN topics t ON mc.topic_id = t.id WHERE t.course_id = $1 AND t.deleted_at IS NULL ORDER BY t.order_index ASC, mc.chunk_index ASC`.
  Verificación: query retorna chunks solo de topics no-deleted del curso indicado, ordenados por topic y chunk_index.

- [x] **B1-T3: Implementar `SearchSimilarByCourse` en el repositorio concreto**
  Archivo: `backend/internal/repositories/chunk_repository.go`.
  SQL: `SELECT mc.*, (mc.embedding <=> $2) AS similarity FROM material_chunks mc JOIN topics t ON mc.topic_id = t.id WHERE t.course_id = $1 AND t.deleted_at IS NULL ORDER BY similarity ASC LIMIT $3`.
  Verificación: retorna top-K chunks ordenados por similitud, filtrados al curso correcto.

### Fase 1.2: Backend — Use case `GetCourseContext`

- [x] **B1-T4: Agregar método `GetCourseContext` a `RAGUseCase`**
  Archivo: `backend/internal/core/usecases/rag_usecase.go`.
  Definir constantes `maxChunksPerTopic = 3`, `maxCourseChunks = 30`.
  Lógica:
  - Sin query → `GetChunksByCourse` + agrupar por topic + top-3 por topic + max 30 total + concatenar con encabezados `### <título topic>`.
  - Con query → `Embed(query)` + `SearchSimilarByCourse(courseID, embedding, 10)` + concatenar con encabezados.
  Agregar dependencia de `topicRepo` al struct si no existe.
  Verificación: el método retorna contexto truncado correctamente y el campo `truncated` refleja si hubo recorte (criterio B1-AC3).

### Fase 1.3: Backend — Handler y ruta del endpoint

- [x] **B1-T5: Agregar handler `GetCourseContext` en `RAGHandler` y registrar ruta**
  Archivo: `backend/internal/handlers/http/rag_handler.go`.
  Registrar `rg.GET("/courses/:course_id/context", h.GetCourseContext)`.
  Handler: extraer `course_id` y `query` de params, validar ownership (inyectar `courseUseCase` como dependencia o verificar directamente), llamar `ragUseCase.GetCourseContext(ctx, courseID, query)`, retornar JSON `{ course_id, context, query, truncated }`.
  Errores: 401, 403, 404, 500 según spec.
  Verificación: criterios B1-AC3, B1-AC4, B1-AC5.

- [x] **B1-T6: Actualizar wiring en `main.go` para las nuevas dependencias de `RAGUseCase` y `RAGHandler`**
  Archivo: `backend/cmd/api/main.go`.
  Pasar `topicRepo` al constructor de `RAGUseCase`. Pasar `courseUseCase` (o `courseRepo`) a `RAGHandler` si se requiere para validación de ownership.
  Verificación: la app arranca sin errores de inyección de dependencias.

### Fase 1.4: Mobile — Sesión de tutoría a nivel de curso

- [x] **B1-T7: Refactorizar `TutorSessionController` para iniciar sesión a nivel de curso**
  Archivo: `mobile/lib/features/tutor/presentation/tutor_session_controller.dart`.
  Cambios:
  - `startSession(courseId)` sin `topicId` obligatorio (pasa a ser opcional).
  - Agregar estado `currentTopicId` y `loadedTopicIds` (Set).
  - Agregar método `loadTopicContext(courseId, topicId)` que llama a `GET /courses/:course_id/topics/:topic_id/context` e inyecta el contexto vía `GeminiLiveService.sendContextUpdate(...)`.
  - Agregar método `loadCourseContext(courseId)` que llama a `GET /courses/:course_id/context`.
  - Agregar estado `isLoadingContext`.
  Verificación: el controller arranca sesión sin cargar contexto masivo; `loadTopicContext` actualiza `currentTopicId` y envía contexto al WebSocket.

- [x] **B1-T8: Agregar `sendContextUpdate` y refactorizar system prompt en `GeminiLiveService`**
  Archivo: `mobile/lib/features/tutor/data/gemini_live_service.dart`.
  Agregar método `sendContextUpdate(String contextText)` que envía un mensaje `client_content` al WebSocket con formato `[CONTEXTO DEL TEMA SELECCIONADO]\n$contextText`.
  Refactorizar `_buildSystemPrompt` para recibir lista de topic titles y generar prompt sin chunks (solo instrucciones de tutor + lista de topics + directiva de preguntar al estudiante qué tema quiere).
  Verificación: el system prompt inicial tiene ≤ ~500 tokens; `sendContextUpdate` envía el JSON correcto al WebSocket (criterio B1-AC1).

- [x] **B1-T9: Agregar `fetchCourseContext` al datasource y repositorio mobile**
  Archivo: `mobile/lib/features/course/data/course_remote_datasource.dart`.
  Agregar método `fetchCourseContext(String courseId, {String? query})` → `GET /courses/:course_id/context`.
  Archivo: `mobile/lib/features/course/data/course_repository.dart`.
  Exponer `fetchCourseContext` hacia el controller.
  Verificación: la llamada HTTP retorna el JSON con `context`, `course_id`, `truncated`.

- [x] **B1-T10: Refactorizar `TutorSessionScreen` con selector de topics**
  Archivo: `mobile/lib/features/tutor/presentation/tutor_session_screen.dart`.
  Cambios:
  - Recibir `courseId` como parámetro obligatorio; `topicId` como opcional.
  - Mostrar selector de topics (chips o dropdown) con la lista de topics del curso.
  - Al seleccionar un topic, llamar a `controller.loadTopicContext(courseId, topicId)`.
  - Agregar botón/chip "Curso completo" que llama a `controller.loadCourseContext(courseId)`.
  - Si se proporcionó `topicId` inicial, pre-seleccionarlo y cargar su contexto automáticamente.
  - Mostrar indicador de carga mientras se obtiene el contexto.
  Verificación: criterios B1-AC2, B1-AC6, B1-AC7.

---

## Bloque 3 — CRUD Completo (paralelo a Bloque 1)

### Fase 3.1: Backend — Interfaces (ports)

- [x] **B3-T1: Extender interfaces `CourseRepository` y `TopicRepository` en ports**
  Archivo: `backend/internal/core/ports/course_port.go`.
  Agregar a `CourseRepository`:
  - `Update(ctx context.Context, course *domain.Course) error`
  - `SoftDelete(ctx context.Context, id string) error`
  Agregar a `TopicRepository`:
  - `Update(ctx context.Context, topic *domain.Topic) error`
  - `SoftDelete(ctx context.Context, id string) error`
  - `FindByCourseForCascade(ctx context.Context, courseID string) ([]domain.Topic, error)`
  Agregar a `MaterialRepository` (si no existe):
  - `SoftDeleteByTopicIDs(ctx context.Context, topicIDs []string) error`
  Agregar a `ChunkRepository` (en `rag_port.go`):
  - `SoftDeleteByTopicIDs(ctx context.Context, topicIDs []string) error`
  **Nota sobre chunks**: Según design, se recomienda hard delete de chunks en cascada (son datos derivados regenerables). Si se elige esta ruta, el método se llamará `HardDeleteByTopicIDs`. Decidir en implementación.
  Verificación: compila sin errores.

- [x] **B3-T2: Adaptar `course_repository.go` para satisfacer la interfaz de ports**
  Archivo: `backend/internal/repositories/course_repository.go`.
  Adaptar métodos `UpdateCourse`/`DeleteCourse` existentes (o crear nuevos) para la firma `Update(ctx, *domain.Course) error` y `SoftDelete(ctx, id string) error`. Usar `context.Context` y `string` para IDs.
  Verificación: `CourseRepository` implementa la interfaz completa de ports.

- [x] **B3-T3: Adaptar `topic_repository.go` para satisfacer la interfaz de ports**
  Archivo: `backend/internal/repositories/topic_repository.go`.
  Adaptar/crear `Update(ctx, *domain.Topic) error`, `SoftDelete(ctx, id string) error`, `FindByCourseForCascade(ctx, courseID string) ([]domain.Topic, error)`.
  `FindByCourseForCascade` retorna todos los topics del curso (incluyendo soft-deleted) para obtener IDs para la cascada.
  Verificación: `TopicRepository` implementa la interfaz completa.

- [x] **B3-T4: Agregar `SoftDeleteByTopicIDs` (o `HardDeleteByTopicIDs`) a repositorios de Material y Chunk**
  Archivos: repositorios concretos de Material y Chunk.
  Para Material: `UPDATE materials SET deleted_at = NOW() WHERE topic_id IN (?) AND deleted_at IS NULL`.
  Para Chunk: `DELETE FROM material_chunks WHERE topic_id IN (?)` (hard delete según design; alternativamente soft delete si se agrega migración).
  Verificación: ambos métodos eliminan/marcan correctamente los registros asociados a los topic IDs indicados.

### Fase 3.2: Backend — Use cases CRUD

- [x] **B3-T5: Agregar `UpdateCourse` y `DeleteCourse` a `CourseUseCase`**
  Archivo: `backend/internal/core/usecases/course_usecase.go`.
  `UpdateCourse(ctx, courseID, userID, input)`: obtener curso → verificar ownership → aplicar actualización parcial (solo campos no vacíos) → `courseRepo.Update(ctx, course)` → retornar curso actualizado.
  `DeleteCourse(ctx, courseID, userID)`: obtener curso → verificar ownership → obtener topic IDs (`topicRepo.FindByCourseForCascade`) → en transacción: `chunkRepo.HardDeleteByTopicIDs` → `materialRepo.SoftDeleteByTopicIDs` → `topicRepo.SoftDelete` cada uno → `courseRepo.SoftDelete` → retornar nil.
  Agregar dependencias `materialRepo`, `chunkRepo`, `db` al constructor del use case.
  Verificación: ownership check retorna error si `userID` no coincide; cascada ejecuta todas las operaciones en orden dentro de transacción (criterio B3-AC4, B3-AC16).

- [x] **B3-T6: Agregar `UpdateTopic` y `DeleteTopic` a `CourseUseCase`**
  Archivo: `backend/internal/core/usecases/course_usecase.go`.
  `UpdateTopic(ctx, courseID, topicID, userID, input)`: obtener curso → verificar ownership → obtener topic → verificar `topic.CourseID == courseID` → actualizar `title` → `topicRepo.Update(ctx, topic)` → retornar topic.
  `DeleteTopic(ctx, courseID, topicID, userID)`: obtener curso → verificar ownership → verificar topic pertenece al curso → en transacción: `chunkRepo.HardDeleteByTopicIDs([]string{topicID})` → `materialRepo.SoftDeleteByTopicIDs([]string{topicID})` → `topicRepo.SoftDelete(topicID)` → retornar nil.
  Verificación: criterios B3-AC7, B3-AC8, B3-AC9.

### Fase 3.3: Backend — Handlers y rutas PATCH/DELETE

- [x] **B3-T7: Agregar handlers `UpdateCourse` y `DeleteCourse` en `CourseHandler`**
  Archivo: `backend/internal/handlers/http/course_handler.go`.
  Registrar rutas:
  - `rg.PATCH("/courses/:course_id", h.UpdateCourse)`
  - `rg.DELETE("/courses/:course_id", h.DeleteCourse)`
  `UpdateCourse`: bind JSON body → extraer `user_id` y `course_id` → llamar `courseUseCase.UpdateCourse(...)` → retornar 200 con curso actualizado.
  `DeleteCourse`: extraer `user_id` y `course_id` → llamar `courseUseCase.DeleteCourse(...)` → retornar 200 `{"message": "course deleted"}`.
  Errores: 400, 401, 403, 404, 500.
  Verificación: criterios B3-AC1, B3-AC2, B3-AC3, B3-AC5, B3-AC6.

- [x] **B3-T8: Agregar handlers `UpdateTopic` y `DeleteTopic` en `TopicHandler`**
  Archivo: `backend/internal/handlers/http/topic_handler.go`.
  Registrar rutas:
  - `rg.PATCH("/courses/:course_id/topics/:topic_id", h.UpdateTopic)`
  - `rg.DELETE("/courses/:course_id/topics/:topic_id", h.DeleteTopic)`
  Inyectar `courseUseCase` como dependencia del handler para acceder a `UpdateTopic`/`DeleteTopic`.
  `UpdateTopic`: bind body → extraer params → llamar `courseUseCase.UpdateTopic(...)` → retornar 200 con topic.
  `DeleteTopic`: extraer params → llamar `courseUseCase.DeleteTopic(...)` → retornar 200 `{"message": "topic deleted"}`.
  Errores: 400, 401, 403, 404, 500.
  Verificación: criterios B3-AC7, B3-AC8, B3-AC9, B3-AC10.

- [x] **B3-T9: Actualizar wiring en `main.go` para las nuevas dependencias del Bloque 3**
  Archivo: `backend/cmd/api/main.go`.
  Pasar `materialRepo`, `chunkRepo`, `db` al constructor de `CourseUseCase`.
  Pasar `courseUseCase` al constructor de `TopicHandler`.
  Verificación: la app arranca sin errores; las nuevas rutas responden.

### Fase 3.4: Mobile — Datasource y repositorio

- [x] **B3-T10: Agregar métodos CRUD en `CourseRemoteDataSource`**
  Archivo: `mobile/lib/features/course/data/course_remote_datasource.dart`.
  Agregar:
  - `updateCourse(String courseId, {String? name, String? educationLevel})` → `PATCH /courses/:course_id`
  - `deleteCourse(String courseId)` → `DELETE /courses/:course_id`
  - `updateTopic(String courseId, String topicId, {String? title})` → `PATCH /courses/:course_id/topics/:topic_id`
  - `deleteTopic(String courseId, String topicId)` → `DELETE /courses/:course_id/topics/:topic_id`
  Verificación: cada método envía el request HTTP correcto y parsea la respuesta.

- [x] **B3-T11: Exponer métodos CRUD en `CourseRepository` mobile**
  Archivo: `mobile/lib/features/course/data/course_repository.dart`.
  Exponer `updateCourse`, `deleteCourse`, `updateTopic`, `deleteTopic` delegando al datasource.
  Verificación: el repositorio compila y expone los cuatro métodos.

### Fase 3.5: Mobile — Controller

- [x] **B3-T12: Agregar acciones CRUD en `CourseController`**
  Archivo: `mobile/lib/features/course/presentation/course_controller.dart`.
  Agregar métodos:
  - `updateCourse(courseId, {name, educationLevel})` → llama al repo → invalida estado → refresca lista.
  - `deleteCourse(courseId)` → llama al repo → invalida estado → refresca lista.
  - `updateTopic(courseId, topicId, {title})` → llama al repo → invalida estado → refresca.
  - `deleteTopic(courseId, topicId)` → llama al repo → invalida estado → refresca.
  Patrón: `state = AsyncValue.loading()` → ejecutar → `_fetchCourses()`.
  Verificación: tras editar/borrar, la lista de cursos/topics se actualiza sin pull-to-refresh manual (criterio B3-AC13).

### Fase 3.6: Mobile — UI (PopupMenuButton + diálogos)

- [x] **B3-T13: Agregar `PopupMenuButton` y diálogos de edición/borrado en `CourseDashboardScreen`**
  Archivo: `mobile/lib/features/course/presentation/course_dashboard_screen.dart` (o el archivo que contiene `_CourseCard`).
  Agregar `PopupMenuButton` (icono `⋮`) en la esquina superior derecha de cada card de curso con opciones:
  - "Editar nombre" → `AlertDialog` con `TextField` pre-poblado + selector de nivel educativo. Al guardar → `controller.updateCourse(...)`.
  - "Eliminar curso" → `AlertDialog` de confirmación: "¿Eliminar curso? Se eliminará «<nombre>» y todos sus temas, materiales y contenido. Esta acción no se puede deshacer." Botón "Eliminar" en rojo. Al confirmar → `controller.deleteCourse(...)`.
  Verificación: criterios B3-AC11, B3-AC12, B3-AC13.

- [x] **B3-T14: Agregar `PopupMenuButton` y diálogos de edición/borrado en `CourseDetailScreen`**
  Archivo: `mobile/lib/features/course/presentation/course_detail_screen.dart` (o el archivo que contiene `_TopicSection`).
  Agregar `PopupMenuButton` junto al título de cada topic con opciones:
  - "Editar título" → `AlertDialog` con `TextField` pre-poblado. Al guardar → `controller.updateTopic(...)`.
  - "Eliminar tema" → `AlertDialog` de confirmación: "¿Eliminar tema? Se eliminará «<título>» y todo su material asociado. Esta acción no se puede deshacer." Botón "Eliminar" en rojo. Al confirmar → `controller.deleteTopic(...)`.
  Verificación: criterios B3-AC14, B3-AC15.

---

## Fase Final — Verificación E2E / Smoke Test

- [ ] **E2E-T1: Smoke test del flujo completo Bloque 2 → Bloque 1**
  Ejecutar en dispositivo/emulador:
  1. Subir un PDF a un topic (Bloque 2 fix) → verificar 201 y que el material llega a `validated`.
  2. Abrir sesión de tutor a nivel de curso → verificar que el system prompt NO contiene chunks.
  3. Seleccionar el topic en el selector → verificar que el contexto se inyecta y el tutor responde con referencia al material.
  4. Seleccionar "Curso completo" → verificar que se carga contexto resumido.
  Criterios cubiertos: B2-AC1–AC3, B1-AC1, B1-AC2, B1-AC6.

- [ ] **E2E-T2: Smoke test del flujo CRUD (Bloque 3)**
  Ejecutar en dispositivo/emulador:
  1. Editar nombre de un curso desde la UI → verificar que el nombre se actualiza en la lista.
  2. Editar título de un topic → verificar actualización.
  3. Eliminar un topic con confirmación → verificar que desaparece y sus materials/chunks están borrados en BD.
  4. Eliminar un curso con confirmación → verificar cascada completa.
  5. Intentar editar/borrar un recurso ajeno (con otro JWT) → verificar 403.
  Criterios cubiertos: B3-AC1–AC16.
