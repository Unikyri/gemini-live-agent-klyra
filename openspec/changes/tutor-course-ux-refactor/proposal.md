# Proposal: Refactor UX del Tutor por Curso y Robustecimiento de Upload

## Intent

Este cambio aborda cuatro fricciones que degradan la experiencia del estudiante en Klyra tras la implementación de `tutor-course-upload-crud` (archivado 2026-03-10):

1. **Upload frágil**: la subida de archivos falla intermitentemente con HTTP 415 debido a discrepancias en la detección de tipos MIME entre Flutter/Dio y Go/Gin. Aunque el fix de boundary (Bloque 2 del cambio anterior) resolvió el error 400, persisten rechazos por validaciones MIME/extensión demasiado estrictas en el backend cuando el `Content-Type` reportado por Dio no coincide exactamente con el esperado por Gin.

2. **Botón `Review Summary & Start` por topic**: existe un botón de inicio de sesión por cada topic que requiere material validado para habilitarse. Si no hay material, el usuario queda bloqueado sin poder acceder al tutor. Este flujo es confuso e innecesariamente restrictivo.

3. **UI organizada por topic, no por curso**: aunque el backend ya soporta sesiones de tutoría a nivel de curso con contexto bajo demanda (implementado en el cambio anterior), la interfaz móvil todavía presenta botones individuales por topic, fragmentando la experiencia.

4. **Material obligatorio**: el sistema actual exige material validado para iniciar la tutoría. Sin embargo, el tutor debería poder funcionar en modo "zero-material", usando los títulos de los topics y su conocimiento base (Gemini) para guiar la conversación.

Este cambio **se construye encima de `tutor-course-upload-crud`** y reutiliza la arquitectura de tutor por curso con contexto bajo demanda ya implementada. El foco aquí es UX, robustez del upload y eliminación de bloqueos artificiales.

## Scope

### In Scope

- **Upload robusto**:
  - Relajar validaciones MIME/extensión en el backend (`material_handler.go`) para aceptar variantes comunes de PDF, imágenes y audio sin falsos 415 (ej: `application/pdf` vs `application/x-pdf`; `image/jpg` vs `image/jpeg`).
  - Asegurar que Flutter/Dio envía `MultipartFile` con `contentType` y `filename` coherentes para cada tipo de archivo, usando la extensión real del archivo como fuente de verdad.
  - Agregar logging de diagnóstico en backend cuando se rechaza un archivo por MIME, para facilitar la depuración futura.

- **Botón global de tutor por curso (Flutter)**:
  - En `CourseDetailScreen` (pantalla de detalle de curso con lista de topics): eliminar el botón `Review Summary & Start` de cada topic.
  - Agregar un **botón prominente de tutor a nivel de curso**, vinculado al avatar del tutor, con copy tipo: _"Da click sobre mí para charlar conmigo"_. Al pulsarlo, se abre la sesión de tutor a nivel de curso (sin pre-seleccionar topic).
  - Mantener la posibilidad de navegar al tutor desde un topic específico (como acceso directo con topic pre-seleccionado), pero sin el paso intermedio de "Summary & Readiness".

- **Selector de topic interno en la sesión del tutor**:
  - Dentro de `TutorSessionScreen`, mantener/reforzar el selector de topics ya implementado (chips o dropdown).
  - Al elegir un topic: cargar contexto si hay materiales; si no hay materiales, inyectar solo el título del topic como contexto mínimo y dejar que el tutor use su conocimiento base.

- **Readiness no bloqueante**:
  - Modificar el uso de `CheckReadiness` / endpoint de readiness para que **no bloquee** el acceso al tutor.
  - Si no hay material validado para un topic: el tutor funciona en modo "zero-material" — usa el título del topic, la estructura del curso y el conocimiento de Gemini para generar una conversación útil.
  - La pantalla de `MaterialSummaryScreen` puede seguir existiendo como informativa, pero no como gate obligatorio.

- **Endpoints RAG con soporte zero-material**:
  - Confirmar y ajustar `GET /courses/:course_id/topics/:topic_id/context` para que retorne una respuesta útil cuando no hay materiales (ej: `{ "context": "", "topic_id": "...", "message": "No hay materiales para este tema. El tutor usará su conocimiento base." }`).
  - Confirmar que `GET /courses/:course_id/context` funciona correctamente cuando no hay chunks (retorna contexto vacío sin error 500).

### Out of Scope

- Integración de voz bidireccional (futuro).
- Cambios en el prompt de Gemini Live más allá de lo necesario para soportar contexto vacío y títulos de topics.
- Rediseño visual completo de la app (solo se ajustan las pantallas de detalle de curso y sesión de tutor).
- Edición/borrado de materiales individuales (cubierto en un cambio futuro).
- Cambios en el pipeline de procesamiento de materiales (extracción de texto, embedding, chunking) — solo se ajustan las validaciones de entrada.

## Approach

### Upload robusto (Backend + Mobile)

En el **backend** (`material_handler.go`), se amplía el mapa de tipos MIME aceptados para incluir variantes comunes que los clientes HTTP reportan de forma inconsistente. Por ejemplo, para PDFs se aceptan tanto `application/pdf` como `application/x-pdf` y `application/octet-stream` (cuando la extensión es `.pdf`). La estrategia de validación pasa a ser **extensión primero, MIME como fallback**: si la extensión del archivo está en la lista permitida, se acepta aunque el MIME reportado no coincida exactamente. Se agrega logging estructurado (nivel `warn`) cuando se detecta discrepancia MIME vs extensión para monitoreo futuro.

En **Flutter** (`material_remote_datasource.dart`), se asegura que al construir el `MultipartFile`, el campo `contentType` se derive de la extensión real del archivo usando `lookupMimeType` del paquete `mime`, y que `filename` siempre incluya la extensión correcta.

### UX de tutor por curso (Flutter)

Se refactoriza `CourseDetailScreen` para reemplazar los botones individuales `Review Summary & Start` por un **botón global de tutor** asociado al avatar del curso. Este botón navega directamente a `TutorSessionScreen` con solo el `courseId`, sin requerir topic ni material previo.

Los topics en la lista de detalle de curso conservan la opción de "ir al tutor hablando de este tema" (acceso directo), que navega a `TutorSessionScreen` con `courseId` + `topicId` pre-seleccionado. Se elimina la dependencia de `MaterialSummaryScreen` como paso previo obligatorio.

El flujo de `CheckReadiness` se modifica para ser **informativo, no bloqueante**: si el topic no tiene material validado, se muestra una indicación visual (ej: badge o texto) pero no se impide el acceso al tutor.

### Zero-material en sesión del tutor (Backend + Flutter)

En la sesión del tutor (`TutorSessionScreen` / `TutorSessionController`), cuando el estudiante selecciona un topic sin materiales:

1. Se llama a `GET /courses/:course_id/topics/:topic_id/context`.
2. El backend retorna contexto vacío (no error).
3. El controller inyecta un contexto mínimo al tutor: _"El estudiante quiere hablar del tema: [título del topic]. No hay material de referencia para este tema. Usa tu conocimiento para guiar la conversación."_
4. El tutor (Gemini) usa ese contexto junto con su conocimiento base para generar respuestas relevantes.

En el **backend**, se ajusta `RAGUseCase.GetTopicContext` para que retorne un contexto vacío con un campo `message` informativo cuando no hay chunks, en lugar de un string vacío sin contexto.

### Ajuste de endpoints existentes

Se verifica que tanto `GET /courses/:course_id/topics/:topic_id/context` como `GET /courses/:course_id/context` manejan correctamente el caso sin materiales:
- Sin chunks → `context: ""`, `message: "No hay materiales..."`, status 200.
- Sin curso/topic → 404 como siempre.
- El frontend interpreta `context` vacío como "zero-material mode".

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/internal/handlers/http/material_handler.go` | Modified | Ampliar mapa de MIME aceptados; lógica de validación extensión-primero; logging de discrepancias |
| `backend/internal/core/usecases/rag_usecase.go` | Modified | Ajustar `GetTopicContext` y `GetCourseContext` para retornar respuesta útil cuando no hay chunks |
| `backend/internal/handlers/http/rag_handler.go` | Modified | Agregar campo `message` en respuestas de contexto vacío |
| `mobile/lib/features/course/data/material_remote_datasource.dart` | Modified | Asegurar `contentType` coherente derivado de extensión del archivo |
| `mobile/lib/features/course/presentation/course_detail_screen.dart` | Modified | Eliminar botón `Review Summary & Start` por topic; agregar botón global de tutor por curso |
| `mobile/lib/features/course/presentation/screens/material_summary_screen.dart` | Modified | Convertir de gate bloqueante a pantalla informativa opcional |
| `mobile/lib/features/tutor/presentation/tutor_session_screen.dart` | Modified | Reforzar manejo de zero-material en selector de topics |
| `mobile/lib/features/tutor/presentation/tutor_session_controller.dart` | Modified | Lógica de contexto mínimo cuando no hay materiales para un topic |
| `mobile/lib/features/tutor/data/gemini_live_service.dart` | Modified | Asegurar que `sendContextUpdate` funciona con contexto mínimo (solo título) |
| `mobile/lib/features/course/presentation/widgets/*` | Modified | Ajustes en widgets de topic card para eliminar botón de review y añadir acceso directo al tutor |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Relajar MIME permite subir archivos corruptos o no soportados que fallan en extracción | Medium | La validación por extensión sigue siendo estricta (lista cerrada); el pipeline de extracción ya maneja fallos con `status: "rejected"`. Se agrega logging para detectar patrones anómalos. |
| Eliminar el botón de review pierde la oportunidad de mostrar al usuario el estado de sus materiales | Low | El estado de materiales se puede mostrar como badge o indicador en la lista de topics sin bloquear el acceso al tutor. Se puede acceder a la vista de materiales como pantalla secundaria. |
| Modo zero-material genera respuestas genéricas poco útiles si los títulos de topics son vagos | Medium | El system prompt del tutor incluye instrucciones para pedir al estudiante que elabore sobre lo que quiere aprender. Si el título es insuficiente, el tutor pregunta activamente. Se recomienda UX hint: "Puedes subir material para mejorar las respuestas del tutor." |
| Cambios en la validación de upload no cubren todos los edge cases de cada plataforma (iOS/Android/Web) | Low | Se prioriza Android (plataforma principal). Se documenta la lista completa de MIME aceptados. Se mantiene logging para detectar nuevos casos en producción. |
| El campo `message` en la respuesta de contexto vacío rompe parseo en clientes existentes | Low | El campo `message` es aditivo (nuevo); los clientes existentes simplemente lo ignoran si no lo esperan. Se actualiza el modelo Dart para incluirlo opcionalmente. |

## Rollback Plan

- **Upload**: Los cambios en el mapa de MIME son aditivos; si causan problemas, se restaura el mapa original (un diff acotado en `material_handler.go`). El cambio en Flutter de `contentType` es un ajuste menor reversible.
- **Botón global de tutor**: Si la nueva UX genera confusión, se puede restaurar el botón por topic en `CourseDetailScreen` con un revert del diff de esa pantalla. El botón global es aditivo y puede coexistir temporalmente con los botones por topic.
- **Readiness no bloqueante**: La lógica de `CheckReadiness` se desacopla pero no se elimina. Si se necesita restaurar el bloqueo, se re-habilita la verificación en la navegación a `TutorSessionScreen`.
- **Zero-material**: El campo `message` en la respuesta del endpoint es aditivo. Si el modo zero-material genera mala experiencia, se puede deshabilitar en el controller mostrando un dialog informativo antes de abrir la sesión ("No hay material para este tema. ¿Quieres continuar de todas formas?").

## Dependencies

- **`tutor-course-upload-crud` (archivado)**: Este cambio asume que toda la infraestructura de tutor por curso con contexto bajo demanda, CRUD, y fix de boundary de upload ya está implementada y funcional. Se construye directamente sobre esa base.
- **No se requieren nuevas migraciones SQL**: no se agregan tablas ni columnas.
- **No se requieren dependencias externas nuevas**: el paquete `mime` de Dart ya existe en el proyecto (verificar); las validaciones de MIME en Go usan la stdlib (`mime` package).
- **Verificaciones pendientes de `tutor-course-upload-crud`**: las tareas B2-T2, E2E-T1 y E2E-T2 del cambio anterior quedaron como verificación manual pendiente. Se recomienda completarlas antes o durante la implementación de este cambio para confirmar la base.

## Success Criteria

- [ ] Un PDF subido desde Android con `contentType` reportado como `application/octet-stream` (caso típico de Dio con ciertos archivos) se acepta sin error 415 y completa el pipeline hasta `validated`.
- [ ] Una imagen JPG subida con extensión `.jpg` y MIME `image/jpeg` (o `image/jpg`) se acepta correctamente.
- [ ] En `CourseDetailScreen`, no existe botón `Review Summary & Start` por topic. Existe un botón global de tutor asociado al avatar del curso con copy invitando a la interacción.
- [ ] Al pulsar el botón global de tutor, se abre `TutorSessionScreen` a nivel de curso sin topic pre-seleccionado. El tutor saluda y ofrece los temas disponibles.
- [ ] Al seleccionar un topic **sin materiales** en el selector de topics del tutor, el tutor inicia conversación usando el título del topic y su conocimiento base, sin mostrar error ni bloquear.
- [ ] Al seleccionar un topic **con materiales**, el contexto RAG se carga y el tutor lo utiliza como referencia.
- [ ] `GET /courses/:course_id/topics/:topic_id/context` retorna status 200 con `context: ""` y un `message` informativo cuando no hay materiales, sin error 500.
- [ ] `GET /courses/:course_id/context` retorna status 200 con contexto vacío cuando el curso no tiene materiales en ningún topic, sin error 500.
- [ ] La pantalla de `MaterialSummaryScreen` no bloquea la navegación al tutor aunque no haya material validado.
