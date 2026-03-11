# Design: Refactor UX del Tutor por Curso y Robustecimiento de Upload

## Enfoque Técnico

Este cambio se construye sobre la infraestructura implementada en `tutor-course-upload-crud` (archivada) y se centra en cuatro ejes: (1) relajar la validación MIME/extensión del upload para eliminar falsos 415, (2) refactorizar la UX de `CourseDetailScreen` para reemplazar botones individuales `Review Summary & Start` por un botón global de tutor vinculado al avatar, (3) adaptar `TutorSessionScreen` y el pipeline RAG para funcionar en modo "zero-material" cuando un topic no tiene materiales, y (4) convertir `MaterialSummaryScreen` de gate bloqueante a pantalla informativa opcional.

La estrategia técnica es mínimamente invasiva: se amplía el mapa de MIME aceptados en `material_handler.go` con una estrategia "extensión primero", se agrega un campo `message` informativo en las respuestas de contexto vacío del backend, se agrega una ruta de tutor a nivel de curso sin topic obligatorio en el router Flutter, y se ajusta el controller del tutor para inyectar contexto mínimo (título del topic) cuando no hay materiales.

---

## Decisiones de Arquitectura

### Decisión 1: Validación "extensión primero, MIME como fallback" en lugar de MIME estricto

**Elección**: Usar la extensión del archivo como fuente de verdad primaria para determinar el formato. El MIME detectado por `http.DetectContentType` (magic bytes) se usa como validación secundaria: solo se rechaza si el MIME detectado contradice la extensión de forma irreconciliable (ej: extensión `.pdf` pero magic bytes de imagen).

**Alternativas consideradas**:
- MIME estricto (enfoque actual): rechaza variantes legítimas como `application/x-pdf` o `application/octet-stream` para archivos cuya extensión es correcta.
- Aceptar cualquier archivo sin validar MIME (inseguro; permite renombrar ejecutables).

**Razonamiento**: El problema raíz es que `http.DetectContentType` de Go es limitado — solo examina los primeros 512 bytes y su tabla MIME es pequeña. Diferentes clientes HTTP (Dio, navegadores, Android intent) reportan MIMEs distintos para el mismo archivo. La extensión del archivo, en cambio, es consistente y ya se valida contra una lista cerrada (`allowedExtensions`). Relajar la validación MIME mientras se mantiene la extensión estricta es el balance correcto entre seguridad y compatibilidad. Se agrega logging `warn` para discrepancias MIME vs extensión para monitoreo en producción.

---

### Decisión 2: Campo `message` aditivo en respuestas de contexto vacío (sin romper contrato existente)

**Elección**: Agregar un campo `message` informativo en las respuestas JSON de `GetTopicContext` y `GetCourseContext` cuando no hay chunks. El campo `context` sigue siendo `""` (string vacío). El campo `message` es aditivo (nuevo) y los clientes existentes lo ignoran si no lo esperan.

**Alternativas consideradas**:
- Retornar 404 cuando no hay materiales (semántica incorrecta: el topic/curso sí existe).
- Incluir un texto genérico dentro del campo `context` (contamina el contexto real enviado a Gemini con texto que no es material de estudio).

**Razonamiento**: El enfoque actual del backend ya retorna un mensaje embebido dentro del campo `context` (ej: `"Aún no hay material validado para este tema..."`). Esto funciona pero mezcla metadatos con contenido real. El refactor separa estas preocupaciones: `context` contiene solo material de estudio (o vacío), y `message` contiene el mensaje informativo. El frontend usa `context` para inyectar al tutor y `message` para mostrar en UI. La migración es compatible hacia atrás: si el campo `context` no está vacío, se usa como antes; si está vacío y hay `message`, el frontend entra en modo zero-material.

---

### Decisión 3: Botón global de tutor en `CourseDetailScreen` en lugar de botón por topic

**Elección**: Eliminar el botón `Review Summary & Start` de cada `_TopicSection` y agregar un botón prominente de tutor a nivel de curso, vinculado visualmente al avatar. Mantener la posibilidad de navegar al tutor desde un topic específico como acceso directo (chip "Hablar de este tema" en el menú del topic), pero sin pasar por `MaterialSummaryScreen` como gate.

**Alternativas consideradas**:
- Mantener ambos botones (por topic y global): confunde al usuario sobre cuál usar.
- Eliminar toda forma de pre-seleccionar topic: pierde la posibilidad de acceso directo contextual.

**Razonamiento**: La UX actual fragmenta la experiencia al exigir que el estudiante seleccione un topic, revise el resumen, y luego inicie la sesión. El botón global invita al estudiante a interactuar con el tutor a nivel de curso; una vez dentro, el selector de topics en `TutorSessionScreen` permite cambiar de tema dinámicamente. El acceso directo desde un topic (ej: desde el menú de 3 puntos del topic) es un atajo útil que navega a `TutorSessionScreen` con el topicId pre-seleccionado.

---

### Decisión 4: Contexto mínimo "zero-material" construido en el frontend, no en el backend

**Elección**: Cuando `context` viene vacío del backend, el `TutorSessionController` construye un mensaje de contexto mínimo local: `"El estudiante quiere hablar del tema: [título]. No hay material de referencia. Usa tu conocimiento base."`. Este mensaje se envía a Gemini como `clientContent`, igual que el contexto RAG normal.

**Alternativas consideradas**:
- Backend retorna un contexto pre-construido para zero-material (acopla lógica de presentación al backend).
- No enviar nada a Gemini (el tutor no tiene idea del tema seleccionado).

**Razonamiento**: El título del topic ya está disponible en el state de Flutter (cargado como parte del curso). Construir el contexto mínimo en el frontend evita un round-trip extra y desacopla la lógica de presentación del backend. El backend se limita a devolver datos (contexto vacío + mensaje informativo), y el frontend decide cómo presentarlos al tutor.

---

### Decisión 5: Mantener `CheckReadiness` como endpoint informativo, no eliminarlo

**Elección**: El endpoint `GET /courses/:course_id/topics/:topic_id/readiness` se mantiene operativo. Lo que cambia es cómo el frontend lo usa: ya no bloquea la navegación al tutor. Si `isReady == false`, se puede mostrar un badge/indicador en la UI, pero no se impide el acceso.

**Alternativas consideradas**:
- Eliminar el endpoint y la lógica de readiness (pierde información útil sobre el estado de materiales).
- Modificar la lógica de `CheckReadiness` para que siempre retorne `isReady: true` (semántica confusa).

**Razonamiento**: `CheckReadiness` sigue siendo útil como indicador visual. El estudiante puede ver qué topics tienen material validado y cuáles no. Lo que se elimina es el bloqueo: antes, `MaterialSummaryScreen` impedía continuar si `isReady == false`. Ahora, la navegación al tutor es libre. La pantalla de resumen (`MaterialSummaryScreen`) puede seguir existiendo como acceso informativo pero no como gate obligatorio.

---

### Decisión 6: Ruta de tutor a nivel de curso con topicId opcional en GoRouter

**Elección**: Agregar una nueva ruta `/tutor/:courseId` (sin topicId) al router, manteniendo la ruta existente `/tutor/:courseId/:topicId`. `TutorSessionScreen` ya acepta `topicId` como parámetro opcional, así que solo hay que registrar la ruta.

**Alternativas consideradas**:
- Usar query parameters en lugar de path parameters (menos limpio en GoRouter).
- Eliminar la ruta con topicId y usar solo la de curso (pierde la funcionalidad de acceso directo con topic pre-seleccionado).

**Razonamiento**: GoRouter requiere rutas explícitas. Mantener ambas rutas es la forma más limpia de soportar ambos casos de uso (curso completo y topic pre-seleccionado). `TutorSessionScreen` ya maneja `topicId: null`, así que no se requieren cambios en el widget.

---

## Flujo de Datos

### Flujo 1: Estudiante entra al curso y abre sesión de tutor sin materiales

```
CourseDetailScreen
  │
  ├─ 1. Estudiante ve la lista de topics (pueden tener o no materiales)
  │    Avatar del curso visible en el SliverAppBar
  │
  ├─ 2. Estudiante pulsa botón global de tutor (asociado al avatar)
  │    → context.push('/tutor/$courseId')   // sin topicId
  │
  └─ TutorSessionScreen(courseId: "...", topicId: null)
       │
       ├─ 3. startSession(courseId)
       │    ├─ Obtiene curso del state (topics, nombre, nivel)
       │    ├─ Conecta WebSocket a Gemini Live
       │    │   System prompt: instrucciones tutor + lista de topics (SIN chunks)
       │    └─ Inicia micrófono
       │
       ├─ 4. Tutor saluda y ofrece los temas disponibles
       │    "¡Hola! Soy Klyra. Estos son los temas de tu curso: ..."
       │
       ├─ 5. Estudiante selecciona topic "Capítulo 1" en chip selector
       │    │
       │    ├─ GET /api/v1/courses/:course_id/topics/:topic_id/context
       │    │   │
       │    │   └─ Backend: RAGUseCase.GetTopicContext → 0 chunks
       │    │       Response: { context: "", message: "No hay materiales...", topic_id: "..." }
       │    │
       │    └─ 6. Controller detecta context == "" → modo zero-material
       │         Construye contexto mínimo: "El estudiante quiere hablar del tema:
       │         Capítulo 1. No hay material de referencia. Usa tu conocimiento base."
       │         │
       │         └─ GeminiLiveService.sendContextUpdate(contextoMinimo)
       │              → clientContent { parts: [{ text: "[CONTEXTO...]\n..." }], turnComplete: true }
       │
       └─ 7. Tutor responde usando su conocimiento base sobre el tema
```

### Flujo 2: Estudiante sube materiales a un topic y luego habla del topic

```
CourseDetailScreen
  │
  ├─ 1. Estudiante sube PDF a "Capítulo 2" via MaterialListView
  │    │
  │    ├─ material_remote_datasource.uploadMaterial()
  │    │   MultipartFile con contentType derivado de extensión (lookupMimeType)
  │    │
  │    ├─ MaterialHandler.UploadMaterial()
  │    │   ├─ ext = ".pdf" → formatType = PDF (extensión primero)
  │    │   ├─ DetectContentType → "application/pdf" ✓
  │    │   └─ materialUseCase.UploadMaterial → status: pending
  │    │
  │    └─ (async) extractTextAsync → status: validated
  │         └─ ragUseCase.ProcessMaterialChunks → genera chunks + embeddings
  │
  ├─ 2. Estudiante pulsa "Hablar de este tema" en menú del topic "Capítulo 2"
  │    → context.push('/tutor/$courseId/$topicId')   // con topicId pre-seleccionado
  │
  └─ TutorSessionScreen(courseId: "...", topicId: "cap2-id")
       │
       ├─ 3. startSession(courseId, topicId: "cap2-id")
       │    ├─ Conecta WebSocket (system prompt sin chunks)
       │    └─ Llama loadTopicContext(courseId, "cap2-id") automáticamente
       │
       ├─ 4. GET /api/v1/courses/:course_id/topics/cap2-id/context
       │    │
       │    └─ Backend: RAGUseCase.GetTopicContext → N chunks encontrados
       │        Response: { context: "contenido de chunks...", topic_id: "cap2-id" }
       │
       ├─ 5. Controller detecta context != "" → modo con material
       │    GeminiLiveService.sendContextUpdate(contextText)
       │
       └─ 6. Tutor usa el contexto RAG para responder preguntas del tema
```

### Flujo 3: Estudiante borra un topic/curso y luego intenta acceder

```
Caso A: Borrar un topic

CourseDetailScreen → menú del topic → "Eliminar tema"
  │
  ├─ CourseController.deleteTopic(courseId, topicId)
  │   └─ DELETE /api/v1/courses/:courseId/topics/:topicId
  │       └─ CourseUseCase.DeleteTopic (cascade transaccional):
  │           1. hard delete MaterialChunk WHERE topic_id = X
  │           2. soft delete Material WHERE topic_id = X
  │           3. soft delete Topic WHERE id = X
  │
  ├─ courseControllerProvider se invalida → lista de topics se actualiza
  │   El topic eliminado desaparece de la UI
  │
  └─ Si el tutor estaba abierto con ese topic seleccionado:
     ├─ El chip del topic desaparece del selector (topics viene del state actualizado)
     ├─ currentTopicId ya no existe → se resetea a null
     └─ Tutor continúa en modo "curso completo"

Caso B: Borrar un curso

CourseDashboardScreen → menú del curso → "Eliminar"
  │
  ├─ CourseController.deleteCourse(courseId)
  │   └─ DELETE /api/v1/courses/:courseId
  │       └─ CourseUseCase.DeleteCourse (cascade transaccional):
  │           1. hard delete MaterialChunk WHERE topic_id IN (topics del curso)
  │           2. soft delete Material WHERE topic_id IN (topics del curso)
  │           3. soft delete Topic WHERE course_id = X
  │           4. soft delete Course WHERE id = X
  │
  ├─ courseControllerProvider se invalida → curso desaparece del dashboard
  │
  └─ Si el tutor estaba abierto para ese curso:
     ├─ Al volver atrás, CourseDetailScreen no encuentra el curso → "Course not found"
     └─ Si se intenta acceder directamente por URL:
         GET /courses/:courseId → 404 (curso soft-deleted no aparece)
         GET /courses/:courseId/context → 404

Caso C: Acceder a contexto/materiales de un topic borrado

  GET /courses/:courseId/topics/:deletedTopicId/context
  │
  └─ RAGHandler.GetTopicContext:
      ├─ courseUseCase.GetCourseByID → curso existe ✓
      ├─ Recorre course.Topics buscando topicId con DeletedAt == nil
      │   → No encontrado (topic soft-deleted)
      └─ 404: "topic not found"

  GET /courses/:courseId/topics/:deletedTopicId/materials
  │
  └─ MaterialHandler.ListMaterials:
      └─ materialUseCase.GetMaterialsByTopic → nil (topic no pertenece o borrado)
         → 404: "course or topic not found"
```

---

## Cambios en Archivos

| Archivo | Acción | Descripción |
|---------|--------|-------------|
| `backend/internal/handlers/http/material_handler.go` | Modificar | Ampliar `allowedMaterialFormats` con variantes MIME comunes; cambiar lógica de validación a "extensión primero"; agregar logging `warn` para discrepancias MIME vs extensión |
| `backend/internal/core/usecases/rag_usecase.go` | Modificar | Refactorizar `GetTopicContext` y `GetCourseContext` para retornar struct con `Context` + `Message` en vez de solo string; separar mensaje informativo del contexto real |
| `backend/internal/handlers/http/rag_handler.go` | Modificar | Agregar campo `message` en respuestas JSON de `GetTopicContext` y `GetCourseContext` cuando el contexto está vacío |
| `backend/internal/core/usecases/material_usecase.go` | Modificar | En `GetMaterialsByTopic`, filtrar explícitamente materiales con `deleted_at IS NULL` (refuerzo de consistencia con soft-delete) |
| `mobile/lib/features/course/data/material_remote_datasource.dart` | Modificar | Agregar `contentType` explícito derivado de extensión del archivo usando `lookupMimeType` del paquete `mime`; asegurar que `filename` siempre incluya extensión correcta |
| `mobile/lib/features/course/presentation/course_detail_screen.dart` | Modificar | Eliminar botón `Review Summary & Start` de `_TopicSection`; agregar botón global de tutor vinculado al avatar en `_CourseDetailView`; agregar opción "Hablar de este tema" en `PopupMenuButton` del topic |
| `mobile/lib/features/course/presentation/screens/material_summary_screen.dart` | Modificar | Convertir gate bloqueante a pantalla informativa: cuando `isReady == false`, mostrar indicador pero permitir navegar al tutor con botón "Iniciar de todas formas" |
| `mobile/lib/features/tutor/presentation/tutor_session_screen.dart` | Modificar | Mostrar selector de topics siempre que la sesión esté activa (ya implementado); ajustar para que funcione sin topicId inicial |
| `mobile/lib/features/tutor/presentation/tutor_session_controller.dart` | Modificar | En `loadTopicContext`, detectar contexto vacío y construir contexto mínimo con título del topic; agregar lógica de invalidación de `loadedTopicIds` cuando los topics cambian |
| `mobile/lib/features/tutor/data/gemini_live_service.dart` | Modificar | Agregar manejo de contexto vacío en `sendContextUpdate` (no enviar si vacío); ajustar system prompt para incluir instrucción de modo zero-material |
| `mobile/lib/core/router/app_router.dart` | Modificar | Agregar ruta `/tutor/:courseId` (sin topicId) como nueva entrada en GoRouter |

---

## Interfaces / Contratos

### Backend: Nuevo tipo de retorno para contexto RAG

```go
// ContextResult separa el contexto real del mensaje informativo.
type ContextResult struct {
    Context   string // texto de chunks concatenados, o "" si no hay materiales
    Message   string // mensaje informativo para UI ("No hay materiales...", etc.)
    Truncated bool   // true si se truncaron chunks (solo para curso)
}
```

### Backend: Cambios en firmas de `RAGUseCase`

```go
// GetTopicContext ahora retorna ContextResult en vez de string.
func (uc *RAGUseCase) GetTopicContext(ctx context.Context, topicID, query string) (*ContextResult, error)

// GetCourseContext ahora retorna ContextResult en vez de (string, bool, error).
func (uc *RAGUseCase) GetCourseContext(ctx context.Context, courseID, query string) (*ContextResult, error)
```

### Backend: Respuestas JSON actualizadas

```json
// GET /courses/:course_id/topics/:topic_id/context (con materiales)
{
  "topic_id": "uuid",
  "context": "contenido de chunks concatenados...",
  "message": "",
  "query": ""
}

// GET /courses/:course_id/topics/:topic_id/context (sin materiales)
{
  "topic_id": "uuid",
  "context": "",
  "message": "No hay materiales para este tema. El tutor usará su conocimiento base.",
  "query": ""
}

// GET /courses/:course_id/context (sin materiales)
{
  "course_id": "uuid",
  "context": "",
  "message": "No hay materiales en este curso. El tutor usará su conocimiento base.",
  "query": "",
  "truncated": false
}
```

### Backend: Mapa MIME ampliado en `material_handler.go`

```go
var allowedMaterialFormats = map[string]domain.MaterialFormatType{
    // PDF
    "application/pdf":   domain.MaterialFormatPDF,
    "application/x-pdf": domain.MaterialFormatPDF,
    // Texto
    "text/plain":        domain.MaterialFormatTXT,
    "text/markdown":     domain.MaterialFormatMD,
    // Imágenes
    "image/png":         domain.MaterialFormatPNG,
    "image/jpeg":        domain.MaterialFormatJPEG,
    "image/jpg":         domain.MaterialFormatJPEG,  // variante no estándar
    "image/webp":        domain.MaterialFormatWEBP,
    // Audio
    "audio/mpeg":        domain.MaterialFormatAudio,
    "audio/mp3":         domain.MaterialFormatAudio,  // variante no estándar
    "audio/wav":         domain.MaterialFormatAudio,
    "audio/x-wav":       domain.MaterialFormatAudio,
    "audio/mp4":         domain.MaterialFormatAudio,
    "audio/x-m4a":       domain.MaterialFormatAudio,
    "audio/m4a":         domain.MaterialFormatAudio,  // variante no estándar
    // Fallback: application/octet-stream se acepta si la extensión es válida
    "application/octet-stream": domain.MaterialFormatTXT, // solo se aplica si extensión validada
}
```

### Backend: Lógica de validación MIME actualizada (pseudocódigo)

```go
// 1. Extensión primero (ya existente)
ext := strings.ToLower(filepath.Ext(header.Filename))
formatType, extOK := allowedExtensions[ext]
if !extOK {
    return 415
}

// 2. MIME detection
detectedMIME := http.DetectContentType(fileData)
// Normalizar
if i := strings.IndexByte(detectedMIME, ';'); i >= 0 {
    detectedMIME = strings.TrimSpace(detectedMIME[:i])
}

// 3. Validación relajada: aceptar application/octet-stream para cualquier extensión validada
if detectedMIME == "application/octet-stream" {
    log.Printf("[Material] MIME=application/octet-stream para ext=%s — aceptado por extensión", ext)
    // continuar — la extensión ya fue validada
} else if formatType == domain.MaterialFormatPDF {
    // PDF requiere match estricto de magic bytes
    if detectedMIME != "application/pdf" && detectedMIME != "application/x-pdf" {
        log.Printf("[Material] WARN: ext=%s MIME=%s — rechazado", ext, detectedMIME)
        return 415
    }
} else if formatType == domain.MaterialFormatMD || formatType == domain.MaterialFormatTXT {
    // Texto: aceptar text/plain para .txt y .md
    if detectedMIME != "text/plain" && detectedMIME != "application/octet-stream" {
        log.Printf("[Material] WARN: ext=%s MIME=%s — discrepancia", ext, detectedMIME)
        // aceptar igualmente, la extensión es la fuente de verdad para texto
    }
} else {
    // Para imágenes y audio: verificar que el MIME esté en el mapa ampliado
    if _, ok := allowedMaterialFormats[detectedMIME]; !ok {
        log.Printf("[Material] WARN: ext=%s MIME=%s — rechazado", ext, detectedMIME)
        return 415
    }
}
```

### Mobile: Contexto mínimo en `TutorSessionController`

```dart
Future<void> loadTopicContext(String courseId, String topicId) async {
  if (state.loadedTopicIds.contains(topicId)) {
    state = state.copyWith(currentTopicId: topicId);
    return;
  }
  state = state.copyWith(isLoadingContext: true);
  try {
    final dio = ref.read(dioClientProvider);
    final response = await dio.get('/courses/$courseId/topics/$topicId/context');
    final contextText = (response.data['context'] as String?) ?? '';
    final message = (response.data['message'] as String?) ?? '';

    if (contextText.isNotEmpty) {
      _geminiService.sendContextUpdate(contextText);
    } else {
      // Zero-material: construir contexto mínimo con título del topic
      final course = await ref.read(courseRepositoryProvider).getCourse(courseId);
      final topicTitle = course.topics
          .where((t) => t.id == topicId)
          .firstOrNull?.title ?? topicId;
      final minimalContext =
          'El estudiante quiere hablar del tema: "$topicTitle". '
          'No hay material de referencia para este tema. '
          'Usa tu conocimiento para guiar la conversación de forma útil. '
          'Pregunta al estudiante qué aspectos del tema le interesan.';
      _geminiService.sendContextUpdate(minimalContext);
    }

    state = state.copyWith(
      currentTopicId: topicId,
      loadedTopicIds: {...state.loadedTopicIds, topicId},
      isLoadingContext: false,
    );
  } catch (e) {
    debugPrint('[TutorSession] loadTopicContext error: $e');
    state = state.copyWith(isLoadingContext: false);
  }
}
```

### Mobile: Upload con `contentType` explícito en `material_remote_datasource.dart`

```dart
import 'package:mime/mime.dart';

Future<Material> uploadMaterial(
    String courseId, String topicId, PlatformFile file) async {
  // Derivar contentType de la extensión real del archivo
  final mimeType = lookupMimeType(file.name) ?? 'application/octet-stream';

  MultipartFile multipart;
  if (file.bytes != null) {
    multipart = MultipartFile.fromBytes(
      file.bytes!,
      filename: file.name,
      contentType: DioMediaType.parse(mimeType),
    );
  } else if (file.path != null) {
    multipart = await MultipartFile.fromFile(
      file.path!,
      filename: file.name,
      contentType: DioMediaType.parse(mimeType),
    );
  } else {
    throw Exception('Picked file has no bytes or path');
  }

  final formData = FormData.fromMap({'file': multipart});
  final response = await _dio.post(
    '/courses/$courseId/topics/$topicId/materials',
    data: formData,
  );
  if (response.statusCode == 201) {
    return Material.fromJson(response.data);
  }
  throw Exception('Failed to upload material');
}
```

### Mobile: Nueva ruta de tutor sin topicId en `app_router.dart`

```dart
GoRoute(
  path: '/tutor/:courseId',
  builder: (context, state) {
    final courseId = state.pathParameters['courseId']!;
    return TutorSessionScreen(courseId: courseId);
  },
),
// Mantener ruta existente con topicId
GoRoute(
  path: '/tutor/:courseId/:topicId',
  builder: (context, state) {
    final courseId = state.pathParameters['courseId']!;
    final topicId = state.pathParameters['topicId']!;
    return TutorSessionScreen(courseId: courseId, topicId: topicId);
  },
),
```

### Mobile: Botón global de tutor en `CourseDetailScreen`

```dart
// En _CourseDetailView, después de la lista de topics o como parte del SliverAppBar
// Se agrega un botón flotante de tutor vinculado al avatar
Widget _buildTutorButton(BuildContext context, Course course) {
  return Padding(
    padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
    child: ElevatedButton.icon(
      onPressed: () => context.push('/tutor/${course.id}'),
      icon: const Icon(Icons.smart_toy_rounded, size: 24),
      label: const Text('Da click sobre mí para charlar conmigo'),
      style: ElevatedButton.styleFrom(
        padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 20),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      ),
    ),
  );
}
```

---

## Estrategia de Testing

| Capa | Qué Testear | Enfoque |
|------|-------------|---------|
| Unit (Backend) | Validación MIME relajada: PDF con MIME `application/x-pdf`, TXT con MIME `application/octet-stream`, imagen con MIME `image/jpg` (no estándar), audio con variantes | Tabla de tests en `material_handler_test.go` con `httptest`, verificar que devuelve 201 (no 415) para cada variante |
| Unit (Backend) | `RAGUseCase.GetTopicContext` retorna `ContextResult` con `Context==""` y `Message` informativo cuando no hay chunks | Mock de `ChunkRepository` en `rag_usecase_test.go` |
| Unit (Backend) | `RAGUseCase.GetCourseContext` retorna `ContextResult` con `Message` cuando el curso no tiene materiales | Mock de repos |
| Unit (Backend) | Logging de discrepancias MIME: verificar que se emite log `warn` cuando extensión y MIME no coinciden pero se acepta | Captura de logs en test |
| Integration (Backend) | Endpoints `/context` retornan campo `message` correctamente con y sin materiales | `httptest` contra handlers con mocks |
| Unit (Mobile) | `TutorSessionController.loadTopicContext` con contexto vacío: verifica que construye contexto mínimo y llama `sendContextUpdate` con título del topic | Mock de Dio y GeminiLiveService |
| Unit (Mobile) | `material_remote_datasource.uploadMaterial` envía `contentType` correcto derivado de extensión | Mock de Dio, verificar headers del MultipartFile |
| Unit (Mobile) | Navegación desde botón global de tutor navega a `/tutor/:courseId` (sin topicId) | Widget test de `CourseDetailScreen` |
| Manual (E2E) | Subir PDF desde Android con MIME `application/octet-stream` → se acepta sin error 415 | Dispositivo físico |
| Manual (E2E) | Abrir tutor sin materiales → seleccionar topic → tutor responde con conocimiento base (no error) | Dispositivo físico o emulador |
| Manual (E2E) | Abrir tutor con materiales → seleccionar topic → contexto RAG se inyecta → tutor lo usa | Dispositivo físico o emulador |
| Manual (E2E) | Borrar topic mientras tutor abierto → chip desaparece → sesión continúa sin crash | Emulador |

---

## Migración / Rollout

No se requieren migraciones de base de datos. Todos los campos y tablas necesarios ya existen desde `tutor-course-upload-crud`.

**Cambios aditivos (compatible hacia atrás)**:
- Campo `message` en respuestas JSON: los clientes existentes lo ignoran.
- Nuevas entradas en `allowedMaterialFormats`: solo amplían aceptación, no restringen.
- Nueva ruta `/tutor/:courseId` en Flutter: aditiva, no modifica la ruta existente.

**Orden de deploy**:
1. **Backend primero**: los cambios son aditivos (MIME relajado + campo `message`). Clientes Flutter antiguos funcionan sin cambios.
2. **Mobile después**: una vez que el backend acepta variantes MIME y retorna `message`, el mobile puede usar las nuevas funcionalidades.

**Rollback**:
- MIME relajado: restaurar `allowedMaterialFormats` original (un diff acotado).
- Botón global: revert del diff de `CourseDetailScreen` restaura botones por topic.
- Zero-material: el controller puede mostrar un dialog de confirmación en vez de inyectar contexto mínimo.
- Campo `message`: es aditivo; clientes que no lo leen siguen funcionando.

---

## Preguntas Abiertas

- [ ] ¿Se debe verificar que el paquete `mime` de Dart ya está en `pubspec.yaml`? Si no, hay que agregarlo como dependencia (es un paquete estándar de Dart: `package:mime/mime.dart`).
- [ ] ¿La opción "Hablar de este tema" en el `PopupMenuButton` del topic debería ser un ítem del menú existente (junto a "Editar" y "Eliminar") o un botón separado (ej: chip o icono de chat en la card)?
- [ ] ¿El botón global de tutor debe ser un FAB flotante sobre el avatar (reemplazando el FAB actual de "Add Topic"), un widget dentro del SliverAppBar, o un botón sticky en el bottom de la pantalla? Recomendación: widget tappable sobre el avatar del curso en el SliverAppBar, con un tooltip invitando a la interacción, más un acceso alternativo en el bottom de la pantalla si no hay avatar visible.

---

## Riesgos

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|-------------|---------|------------|
| Relajar MIME permite archivos corruptos que fallan en el pipeline de extracción/chunking | Media | Bajo | La validación por extensión sigue siendo estricta (lista cerrada). El pipeline de extracción ya maneja fallos con `status: "rejected"`. Se agrega logging para detectar patrones. |
| El paquete `mime` de Dart no cubre todas las extensiones soportadas (ej: `.m4a`) | Baja | Bajo | `lookupMimeType` cubre los tipos estándar. Para extensiones no reconocidas se usa `application/octet-stream` como fallback, y el backend lo acepta si la extensión es válida. |
| El modo zero-material genera respuestas genéricas poco útiles si los títulos de topics son vagos | Media | Medio | El contexto mínimo incluye instrucción al tutor de preguntar activamente al estudiante. La UI puede mostrar un hint: "Puedes subir material para mejorar las respuestas del tutor." |
| Eliminar el botón de review pierde la oportunidad de mostrar al usuario el estado de sus materiales antes de la sesión | Baja | Bajo | Se mantiene `MaterialSummaryScreen` como pantalla accesible (no eliminada, solo desacoplada). Se puede agregar un badge en el topic card indicando cantidad de materiales validados. |
| El campo `message` en la respuesta JSON podría romper el parseo en clientes Flutter que usen tipado estricto (Freezed) | Baja | Bajo | El campo es nuevo y opcional. Dart ignora campos desconocidos en `fromJson` por defecto. Si se usa Freezed, se agrega el campo como nullable. |
| Conflicto entre la ruta `/tutor/:courseId` y `/tutor/:courseId/:topicId` en GoRouter | Baja | Medio | GoRouter matchea la ruta más específica primero. Se verifica en tests que ambas rutas funcionan independientemente. La ruta sin topicId va primero (match greedy). |
| `clientContent` en Gemini Live WebSocket podría no funcionar como mecanismo de inyección de contexto dinámico | Media | Alto | Ya implementado y probado en `tutor-course-upload-crud`. Si falla, Plan B: reconectar WebSocket con nuevo system prompt. |
