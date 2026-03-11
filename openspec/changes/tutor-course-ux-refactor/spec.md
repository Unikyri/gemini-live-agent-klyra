# Spec: Refactor UX del Tutor por Curso y Robustecimiento de Upload

> Generada a partir de `openspec/changes/tutor-course-ux-refactor/proposal.md`.
> Depende de: `tutor-course-upload-crud` (archivado 2026-03-10).
> Objetivo: detallar contratos, comportamiento y criterios de aceptación para que `sdd-tasks` pueda desglosar las tareas de implementación.

---

## Bloque 1 — Upload robusto

### 1.1 Endpoint: subir material (existente, modificado)

| Campo | Valor |
|-------|-------|
| **Método** | `POST` |
| **Path** | `/api/v1/courses/:course_id/topics/:topic_id/materials` |
| **Auth** | Bearer JWT. El middleware inyecta `user_id` en contexto Gin. Validación de ownership del curso. |
| **Content-Type** | `multipart/form-data` (generado automáticamente por el cliente HTTP; **nunca** forzado manualmente). |
| **Body** | `FormData` con campo `file` (archivo binario). |

#### 1.1.1 Extensiones y tipos MIME aceptados

La estrategia de validación pasa de **MIME-primero** a **extensión-primero, MIME como fallback**:

1. Se extrae la extensión del archivo (`filepath.Ext(header.Filename)`).
2. Si la extensión está en `allowedExtensions`, se acepta el archivo. Se omite la verificación estricta de MIME.
3. Se usa `http.DetectContentType` (magic bytes) solo como verificación complementaria para PDFs (que pueden ser detectados de forma confiable) y para logging de discrepancias.

**Tabla de extensiones aceptadas (sin cambios)**

| Extensión | `MaterialFormatType` | Tamaño máx. |
|-----------|---------------------|-------------|
| `.pdf` | `pdf` | 20 MB |
| `.txt` | `txt` | 20 MB |
| `.md` | `md` | 20 MB |
| `.png` | `png` | 20 MB |
| `.jpg` | `jpg` | 20 MB |
| `.jpeg` | `jpeg` | 20 MB |
| `.webp` | `webp` | 20 MB |
| `.mp3` | `audio` | 50 MB |
| `.wav` | `audio` | 50 MB |
| `.m4a` | `audio` | 50 MB |

**Tabla de MIME ampliada (mapa `allowedMaterialFormats`)**

Se agregan las siguientes variantes al mapa existente:

| MIME actual | Ya aceptado | MIME nuevo a agregar | Motivo |
|-------------|-------------|---------------------|--------|
| `application/pdf` | Sí | `application/x-pdf` | Variante que reportan algunos clientes HTTP |
| — | No | `application/octet-stream` (cuando ext = `.pdf`, `.png`, `.jpg`, `.jpeg`, `.webp`) | Dio en Android a veces reporta octet-stream para archivos válidos |
| `image/jpeg` | Sí | `image/jpg` | Variante no estándar pero frecuente |
| `audio/mpeg` | Sí | `audio/mp3` | Variante no estándar reportada en iOS |

**Lógica de validación actualizada (pseudocódigo)**

```
ext := lowercase(filepath.Ext(filename))
formatType, extOK := allowedExtensions[ext]
if !extOK → 415 "only PDF, TXT, MD, ... are accepted"

# Verificar tamaño según formatType
if formatType == audio && size > 50MB → 413
if formatType != audio && size > 20MB → 413

# Detección MIME por magic bytes
detectedMIME := http.DetectContentType(fileData)
detectedMIME = stripParams(detectedMIME)  # quitar "; charset=utf-8"

# Para PDF: verificación estricta de contenido (magic bytes confiables)
if formatType == pdf && detectedMIME != "application/pdf" && detectedMIME != "application/x-pdf" && detectedMIME != "application/octet-stream":
    log.Warn("MIME mismatch", ext, detectedMIME)
    → 415 "file content does not match the .pdf extension"

# Para otros formatos: logging de discrepancia pero NO rechazo
if detectedMIME not in allowedMaterialFormats && detectedMIME != "application/octet-stream":
    log.Warn("MIME mismatch", ext, detectedMIME, "accepted by extension")
    # Se acepta igualmente porque la extensión es válida

# Aceptar el archivo
```

**Logging de discrepancias**

Cuando `detectedMIME` no coincide con lo esperado para la extensión, se emite un log nivel `warn` con:
- `filename`, `extension`, `detectedMIME`, `expectedMIME`, `action: "accepted"` o `"rejected"`.

#### 1.1.2 Respuesta exitosa (201)

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

Sin cambios en la forma de la respuesta respecto a `tutor-course-upload-crud`.

#### 1.1.3 Errores

| Código | Condición |
|--------|-----------|
| 400 | Falta campo `file`, archivo ilegible o `FormData` malformado |
| 401 | JWT ausente o inválido |
| 403 | El curso no pertenece al usuario autenticado |
| 404 | Curso o topic no encontrado (o soft-deleted). También se retorna cuando la validación de ownership falla (`material == nil`) para evitar enumeración. |
| 413 | Archivo excede el límite de tamaño (20 MB general, 50 MB audio) |
| 415 | Extensión no soportada **o** contenido de PDF no coincide con magic bytes |
| 500 | Error interno (storage, DB) |

### 1.2 Cambio en Flutter: `contentType` coherente

**Archivo**: `mobile/lib/features/course/data/material_remote_datasource.dart`

**Estado actual**: El datasource ya NO fuerza `Options(contentType: 'multipart/form-data')` (fix de `tutor-course-upload-crud`). Sin embargo, `MultipartFile.fromFile` y `MultipartFile.fromBytes` no reciben `contentType` explícito.

**Cambio requerido**: Al construir `MultipartFile`, derivar `contentType` de la extensión real del archivo usando el paquete `mime` de Dart (`lookupMimeType`):

```dart
import 'package:mime/mime.dart';

// Al construir el MultipartFile:
final mimeType = lookupMimeType(file.name) ?? 'application/octet-stream';
final mediaType = MediaType.parse(mimeType);

MultipartFile multipart;
if (file.bytes != null) {
  multipart = MultipartFile.fromBytes(
    file.bytes!,
    filename: file.name,
    contentType: mediaType,
  );
} else if (file.path != null) {
  multipart = await MultipartFile.fromFile(
    file.path!,
    filename: file.name,
    contentType: mediaType,
  );
}
```

Esto asegura que el `Content-Type` de cada parte multipart sea coherente con la extensión, reduciendo los falsos 415 por discrepancia MIME.

**Verificación previa**: confirmar que el paquete `mime` ya está en `pubspec.yaml`; si no, agregarlo.

---

### 1.3 Endpoint: listar materiales (existente, sin cambios)

| Campo | Valor |
|-------|-------|
| **Método** | `GET` |
| **Path** | `/api/v1/courses/:course_id/topics/:topic_id/materials` |
| **Auth** | Bearer JWT. Validación de ownership del curso. |

**Respuesta exitosa (200)**

```json
{
  "materials": [
    {
      "id": "<uuid>",
      "topic_id": "<uuid>",
      "format_type": "pdf",
      "storage_url": "...",
      "extracted_text": "...",
      "status": "validated",
      "original_name": "apuntes.pdf",
      "size_bytes": 1048576,
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "total": 1
}
```

- Solo devuelve materiales con `deleted_at IS NULL` (filtro ya existente en `MaterialRepository.FindByTopic`).
- `total` es `len(materials)`.

**Errores**

| Código | Condición |
|--------|-----------|
| 401 | JWT ausente o inválido |
| 403 | El curso no pertenece al usuario |
| 404 | Curso o topic no encontrado |
| 500 | Error interno |

Sin cambios de comportamiento en este endpoint.

---

## Bloque 2 — Endpoints de contexto RAG con soporte zero-material

### 2.1 Endpoint: contexto de un topic (existente, modificado)

| Campo | Valor |
|-------|-------|
| **Método** | `GET` |
| **Path** | `/api/v1/courses/:course_id/topics/:topic_id/context` |
| **Auth** | Bearer JWT. Validación de ownership del curso + verificación de que el topic pertenece al curso y no está soft-deleted. |
| **Path params** | `course_id` (UUID), `topic_id` (UUID) |
| **Query params** | `query` (string, opcional) |

#### 2.1.1 Respuesta exitosa (200) — con materiales

```json
{
  "topic_id": "<uuid>",
  "context": "<texto concatenado de chunks>",
  "query": "",
  "has_materials": true,
  "message": ""
}
```

#### 2.1.2 Respuesta exitosa (200) — sin materiales (zero-material)

```json
{
  "topic_id": "<uuid>",
  "context": "",
  "query": "",
  "has_materials": false,
  "message": "No hay materiales para este tema. El tutor usará su conocimiento base."
}
```

**Cambio clave**: en el estado actual, `GetTopicContext` retorna el string `"Aún no hay material validado para este tema..."` dentro del campo `context`. El cambio es:
- Retornar `context: ""` (string vacío).
- Agregar campo `has_materials: bool` para que el cliente pueda distinguir sin parsear el texto.
- Agregar campo `message: string` con el texto informativo.

Esto requiere un cambio en `RAGUseCase.GetTopicContext` para que retorne una struct en lugar de un `string`:

```go
type TopicContextResult struct {
    Context      string
    HasMaterials bool
    Message      string
}
```

Y en `RAGHandler.GetTopicContext`, serializar los campos adicionales.

#### 2.1.3 Comportamiento sin embedder (entorno local/dev)

Cuando `uc.embedder == nil` y hay `query`:
- Fallback a contexto completo del topic (sin similarity search).
- Si no hay chunks → misma respuesta zero-material (200, `context: ""`, `has_materials: false`).

#### 2.1.4 Errores

| Código | Condición |
|--------|-----------|
| 401 | JWT ausente o inválido |
| 403 | El curso no pertenece al usuario autenticado |
| 404 | Curso no encontrado (o soft-deleted), o topic no encontrado / no pertenece al curso / soft-deleted |
| 500 | Error interno (DB, embedder) |

---

### 2.2 Endpoint: contexto de un curso (existente, modificado)

| Campo | Valor |
|-------|-------|
| **Método** | `GET` |
| **Path** | `/api/v1/courses/:course_id/context` |
| **Auth** | Bearer JWT. Validación de ownership del curso. |
| **Path params** | `course_id` (UUID) |
| **Query params** | `query` (string, opcional) |

#### 2.2.1 Respuesta exitosa (200) — con materiales

```json
{
  "course_id": "<uuid>",
  "context": "### Tema 1\n\n<chunks>\n\n### Tema 2\n\n<chunks>",
  "query": "",
  "truncated": true,
  "has_materials": true,
  "message": ""
}
```

#### 2.2.2 Respuesta exitosa (200) — sin materiales (zero-material)

```json
{
  "course_id": "<uuid>",
  "context": "",
  "query": "",
  "truncated": false,
  "has_materials": false,
  "message": "No hay materiales en ningún tema de este curso. El tutor usará su conocimiento base."
}
```

**Cambio clave**: en el estado actual, `GetCourseContext` retorna `"Aún no hay material validado para este curso..."` como string en `context`. El cambio es análogo al de topic:
- `context: ""` cuando no hay chunks.
- Agregar `has_materials` y `message`.

Requiere que `RAGUseCase.GetCourseContext` retorne una struct similar:

```go
type CourseContextResult struct {
    Context      string
    Truncated    bool
    HasMaterials bool
    Message      string
}
```

#### 2.2.3 Errores

| Código | Condición |
|--------|-----------|
| 401 | JWT ausente o inválido |
| 403 | El curso no pertenece al usuario autenticado |
| 404 | Curso no encontrado (o soft-deleted) |
| 500 | Error interno (DB, embedder, topic repo no configurado) |

---

## Bloque 3 — Readiness no bloqueante

### 3.1 Estado actual de `CheckReadiness`

**Backend** (`TopicUseCase.CheckReadiness` en `topic_usecase.go`):
- Cuenta materiales totales y validados para un topic.
- Retorna `is_ready: true` si `validatedCount > 0`.
- Endpoint: `GET /api/v1/courses/:course_id/topics/:topic_id/readiness`.

**Mobile** (`MaterialSummaryScreen`):
- Al navegar a un topic → se abre `MaterialSummaryScreen`.
- Llama a `checkTopicReadiness`. Si `isReady == false`, muestra un candado y botón "Go back to upload materials" → **bloquea** el acceso al tutor.
- Si `isReady == true`, muestra resumen y botón "Start Tutor Session".

### 3.2 Comportamiento deseado tras el refactor

**El endpoint de readiness NO se elimina.** Sigue disponible como endpoint informativo.

**Cambios en el flujo mobile**:

1. `MaterialSummaryScreen` deja de ser **gate obligatorio** para el tutor:
   - El botón "Review Summary & Start" de `_TopicSection` se elimina (ver Bloque 4).
   - `MaterialSummaryScreen` puede seguir existiendo como pantalla accesible desde un enlace secundario (ej. desde la lista de materiales) para que el usuario vea el resumen si quiere.
   - **No** bloquea la navegación a `TutorSessionScreen`.

2. `CheckReadiness` puede usarse opcionalmente en la UI para mostrar **indicadores visuales** (badges) en la lista de topics:
   - Topic con material validado → badge verde o check.
   - Topic sin material → badge gris o texto "Sin material".
   - Estos badges son **informativos**, no bloquean nada.

3. El endpoint de readiness **no cambia** en el backend. Los cambios son puramente de flujo en Flutter.

---

## Bloque 4 — Flujos de UI en Flutter

### 4.1 Pantalla de detalle de curso (`CourseDetailScreen`)

#### 4.1.1 Estado actual

- `SliverAppBar` con avatar, nombre del curso.
- Lista de topics (`_TopicSection`), cada uno con:
  - Título del topic.
  - `PopupMenuButton` (editar/eliminar).
  - `MaterialListView` (lista de materiales subidos).
  - Botón `OutlinedButton.icon` "Review Summary & Start" → navega a `/course/$courseId/topic/$topicId/summary` (paso bloqueante).
- `FloatingActionButton` "Add Topic".

#### 4.1.2 Cambios requeridos

**A. Eliminar el botón "Review Summary & Start" de `_TopicSection`**

Eliminar completamente el `OutlinedButton.icon` que navega a `/course/$courseId/topic/$topicId/summary`. Los topics conservan:
- Título + `PopupMenuButton`.
- `MaterialListView`.
- (Opcional) Nuevo acceso directo al tutor por topic: un chip o icono discreto (ej. `IconButton` con icono de chat/tutor) que navega directamente a `TutorSessionScreen` con `courseId` y `topicId` pre-seleccionado, **sin** pasar por `MaterialSummaryScreen`.

**B. Agregar botón global de tutor a nivel de curso**

Posición: Superpuesto sobre el avatar en el `SliverAppBar`, o como un `FloatingActionButton` secundario o botón prominente debajo del `SliverAppBar`. La posición exacta puede variar, pero debe cumplir:

- **Visible sin scroll** (en la parte superior de la pantalla o como FAB flotante).
- **Asociado visualmente al avatar** del tutor (sobre o junto a la imagen del avatar en el hero).
- **Copy del botón**: texto tipo _"Haz clic sobre mí para charlar conmigo"_ o _"Hablar con el tutor"_.
- **Acción**: navega a `TutorSessionScreen` con solo `courseId`, sin `topicId` (el tutor arranca en modo "elige un tema").

Implementación sugerida: `GestureDetector` sobre el `AvatarImage` en el `SliverAppBar`, o un botón `ElevatedButton`/`FloatingActionButton` posicionado sobre el avatar con un `Positioned` widget dentro del `Stack` del `FlexibleSpaceBar`.

**C. Reemplazo del FAB actual**

El `FloatingActionButton.extended` "Add Topic" actual puede coexistir con el botón de tutor. Opciones:
- Mantener FAB de "Add Topic" y agregar botón de tutor en el hero area (preferido).
- O convertir el FAB en un `SpeedDial` con dos opciones: "Add Topic" y "Hablar con el tutor".

### 4.2 Pantalla de sesión del tutor (`TutorSessionScreen`)

#### 4.2.1 Estado actual

- Recibe `courseId` (obligatorio) y `topicId` (opcional).
- `AppBar` muestra nombre del curso y topic seleccionado.
- **Selector de topics** (chips): visible cuando la sesión está activa y hay topics. Incluye chip "Curso completo" y un `FilterChip` por cada topic.
- Avatar animado central.
- Panel de transcripción.
- Botón de micrófono (start/stop session).

#### 4.2.2 Cambios requeridos

**A. Manejo de zero-material en el selector de topics**

Cuando el usuario selecciona un topic en los chips:

1. `loadTopicContext` llama a `GET /courses/:course_id/topics/:topic_id/context`.
2. El controller inspecciona la respuesta:
   - Si `has_materials == true` → inyecta `context` en Gemini vía `sendContextUpdate(contextText)` (flujo existente).
   - Si `has_materials == false` → inyecta un **contexto mínimo** al tutor:

```
[Contexto actualizado — Tema: <topicTitle>]
El estudiante quiere hablar del tema: "<topicTitle>".
No hay material de referencia para este tema.
Usa tu conocimiento para guiar la conversación.
Si el estudiante quiere respuestas más precisas, puede subir material de estudio.
```

3. El controller actualiza `currentTopicId` normalmente, sin marcar error.

**B. Indicador visual de estado zero-material**

Cuando un topic sin materiales está seleccionado, mostrar un indicador sutil en la UI:
- Un `Container` con texto tipo: _"Sin material de referencia — el tutor usará su conocimiento base"_.
- Posición: debajo de los chips de topic, antes del área de avatar.
- Color: amarillo/ámbar tenue, no rojo (no es un error, es informativo).

**C. Cambio en `TutorSessionController.loadTopicContext`**

El método debe:
1. Parsear los nuevos campos `has_materials` y `message` de la respuesta.
2. Si `has_materials == false`:
   - Obtener el título del topic desde `course.topics` (ya disponible en estado local).
   - Construir el contexto mínimo (string con título y instrucciones).
   - Enviar a Gemini vía `sendContextUpdate`.
3. Actualizar estado con un campo nuevo `hasCurrentTopicMaterials: bool` para que la UI muestre el indicador.

**D. Cambio en `TutorSessionState`**

Agregar campo:

```dart
final bool hasCurrentTopicMaterials;
```

Default: `true`. Se actualiza a `false` cuando se selecciona un topic sin materiales.

**E. `sendContextUpdate` ya funciona con contexto mínimo**

El método `GeminiLiveService.sendContextUpdate` envía `clientContent` con el texto provisto. No requiere cambios de interfaz, solo se invoca con texto diferente (contexto mínimo vs contexto RAG).

---

## Bloque 5 — Criterios de aceptación

### 5.1 Upload robusto

| ID | Criterio | Verificación |
|----|----------|-------------|
| UPL-AC1 | Un PDF subido desde Android con `Content-Type: application/octet-stream` (caso típico de Dio con ciertos archivos) se acepta sin error 415 y completa el pipeline hasta `validated`. | Subir PDF cuyo MIME detectado por magic bytes es `application/pdf`; verificar 201. Subir PDF cuyo MIME reportado por Dio es `application/octet-stream`; verificar 201. |
| UPL-AC2 | Una imagen JPG subida con extensión `.jpg` y MIME `image/jpeg` (o `image/jpg`) se acepta correctamente. | Subir imagen con ambas variantes de MIME; ambos retornan 201. |
| UPL-AC3 | Un archivo con extensión `.docx` (no soportada) se rechaza con 415, independientemente del MIME. | Subir `.docx`; esperar 415. |
| UPL-AC4 | Un archivo con extensión `.pdf` pero cuyo contenido no es un PDF real (magic bytes de imagen, por ejemplo) se rechaza con 415 y se logea la discrepancia. | Subir archivo falso; esperar 415 y verificar log `warn`. |
| UPL-AC5 | En Flutter, `MultipartFile` se construye con `contentType` derivado de la extensión vía `lookupMimeType`, no hardcodeado ni omitido. | Inspeccionar código de `material_remote_datasource.dart`. |
| UPL-AC6 | Cuando se detecta discrepancia entre MIME detectado y extensión (pero se acepta por extensión válida), se emite un log `warn` con `filename`, `extension`, `detectedMIME`. | Revisar logs del backend tras subida con discrepancia. |

### 5.2 Botón global de tutor

| ID | Criterio | Verificación |
|----|----------|-------------|
| BTN-AC1 | En `CourseDetailScreen`, **no existe** el botón "Review Summary & Start" en ningún topic. | Inspeccionar UI; confirmar que `_TopicSection` no contiene `OutlinedButton` de review. |
| BTN-AC2 | En `CourseDetailScreen`, existe un botón de tutor a nivel de curso, visible sin scroll, asociado visual o espacialmente al avatar del tutor. | Abrir pantalla de detalle de un curso con avatar; verificar botón visible. |
| BTN-AC3 | El botón de tutor muestra copy invitando a la interacción (ej. "Hablar con el tutor", "Haz clic para charlar conmigo"). | Verificar texto del botón. |
| BTN-AC4 | Al pulsar el botón de tutor, se abre `TutorSessionScreen` con solo `courseId`, sin `topicId`. | Pulsar botón; verificar que la pantalla de tutor arranca sin topic pre-seleccionado. |
| BTN-AC5 | El tutor saluda y ofrece los temas disponibles del curso al iniciar sin topic. | Verificar que el system prompt incluye la lista de topics y el tutor pregunta cuál elegir. |
| BTN-AC6 | Los topics en la lista conservan la opción de navegar al tutor con topic pre-seleccionado (acceso directo). | Si se implementa el acceso directo por topic (chip/icono), verificar que navega a `TutorSessionScreen` con `courseId` + `topicId`. |

### 5.3 Tutor en modo zero-material

| ID | Criterio | Verificación |
|----|----------|-------------|
| ZM-AC1 | `GET /courses/:course_id/topics/:topic_id/context` retorna 200 con `context: ""`, `has_materials: false` y un `message` informativo cuando el topic no tiene materiales validados. | Llamar endpoint para topic sin materiales; verificar forma de la respuesta. |
| ZM-AC2 | `GET /courses/:course_id/topics/:topic_id/context` retorna 200 con `context: "<texto>"`, `has_materials: true` cuando el topic tiene materiales. | Llamar endpoint para topic con materiales; verificar `has_materials: true`. |
| ZM-AC3 | `GET /courses/:course_id/context` retorna 200 con `context: ""`, `has_materials: false` cuando ningún topic del curso tiene materiales. Sin error 500. | Llamar endpoint para curso vacío; verificar respuesta limpia. |
| ZM-AC4 | `GET /courses/:course_id/context` retorna 200 con `has_materials: true` cuando al menos un topic tiene materiales. | Llamar endpoint; verificar campo. |
| ZM-AC5 | Al seleccionar un topic **sin materiales** en el selector de topics del tutor, el tutor inicia conversación usando el título del topic y su conocimiento base, sin mostrar error ni bloquear. | Seleccionar topic vacío en sesión activa; verificar que el tutor responde sobre el tema. |
| ZM-AC6 | Al seleccionar un topic **con materiales**, el contexto RAG se carga y el tutor lo utiliza como referencia. | Seleccionar topic con material; preguntar algo del material; verificar que la respuesta lo referencia. |
| ZM-AC7 | En la UI, cuando un topic sin materiales está seleccionado, se muestra un indicador informativo (no bloqueante) tipo "Sin material de referencia — el tutor usará su conocimiento base". | Verificar indicador visual en `TutorSessionScreen`. |
| ZM-AC8 | `GET /courses/:course_id/topics/:topic_id/context` con `query` y sin embedder retorna fallback a contexto completo, o zero-material si no hay chunks. Sin error 500. | Probar en entorno local sin embedder configurado. |

### 5.4 No más bloqueos de UX por falta de materiales

| ID | Criterio | Verificación |
|----|----------|-------------|
| NB-AC1 | `MaterialSummaryScreen` **no** bloquea la navegación al tutor aunque no haya material validado. | Si la pantalla sigue existiendo, verificar que no impide el acceso al tutor. |
| NB-AC2 | Un usuario puede llegar a `TutorSessionScreen` desde `CourseDetailScreen` sin haber subido ningún material a ningún topic. | Crear curso con topics vacíos; pulsar botón de tutor; verificar que se abre la sesión. |
| NB-AC3 | El flujo completo "crear curso → agregar topic → abrir tutor → conversar" funciona sin subir material. | Ejecutar el flujo end-to-end; verificar que no hay errores ni pantallas bloqueantes. |
| NB-AC4 | El endpoint de readiness (`GET /courses/:course_id/topics/:topic_id/readiness`) sigue funcionando y retorna la misma estructura de respuesta. No se rompen consumidores existentes. | Llamar endpoint; verificar respuesta sin cambios de forma. |

---

## Resumen de endpoints afectados

| Método | Path | Cambio | Bloque |
|--------|------|--------|--------|
| POST | `/api/v1/courses/:course_id/topics/:topic_id/materials` | Modificado (validación MIME relajada, logging) | 1 |
| GET | `/api/v1/courses/:course_id/topics/:topic_id/materials` | Sin cambios | 1 (referencia) |
| GET | `/api/v1/courses/:course_id/topics/:topic_id/context` | Modificado (campos `has_materials`, `message`, `context` vacío) | 2 |
| GET | `/api/v1/courses/:course_id/context` | Modificado (campos `has_materials`, `message`, `context` vacío) | 2 |
| GET | `/api/v1/courses/:course_id/topics/:topic_id/readiness` | Sin cambios en backend | 3 (referencia) |

---

## Archivos afectados (resumen)

| Archivo | Tipo de cambio | Bloque |
|---------|---------------|--------|
| `backend/internal/handlers/http/material_handler.go` | Ampliar `allowedMaterialFormats`, relajar lógica de validación MIME, agregar logging | 1 |
| `backend/internal/core/usecases/rag_usecase.go` | Cambiar retorno de `GetTopicContext` y `GetCourseContext` a structs con `HasMaterials` y `Message` | 2 |
| `backend/internal/handlers/http/rag_handler.go` | Serializar campos `has_materials` y `message` en respuestas | 2 |
| `mobile/lib/features/course/data/material_remote_datasource.dart` | Agregar `contentType` derivado de extensión con `lookupMimeType` | 1 |
| `mobile/lib/features/course/presentation/course_detail_screen.dart` | Eliminar botón "Review Summary & Start", agregar botón global de tutor | 4 |
| `mobile/lib/features/course/presentation/screens/material_summary_screen.dart` | Convertir de gate bloqueante a pantalla informativa (o deprecar) | 3, 4 |
| `mobile/lib/features/tutor/presentation/tutor_session_screen.dart` | Agregar indicador visual de zero-material | 4 |
| `mobile/lib/features/tutor/presentation/tutor_session_controller.dart` | Parsear `has_materials`, construir contexto mínimo, agregar `hasCurrentTopicMaterials` a estado | 4 |
| `mobile/lib/features/tutor/data/gemini_live_service.dart` | Sin cambios de interfaz (ya soporta contexto mínimo vía `sendContextUpdate`) | — |

---

## Dependencias entre bloques

```
Bloque 1 (upload robusto) ─── independiente, puede ir primero ───
Bloque 2 (context zero-material) ─── independiente de 1, prerequisito de 4 ───
Bloque 3 (readiness no bloqueante) ─── independiente ───
Bloque 4 (flujos UI) ─── depende de 2 y 3 ───
```

## Migraciones de BD requeridas

Ninguna. Este cambio no agrega tablas ni columnas. Los cambios son de lógica de validación, forma de respuesta JSON y flujo de UI.
