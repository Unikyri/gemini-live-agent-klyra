# Tasks: Sprint 8 – Klyra MVP Extendido

> Secuencia recomendada: **Fase 1 (A+B)** → **Fase 2 (C+D)** → **Fase 3 (E)** → **QA/Testing** → **Feature Flags/Config**.
> Dentro de cada bloque las tareas están ordenadas por dependencia.

---

## Bloque A — Interpretación de materiales (Vertex Gemini 2.0 Flash)

### Fase A1: Fundación backend (dominio, migración, repositorio)

- [x] **A1.1** — Crear `backend/internal/core/domain/interpretation.go` con structs `InterpretationResult` y `InterpretationBlock` (`BlockIndex`, `BlockType` enum text/equation/figure/audio_transcript, `Content`, `LaTeX`, `FigureDescription`, `Confidence`). *(backend)*

- [x] **A1.2** — Crear migración `backend/migrations/000006_add_interpretation_and_corrections.up.sql`: agregar columna `interpretation_json JSONB` a tabla `materials`; crear tabla `material_corrections` con índices sobre `material_id` y `chunk_id`; crear migración `.down.sql` correspondiente. *(backend)*

- [x] **A1.3** — Modificar `backend/internal/core/domain/material.go`: agregar campo `InterpretationJSON datatypes.JSON` al struct `Material` y nuevo estado `MaterialStatusInterpreted`. *(backend)*

- [x] **A1.4** — Definir interfaz `MaterialInterpreter` en `backend/internal/core/ports/material_port.go` con método `Interpret(ctx, gcsURI, formatType) (*InterpretationResult, error)`. *(backend)*

### Fase A2: Implementación del intérprete Gemini

- [ ] **A2.1** — Crear `backend/internal/repositories/gemini_interpreter.go` implementando `MaterialInterpreter`. Debe: construir el body con `fileData` (GCS URI) + `response_mime_type: "application/json"` + `response_schema` (schema de interpretación del design); invocar `POST https://{GCP_REGION}-aiplatform.googleapis.com/v1/projects/{GCP_PROJECT_ID}/.../gemini-2.0-flash:generateContent`; parsear la respuesta JSON a `InterpretationResult`; validar contra el schema; devolver error descriptivo si falla la validación. *(backend)* — Depende de A1.1, A1.4.

- [ ] **A2.2** — Modificar `backend/internal/repositories/text_extractor.go` (`PlainTextExtractor.Extract`): aceptar flag `useGeminiInterpretation`; si es `true`, delegar a `GeminiInterpreter`, persistir `InterpretationJSON` en el material y concatenar el texto plano de los bloques para chunking/embedding; si es `false`, mantener flujo OCR/Speech actual. *(backend)* — Depende de A2.1.

- [ ] **A2.3** — Modificar `backend/internal/core/usecases/material_usecase.go` en `extractTextAsync`: leer `FF_GEMINI_INTERPRETATION` del entorno; si está activo, invocar `GeminiInterpreter` via `text_extractor.go` modificado; persistir resultado en `materials.interpretation_json`; pasar texto concatenado a chunking/embedding. *(backend)* — Depende de A2.2.

### Fase A3: Endpoints HTTP

- [x] **A3.1** — Agregar endpoint `GET /api/v1/courses/:cid/topics/:tid/materials/:mid/interpretation` en `backend/internal/handlers/http/material_handler.go`: leer `interpretation_json` del material, retornar 200 con `InterpretationResult` JSON o 404 si no existe/está procesando. *(backend)* — Depende de A2.3.

- [ ] **A3.2** — (Opcional) Si se implementa procesamiento asíncrono para PDFs grandes: el `POST .../interpret` retorna 202 Accepted con `status: processing`; el cliente hace polling con `GET .../interpretation` hasta `completed`/`failed`. Evaluar si se requiere endpoint explícito `POST .../interpret` o si se dispara automáticamente al subir material. *(backend)* — Depende de A3.1.

### Fase A4: Pantalla de revisión en Flutter

- [x] **A4.1** — Crear modelos Freezed en `mobile/lib/features/course/domain/interpretation_models.dart`: `InterpretationResult`, `InterpretationBlock`, con `fromJson`/`toJson`. *(mobile)* — Depende de A1.1 (contrato).

- [x] **A4.2** — Crear `mobile/lib/features/course/data/interpretation_remote_datasource.dart` con métodos `getInterpretation(materialId)` (GET) y `triggerInterpretation(materialId)` (POST, si aplica A3.2). *(mobile)* — Depende de A3.1.

- [x] **A4.3** — Crear `mobile/lib/features/course/presentation/material_review_controller.dart`: Riverpod controller que carga la interpretación, gestiona estados (cargando, completado, error, sin interpretación), y expone la lista de bloques. *(mobile)* — Depende de A4.1, A4.2.

- [x] **A4.4** — Crear `mobile/lib/features/course/presentation/screens/material_review_screen.dart`: pantalla con lista scrollable de bloques renderizados condicionalmente (`Math.tex()` para ecuaciones via `flutter_math_fork`, `MarkdownBody` para texto, `Text` en itálica para figuras, `Text` con estilo cita para transcripciones); barra superior con resumen (`summary`); estados de cargando / error con reintento / sin interpretación con botón "Interpretar". *(mobile)* — Depende de A4.3.

- [x] **A4.5** — Agregar navegación desde `material_list_view.dart` o `material_summary_screen.dart` hacia `material_review_screen` al tocar un material con interpretación disponible. *(mobile)* — Depende de A4.4.

---

## Bloque B — Chat de corrección (overrides + merge en retrieval)

### Fase B1: Fundación backend (dominio, repositorio)

- [x] **B1.1** — Crear `backend/internal/core/domain/correction.go` con struct `MaterialCorrection`: `ID`, `MaterialID`, `ChunkID` (nullable), `BlockIndex`, `OriginalText`, `CorrectedText`, `CreatedAt`. *(backend)*

- [x] **B1.2** — Definir interfaz `CorrectionRepository` en `backend/internal/core/ports/material_port.go` con métodos `Create`, `FindByMaterial`, `FindByChunkIDs`, `Delete`. *(backend)* — Depende de B1.1.

- [x] **B1.3** — Crear `backend/internal/repositories/correction_repository.go` (`PostgresCorrectionRepository`) implementando `CorrectionRepository`. En `Create`: hacer UPSERT por `(material_id, block_index)`; al insertar, buscar el `chunk_id` más cercano al bloque corregido (por overlap de texto) y guardarlo en la corrección. *(backend)* — Depende de B1.2, A1.2 (migración con tabla `material_corrections`).

### Fase B2: Merge de overrides en RAG

- [x] **B2.1** — Modificar `backend/internal/core/usecases/rag_usecase.go` en `GetTopicContext` y `GetCourseContext`: después de `SearchSimilar`/`GetChunksByTopic`, consultar `CorrectionRepository.FindByChunkIDs()` con los IDs de los chunks devueltos; para cada chunk con corrección, reemplazar `Content` por `CorrectedText` del override. Añadir `CorrectionRepository` como dependencia del use case. *(backend)* — Depende de B1.3.

### Fase B3: Endpoints HTTP de correcciones

- [x] **B3.1** — Agregar endpoint `POST /api/v1/courses/:cid/topics/:tid/materials/:mid/corrections` en `material_handler.go`: validar que existe interpretación `completed`, que `block_index` está dentro de rango, UPSERT via `CorrectionRepository.Create()`, retornar 201/200 con la corrección. *(backend)* — Depende de B1.3, A3.1.

- [x] **B3.2** — Agregar endpoint `GET /api/v1/courses/:cid/topics/:tid/materials/:mid/corrections` en `material_handler.go`: retornar lista de correcciones via `CorrectionRepository.FindByMaterial()`. *(backend)* — Depende de B1.3.

- [x] **B3.3** — Agregar endpoint `DELETE /api/v1/courses/:cid/topics/:tid/materials/:mid/corrections/:correction_id` en `material_handler.go`: eliminar corrección, retornar 204. *(backend)* — Depende de B1.3.

### Fase B4: UI de corrección en Flutter

- [x] **B4.1** — Agregar modelo Freezed `MaterialCorrection` en `interpretation_models.dart` (`id`, `materialId`, `blockIndex`, `originalContent`, `correctedContent`, `correctedType`, `createdAt`). *(mobile)*

- [x] **B4.2** — Agregar métodos `submitCorrection(materialId, blockIndex, original, corrected)`, `getCorrections(materialId)`, `deleteCorrection(correctionId)` en `interpretation_remote_datasource.dart`. *(mobile)* — Depende de B3.1, B3.2, B3.3.

- [x] **B4.3** — Ampliar `material_review_controller.dart` con lógica de correcciones: cargar correcciones existentes, enviar nueva corrección, eliminar corrección, marcar bloques como corregidos en la UI. *(mobile)* — Depende de B4.1, B4.2.

- [x] **B4.4** — Modificar `material_review_screen.dart`: al tocar un bloque, abrir bottom sheet/modal con contenido editable; campo de texto con el contenido original pre-llenado; dropdown para cambiar tipo de bloque; botón "Guardar corrección"; bloques corregidos se marcan visualmente (borde de color o badge). *(mobile)* — Depende de B4.3.

### Fase B5: Wiring backend

- [x] **B5.1** — Modificar `backend/cmd/api/main.go`: inyectar `GeminiInterpreter`, `PostgresCorrectionRepository`; conectar `CorrectionRepository` al `RAGUseCase`; registrar rutas de interpretación y correcciones en el router. *(backend)* — Depende de A2.1, B1.3, B2.1, B3.1, B3.2, B3.3.

---

## Bloque C — Perfil de aprendizaje invisible (resumen incremental)

### Fase C1: Fundación backend

- [x] **C1.1** — Crear `backend/internal/core/usecases/learning_profile_usecase.go` con `LearningProfileUseCase`: método `UpdateProfile(ctx, userID, recentMessages []string)` que construye prompt de resumen con los últimos mensajes (ventana de 4000 tokens) + perfil existente JSONB, invoca Gemini 2.0 Flash con `response_schema` del perfil, parsea resultado, y hace merge con perfil existente (promedio ponderado en `style_scores`, append/increment en `difficult_topics`, suma en `total_session_minutes`, incremento en `total_sessions`). *(backend)*

- [x] **C1.2** — Modificar `backend/internal/repositories/user_repository.go`: agregar método `UpdateLearningProfile(ctx, userID string, profile map[string]interface{}) error` que hace `UPDATE users SET learning_profile = $1 WHERE id = $2`. *(backend)*

- [x] **C1.3** — Modificar `backend/internal/core/ports/auth_port.go`: agregar `UpdateLearningProfile` a la interfaz `UserRepository`. *(backend)* — Depende de C1.2.

### Fase C2: Endpoints HTTP

- [x] **C2.1** — Crear `backend/internal/handlers/http/learning_profile_handler.go` con: `GET /api/v1/users/me/learning-profile` (retorna JSONB del usuario autenticado); `POST /api/v1/users/me/learning-profile/update` (body: `{ recent_messages: [...] }`, invoca `LearningProfileUseCase.UpdateProfile`). Verificar `FF_LEARNING_PROFILE`: si está desactivado, solo actualizar contadores (`total_session_minutes`, `total_sessions`) sin invocar a Gemini. *(backend)* — Depende de C1.1, C1.3.

- [x] **C2.2** — Registrar rutas de learning profile en `backend/cmd/api/main.go`; inyectar `LearningProfileUseCase`. *(backend)* — Depende de C2.1.

### Fase C3: Resumen incremental y envío desde Flutter

- [x] **C3.1** — Modificar `mobile/lib/features/tutor/presentation/tutor_session_controller.dart`: implementar contador de intercambios; cada N=10 pares (usuario-tutor), recopilar los últimos mensajes (hasta 4000 tokens), enviar `POST /api/v1/users/me/learning-profile/update` con `recent_messages`. Parámetro N configurable (env var `LEARNING_PROFILE_UPDATE_INTERVAL=10`). *(mobile)* — Depende de C2.1.

- [x] **C3.2** — En `tutor_session_controller.dart`: al cerrar sesión, si quedan intercambios pendientes desde la última actualización incremental, enviar resumen final con los mensajes restantes (máximo 2000 tokens). *(mobile)* — Depende de C3.1.

### Fase C4: Inyección del perfil en el tutor

- [ ] **C4.1** — En el backend (o en `gemini_live_service.dart` si el sistema monta el prompt en el cliente): al iniciar sesión de tutoría, obtener `learning_profile` del usuario y construir bloque de texto `[Perfil del estudiante] Estilo: ..., Temas difíciles: ..., Sesiones: ...` para concatenar al system instruction de Gemini Live. *(backend/mobile)* — Depende de C1.1.

---

## Bloque D — Exportación PDF (Flutter, share_plus)

### Fase D1: Dependencias y servicio

- [x] **D1.1** — Modificar `mobile/pubspec.yaml`: agregar dependencias `pdf` (última versión estable) y `share_plus` (última versión estable). Ejecutar `flutter pub get`. *(mobile)*

- [x] **D1.2** — Crear `mobile/lib/features/export/pdf_export_service.dart`: servicio que recibe `InterpretationResult` + lista de `MaterialCorrection` + metadatos (nombre material, nombre curso, fecha); genera PDF con paquete `pdf`; contenido: encabezado, resumen, bloques renderizados por tipo. *(mobile)* — Depende de D1.1.

### Fase D2: Renderizado de LaTeX a imagen

- [ ] **D2.1** — En `pdf_export_service.dart`: para bloques `equation`, renderizar LaTeX con `flutter_math_fork` en un widget offscreen + `RepaintBoundary.toImage()` para obtener `Uint8List` PNG; embeber la imagen en el PDF con `pw.MemoryImage`. Para bloques `text`: texto con formato en `pw.Paragraph`. Para `figure`: texto en itálica. Para `transcription`: texto en bloque cita con estilo. *(mobile)* — Depende de D1.2.

### Fase D3: Compartir y UI

- [x] **D3.1** — En `pdf_export_service.dart`: guardar el PDF generado en `getTemporaryDirectory()` y lanzar `Share.shareXFiles([XFile(path)])` via `share_plus` para abrir la hoja de compartir nativa del SO. *(mobile)* — Depende de D2.1.

- [x] **D3.2** — Crear `mobile/lib/features/export/presentation/export_button.dart`: widget botón "Exportar PDF" que invoca `PdfExportService`, muestra `CircularProgressIndicator` mientras genera, y maneja errores con snackbar. *(mobile)* — Depende de D3.1.

- [x] **D3.3** — Integrar `ExportButton` en `material_review_screen.dart` (visible cuando `FF_PDF_EXPORT = true` y hay interpretación completada). *(mobile)* — Depende de D3.2, A4.4.

---

## Bloque E — Experiencia audiovisual (avatar, barge-in, fondos, snapshot, reconexión)

### Fase E1: Audio 16 kHz y downsampler

- [x] **E1.1** — Verificar en `mobile/lib/features/tutor/presentation/tutor_session_controller.dart` que `RecordConfig(sampleRate: 16000, numChannels: 1, encoder: pcm16bits)` funciona correctamente con el paquete `record` (v6.x ya integrado). Documentar resultados en dispositivos Android e iOS de prueba. *(mobile)*

- [ ] **E1.2** — (Fallback, solo si E1.1 falla en algún dispositivo) Implementar downsampler en Dart en un archivo auxiliar: grabar a tasa nativa del hardware, aplicar filtro anti-aliasing + decimación (ej. 48 kHz → 16 kHz = factor 3) sobre el buffer PCM antes de enviar al WebSocket. *(mobile)* — Depende de E1.1.

### Fase E2: VAD local (detección de actividad vocal)

- [x] **E2.1** — Crear `mobile/lib/features/tutor/data/vad_detector.dart`: clase abstracta `VadDetector` con `Stream<bool> get isSpeakingStream`, `processAudioChunk(Uint8List)`, `dispose()`. Implementación `RmsVadDetector`: calcula RMS por ventana de 20 ms sobre PCM16; si RMS > threshold (default 0.03) durante >= 250 ms consecutivos, emite `true` en `isSpeakingStream`; emite `false` cuando RMS baja del threshold. Threshold y duración mínima configurables. *(mobile)*

### Fase E3: Barge-in con feedback visual

- [x] **E3.1** — Modificar `mobile/lib/features/tutor/presentation/tutor_session_controller.dart`: integrar `VadDetector`; al detectar `isSpeaking = true` mientras el estado de sesión es `speaking` (tutor hablando): (1) llamar `player.stop()` + `player.release()` para detener audio del tutor y limpiar buffer; (2) cambiar estado del avatar a `listening`; (3) emitir feedback visual inmediato (< 100 ms desde detección); (4) comenzar a enviar audio del estudiante como `realtimeInput.mediaChunks` al WebSocket. *(mobile)* — Depende de E2.1.

- [x] **E3.2** — Modificar `mobile/lib/features/tutor/presentation/tutor_session_screen.dart`: agregar indicador visual de barge-in (ícono de micrófono iluminado o badge animado) que se activa cuando el controller emite estado `listening` post-VAD. Visible solo cuando `FF_BARGE_IN = true`. *(mobile)* — Depende de E3.1.

### Fase E4: AEC (cancelación de eco acústico)

- [x] **E4.1** — Configurar AEC nativo por plataforma: en Android, establecer `AudioManager.MODE_IN_COMMUNICATION` para habilitar AEC a nivel de sistema al iniciar sesión de tutoría; en iOS, configurar `AVAudioSession` con categoría `.playAndRecord` y modo `.voiceChat` (aplica AEC automáticamente). Verificar que el paquete `record` permite controlar la fuente de audio (`VOICE_COMMUNICATION` en Android). *(mobile)* — Depende de E1.1.

### Fase E5: Avatar Rive con lip-sync

- [x] **E5.1** — Crear `mobile/lib/features/tutor/data/audio_amplitude_tracker.dart`: clase que recibe stream PCM de salida (audio del tutor), calcula RMS normalizado (0.0–1.0) por ventana de 50 ms, y expone `Stream<double> amplitudeStream`. *(mobile)*

- [x] **E5.2** — Agregar dependencia `rive` (última versión estable) en `mobile/pubspec.yaml`. Ejecutar `flutter pub get`. *(mobile)*

- [ ] **E5.3** — Crear (o recibir del equipo de diseño) `mobile/assets/rive/tutor_avatar.riv` con: artboard `TutorAvatar`, state machine `MainStateMachine` con estados `idle`, `speaking`, `listening`, `thinking`, `reconnecting`; propiedad numérica `mouthOpen` (0.0–1.0) vinculada via ViewModel; triggers `onBargeIn` → `listening`, `onReconnecting` → `thinking`. Si no hay asset final, crear placeholder geométrico simple (círculo + boca animada). *(mobile/diseño)*

- [x] **E5.4** — Crear `mobile/lib/features/tutor/presentation/widgets/rive_avatar_widget.dart`: widget `StatefulWidget` que carga el `.riv`, obtiene `ViewModelInstanceNumber` para `mouthOpen`, suscribe a `amplitudeStream` y actualiza `mouthOpen` a 15–30 fps; gestiona transiciones de state machine (`idle` ↔ `speaking` ↔ `listening` ↔ `thinking` ↔ `reconnecting`) según el estado del `TutorSessionController`. *(mobile)* — Depende de E5.1, E5.2, E5.3.

- [x] **E5.5** — Modificar `mobile/lib/features/tutor/presentation/tutor_session_screen.dart`: reemplazar `_AvatarDisplay` actual con `RiveAvatarWidget` cuando `FF_AVATAR_RIVE = true`; mantener UI actual (audio/transcripción) cuando el flag está desactivado. *(mobile)* — Depende de E5.4.

### Fase E6: Fondos dinámicos (function calling)

- [ ] **E6.1** — Crear assets de fondo en `mobile/assets/backgrounds/`: `bg_math.webp`, `bg_science.webp`, `bg_history.webp`, `bg_default.webp` (4 imágenes estáticas pre-diseñadas, o placeholders de color sólido si no hay arte final). *(mobile/diseño)*

- [x] **E6.2** — Modificar `mobile/lib/features/tutor/data/gemini_live_service.dart`: en el mensaje de `setup` del WebSocket, agregar `tools` con `function_declarations` para `change_background` (parámetro `context_type`: enum `math|science|history|default`). Exponer `Stream<String> toolCallStream` que emite `context_type` cuando llega un `functionCall` con `name = "change_background"`. Enviar `toolResponse` de confirmación tras procesar. *(mobile)* — Depende de E6.1.

- [x] **E6.3** — Crear `mobile/lib/features/tutor/presentation/widgets/dynamic_background.dart`: widget `AnimatedSwitcher` que escucha el `context_type` desde el controller y cambia entre assets de fondo con fade de 300 ms. *(mobile)* — Depende de E6.2.

- [x] **E6.4** — Integrar `DynamicBackground` en `tutor_session_screen.dart` como capa de fondo, visible cuando `FF_DYNAMIC_BACKGROUNDS = true`; fondo estático `bg_default` si el flag está desactivado. *(mobile)* — Depende de E6.3.

### Fase E7: Snapshot de cámara

- [x] **E7.1** — Agregar dependencia `camera` (o `image_picker`, última versión estable) en `mobile/pubspec.yaml`. Ejecutar `flutter pub get`. *(mobile)*

- [x] **E7.2** — Crear `mobile/lib/features/tutor/presentation/widgets/camera_snapshot_button.dart`: botón que abre cámara trasera, captura foto, comprime a JPEG (calidad 80%, max 1024 px de lado largo), codifica a base64. *(mobile)* — Depende de E7.1.

- [x] **E7.3** — Modificar `mobile/lib/features/tutor/data/gemini_live_service.dart`: agregar método `sendImageData(base64Jpeg, promptText)` que envía `clientContent` con `inlineData` (mimeType `image/jpeg`) + texto prompt como parte del turno del usuario. *(mobile)*

- [x] **E7.4** — Integrar `CameraSnapshotButton` en `tutor_session_screen.dart` (toolbar o FAB secundario), visible cuando `FF_CAMERA_SNAPSHOT = true`. Al capturar, invocar `geminiService.sendImageData(base64, "Mira mis apuntes y explícame lo que ves")`. *(mobile)* — Depende de E7.2, E7.3.

### Fase E8: Permisos (AndroidManifest + Info.plist)

- [x] **E8.1** — Modificar `mobile/android/app/src/main/AndroidManifest.xml`: agregar `<uses-permission android:name="android.permission.CAMERA"/>`. Verificar que `RECORD_AUDIO` y `INTERNET` ya existen. *(mobile)*

- [x] **E8.2** — Modificar `mobile/ios/Runner/Info.plist`: agregar `NSCameraUsageDescription` con rationale "Klyra necesita la cámara para que puedas mostrar tus apuntes escritos a mano al tutor interactivo y recibir ayuda visual inmediata." Verificar que `NSMicrophoneUsageDescription` ya existe. *(mobile)*

### Fase E9: Reconexión WebSocket

- [x] **E9.1** — Modificar `mobile/lib/features/tutor/data/gemini_live_service.dart`: implementar detección de desconexión (`WebSocketChannel.stream.done` o `closeCode != null`); reconexión automática con backoff exponencial (1s, 2s, 5s, 10s, 30s max, 5 reintentos max); al reconectar, re-enviar mensaje de `setup` y último contexto RAG cargado (historial de turnos almacenado en memoria en el controller). *(mobile)*

- [x] **E9.2** — Modificar `tutor_session_controller.dart`: al detectar desconexión, transicionar avatar a estado `reconnecting`; deshabilitar input de voz; mostrar indicador visual de reconexión en la UI. Al reconectar exitosamente, restaurar estado `idle` y habilitar input. *(mobile)* — Depende de E9.1, E5.4 (avatar con estado `reconnecting`).

- [x] **E9.3** — En `tutor_session_screen.dart`: agregar estado "reconectando" en `_StatusBadge` con texto y color diferenciado; deshabilitar controles de voz hasta que la sesión esté activa de nuevo. *(mobile)* — Depende de E9.2.

- [x] **E9.4** — Tras agotar reintentos (5), mostrar diálogo de error con opciones "Reconectar manualmente" y "Volver al curso". *(mobile)* — Depende de E9.1.

---

## QA / Testing

### Fase QA1: Mocks e infraestructura de testing

- [ ] **QA1.1** — Crear `FakeGeminiLiveService` en tests Flutter: implementación de `GeminiLiveService` que usa streams locales (sin WebSocket real); emite mensajes predefinidos (audio PCM sintético, respuestas de texto, toolCalls de fondos); permite simular desconexión y reconexión. *(tests/mobile)*

- [ ] **QA1.2** — Crear `FakeVadDetector` en tests Flutter: implementación de `VadDetector` con `StreamController<bool>` manual para simular `isSpeakingStream` sin audio real del micrófono. *(tests/mobile)*

- [ ] **QA1.3** — Crear mock de `MaterialInterpreter` en tests Go: struct que retorna `InterpretationResult` hardcoded sin llamar a Vertex AI. *(tests/backend)*

- [ ] **QA1.4** — Crear mock in-memory de `CorrectionRepository` en tests Go: mapa en memoria que implementa `Create`, `FindByMaterial`, `FindByChunkIDs`. *(tests/backend)*

### Fase QA2: Tests unitarios clave por bloque

- [ ] **QA2.1** — **(Bloque A)** Test unitario Go: `GeminiInterpreter.Interpret()` parsea correctamente un JSON de respuesta simulado y devuelve `InterpretationResult` con bloques de tipo `text`, `equation`, `figure`. Mock de HTTP client. *(tests/backend)* — Depende de A2.1, QA1.3.

- [ ] **QA2.2** — **(Bloque B)** Test de integración Go: crear material con chunks, insertar corrección via `CorrectionRepository`, llamar a `RAGUseCase.GetTopicContext()` y verificar que el contenido devuelto contiene `CorrectedText` en lugar del original. *(tests/backend)* — Depende de B2.1, QA1.4.

- [ ] **QA2.3** — **(Bloque C)** Test unitario Go: `LearningProfileUseCase.UpdateProfile()` hace merge correcto de perfiles (promedio ponderado en `style_scores`, incremento en contadores, append de `difficult_topics`). Mock de Gemini que retorna perfil pre-definido. *(tests/backend)* — Depende de C1.1.

- [ ] **QA2.4** — **(Bloque D)** Test unitario Flutter: `PdfExportService` genera un archivo PDF sin errores; inyectar `InterpretationResult` de prueba con bloques de tipo text y equation; verificar que el archivo temporal se crea y tiene tamaño > 0. *(tests/mobile)* — Depende de D1.2.

- [ ] **QA2.5** — **(Bloque E — VAD)** Test unitario Flutter: `RmsVadDetector` detecta voz con audio PCM sintético (onda sinusoidal de alta amplitud); no dispara con silencio (buffer de ceros); no dispara con ruido corto (< 250 ms de alta amplitud). *(tests/mobile)* — Depende de E2.1.

- [ ] **QA2.6** — **(Bloque E — Amplitud)** Test unitario Flutter: `AudioAmplitudeTracker` calcula RMS normalizado correcto dado PCM conocido (onda de amplitud constante); verificar que output ≈ valor esperado. *(tests/mobile)* — Depende de E5.1.

### Fase QA3: Tests de widget e integración

- [ ] **QA3.1** — Test de widget Flutter: `MaterialReviewScreen` renderiza bloques de interpretación correctamente (widget `Math.tex()` para ecuaciones, `MarkdownBody` para texto); mock de `InterpretationRemoteDatasource`. *(tests/mobile)* — Depende de A4.4, QA1.1.

- [ ] **QA3.2** — Test de integración Flutter: reconexión WebSocket con backoff; mock de `WebSocketChannel` que desconecta después de N mensajes; verificar que el servicio reintenta y los estados de UI cambian (`active` → `reconnecting` → `active`). *(tests/mobile)* — Depende de E9.1, QA1.1.

- [ ] **QA3.3** — Test de integración Flutter: barge-in end-to-end; inyectar stream de audio PCM con segmento de "voz" (alta amplitud); verificar que `VadDetector` emite `true`, el controller llama a `player.stop()`, y el estado cambia a feedback visual (`listening`). *(tests/mobile)* — Depende de E3.1, QA1.2.

- [ ] **QA3.4** — Test de widget Flutter: `RiveAvatarWidget` no lanza excepciones al recibir stream de amplitudes; verificar que el widget se monta y desmonta correctamente. *(tests/mobile)* — Depende de E5.4.

### Fase QA4: Smoke test (opcional, bajo demanda)

- [ ] **QA4.1** — Smoke test con Gemini 2.0 Flash real: subir un PDF de prueba con ecuaciones; verificar que la respuesta JSON tiene bloques con `block_type: equation` y `latex` no vacío. No ejecutar en CI; solo bajo demanda con créditos de Vertex AI. *(tests/backend)*

- [ ] **QA4.2** — Test unitario Go: `FeatureFlags.LoadFromEnv()` lee correctamente variables `FF_*` del entorno usando `t.Setenv()`. *(tests/backend)* — Depende de FF1.1.

---

## Feature Flags y Configuración

### Fase FF1: Backend

- [ ] **FF1.1** — Crear `backend/internal/core/domain/feature_flags.go`: struct `FeatureFlags` con campos booleanos (`GeminiInterpretation`, `MaterialCorrections`, `LearningProfile`, `PdfExport`, `AvatarRive`, `BargeIn`, `DynamicBackgrounds`, `CameraSnapshot`); función `LoadFromEnv()` que lee `os.Getenv("FF_*")` y retorna la struct. *(backend)*

- [ ] **FF1.2** — Crear `backend/internal/handlers/http/config_handler.go`: handler `GET /api/v1/config` que invoca `FeatureFlags.LoadFromEnv()` y retorna JSON con los flags relevantes para el móvil. Sin auth (público, cacheable). *(backend)* — Depende de FF1.1.

- [ ] **FF1.3** — Registrar ruta `GET /api/v1/config` en `backend/cmd/api/main.go`. *(backend)* — Depende de FF1.2.

- [ ] **FF1.4** — Agregar variables `FF_GEMINI_INTERPRETATION`, `FF_MATERIAL_CORRECTIONS`, `FF_LEARNING_PROFILE`, `FF_PDF_EXPORT`, `FF_AVATAR_RIVE`, `FF_BARGE_IN`, `FF_DYNAMIC_BACKGROUNDS`, `FF_CAMERA_SNAPSHOT`, `LEARNING_PROFILE_UPDATE_INTERVAL`, `LEARNING_PROFILE_CONTEXT_WINDOW` al archivo `.env` de ejemplo / documentación. *(backend/docs)*

### Fase FF2: Mobile

- [ ] **FF2.1** — Crear `mobile/lib/core/services/feature_flags_service.dart`: servicio Riverpod que hace `GET /api/v1/config` al inicio de sesión, cachea resultado en memoria, y expone flags como propiedades booleanas (`isGeminiInterpretationEnabled`, `isCorrectionEnabled`, etc.). *(mobile)* — Depende de FF1.2.

- [ ] **FF2.2** — Integrar `FeatureFlagsService` en los controllers y screens relevantes: condicionar visibilidad de avatar (`FF_AVATAR_RIVE`), barge-in (`FF_BARGE_IN`), fondos (`FF_DYNAMIC_BACKGROUNDS`), snapshot de cámara (`FF_CAMERA_SNAPSHOT`), botón de exportación PDF (`FF_PDF_EXPORT`), chat de corrección (`FF_MATERIAL_CORRECTIONS`). *(mobile)* — Depende de FF2.1 y de las tareas de cada bloque correspondiente.

---

## Resumen de dependencias entre bloques

| Relación | Descripción |
|----------|-------------|
| **A → B** | Corrección (B) requiere interpretación (A) completada. |
| **A + B → D** | Exportación PDF (D) requiere bloques interpretados y correcciones. |
| **E2 (barge-in) → E5 (avatar)** | El barge-in debe pausar animación del avatar. |
| **C independiente** | Puede desarrollarse en paralelo con A+B. |
| **FF transversal** | Feature flags se integran al final de cada bloque. |
| **QA post-implementación** | Los tests refieren componentes de cada bloque. |

---

## Conteo de tareas

| Bloque | Tareas | Tipo principal |
|--------|--------|----------------|
| A — Interpretación | 10 | backend + mobile |
| B — Corrección | 10 | backend + mobile |
| C — Perfil de aprendizaje | 7 | backend + mobile |
| D — Exportación PDF | 5 | mobile |
| E — Experiencia audiovisual | 18 | mobile |
| QA / Testing | 12 | tests |
| Feature Flags / Config | 6 | backend + mobile |
| **Total** | **68** | |
