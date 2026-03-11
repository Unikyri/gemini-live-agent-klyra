# Spec: Tutor por Curso, Upload de Material y CRUD Completo

> Generada a partir de `openspec/changes/tutor-course-upload-crud/proposal.md`.
> Objetivo: definir contratos, comportamiento y criterios de aceptación con detalle suficiente para que `sdd-tasks` pueda desglosar las tareas de implementación.

---

## Bloque 1 — Tutor por curso con contexto bajo demanda

### 1.1 Endpoint existente: obtener contexto de un topic

| Campo | Valor |
|-------|-------|
| **Método** | `GET` |
| **Path** | `/api/v1/courses/:course_id/topics/:topic_id/context` |
| **Auth** | Bearer JWT (header `Authorization`). El middleware inyecta `user_id` en el contexto de Gin. |
| **Path params** | `course_id` (UUID), `topic_id` (UUID) |
| **Query params** | `query` (string, opcional). Si vacío → devuelve todo el contexto del topic (concatenación de chunks). Si presente → embedding de la query + similarity search (top 5 chunks). |

**Respuesta exitosa (200)**

```json
{
  "topic_id": "<uuid>",
  "context": "<texto concatenado de los chunks relevantes>",
  "query": "<string vacío o la query enviada>"
}
```

**Errores**

| Código | Condición |
|--------|-----------|
| 401 | JWT ausente o inválido |
| 500 | Error interno al recuperar chunks o generar embedding |

**Comportamiento actual (sin cambios en este endpoint)**
- `query` vacío → `RAGUseCase.GetTopicContext` llama a `ChunkRepository.GetChunksByTopic(topicID)` y concatena todos los chunks separados por `\n\n`.
- `query` presente → genera embedding con `Embedder.Embed(query)`, luego `ChunkRepository.SearchSimilar(topicID, embedding, 5)` y concatena los 5 chunks más similares.

---

### 1.2 Nuevo endpoint: obtener contexto resumido de un curso

| Campo | Valor |
|-------|-------|
| **Método** | `GET` |
| **Path** | `/api/v1/courses/:course_id/context` |
| **Auth** | Bearer JWT. Validación de ownership (el curso debe pertenecer al `user_id` del JWT). |
| **Path params** | `course_id` (UUID) |
| **Query params** | `query` (string, opcional). Si presente → similarity search entre todos los chunks del curso. Si vacío → selección por truncamiento (ver lógica abajo). |

**Respuesta exitosa (200)**

```json
{
  "course_id": "<uuid>",
  "context": "<texto con los chunks seleccionados/resumidos>",
  "query": "<string vacío o la query enviada>",
  "truncated": true
}
```

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `course_id` | string (UUID) | ID del curso solicitado |
| `context` | string | Texto concatenado de los chunks seleccionados, separados por `\n\n`. Cada sección precedida por un encabezado `### <título del topic>` |
| `query` | string | Query enviada (o vacío) |
| `truncated` | boolean | `true` si el contexto fue recortado respecto al total disponible |

**Errores**

| Código | Condición |
|--------|-----------|
| 401 | JWT ausente o inválido |
| 403 | El curso no pertenece al usuario autenticado |
| 404 | Curso no encontrado (o soft-deleted) |
| 500 | Error interno |

**Lógica de truncamiento y selección**

El método `RAGUseCase.GetCourseContext(ctx, courseID, query)` aplica la siguiente estrategia:

1. **Sin query** (contexto general del curso):
   - Obtener todos los chunks del curso mediante `ChunkRepository.GetChunksByCourse(courseID)` (JOIN `material_chunks.topic_id → topics.course_id`, filtrando `topics.deleted_at IS NULL`).
   - Ordenar por `topics.order_index ASC`, luego `material_chunks.chunk_index ASC`.
   - **Límite de truncamiento**: máximo **30 chunks** en total (configurable con constante `maxCourseContextChunks = 30`). Si el curso tiene más, se reparten equitativamente entre topics (floor division), priorizando los primeros chunks de cada topic.
   - Agrupar el texto resultante por topic, precediendo cada grupo con `### <título del topic>`.
   - Campo `truncated = true` si se descartaron chunks.

2. **Con query** (búsqueda semántica cross-topic):
   - Generar embedding de la query.
   - Ejecutar similarity search en todos los chunks del curso (nuevo método `ChunkRepository.SearchSimilarByCourse(courseID, embedding, topK=10)`).
   - Devolver los 10 chunks más similares, agrupados por topic con encabezado.
   - `truncated = true` siempre (es una selección por relevancia).

**Nuevo método en ChunkRepository (port)**

```go
GetChunksByCourse(ctx context.Context, courseID string) ([]domain.MaterialChunk, error)
SearchSimilarByCourse(ctx context.Context, courseID string, queryEmbedding []float32, topK int) ([]domain.RAGResult, error)
```

**SQL subyacente para `GetChunksByCourse`**

```sql
SELECT mc.* FROM material_chunks mc
JOIN topics t ON mc.topic_id = t.id
WHERE t.course_id = $1
  AND t.deleted_at IS NULL
  AND mc.deleted_at IS NULL
ORDER BY t.order_index ASC, mc.chunk_index ASC
```

**SQL subyacente para `SearchSimilarByCourse`**

```sql
SELECT mc.*, (mc.embedding <=> $2) AS similarity
FROM material_chunks mc
JOIN topics t ON mc.topic_id = t.id
WHERE t.course_id = $1
  AND t.deleted_at IS NULL
ORDER BY similarity ASC
LIMIT $3
```

**Registro de ruta en `RAGHandler`**

```go
rg.GET("/courses/:course_id/context", h.GetCourseContext)
```

---

### 1.3 Flujo en mobile: sesión de tutoría a nivel de curso

#### 1.3.1 Inicio de sesión

1. El usuario navega a la pantalla del tutor desde cualquier topic o desde el detalle del curso.
2. `TutorSessionScreen` recibe `courseId` como parámetro obligatorio. El parámetro `topicId` pasa a ser **opcional** (si viene, se pre-carga ese topic).
3. Al iniciar sesión (`startSession`), el controlador:
   a. Llama a `GET /api/v1/courses/:course_id` para obtener la lista de topics del curso.
   b. **No** solicita contexto RAG en este paso.
   c. Conecta a Gemini Live con un **system prompt inicial sin contexto masivo**:

```
Eres Klyra, una tutora de IA entusiasta, paciente y motivadora.
Tu objetivo es ayudar al estudiante a comprender el material de su curso a través de conversación natural.
Habla de forma clara y a un ritmo apropiado.
Haz preguntas para verificar la comprensión.
Celebra las respuestas correctas y corrige los errores con amabilidad.

Este curso se llama "<nombre del curso>" (nivel: <education_level>).
Los temas disponibles son:
1. <título topic 1>
2. <título topic 2>
...

Pregunta al estudiante de qué tema quiere hablar. Si no elige uno, ofrécele las opciones.
NO tienes contexto del material aún — se cargará cuando el estudiante elija un tema.
```

   d. Si se proporcionó `topicId`, automáticamente se solicita el contexto de ese topic (paso 1.3.2).

#### 1.3.2 Carga de contexto por topic (on-demand)

Cuando el estudiante indica que quiere hablar de un topic específico (selección explícita en UI o detección en el mensaje):

1. El controlador llama a `GET /api/v1/courses/:course_id/topics/:topic_id/context` (sin query param; contexto completo del topic).
2. Se actualiza el contexto de la sesión de Gemini Live inyectando el texto recibido. `GeminiLiveService` expone un nuevo método:

```dart
Future<void> updateContext(String newContext, String topicTitle)
```

   Este método envía un mensaje de sistema a la sesión activa que indica:

```
[Contexto actualizado — Tema: <topicTitle>]
<newContext>
```

3. El controlador guarda en estado local qué topic está activo (`currentTopicId`) para evitar recargar el mismo contexto.

#### 1.3.3 Carga de contexto de curso completo (fallback)

Si el estudiante pide explícitamente "quiero hablar de todo el curso" o "dame un resumen general":

1. El controlador llama a `GET /api/v1/courses/:course_id/context` (sin query param).
2. Se inyecta el contexto resumido del curso usando el mismo método `updateContext`.
3. Se marca `currentTopicId = null` para indicar modo "curso completo".

#### 1.3.4 Cambio de tema durante la sesión

1. Si el estudiante dice "ahora quiero hablar de <otro tema>":
   - En MVP: el usuario selecciona el topic desde un selector en la UI (menú desplegable o lista lateral).
   - Futuro: detección automática por análisis del mensaje.
2. Se ejecuta el flujo 1.3.2 con el nuevo `topicId`.
3. El contexto anterior se reemplaza por el nuevo (no se acumula).

---

### 1.4 Criterios de aceptación — Bloque 1

| ID | Criterio | Verificación |
|----|----------|-------------|
| B1-AC1 | La sesión de tutoría se abre a nivel de curso. El system prompt inicial contiene instrucciones del tutor y lista de topics, pero **no** contiene chunks de material. | Inspeccionar el payload WebSocket enviado a Gemini al conectar; el campo `system_instruction` no debe superar ~500 tokens. |
| B1-AC2 | Al seleccionar un topic en la UI, el contexto de ese topic se carga dinámicamente y la siguiente respuesta del tutor lo utiliza como referencia. | Enviar pregunta sobre un topic después de cargarlo; la respuesta debe referenciar contenido del material subido. |
| B1-AC3 | Al solicitar contexto de curso completo, `GET /courses/:course_id/context` devuelve un contexto truncado a máximo 30 chunks, agrupado por topic con encabezados. | Llamar al endpoint con un curso que tenga >30 chunks; verificar que `truncated=true` y que el texto contiene encabezados `### <topic>`. |
| B1-AC4 | El endpoint `GET /courses/:course_id/context` valida ownership. Un usuario no puede obtener contexto de un curso ajeno. | Llamar con JWT de otro usuario; esperar 403. |
| B1-AC5 | El endpoint `GET /courses/:course_id/context?query=<texto>` devuelve los 10 chunks más relevantes del curso por similarity search. | Enviar query relacionada con un tema; verificar que los chunks devueltos son del topic correcto. |
| B1-AC6 | El selector de topics está visible en la UI de la sesión de tutoría, permitiendo cambiar de tema sin reiniciar la sesión. | Abrir sesión de tutor, cambiar de topic, verificar que el contexto se actualiza y las respuestas reflejan el nuevo tema. |
| B1-AC7 | Si se navega al tutor desde un topic específico (parámetro `topicId`), el contexto de ese topic se carga automáticamente al iniciar. | Navegar desde la pantalla de summary de un topic; verificar que el contexto ya está cargado al empezar a conversar. |

---

## Bloque 2 — Fix upload de material

### 2.1 Endpoint de upload (existente, sin cambios en backend)

| Campo | Valor |
|-------|-------|
| **Método** | `POST` |
| **Path** | `/api/v1/courses/:course_id/topics/:topic_id/materials` |
| **Auth** | Bearer JWT. Validación de ownership del curso. |
| **Content-Type** | `multipart/form-data` (generado automáticamente por el cliente HTTP; **no** se debe forzar manualmente). |
| **Body** | `FormData` con campo `file` (archivo binario). |
| **Límites** | 20 MB general, 50 MB audio. Formatos: PDF, TXT, MD, PNG, JPG, JPEG, WEBP, MP3, WAV, M4A. |

**Respuesta exitosa (201)**

```json
{
  "id": "<uuid>",
  "topic_id": "<uuid>",
  "format_type": "pdf",
  "storage_url": "https://storage.googleapis.com/...",
  "extracted_text": "",
  "status": "pending",
  "original_name": "apuntes-calculo.pdf",
  "size_bytes": 1048576,
  "created_at": "2026-03-10T...",
  "updated_at": "2026-03-10T..."
}
```

El material se crea con `status: "pending"`. El pipeline asíncrono lo procesa: `pending → processing → validated` (con `extracted_text` y chunks generados) o `rejected` (si falla la extracción).

**Errores**

| Código | Condición |
|--------|-----------|
| 400 | Falta campo `file` o archivo ilegible |
| 401 | JWT ausente o inválido |
| 403 | El curso no pertenece al usuario |
| 404 | Curso o topic no encontrado |
| 413 | Archivo excede límite de tamaño |
| 415 | Formato no soportado (extensión o MIME) |
| 500 | Error interno |

### 2.2 Cambio requerido en mobile

**Archivo**: `mobile/lib/features/course/data/material_remote_datasource.dart`

**Problema**: La línea `options: Options(contentType: 'multipart/form-data')` fuerza el header `Content-Type` sin boundary. Dio, al enviar un `FormData`, genera automáticamente el header `Content-Type: multipart/form-data; boundary=----...`. Al forzar el header manualmente, se sobrescribe el boundary, causando que el backend (Gin) no pueda parsear el multipart y devuelva HTTP 400.

**Fix**: Eliminar el parámetro `options` de la llamada `_dio.post(...)`:

```dart
// ANTES (roto)
final response = await _dio.post(
  '/courses/$courseId/topics/$topicId/materials',
  data: formData,
  options: Options(contentType: 'multipart/form-data'),
);

// DESPUÉS (correcto)
final response = await _dio.post(
  '/courses/$courseId/topics/$topicId/materials',
  data: formData,
);
```

No se requieren cambios en backend.

### 2.3 Criterios de aceptación — Bloque 2

| ID | Criterio | Verificación |
|----|----------|-------------|
| B2-AC1 | Un archivo PDF subido desde la app móvil (Android) llega al backend y se crea con `status: "pending"`. | Subir un PDF, verificar respuesta 201 con los campos esperados. |
| B2-AC2 | El material subido completa el pipeline asíncrono: su `status` transiciona de `pending` → `processing` → `validated`, con `extracted_text` no vacío. | Tras subir, hacer polling de `GET .../materials` hasta que `status == "validated"`. |
| B2-AC3 | Se generan chunks con embeddings para el material validado. | Verificar que `GET .../context` devuelve contenido no vacío para el topic donde se subió el material. |
| B2-AC4 | El cliente Dart **no** fuerza `Content-Type` manualmente; Dio genera el boundary automáticamente. | Inspeccionar el código de `material_remote_datasource.dart`; confirmar ausencia de `Options(contentType: ...)` en la llamada de upload. |
| B2-AC5 | El upload funciona tanto en Android nativo (file path) como en web (bytes). | Probar upload desde ambas plataformas; ambos deben retornar 201. |

---

## Bloque 3 — CRUD completo de cursos y topics

### 3.1 Contratos de endpoints

#### 3.1.1 PATCH /courses/:course_id — Actualizar curso

| Campo | Valor |
|-------|-------|
| **Método** | `PATCH` |
| **Path** | `/api/v1/courses/:course_id` |
| **Auth** | Bearer JWT. Validación de ownership. |
| **Content-Type** | `application/json` |
| **Body** | JSON con campos opcionales (actualización parcial) |

**Request body**

```json
{
  "name": "Nuevo nombre del curso",
  "education_level": "university"
}
```

| Campo | Tipo | Requerido | Validación |
|-------|------|-----------|------------|
| `name` | string | No | Si presente, no vacío, max 200 caracteres |
| `education_level` | string | No | Si presente, uno de: `"elementary"`, `"middle_school"`, `"high_school"`, `"university"`, `"postgraduate"`, `"other"` |

Solo se actualizan los campos enviados (los ausentes o `null` no se modifican).

**Respuesta exitosa (200)**

```json
{
  "id": "<uuid>",
  "user_id": "<uuid>",
  "name": "Nuevo nombre del curso",
  "education_level": "university",
  "avatar_model_url": "...",
  "avatar_status": "ready",
  "reference_image_url": "...",
  "created_at": "...",
  "updated_at": "...",
  "topics": [...]
}
```

Devuelve el curso completo actualizado (misma estructura que `GET /courses/:course_id`).

**Errores**

| Código | Condición |
|--------|-----------|
| 400 | Body inválido o `name` vacío si se envía |
| 401 | JWT ausente o inválido |
| 403 | El curso no pertenece al usuario |
| 404 | Curso no encontrado (o soft-deleted) |
| 500 | Error interno |

---

#### 3.1.2 DELETE /courses/:course_id — Eliminar curso (soft delete)

| Campo | Valor |
|-------|-------|
| **Método** | `DELETE` |
| **Path** | `/api/v1/courses/:course_id` |
| **Auth** | Bearer JWT. Validación de ownership. |
| **Body** | Ninguno |

**Respuesta exitosa (204)** — Sin body.

**Comportamiento de soft delete en cascada**

Dentro de una **transacción** de base de datos:

1. Marcar `courses.deleted_at = NOW()` para el curso.
2. Marcar `topics.deleted_at = NOW()` para todos los topics donde `course_id = :course_id` y `deleted_at IS NULL`.
3. Marcar `materials.deleted_at = NOW()` para todos los materials donde `topic_id IN (topics del curso)` y `deleted_at IS NULL`.
4. Marcar `material_chunks.deleted_at = NOW()` para todos los chunks donde `topic_id IN (topics del curso)` y `deleted_at IS NULL`.

> Nota: `material_chunks` actualmente no tiene columna `deleted_at`. Se debe agregar como parte de este cambio (migración: `ALTER TABLE material_chunks ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL`). Alternativa: omitir soft delete de chunks y filtrarlos por el `deleted_at` del topic en las queries de lectura (JOIN).

Si la transacción falla en cualquier paso, se ejecuta rollback completo.

**Errores**

| Código | Condición |
|--------|-----------|
| 401 | JWT ausente o inválido |
| 403 | El curso no pertenece al usuario |
| 404 | Curso no encontrado (o ya eliminado) |
| 500 | Error interno (fallo en transacción) |

---

#### 3.1.3 PATCH /courses/:course_id/topics/:topic_id — Actualizar topic

| Campo | Valor |
|-------|-------|
| **Método** | `PATCH` |
| **Path** | `/api/v1/courses/:course_id/topics/:topic_id` |
| **Auth** | Bearer JWT. Validación de ownership del curso padre. |
| **Content-Type** | `application/json` |
| **Body** | JSON con campos opcionales |

**Request body**

```json
{
  "title": "Nuevo título del tema"
}
```

| Campo | Tipo | Requerido | Validación |
|-------|------|-----------|------------|
| `title` | string | No | Si presente, no vacío, max 200 caracteres |

**Respuesta exitosa (200)**

```json
{
  "id": "<uuid>",
  "course_id": "<uuid>",
  "title": "Nuevo título del tema",
  "order_index": 0,
  "created_at": "...",
  "updated_at": "..."
}
```

**Errores**

| Código | Condición |
|--------|-----------|
| 400 | Body inválido o `title` vacío si se envía |
| 401 | JWT ausente o inválido |
| 403 | El curso padre no pertenece al usuario |
| 404 | Curso o topic no encontrado (o soft-deleted) |
| 500 | Error interno |

---

#### 3.1.4 DELETE /courses/:course_id/topics/:topic_id — Eliminar topic (soft delete)

| Campo | Valor |
|-------|-------|
| **Método** | `DELETE` |
| **Path** | `/api/v1/courses/:course_id/topics/:topic_id` |
| **Auth** | Bearer JWT. Validación de ownership del curso padre. |
| **Body** | Ninguno |

**Respuesta exitosa (204)** — Sin body.

**Comportamiento de soft delete en cascada**

Dentro de una **transacción**:

1. Verificar que el topic pertenece al curso indicado y que el curso pertenece al usuario.
2. Marcar `topics.deleted_at = NOW()` para el topic.
3. Marcar `materials.deleted_at = NOW()` para todos los materials donde `topic_id = :topic_id` y `deleted_at IS NULL`.
4. Marcar `material_chunks.deleted_at = NOW()` (o filtrar por JOIN) para los chunks del topic.

Rollback completo si cualquier paso falla.

**Errores**

| Código | Condición |
|--------|-----------|
| 401 | JWT ausente o inválido |
| 403 | El curso padre no pertenece al usuario |
| 404 | Curso o topic no encontrado (o ya eliminado) |
| 500 | Error interno |

---

### 3.2 Interfaces de port a extender

**`CourseRepository`** — agregar:

```go
Update(ctx context.Context, course *domain.Course) error
SoftDelete(ctx context.Context, courseID string) error
```

**`TopicRepository`** — agregar:

```go
Update(ctx context.Context, topic *domain.Topic) error
SoftDelete(ctx context.Context, topicID string) error
FindByCourseForCascade(ctx context.Context, courseID string) ([]domain.Topic, error)
```

**`MaterialRepository`** — agregar (si no existe):

```go
SoftDeleteByTopicIDs(ctx context.Context, topicIDs []string) error
```

**`ChunkRepository`** — agregar:

```go
SoftDeleteByTopicIDs(ctx context.Context, topicIDs []string) error
```

> `FindByCourseForCascade` retorna topics incluyendo soft-deleted para evitar re-borrar. En cascada se necesitan los IDs de todos los topics del curso para luego borrar sus materials y chunks.

---

### 3.3 Lógica de use case

**`CourseUseCase.UpdateCourse(ctx, courseID, userID, input)`**

1. Obtener curso por ID.
2. Verificar ownership (`course.UserID == userID`); si no → `ErrCourseForbidden`.
3. Aplicar actualización parcial: solo campos no vacíos del input.
4. Llamar `courseRepo.Update(ctx, course)`.
5. Retornar curso actualizado.

**`CourseUseCase.DeleteCourse(ctx, courseID, userID)`**

1. Obtener curso por ID.
2. Verificar ownership; si no → `ErrCourseForbidden`.
3. Obtener todos los topic IDs del curso (`topicRepo.FindByCourseForCascade`).
4. En transacción:
   a. `courseRepo.SoftDelete(courseID)`
   b. `topicRepo.SoftDelete(topicID)` para cada topic (o batch).
   c. `materialRepo.SoftDeleteByTopicIDs(topicIDs)`
   d. `chunkRepo.SoftDeleteByTopicIDs(topicIDs)`
5. Retornar nil.

**`CourseUseCase.UpdateTopic(ctx, courseID, topicID, userID, input)`**

1. Obtener curso y verificar ownership.
2. Obtener topic por ID; verificar que `topic.CourseID == courseID`.
3. Aplicar actualización parcial.
4. Llamar `topicRepo.Update(ctx, topic)`.
5. Retornar topic actualizado.

**`CourseUseCase.DeleteTopic(ctx, courseID, topicID, userID)`**

1. Obtener curso y verificar ownership.
2. Obtener topic; verificar pertenencia al curso.
3. En transacción:
   a. `topicRepo.SoftDelete(topicID)`
   b. `materialRepo.SoftDeleteByTopicIDs([]string{topicID})`
   c. `chunkRepo.SoftDeleteByTopicIDs([]string{topicID})`
4. Retornar nil.

---

### 3.4 Registro de nuevas rutas

En `CourseHandler.RegisterRoutes`:

```go
rg.PATCH("/courses/:course_id", h.UpdateCourse)
rg.DELETE("/courses/:course_id", h.DeleteCourse)
rg.PATCH("/courses/:course_id/topics/:topic_id", h.UpdateTopic)
rg.DELETE("/courses/:course_id/topics/:topic_id", h.DeleteTopic)
```

---

### 3.5 Flujo de UI mobile

#### 3.5.1 Listado de cursos (`CourseDashboardScreen`)

- Cada `_CourseCard` muestra un **`PopupMenuButton`** (icono `⋮`) en la esquina superior derecha con dos opciones:
  - **"Editar nombre"** → abre `AlertDialog` con `TextField` pre-poblado con el nombre actual del curso y un selector de nivel educativo. Botones: "Cancelar", "Guardar". Al guardar → `PATCH /courses/:course_id`.
  - **"Eliminar curso"** → abre `AlertDialog` de confirmación:
    - Título: "¿Eliminar curso?"
    - Mensaje: "Se eliminará el curso «<nombre>» y todos sus temas, materiales y contenido asociado. Esta acción no se puede deshacer."
    - Botones: "Cancelar", "Eliminar" (rojo).
    - Al confirmar → `DELETE /courses/:course_id`.
- Tras editar o borrar, se invalida `courseControllerProvider` para refrescar la lista.

#### 3.5.2 Detalle de curso (`CourseDetailScreen`) — topics

- Cada `_TopicSection` muestra un **`PopupMenuButton`** junto al título del topic con dos opciones:
  - **"Editar título"** → abre `AlertDialog` con `TextField` pre-poblado con el título actual. Botones: "Cancelar", "Guardar". Al guardar → `PATCH /courses/:course_id/topics/:topic_id`.
  - **"Eliminar tema"** → abre `AlertDialog` de confirmación:
    - Título: "¿Eliminar tema?"
    - Mensaje: "Se eliminará el tema «<título>» y todo su material asociado. Esta acción no se puede deshacer."
    - Botones: "Cancelar", "Eliminar" (rojo).
    - Al confirmar → `DELETE /courses/:course_id/topics/:topic_id`.
- Tras editar o borrar, se invalida el provider para refrescar.

#### 3.5.3 Campos editables

| Entidad | Campos editables |
|---------|-----------------|
| Course | `name`, `education_level` |
| Topic | `title` |

No se editan otros campos (IDs, fechas, avatar, etc.).

---

### 3.6 Métodos a agregar en mobile

**`CourseRemoteDataSource`**:

```dart
Future<Course> updateCourse(String courseId, {String? name, String? educationLevel});
Future<void> deleteCourse(String courseId);
Future<Topic> updateTopic(String courseId, String topicId, {String? title});
Future<void> deleteTopic(String courseId, String topicId);
```

**`CourseRepository`** (o interfaz equivalente):

```dart
Future<Course> updateCourse(String courseId, {String? name, String? educationLevel});
Future<void> deleteCourse(String courseId);
Future<Topic> updateTopic(String courseId, String topicId, {String? title});
Future<void> deleteTopic(String courseId, String topicId);
```

**`CourseController`** — nuevos métodos:

```dart
Future<void> updateCourse(String courseId, {String? name, String? educationLevel});
Future<void> deleteCourse(String courseId);
Future<void> updateTopic(String courseId, String topicId, {String? title});
Future<void> deleteTopic(String courseId, String topicId);
```

Cada método sigue el patrón existente: `state = AsyncValue.loading()` → ejecutar → `_fetchCourses()` para refrescar.

---

### 3.7 Criterios de aceptación — Bloque 3

| ID | Criterio | Verificación |
|----|----------|-------------|
| B3-AC1 | `PATCH /courses/:course_id` con `{"name": "X"}` actualiza solo el nombre; los demás campos no cambian. | Enviar PATCH con solo `name`, verificar que `education_level` no cambió en la respuesta. |
| B3-AC2 | `PATCH /courses/:course_id` con `{"education_level": "university"}` actualiza solo el nivel; el nombre no cambia. | Enviar PATCH con solo `education_level`, verificar `name` sin cambio. |
| B3-AC3 | `PATCH /courses/:course_id` retorna 403 si el curso no pertenece al usuario autenticado. | Llamar con JWT de otro usuario; esperar 403. |
| B3-AC4 | `DELETE /courses/:course_id` marca `deleted_at` en el curso, todos sus topics, materials y chunks. | Borrar un curso con 2 topics y materiales; verificar en BD que todos tienen `deleted_at IS NOT NULL`. |
| B3-AC5 | `DELETE /courses/:course_id` retorna 204 sin body. | Verificar status code y body vacío. |
| B3-AC6 | Tras `DELETE /courses/:course_id`, el curso no aparece en `GET /courses`. | Listar cursos; el borrado no debe estar presente. |
| B3-AC7 | `PATCH /courses/:course_id/topics/:topic_id` con `{"title": "Y"}` actualiza el título del topic. | Enviar PATCH, verificar título actualizado en respuesta. |
| B3-AC8 | `PATCH /courses/:course_id/topics/:topic_id` retorna 404 si el topic no pertenece al curso indicado. | Enviar PATCH con `course_id` incorrecto; esperar 404. |
| B3-AC9 | `DELETE /courses/:course_id/topics/:topic_id` marca `deleted_at` en el topic, sus materials y chunks. | Borrar topic; verificar cascada en BD. |
| B3-AC10 | `DELETE /courses/:course_id/topics/:topic_id` retorna 204 sin body. | Verificar status code. |
| B3-AC11 | En la app móvil, el `PopupMenuButton` con opciones "Editar" y "Eliminar" aparece en cada card de curso. | Inspeccionar UI en `CourseDashboardScreen`. |
| B3-AC12 | En la app móvil, el diálogo de confirmación de borrado de curso muestra el nombre del curso y advierte sobre la irreversibilidad. | Pulsar "Eliminar curso"; verificar texto del diálogo. |
| B3-AC13 | En la app móvil, tras editar el nombre de un curso, la lista se actualiza mostrando el nuevo nombre. | Editar nombre; verificar que la lista refleja el cambio sin necesidad de pull-to-refresh manual. |
| B3-AC14 | En la app móvil, el `PopupMenuButton` con opciones "Editar" y "Eliminar" aparece en cada sección de topic. | Inspeccionar UI en `CourseDetailScreen`. |
| B3-AC15 | Todos los endpoints PATCH y DELETE validan ownership mediante JWT. Un usuario no puede modificar/borrar recursos de otro. | Probar cada endpoint con JWT ajeno; todos deben retornar 403. |
| B3-AC16 | El soft delete en cascada se ejecuta dentro de una transacción; si falla un paso intermedio, se ejecuta rollback y ningún registro queda marcado. | Simular fallo en la cascada (mock de repo); verificar que no quedan registros parcialmente borrados. |

---

## Resumen de endpoints (estado final)

| Método | Path | Nuevo | Bloque |
|--------|------|-------|--------|
| GET | `/api/v1/courses` | No | — |
| POST | `/api/v1/courses` | No | — |
| GET | `/api/v1/courses/:course_id` | No | — |
| **PATCH** | `/api/v1/courses/:course_id` | **Sí** | 3 |
| **DELETE** | `/api/v1/courses/:course_id` | **Sí** | 3 |
| POST | `/api/v1/courses/:course_id/topics` | No | — |
| **PATCH** | `/api/v1/courses/:course_id/topics/:topic_id` | **Sí** | 3 |
| **DELETE** | `/api/v1/courses/:course_id/topics/:topic_id` | **Sí** | 3 |
| POST | `/api/v1/courses/:course_id/topics/:topic_id/materials` | No | 2 (fix client) |
| GET | `/api/v1/courses/:course_id/topics/:topic_id/materials` | No | — |
| GET | `/api/v1/courses/:course_id/topics/:topic_id/context` | No | 1 (sin cambios) |
| **GET** | `/api/v1/courses/:course_id/context` | **Sí** | 1 |
| GET | `/api/v1/courses/:course_id/topics/:topic_id/readiness` | No | — |
| GET | `/api/v1/courses/:course_id/topics/:topic_id/summary` | No | — |

---

## Dependencias entre bloques

```
Bloque 2 (fix upload) ──debe completarse antes──▶ Bloque 1 (tutor por curso)
Bloque 3 (CRUD) ──independiente, puede ir en paralelo──
```

## Migración de BD requerida

| Tabla | Cambio | Bloque |
|-------|--------|--------|
| `material_chunks` | Agregar columna `deleted_at TIMESTAMPTZ DEFAULT NULL` | 3 |

No se agrega `course_id` a `material_chunks`; se usa JOIN a través de `topics.course_id` para las queries de curso completo (Bloque 1).
