# Tasks: Refactor UX del Tutor por Curso y Robustecimiento de Upload

> Orden de ejecución recomendado: **B1 (upload robusto) y B2 (contexto zero-material) son independientes y pueden ir en paralelo → B3 (ajustes menores) → M1 (UX Flutter, depende de B2) y M2 (upload Flutter, depende de B1) → T (pruebas).**
> Depende de: `tutor-course-upload-crud` (archivado 2026-03-10).

---

## Alcance

- Relajar la validación MIME/extensión en el backend para eliminar falsos HTTP 415 con estrategia "extensión primero, MIME como fallback".
- Agregar logging de discrepancias MIME para diagnóstico en producción.
- Refactorizar `GetTopicContext` y `GetCourseContext` para retornar structs con campos `has_materials` y `message`, permitiendo el modo zero-material.
- Eliminar el botón `Review Summary & Start` de cada topic en `CourseDetailScreen`; reemplazar por un botón global de tutor a nivel de curso vinculado al avatar.
- Agregar opción "Hablar de este tema" como acceso directo al tutor con topic pre-seleccionado.
- Convertir `MaterialSummaryScreen` de gate bloqueante a pantalla informativa.
- Implementar lógica de contexto mínimo en el controller del tutor cuando un topic no tiene materiales.
- Agregar ruta `/tutor/:courseId` (sin topicId) al router Flutter.
- Asegurar que `MultipartFile` en Flutter envía `contentType` coherente derivado de la extensión del archivo.

## Fuera de alcance

- Integración de voz bidireccional (futuro).
- Cambios en el prompt de Gemini Live más allá de lo necesario para contexto vacío y títulos de topics.
- Rediseño visual completo de la app (solo se ajustan `CourseDetailScreen` y `TutorSessionScreen`).
- Edición/borrado de materiales individuales (cubierto en un cambio futuro).
- Cambios en el pipeline de procesamiento de materiales (extracción, embedding, chunking).
- Nuevas migraciones de BD; no se agregan tablas ni columnas.

---

## B1 — Backend: Upload de Material Robusto

- [x] **B1.1: Ampliar mapa `allowedMaterialFormats` con variantes MIME comunes**
  Archivo: `backend/internal/handlers/http/material_handler.go`.
  Agregar al mapa las siguientes entradas:
  - `"application/x-pdf"` → `MaterialFormatPDF`
  - `"image/jpg"` → `MaterialFormatJPEG` (variante no estándar)
  - `"audio/mp3"` → `MaterialFormatAudio` (variante no estándar)
  - `"audio/x-wav"` → `MaterialFormatAudio`
  - `"audio/x-m4a"` → `MaterialFormatAudio`
  - `"audio/m4a"` → `MaterialFormatAudio`
  - `"application/octet-stream"` como fallback aceptado cuando la extensión es válida.
  Verificación: el mapa contiene todas las variantes listadas en la tabla de la spec §1.1.1.

- [x] **B1.2: Cambiar lógica de validación a "extensión primero, MIME como fallback"**
  Archivo: `backend/internal/handlers/http/material_handler.go`.
  Reemplazar la validación MIME estricta actual por la lógica descrita en design §Interfaces:
  1. Extraer extensión (`filepath.Ext`); si no está en `allowedExtensions` → 415.
  2. `http.DetectContentType` para magic bytes; normalizar (strip `; charset=...`).
  3. Si `detectedMIME == "application/octet-stream"` → aceptar (la extensión ya fue validada).
  4. Si `formatType == PDF` y magic bytes no coinciden (`!= application/pdf`, `!= application/x-pdf`) → 415 con log.
  5. Para texto (`.txt`, `.md`): aceptar aunque MIME no sea `text/plain`, logear discrepancia.
  6. Para imágenes y audio: verificar que el MIME esté en `allowedMaterialFormats`; si no → 415 con log.
  Dependencias: B1.1.
  Verificación: un PDF con MIME `application/octet-stream` se acepta (201); un `.pdf` con magic bytes de imagen se rechaza (415). Criterios UPL-AC1, UPL-AC3, UPL-AC4.

- [x] **B1.3: Agregar logging estructurado `warn` para discrepancias MIME vs extensión**
  Archivo: `backend/internal/handlers/http/material_handler.go`.
  Cuando `detectedMIME` no coincide con lo esperado para la extensión, emitir log nivel `warn` con campos: `filename`, `extension`, `detectedMIME`, `expectedMIME`, `action` (`"accepted"` o `"rejected"`).
  Dependencias: B1.2.
  Verificación: al subir un archivo con discrepancia MIME, aparece un log `warn` en stdout del servidor. Criterio UPL-AC6.

---

## B2 — Backend: Endpoints de Contexto con Soporte Zero-Material

- [x] **B2.1: Crear struct `ContextResult` en `rag_usecase.go`**
  Archivo: `backend/internal/core/usecases/rag_usecase.go`.
  Definir:
  ```go
  type ContextResult struct {
      Context      string
      Truncated    bool
      HasMaterials bool
      Message      string
  }
  ```
  Verificación: compila sin errores.

- [x] **B2.2: Refactorizar `GetTopicContext` para retornar `*ContextResult`**
  Archivo: `backend/internal/core/usecases/rag_usecase.go`.
  Cambiar firma: `func (uc *RAGUseCase) GetTopicContext(ctx context.Context, topicID, query string) (*ContextResult, error)`.
  Cuando no hay chunks → retornar `&ContextResult{Context: "", HasMaterials: false, Message: "No hay materiales para este tema. El tutor usará su conocimiento base."}`.
  Cuando hay chunks → retornar `&ContextResult{Context: <chunks>, HasMaterials: true, Message: ""}`.
  Eliminar el string embebido `"Aún no hay material validado..."` del campo `Context`.
  Dependencias: B2.1.
  Verificación: criterios ZM-AC1, ZM-AC2, ZM-AC8.

- [x] **B2.3: Refactorizar `GetCourseContext` para retornar `*ContextResult`**
  Archivo: `backend/internal/core/usecases/rag_usecase.go`.
  Cambiar firma: `func (uc *RAGUseCase) GetCourseContext(ctx context.Context, courseID, query string) (*ContextResult, error)`.
  Cuando ningún topic tiene chunks → `&ContextResult{Context: "", HasMaterials: false, Truncated: false, Message: "No hay materiales en ningún tema de este curso. El tutor usará su conocimiento base."}`.
  Cuando hay chunks → `&ContextResult{Context: <chunks>, HasMaterials: true, Truncated: <bool>, Message: ""}`.
  Dependencias: B2.1.
  Verificación: criterios ZM-AC3, ZM-AC4.

- [x] **B2.4: Actualizar `RAGHandler.GetTopicContext` para serializar campos `has_materials` y `message`**
  Archivo: `backend/internal/handlers/http/rag_handler.go`.
  Adaptar el handler para recibir `*ContextResult` del use case y serializar la respuesta JSON con campos: `topic_id`, `context`, `query`, `has_materials`, `message`.
  Dependencias: B2.2.
  Verificación: `GET /courses/:cid/topics/:tid/context` sin materiales retorna `{"context":"","has_materials":false,"message":"No hay materiales..."}`. Criterio ZM-AC1.

- [x] **B2.5: Actualizar `RAGHandler.GetCourseContext` para serializar campos `has_materials` y `message`**
  Archivo: `backend/internal/handlers/http/rag_handler.go`.
  Adaptar el handler para serializar la respuesta JSON con campos: `course_id`, `context`, `query`, `truncated`, `has_materials`, `message`.
  Dependencias: B2.3.
  Verificación: `GET /courses/:cid/context` sin materiales retorna `{"context":"","has_materials":false,"message":"No hay materiales en ningún tema...","truncated":false}`. Criterio ZM-AC3.

---

## B3 — Backend: Ajustes Menores (Soft-Delete, Consistencia)

- [ ] **B3.1: Reforzar filtro `deleted_at IS NULL` en `GetMaterialsByTopic`**
  Archivo: `backend/internal/core/usecases/material_usecase.go`.
  Verificar que `GetMaterialsByTopic` filtra materiales con `deleted_at IS NULL`. Si el filtro ya existe en el repositorio, documentar; si no, agregar la condición.
  Verificación: `GET /courses/:cid/topics/:tid/materials` no retorna materiales soft-deleted.

---

## M1 — Mobile Flutter: Nuevo Flujo de UX en Curso y Sesión del Tutor

- [x] **M1.1: Agregar ruta `/tutor/:courseId` (sin topicId) en GoRouter**
  Archivo: `mobile/lib/core/router/app_router.dart`.
  Agregar nueva ruta:
  ```dart
  GoRoute(
    path: '/tutor/:courseId',
    builder: (context, state) {
      final courseId = state.pathParameters['courseId']!;
      return TutorSessionScreen(courseId: courseId);
    },
  ),
  ```
  Mantener la ruta existente `/tutor/:courseId/:topicId` intacta.
  Verificación: `context.push('/tutor/$courseId')` abre `TutorSessionScreen` con `topicId: null`. Criterio BTN-AC4.

- [x] **M1.2: Eliminar botón "Review Summary & Start" de `_TopicSection`**
  Archivo: `mobile/lib/features/course/presentation/course_detail_screen.dart`.
  Eliminar el `OutlinedButton.icon` que navega a `/course/$courseId/topic/$topicId/summary`. El topic conserva: título + `PopupMenuButton` + `MaterialListView`.
  Verificación: no existe botón "Review Summary & Start" en ningún topic de `CourseDetailScreen`. Criterio BTN-AC1.

- [x] **M1.3: Agregar botón global de tutor vinculado al avatar del curso**
  Archivo: `mobile/lib/features/course/presentation/course_detail_screen.dart`.
  Opciones de implementación (según design §4.1.2-B):
  - `GestureDetector` sobre el avatar en el `SliverAppBar`.
  - O `ElevatedButton.icon` prominente debajo del `SliverAppBar` con ícono de tutor (`Icons.smart_toy_rounded`).
  Copy del botón: _"Hablar con el tutor"_ o _"Da click sobre mí para charlar conmigo"_.
  Acción: `context.push('/tutor/${course.id}')` (sin topicId).
  Debe ser visible sin scroll (parte superior de la pantalla).
  Dependencias: M1.1.
  Verificación: criterios BTN-AC2, BTN-AC3, BTN-AC4.

- [x] **M1.4: Agregar opción "Hablar de este tema" en `PopupMenuButton` del topic**
  Archivo: `mobile/lib/features/course/presentation/course_detail_screen.dart`.
  En el `PopupMenuButton` existente de `_TopicSection` (junto a "Editar" y "Eliminar"), agregar un nuevo `PopupMenuItem` con texto "Hablar de este tema" y un ícono de chat.
  Acción: `context.push('/tutor/${course.id}/${topic.id}')` (con topicId pre-seleccionado, sin pasar por `MaterialSummaryScreen`).
  Verificación: criterio BTN-AC6. Al seleccionar la opción, se abre `TutorSessionScreen` con el topic pre-seleccionado.

- [x] **M1.5: Convertir `MaterialSummaryScreen` de gate bloqueante a informativa**
  Archivo: `mobile/lib/features/course/presentation/screens/material_summary_screen.dart`.
  Cuando `isReady == false`, ya no bloquear la navegación al tutor. Opciones:
  - Mantener el resumen como informativo y agregar botón "Iniciar de todas formas" que navega a `TutorSessionScreen`.
  - O deprecar la pantalla y eliminar la navegación hacia ella (ya no se llega desde `CourseDetailScreen`).
  Dependencias: M1.2 (ya no se navega desde el botón eliminado).
  Verificación: criterios NB-AC1, NB-AC2.

- [x] **M1.6: Agregar campo `hasCurrentTopicMaterials` a `TutorSessionState`**
  Archivo: `mobile/lib/features/tutor/presentation/tutor_session_controller.dart`.
  Agregar al estado:
  ```dart
  final bool hasCurrentTopicMaterials; // default: true
  ```
  Se actualiza a `false` cuando se selecciona un topic sin materiales, y a `true` cuando se selecciona uno con materiales.
  Verificación: el campo existe en el estado y se actualiza correctamente al cambiar de topic.

- [x] **M1.7: Implementar lógica zero-material en `loadTopicContext`**
  Archivo: `mobile/lib/features/tutor/presentation/tutor_session_controller.dart`.
  En `loadTopicContext(courseId, topicId)`:
  1. Parsear los nuevos campos `has_materials` y `message` de la respuesta del endpoint.
  2. Si `context` no está vacío → flujo existente (`sendContextUpdate(contextText)`).
  3. Si `context` está vacío (zero-material):
     - Obtener el título del topic desde `course.topics`.
     - Construir contexto mínimo: `"El estudiante quiere hablar del tema: \"<título>\". No hay material de referencia. Usa tu conocimiento para guiar la conversación."`.
     - Enviar vía `sendContextUpdate`.
  4. Actualizar `hasCurrentTopicMaterials` en el estado.
  Dependencias: B2.2, B2.4, M1.6.
  Verificación: criterios ZM-AC5, ZM-AC6.

- [x] **M1.8: Agregar indicador visual de zero-material en `TutorSessionScreen`**
  Archivo: `mobile/lib/features/tutor/presentation/tutor_session_screen.dart`.
  Cuando `state.hasCurrentTopicMaterials == false`, mostrar un `Container` informativo debajo de los chips de topic:
  - Texto: _"Sin material de referencia — el tutor usará su conocimiento base"_.
  - Color de fondo: ámbar/amarillo tenue (informativo, no error).
  Dependencias: M1.6, M1.7.
  Verificación: criterio ZM-AC7. El indicador aparece al seleccionar un topic sin materiales y desaparece al seleccionar uno con materiales.

---

## M2 — Mobile Flutter: Upload Robusto desde App

- [x] **M2.1: Verificar/agregar paquete `mime` en `pubspec.yaml`**
  Archivo: `mobile/pubspec.yaml`.
  Verificar que la dependencia `mime` está declarada. Si no existe, agregar `mime: ^1.0.0` (o la versión más reciente disponible).
  Ejecutar `flutter pub get`.
  Verificación: `import 'package:mime/mime.dart';` compila sin errores.

- [x] **M2.2: Derivar `contentType` de la extensión del archivo en `uploadMaterial`**
  Archivo: `mobile/lib/features/course/data/material_remote_datasource.dart`.
  Al construir `MultipartFile`, derivar `contentType` usando `lookupMimeType(file.name) ?? 'application/octet-stream'` y pasarlo como `DioMediaType.parse(mimeType)`.
  Aplicar tanto a `MultipartFile.fromBytes` como a `MultipartFile.fromFile`.
  Asegurar que `filename` siempre incluya la extensión correcta del archivo original.
  Dependencias: M2.1.
  Verificación: criterio UPL-AC5. El `Content-Type` de cada parte multipart es coherente con la extensión del archivo.

---

## T — Pruebas

### T1: Unitarias Backend

- [x] **T1.1: Tests de validación MIME relajada en `material_handler_test.go`**
  Archivo: `backend/internal/handlers/http/material_handler_test.go` (crear si no existe).
  Tabla de tests con `httptest`:
  - PDF con MIME `application/pdf` → 201.
  - PDF con MIME `application/x-pdf` → 201.
  - PDF con MIME `application/octet-stream` → 201.
  - JPG con MIME `image/jpeg` → 201. JPG con MIME `image/jpg` → 201.
  - `.docx` (extensión no soportada) → 415.
  - `.pdf` con magic bytes de imagen → 415.
  Dependencias: B1.1, B1.2, B1.3.
  Verificación: criterios UPL-AC1, UPL-AC2, UPL-AC3, UPL-AC4.

- [x] **T1.2: Tests de `GetTopicContext` con y sin materiales en `rag_usecase_test.go`**
  Archivo: `backend/internal/core/usecases/rag_usecase_test.go` (crear o extender).
  - Mock de `ChunkRepository` retornando 0 chunks → verificar `ContextResult{Context: "", HasMaterials: false, Message: "No hay materiales..."}`.
  - Mock retornando N chunks → verificar `ContextResult{Context: "<texto>", HasMaterials: true, Message: ""}`.
  Dependencias: B2.1, B2.2.
  Verificación: criterios ZM-AC1, ZM-AC2.

- [x] **T1.3: Tests de `GetCourseContext` con y sin materiales en `rag_usecase_test.go`**
  Archivo: `backend/internal/core/usecases/rag_usecase_test.go`.
  - Mock con 0 chunks en todos los topics → `ContextResult{HasMaterials: false}`.
  - Mock con chunks en al menos un topic → `ContextResult{HasMaterials: true}`.
  Dependencias: B2.1, B2.3.
  Verificación: criterios ZM-AC3, ZM-AC4.

- [x] **T1.4: Tests de handlers RAG con campos `has_materials` y `message`**
  Archivo: `backend/internal/handlers/http/rag_handler_test.go` (crear o extender).
  Usar `httptest` contra handlers con mocks del use case:
  - Topic sin materiales → JSON con `has_materials: false`, `message: "..."`.
  - Curso sin materiales → JSON con `has_materials: false`, `truncated: false`.
  Dependencias: B2.4, B2.5.

### T2: Unitarias Mobile

- [x] **T2.1: Test de `loadTopicContext` con contexto vacío (zero-material)**
  Archivo: `mobile/test/features/tutor/tutor_session_controller_test.dart` (crear o extender).
  Mock de Dio retornando `{"context": "", "has_materials": false, "message": "..."}`.
  Verificar que el controller construye contexto mínimo con el título del topic y llama a `sendContextUpdate` con ese texto.
  Verificar que `hasCurrentTopicMaterials` se actualiza a `false`.
  Dependencias: M1.6, M1.7.
  Verificación: criterio ZM-AC5.

- [x] **T2.2: Test de `uploadMaterial` con `contentType` correcto**
  Archivo: `mobile/test/features/course/material_remote_datasource_test.dart` (crear o extender).
  Mock de Dio; verificar que el `MultipartFile` se construye con `contentType` derivado de la extensión.
  Dependencias: M2.2.
  Verificación: criterio UPL-AC5.

- [x] **T2.3: Widget test de `CourseDetailScreen` — botón global de tutor y ausencia de botón review**
  Archivo: `mobile/test/features/course/course_detail_screen_test.dart` (crear o extender).
  Verificar:
  - No existe widget con texto "Review Summary & Start".
  - Existe botón con texto "Hablar con el tutor" (o el copy elegido).
  - Al pulsar el botón, se navega a `/tutor/:courseId` (sin topicId).
  Dependencias: M1.2, M1.3.
  Verificación: criterios BTN-AC1, BTN-AC2, BTN-AC4.

### T3: E2E / Manuales

- [ ] **T3.1: Smoke test — Upload con variantes MIME desde Android**
  Ejecutar en dispositivo físico/emulador:
  1. Subir PDF cuyo MIME reportado por Dio es `application/octet-stream` → verificar 201 y que el material pasa a `validated`.
  2. Subir imagen JPG → verificar 201.
  3. Intentar subir `.docx` → verificar rechazo.
  Criterios cubiertos: UPL-AC1, UPL-AC2, UPL-AC3.

- [ ] **T3.2: Smoke test — Flujo completo de tutor sin materiales**
  Ejecutar en dispositivo/emulador:
  1. Crear curso con 2 topics, sin subir material a ninguno.
  2. Pulsar botón global de tutor → verificar que se abre `TutorSessionScreen` sin topic pre-seleccionado.
  3. El tutor saluda y ofrece los temas disponibles.
  4. Seleccionar un topic sin materiales → verificar indicador ámbar "Sin material de referencia" y que el tutor responde con conocimiento base.
  Criterios cubiertos: BTN-AC2, BTN-AC4, BTN-AC5, ZM-AC5, ZM-AC7, NB-AC2, NB-AC3.

- [ ] **T3.3: Smoke test — Flujo completo de tutor con materiales**
  Ejecutar en dispositivo/emulador:
  1. Subir PDF a un topic → esperar `validated`.
  2. Abrir tutor desde botón global → seleccionar topic con material → verificar que el contexto RAG se carga y el tutor lo referencia.
  3. Cambiar a topic sin material → verificar modo zero-material.
  4. Cambiar a "Curso completo" → verificar que se carga contexto del curso.
  Criterios cubiertos: ZM-AC5, ZM-AC6, BTN-AC4.

- [ ] **T3.4: Verificar que `MaterialSummaryScreen` no bloquea el acceso al tutor**
  Si la pantalla sigue existiendo tras M1.5, verificar que aunque `isReady == false` se puede navegar a `TutorSessionScreen`.
  Criterio cubierto: NB-AC1.

- [ ] **T3.5: Verificar endpoints de contexto desde consola/Postman**
  - `GET /courses/:cid/topics/:tid/context` para topic sin materiales → 200 con `has_materials: false`.
  - `GET /courses/:cid/topics/:tid/context` para topic con materiales → 200 con `has_materials: true`.
  - `GET /courses/:cid/context` para curso sin materiales → 200 con `has_materials: false`, sin error 500.
  - `GET /courses/:cid/topics/:deletedTid/context` → 404.
  Criterios cubiertos: ZM-AC1, ZM-AC2, ZM-AC3, ZM-AC4, NB-AC4.
