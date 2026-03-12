# Design: Sprint 8 – Klyra MVP Extendido

## Enfoque técnico

Este diseño cubre los bloques A–E del proposal: interpretación multimodal de materiales con Gemini 2.0 Flash, chat de corrección con overrides en RAG, perfil de aprendizaje invisible, exportación PDF, y experiencia audiovisual (avatar Rive, barge-in, fondos dinámicos, snapshot de cámara). Todo se controla por feature flags por bloque para rollback instantáneo sin redespliegue.

La estrategia es **incremental y aditiva**: ningún cambio rompe funcionalidad existente. El extractor actual (`PlainTextExtractor`) sigue operativo cuando el flag de Gemini Vision está desactivado. Las correcciones son registros adicionales que se aplican en tiempo de retrieval sin modificar chunks originales. El perfil de aprendizaje aprovecha el campo `LearningProfile JSONB` que ya existe en `domain.User`.

---

## Decisiones de arquitectura

### Decisión 1: Estrategia de overrides en RAG — Merge en retrieval (no re-embed)

**Elección**: Fusionar overrides sobre los chunks devueltos en tiempo de retrieval.

**Alternativas consideradas**: Re-embedear el chunk corregido y reemplazar el vector en pgvector.

**Justificación**:
- Re-embed requiere una llamada a Vertex Embedding por cada corrección, añadiendo latencia y coste. Si el estudiante corrige 10 bloques en una sesión, son 10 llamadas de embedding adicionales.
- Con merge en retrieval, `GetTopicContext` y `GetCourseContext` simplemente consultan `material_corrections` para los chunk IDs devueltos y sustituyen `Content` antes de montar el prompt. El vector original sigue siendo válido semánticamente (el chunk trata del mismo concepto, solo se corrige un dato).
- Si en el futuro la corrección cambia radicalmente el significado del chunk, se puede migrar a re-embed como mejora. Para MVP, merge es suficiente y 10x más simple.
- **Implementación**: Tras el `SearchSimilar` o `GetChunksByTopic`, un paso adicional consulta `material_corrections` por `chunk_id IN (?)` y reemplaza el `Content` de los chunks afectados.

### Decisión 2: Resumen incremental de perfil — cada N=10 mensajes, con ventana de contexto de X=4000 tokens

**Elección**: Actualizar `learning_profile` cada 10 intercambios (pares usuario-tutor) durante la sesión, no al cierre.

**Alternativas consideradas**:
- Resumen al cierre de sesión con la transcripción completa (costoso y riesgoso si la sesión es larga).
- Resumen solo de los últimos 2k tokens al cierre.

**Justificación**:
- Con N=10, cada actualización envía un prompt corto (~4000 tokens de ventana + perfil existente ~500 tokens) a Gemini 2.0 Flash (modelo barato). Si la sesión dura 40 min (~60 intercambios), se hacen 6 llamadas de resumen en lugar de 1 masiva.
- Si la sesión se interrumpe (desconexión), no se pierde el perfil completo; las actualizaciones previas ya están persistidas.
- El coste total es similar o inferior al resumen final con la transcripción completa, pero con mejor resiliencia.
- **Parámetros**: `LEARNING_PROFILE_UPDATE_INTERVAL=10` y `LEARNING_PROFILE_CONTEXT_WINDOW=4000` como variables de entorno configurables.

### Decisión 3: Paquete de grabación de audio — `record` (ya integrado) con verificación de 16 kHz nativo

**Elección**: Mantener el paquete `record` (v6.x) que ya está en `pubspec.yaml` y valida que el hardware soporta 16 kHz.

**Alternativas consideradas**: `flutter_sound`, `mic_stream`, downsampler manual en Dart.

**Justificación**:
- `record` ya está integrado y funciona con `RecordConfig(encoder: pcm16bits, sampleRate: 16000, numChannels: 1)` en `tutor_session_controller.dart`. No hay evidencia de problemas de compatibilidad.
- `flutter_sound` es más pesado y su API es más compleja. `mic_stream` no gestiona permisos automáticamente.
- Un downsampler manual (de 44.1k a 16k) añade complejidad y artefactos de alias. La gran mayoría de dispositivos Android/iOS soportan grabación nativa a 16 kHz.
- Si un dispositivo no soporta 16 kHz nativo (raro en 2025+), el fallback es grabar a 44.1 kHz y aplicar un downsampler simple (decimación por factor + filtro low-pass) en Dart antes de enviar al WebSocket. Esto se implementaría como mejora futura solo si se detectan dispositivos problemáticos en QA.

### Decisión 4: Reproductor de audio — `audioplayers` con limpieza de buffer + AEC nativo

**Elección**: Mantener `audioplayers` (v6.x, ya integrado) con `stop()` + `release()` para limpieza inmediata. Habilitar AEC mediante la flag del sistema en Android (`AudioManager.MODE_IN_COMMUNICATION`).

**Alternativas consideradas**: `just_audio`, `flutter_soloud` (baja latencia).

**Justificación**:
- `audioplayers` ya reproduce los chunks PCM del tutor. El problema del buffer residual en barge-in (200-500ms) se mitiga con: (1) llamar `stop()` seguido de `release()` y reinicializar el player para el siguiente turno; (2) en Android, configurar `AudioManager.MODE_IN_COMMUNICATION` que habilita AEC a nivel de sistema, evitando que el audio residual del altavoz se capture por el micrófono.
- `just_audio` no mejora significativamente la latencia de `stop()` para PCM raw.
- `flutter_soloud` tiene latencia más baja pero no gestiona streams PCM de manera nativa; requiere conversión a WAV.
- iOS aplica AEC automáticamente cuando se usa `AVAudioSession` con categoría `.playAndRecord`.

### Decisión 5: VAD — Threshold simple de amplitud RMS con duración mínima

**Elección**: VAD por threshold de amplitud RMS sobre el stream del micrófono, con duración mínima de 250ms.

**Alternativas consideradas**: `flutter_silero_vad` (modelo ML), solución nativa por plataforma.

**Justificación**:
- `flutter_silero_vad` añade ~5MB al bundle y complejidad de integración con un modelo ONNX. Para MVP, un threshold RMS es suficiente.
- El stream de audio del micrófono (via `record`) ya entrega chunks PCM16. Calcular RMS es trivial: `sqrt(sum(sample^2) / N)`. Si RMS > umbral (configurable, default 0.03) durante >= 250ms consecutivos, se declara actividad vocal.
- La duración mínima de 250ms filtra ruidos impulsivos (golpes, teclado).
- Si los falsos positivos son problemáticos en QA, se puede migrar a Silero VAD en una iteración futura sin cambiar la interfaz (`VadDetector` abstracta).

### Decisión 6: Generación de PDF — En Flutter con `pdf` + LaTeX renderizado a imagen

**Elección**: Generar el PDF en Flutter usando el paquete `pdf`, con las ecuaciones LaTeX renderizadas a imagen via `flutter_math_fork` + `RepaintBoundary.toImage()`.

**Alternativas consideradas**: Generación server-side con WeasyPrint/Typst.

**Justificación**:
- Generación server-side requiere un servicio adicional con dependencias de renderizado (LaTeX engine, Chromium para WeasyPrint). Añade complejidad de infraestructura y latencia de red.
- `flutter_math_fork` ya renderiza las ecuaciones en la app. Convertir cada fórmula a imagen PNG con `RepaintBoundary.toImage()` y embeber en el PDF con el paquete `pdf` es directo.
- El PDF se genera en un directorio temporal (`getTemporaryDirectory()`) y se comparte via `share_plus` sin requerir permisos de almacenamiento.
- Limitación aceptable para MVP: layout lineal simple, sin fondos decorativos.

### Decisión 7: Modelo de datos de `material_corrections` — Vinculadas a bloque del JSON de interpretación

**Elección**: Cada corrección referencia un `material_id` + `block_index` (posición del bloque en el JSON estructurado de interpretación) + `chunk_id` (nullable, el chunk más cercano que lo contiene).

**Alternativas consideradas**: Vinculación solo por chunk, vinculación por posición de texto libre.

**Justificación**:
- El JSON de interpretación (Bloque A) tiene bloques ordenados con índice explícito. El usuario corrige un bloque específico que ve en pantalla; el `block_index` identifica unívocamente el bloque corregido.
- El `chunk_id` se calcula como "el chunk cuyo `ChunkIndex` contiene la porción de texto del bloque corregido". Esto permite el merge en retrieval sin búsquedas costosas.
- Si el material se re-procesa (nueva interpretación), las correcciones antiguas se invalidan (el `block_index` puede haber cambiado). Esto es aceptable porque re-procesar un material es una acción explícita del usuario.

### Decisión 8: Nombres de feature flags

**Elección**: Variables de entorno en backend, endpoint `GET /api/v1/config` para el móvil.

| Flag (env var) | Bloque | Default | Efecto al desactivar |
|---|---|---|---|
| `FF_GEMINI_INTERPRETATION` | A | `true` | Se usa OCR existente (`PlainTextExtractor`) |
| `FF_MATERIAL_CORRECTIONS` | B | `true` | Se oculta el chat de corrección; overrides no se aplican en retrieval |
| `FF_LEARNING_PROFILE` | C | `true` | No se actualiza el perfil post-sesión |
| `FF_PDF_EXPORT` | D | `true` | Se oculta el botón de exportación |
| `FF_AVATAR_RIVE` | E1 | `true` | Se muestra avatar placeholder (actual) |
| `FF_BARGE_IN` | E2 | `true` | VAD desactivado; estudiante espera turno completo |
| `FF_DYNAMIC_BACKGROUNDS` | E3 | `false` | Fondo estático por defecto |
| `FF_CAMERA_SNAPSHOT` | E4 | `false` | Se oculta botón de cámara |

**Justificación**: Variables de entorno son la forma más simple; no requieren plataforma de feature flags. El endpoint `/config` agrega los flags relevantes para el móvil en un solo GET, cacheado en memoria. E3 y E4 (COULD) arrancan desactivados.

---

## Flujo de datos

### Flujo 1: Interpretación de materiales (Bloque A)

```
  Estudiante                 Flutter App              Go Backend              Vertex AI (Gemini 2.0 Flash)        GCS
      │                          │                        │                              │                        │
      │── sube archivo ─────────>│                        │                              │                        │
      │                          │── POST /materials ────>│                              │                        │
      │                          │                        │── upload raw file ───────────────────────────────────>│
      │                          │                        │<── storageURL ───────────────────────────────────────│
      │                          │                        │                              │                        │
      │                          │                        │ [async goroutine]             │                        │
      │                          │                        │   status → processing        │                        │
      │                          │                        │                              │                        │
      │                          │                        │  if FF_GEMINI_INTERPRETATION: │                        │
      │                          │                        │── POST generateContent ─────>│                        │
      │                          │                        │   (fileData: GCS URI,        │                        │
      │                          │                        │    response_schema: JSON)     │                        │
      │                          │                        │<── JSON estructurado ────────│                        │
      │                          │                        │                              │                        │
      │                          │                        │  else:                        │                        │
      │                          │                        │   PlainTextExtractor (OCR)   │                        │
      │                          │                        │                              │                        │
      │                          │                        │── persist interpretation ──>  DB                       │
      │                          │                        │── chunk + embed ──────────>  DB (pgvector)             │
      │                          │                        │   status → validated         │                        │
      │                          │                        │                              │                        │
      │                          │── GET /materials/:id/interpretation ──>│              │                        │
      │                          │<── JSON { blocks: [...] } ───────────│              │                        │
      │                          │                        │                              │                        │
      │<── material_review_screen│                        │                              │                        │
      │    (LaTeX + texto)       │                        │                              │                        │
```

### Flujo 2: Corrección (Bloque B)

```
  Estudiante                 Flutter App              Go Backend                  DB
      │                          │                        │                        │
      │── selecciona bloque ────>│                        │                        │
      │── escribe corrección ───>│                        │                        │
      │                          │── POST /materials/:id/corrections ──>│          │
      │                          │   { block_index, original, corrected }│          │
      │                          │                        │── INSERT material_corrections ─>│
      │                          │                        │── buscar chunk_id por overlap ──>│
      │                          │                        │<── chunk_id ────────────────────│
      │                          │                        │── UPDATE correction.chunk_id ──>│
      │                          │<── 201 Created ────────│                        │
      │                          │                        │                        │
      │                          │    [Sesión tutor futura]                        │
      │                          │                        │                        │
      │                          │── GET /context ────────>│                       │
      │                          │                        │── SearchSimilar ──────>│
      │                          │                        │<── chunks[] ──────────│
      │                          │                        │── SELECT corrections WHERE chunk_id IN (?) ──>│
      │                          │                        │<── corrections[] ─────│
      │                          │                        │── merge: reemplazar   │
      │                          │                        │   content de chunks   │
      │                          │                        │   con corrected_text  │
      │                          │<── contexto mergeado ──│                       │
```

### Flujo 3: Sesión Live con avatar y barge-in (Bloque E)

```
  Estudiante        Micrófono      Flutter App         WebSocket (Gemini Live)       Altavoz       Avatar Rive
      │                │               │                        │                      │               │
      │── habla ──────>│               │                        │                      │               │
      │                │── PCM 16k ──>│                        │                      │               │
      │                │               │── realtimeInput ──────>│                      │               │
      │                │               │<── serverContent(audio)│                      │               │
      │                │               │                        │                      │               │
      │                │               │── PCM a AudioPlayer ──────────────────────────>│               │
      │                │               │── calcular RMS ──> mouthOpen (0.0–1.0) ──────────────────────>│
      │                │               │                    state: speaking             │               │
      │                │               │                        │                      │               │
      │ [BARGE-IN]     │               │                        │                      │               │
      │── habla ──────>│               │                        │                      │               │
      │                │── PCM ───────>│                        │                      │               │
      │                │               │── VAD detecta voz      │                      │               │
      │                │               │   (RMS > 0.03, >=250ms)│                      │               │
      │                │               │                        │                      │               │
      │                │               │── player.stop()+release()─────────────────────>│(silencio)     │
      │                │               │── mouthOpen = 0.0 ────────────────────────────────────────────>│
      │                │               │   state: idle, feedback visual inmediato       │           🖐️👂│
      │                │               │                        │                      │               │
      │                │               │── enviar audio del estudiante ──>│             │               │
      │                │               │   (nuevo turno del cliente)      │             │               │
      │                │               │<── serverContent(audio respuesta)│             │               │
      │                │               │── reproducir + animar ──────────────────────────>│──────────────>│
```

### Flujo 4: Exportación PDF (Bloque D)

```
  Estudiante        Flutter App                                    Go Backend        Share OS
      │                │                                               │                │
      │── tap Exportar>│                                               │                │
      │                │── GET /materials?topic_id=X ─────────────────>│                │
      │                │<── materials[] con interpretaciones ──────────│                │
      │                │── GET /materials/:id/corrections ────────────>│                │
      │                │<── corrections[] ────────────────────────────│                │
      │                │                                               │                │
      │                │── Construir PDF en memoria:                    │                │
      │                │   1. Por cada material:                       │                │
      │                │      - Título + texto interpretado            │                │
      │                │      - LaTeX → RepaintBoundary.toImage() → PNG│                │
      │                │      - Insertar imagen en pdf.Document        │                │
      │                │      - Correcciones resaltadas en rojo        │                │
      │                │   2. Guardar en getTemporaryDirectory()       │                │
      │                │                                               │                │
      │                │── share_plus.shareXFiles([tempPdf]) ──────────────────────────>│
      │<── hoja de compartir nativa ───────────────────────────────────────────────────│
      │   (WhatsApp, Drive, Archivos, etc.)                            │                │
```

### Flujo 5: Reconexión WebSocket

```
  Flutter App            WebSocket (Gemini Live)
      │                        │
      │── conexión activa ────>│
      │        ...             │
      │<── disconnect ─────── ✖│ (red cae)
      │                        │
      │── UI: estado "reconectando"
      │   avatar: animación pensando
      │   input voz: deshabilitado
      │                        │
      │── retry #1 (1s) ──────>│ ✖ falla
      │── retry #2 (2s) ──────>│ ✖ falla
      │── retry #3 (5s) ──────>│ ✓ conectado
      │                        │
      │── re-enviar setup      │
      │── re-enviar último     │
      │   contexto cargado     │
      │                        │
      │── UI: estado "activo"  │
      │   input voz: habilitado│
```

---

## Cambios en archivos

### Backend (Go)

| Archivo | Acción | Descripción |
|---------|--------|-------------|
| `backend/internal/core/domain/material.go` | Modificar | Agregar campo `InterpretationJSON` (`datatypes.JSON`) al struct `Material`. Nuevo estado `MaterialStatusInterpreted`. |
| `backend/internal/core/domain/interpretation.go` | Crear | Structs para el schema de interpretación: `InterpretationResult`, `InterpretationBlock` (tipo: text/equation/figure/audio_transcript), con campos `Content`, `LaTeX`, `Description`, `Confidence`. |
| `backend/internal/core/domain/correction.go` | Crear | Struct `MaterialCorrection`: `ID`, `MaterialID`, `ChunkID` (nullable), `BlockIndex`, `OriginalText`, `CorrectedText`, `CreatedAt`. |
| `backend/internal/core/domain/feature_flags.go` | Crear | Struct `FeatureFlags` con campos booleanos por bloque (A–E). Función `LoadFromEnv()` que lee `os.Getenv("FF_*")`. |
| `backend/internal/repositories/gemini_interpreter.go` | Crear | `GeminiInterpreter` que implementa `ports.MaterialInterpreter`. Llama a Vertex AI Gemini 2.0 Flash con `fileData` (GCS URI) + `response_schema`. Parsea respuesta JSON a `InterpretationResult`. |
| `backend/internal/repositories/text_extractor.go` | Modificar | `PlainTextExtractor.Extract()` recibe un flag `useGeminiInterpretation`. Si es `true`, delega a `GeminiInterpreter` y persiste el JSON estructurado además del texto plano concatenado. Si es `false`, sigue con OCR/Speech actual. |
| `backend/internal/core/ports/material_port.go` | Modificar | Agregar interface `MaterialInterpreter` con método `Interpret(ctx, gcsURI, formatType) (*InterpretationResult, error)`. Agregar `CorrectionRepository` interface. |
| `backend/internal/repositories/correction_repository.go` | Crear | `PostgresCorrectionRepository` con métodos: `Create`, `FindByMaterial`, `FindByChunkIDs`. |
| `backend/internal/core/usecases/rag_usecase.go` | Modificar | En `GetTopicContext` y `GetCourseContext`: después de recuperar chunks, consultar correcciones por `chunk_id` y reemplazar `Content` de los afectados. Añadir dependencia a `CorrectionRepository`. |
| `backend/internal/core/usecases/material_usecase.go` | Modificar | En `extractTextAsync`: si `FF_GEMINI_INTERPRETATION`, llamar a `GeminiInterpreter`, persistir `InterpretationJSON` en el material, y usar el texto plano concatenado para chunking/embedding. |
| `backend/internal/core/usecases/learning_profile_usecase.go` | Crear | `LearningProfileUseCase` con método `UpdateProfile(ctx, userID, recentMessages []string)`. Construye prompt de resumen, llama a Gemini 2.0 Flash, parsea resultado y hace merge con el perfil existente via `userRepo.UpdateLearningProfile()`. |
| `backend/internal/handlers/http/material_handler.go` | Modificar | Agregar endpoints: `GET /materials/:id/interpretation`, `POST /materials/:id/corrections`, `GET /materials/:id/corrections`. |
| `backend/internal/handlers/http/config_handler.go` | Crear | `ConfigHandler` con endpoint `GET /api/v1/config` que devuelve los feature flags relevantes para el móvil como JSON. Lee de `FeatureFlags.LoadFromEnv()`. |
| `backend/internal/handlers/http/learning_profile_handler.go` | Crear | `GET /api/v1/users/me/learning-profile` y `POST /api/v1/users/me/learning-profile/update` (para trigger manual o desde el móvil al cerrar sesión). |
| `backend/internal/repositories/user_repository.go` | Modificar | Agregar método `UpdateLearningProfile(ctx, userID string, profile map[string]interface{}) error`. |
| `backend/internal/core/ports/auth_port.go` | Modificar | Agregar `UpdateLearningProfile` a la interfaz `UserRepository`. |
| `backend/cmd/api/main.go` | Modificar | Wiring de nuevos use cases, repositorios y handlers. Inyectar `GeminiInterpreter`, `CorrectionRepository`, `LearningProfileUseCase`, `ConfigHandler`. |
| `backend/migrations/000006_add_interpretation_and_corrections.up.sql` | Crear | ALTER TABLE `materials` ADD `interpretation_json JSONB`; CREATE TABLE `material_corrections`; CREATE INDEX sobre `material_id` y `chunk_id`. |
| `backend/migrations/000006_add_interpretation_and_corrections.down.sql` | Crear | DROP TABLE `material_corrections`; ALTER TABLE `materials` DROP COLUMN `interpretation_json`. |

### Mobile (Flutter)

| Archivo | Acción | Descripción |
|---------|--------|-------------|
| `mobile/pubspec.yaml` | Modificar | Agregar dependencias: `rive`, `pdf`, `share_plus`, `camera`, `path_provider`. |
| `mobile/lib/features/course/presentation/screens/material_review_screen.dart` | Crear | Pantalla de revisión de interpretación. Lista de `InterpretationBlock` con renderizado condicional: `LatexMarkdown` para ecuaciones, `Text` para texto, `Container` con descripción para figuras. Chat de corrección embebido en la parte inferior. |
| `mobile/lib/features/course/domain/interpretation_models.dart` | Crear | Modelos Freezed: `InterpretationResult`, `InterpretationBlock`, `MaterialCorrection`. |
| `mobile/lib/features/course/data/interpretation_remote_datasource.dart` | Crear | Datasource con métodos: `getInterpretation(materialId)`, `submitCorrection(materialId, blockIndex, original, corrected)`, `getCorrections(materialId)`. |
| `mobile/lib/features/course/presentation/material_review_controller.dart` | Crear | Riverpod controller para `material_review_screen`: carga interpretación, gestiona correcciones, estado de envío. |
| `mobile/lib/features/export/pdf_export_service.dart` | Crear | Servicio que genera PDF con `pdf` package. Recibe interpretaciones y correcciones, renderiza LaTeX a imagen, compone el documento, guarda en temporal y lanza `share_plus`. |
| `mobile/lib/features/export/presentation/export_button.dart` | Crear | Widget botón de exportación que invoca `PdfExportService` y muestra indicador de progreso. |
| `mobile/lib/features/tutor/data/gemini_live_service.dart` | Modificar | (1) Soporte para `toolCall` de function calling (fondos dinámicos). (2) Método `sendImageData(base64Jpeg)` para snapshot. (3) Lógica de reconexión con backoff exponencial (1s, 2s, 5s, max 30s). (4) Stream de `toolCallStream` para que el controller escuche cambios de fondo. |
| `mobile/lib/features/tutor/data/vad_detector.dart` | Crear | Clase `VadDetector` que recibe stream de PCM16, calcula RMS por ventana (20ms), aplica threshold (configurable, default 0.03) y duración mínima (250ms). Expone `Stream<bool> isSpeakingStream`. Interfaz abstracta para facilitar mock en tests. |
| `mobile/lib/features/tutor/data/audio_amplitude_tracker.dart` | Crear | Clase `AudioAmplitudeTracker` que recibe stream PCM de salida (audio del tutor), calcula RMS normalizado (0.0–1.0) por ventana de 50ms, y expone `Stream<double> amplitudeStream` para alimentar el data binding del avatar Rive. |
| `mobile/lib/features/tutor/presentation/tutor_session_controller.dart` | Modificar | (1) Integrar `VadDetector`: al detectar voz del estudiante, ejecutar barge-in (stop player, pause avatar, feedback visual). (2) Integrar `AudioAmplitudeTracker` para lip-sync. (3) Escuchar `toolCallStream` para cambio de fondos. (4) Gestionar estado de reconexión. (5) Al detener sesión, enviar resumen al endpoint de learning profile. (6) Cargar feature flags desde `GET /config` y condicionar features. |
| `mobile/lib/features/tutor/presentation/tutor_session_screen.dart` | Modificar | (1) Reemplazar `_AvatarDisplay` con widget Rive cuando `FF_AVATAR_RIVE=true`. (2) Agregar indicador de feedback visual para barge-in (ícono de micrófono iluminado). (3) Agregar botón de cámara cuando `FF_CAMERA_SNAPSHOT=true`. (4) Fondo dinámico condicional. (5) Estado "reconectando" en `_StatusBadge`. |
| `mobile/lib/features/tutor/presentation/widgets/rive_avatar_widget.dart` | Crear | Widget que carga `.riv`, obtiene `ViewModelInstanceNumber` para `mouthOpen`, y lo actualiza via `amplitudeStream`. State machine con transiciones idle ↔ speaking ↔ listening (feedback barge-in). |
| `mobile/lib/features/tutor/presentation/widgets/dynamic_background.dart` | Crear | Widget `AnimatedSwitcher` que cambia entre assets de fondo (math, science, history, default) según el state del controller. |
| `mobile/lib/features/tutor/presentation/widgets/camera_snapshot_button.dart` | Crear | Botón que abre `CameraController`, captura imagen, comprime a JPEG, y llama a `geminiService.sendImageData(base64)`. |
| `mobile/lib/core/services/feature_flags_service.dart` | Crear | Servicio que hace `GET /api/v1/config`, cachea resultado en memoria, y expone flags como propiedades booleanas. Riverpod provider. |
| `mobile/android/app/src/main/AndroidManifest.xml` | Modificar | Agregar `<uses-permission android:name="android.permission.CAMERA"/>`. |
| `mobile/ios/Runner/Info.plist` | Modificar | Agregar `NSCameraUsageDescription`: "Klyra necesita la cámara para que puedas mostrar tus apuntes al tutor interactivo durante la sesión de estudio." |
| `mobile/assets/rive/tutor_avatar.riv` | Crear | Archivo Rive con personaje tutor. State machine con estados: idle, speaking, listening. Parámetro numérico `mouthOpen` (0.0–1.0). Parámetro booleano `isListening` para feedback de barge-in. |
| `mobile/assets/backgrounds/` | Crear | 4 imágenes: `bg_math.webp`, `bg_science.webp`, `bg_history.webp`, `bg_default.webp`. |

---

## Interfaces y contratos

### Schema de interpretación (Gemini 2.0 Flash response_schema)

```json
{
  "type": "object",
  "properties": {
    "blocks": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "block_index": { "type": "integer" },
          "block_type": {
            "type": "string",
            "enum": ["text", "equation", "figure", "audio_transcript"]
          },
          "content": { "type": "string" },
          "latex": { "type": "string" },
          "figure_description": { "type": "string" },
          "confidence": { "type": "number" }
        },
        "required": ["block_index", "block_type", "content"]
      }
    },
    "language": { "type": "string" },
    "total_blocks": { "type": "integer" }
  },
  "required": ["blocks", "total_blocks"]
}
```

### Modelo de dominio Go: `InterpretationBlock`

```go
type InterpretationBlock struct {
    BlockIndex        int     `json:"block_index"`
    BlockType         string  `json:"block_type"` // text | equation | figure | audio_transcript
    Content           string  `json:"content"`
    LaTeX             string  `json:"latex,omitempty"`
    FigureDescription string  `json:"figure_description,omitempty"`
    Confidence        float64 `json:"confidence,omitempty"`
}

type InterpretationResult struct {
    Blocks      []InterpretationBlock `json:"blocks"`
    Language    string                `json:"language,omitempty"`
    TotalBlocks int                   `json:"total_blocks"`
}
```

### Modelo de dominio Go: `MaterialCorrection`

```go
type MaterialCorrection struct {
    ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    MaterialID    uuid.UUID  `json:"material_id" gorm:"type:uuid;not null;index"`
    ChunkID       *uuid.UUID `json:"chunk_id,omitempty" gorm:"type:uuid;index"`
    BlockIndex    int        `json:"block_index" gorm:"not null"`
    OriginalText  string     `json:"original_text" gorm:"not null"`
    CorrectedText string     `json:"corrected_text" gorm:"not null"`
    CreatedAt     time.Time  `json:"created_at"`
}
```

### Interface `MaterialInterpreter`

```go
type MaterialInterpreter interface {
    Interpret(ctx context.Context, gcsURI string, formatType domain.MaterialFormatType) (*domain.InterpretationResult, error)
}
```

### Interface `CorrectionRepository`

```go
type CorrectionRepository interface {
    Create(ctx context.Context, correction *domain.MaterialCorrection) error
    FindByMaterial(ctx context.Context, materialID string) ([]domain.MaterialCorrection, error)
    FindByChunkIDs(ctx context.Context, chunkIDs []string) ([]domain.MaterialCorrection, error)
}
```

### Taxonomía del `learning_profile` (JSONB)

```json
{
  "predominant_style": "visual" | "auditory" | "reading",
  "style_scores": {
    "visual": 0.0-1.0,
    "auditory": 0.0-1.0,
    "reading": 0.0-1.0
  },
  "difficult_topics": [
    { "topic": "derivadas parciales", "confusion_count": 3, "last_seen": "2026-03-10" }
  ],
  "total_session_minutes": 125,
  "total_sessions": 8,
  "last_updated": "2026-03-11T15:30:00Z"
}
```

### Nuevos endpoints HTTP

| Método | Ruta | Descripción |
|--------|------|-------------|
| `GET` | `/api/v1/config` | Feature flags para el móvil. Sin auth (público). Cacheable. |
| `GET` | `/api/v1/courses/:cid/topics/:tid/materials/:mid/interpretation` | Retorna `InterpretationResult` JSON. 404 si no existe aún (procesando). |
| `POST` | `/api/v1/courses/:cid/topics/:tid/materials/:mid/corrections` | Body: `{ block_index, original_text, corrected_text }`. Retorna 201 + corrección creada. |
| `GET` | `/api/v1/courses/:cid/topics/:tid/materials/:mid/corrections` | Lista de correcciones para un material. |
| `GET` | `/api/v1/users/me/learning-profile` | Retorna el `learning_profile` JSONB del usuario autenticado. |
| `POST` | `/api/v1/users/me/learning-profile/update` | Body: `{ recent_messages: [...] }`. Trigger manual de actualización. |

### Respuesta de `GET /api/v1/config`

```json
{
  "gemini_interpretation": true,
  "material_corrections": true,
  "learning_profile": true,
  "pdf_export": true,
  "avatar_rive": true,
  "barge_in": true,
  "dynamic_backgrounds": false,
  "camera_snapshot": false
}
```

### Function calling para fondos dinámicos (Gemini Live setup)

Se añaden `tools` en el mensaje de setup del WebSocket:

```json
{
  "setup": {
    "model": "models/gemini-2.0-flash-live-001",
    "tools": [
      {
        "function_declarations": [
          {
            "name": "change_background",
            "description": "Changes the visual background theme based on the current topic of conversation.",
            "parameters": {
              "type": "object",
              "properties": {
                "context_type": {
                  "type": "string",
                  "enum": ["math", "science", "history", "default"]
                }
              },
              "required": ["context_type"]
            }
          }
        ]
      }
    ]
  }
}
```

### Interfaz abstracta `VadDetector` (Dart)

```dart
abstract class VadDetector {
  Stream<bool> get isSpeakingStream;
  void processAudioChunk(Uint8List pcm16Data);
  void dispose();
}

class RmsVadDetector implements VadDetector {
  final double threshold; // default 0.03
  final Duration minDuration; // default 250ms
  // ...
}
```

---

## Estrategia de testing

| Capa | Qué probar | Enfoque |
|------|-----------|---------|
| **Unit (Go)** | `GeminiInterpreter.Interpret()` parsea JSON correctamente; `RAGUseCase` aplica overrides; `LearningProfileUseCase` hace merge de perfiles | Mock de HTTP client para Vertex AI (no llamar a Gemini real). Mock de `CorrectionRepository`. |
| **Unit (Go)** | `FeatureFlags.LoadFromEnv()` lee correctamente las variables | `t.Setenv()` para simular flags. |
| **Unit (Flutter)** | `RmsVadDetector` detecta voz con audio sintético y no dispara con silencio | Generar `Uint8List` con onda sinusoidal (voz) y con ceros (silencio). Verificar que `isSpeakingStream` emite `true`/`false` correctamente. |
| **Unit (Flutter)** | `AudioAmplitudeTracker` calcula RMS normalizado | Inyectar PCM conocido (onda de amplitud constante) y verificar output ≈ valor esperado. |
| **Unit (Flutter)** | `PdfExportService` genera PDF sin errores | Inyectar `InterpretationResult` de prueba; verificar que el archivo temporal se crea y tiene tamaño > 0. |
| **Widget (Flutter)** | `MaterialReviewScreen` renderiza bloques de interpretación y chat de corrección | Mock de `InterpretationRemoteDatasource`. Verificar que aparecen widgets `LatexMarkdown` para ecuaciones. |
| **Widget (Flutter)** | `RiveAvatarWidget` responde a cambios de `mouthOpen` | Inyectar stream de amplitudes; verificar que el widget no lanza excepciones (rendering test). |
| **Integration (Flutter)** | Reconexión WebSocket con backoff | Mock de `WebSocketChannel` que desconecta después de N mensajes; verificar que el servicio reintenta y los estados de UI cambian (connecting → reconnecting → active). |
| **Integration (Flutter)** | Barge-in end-to-end | Inyectar stream de audio PCM con segmento de "voz" (alta amplitud). Verificar que `VadDetector` emite `true`, el controller llama a `player.stop()`, y el state cambia a feedback visual. |
| **Integration (Go)** | Endpoint `/corrections` + retrieval con merge | Crear material con chunks, insertar corrección, llamar a `GetTopicContext` y verificar que el contenido devuelto contiene el texto corregido. |
| **Smoke (opcional)** | Interpretación real con Gemini 2.0 Flash | Subir un PDF de prueba con ecuaciones. Verificar que la respuesta JSON tiene bloques con `block_type: equation` y `latex` no vacío. Ejecutar bajo demanda, no en CI. |

### Puntos de inyección para mocks

| Componente | Interfaz/Abstract | Implementación real | Mock en tests |
|---|---|---|---|
| `GeminiLiveService` | Mismo nombre (constructor con factory provider) | Conecta a WebSocket real | `FakeGeminiLiveService` que emite streams locales |
| `AudioPlayer` | `audioplayers.AudioPlayer` | Reproduce PCM | Stub que registra llamadas a `play()`/`stop()` |
| `AudioRecorder` | `record.AudioRecorder` | Graba del micrófono | Stub que emite PCM sintético |
| `VadDetector` | `abstract VadDetector` | `RmsVadDetector` | `FakeVadDetector` con `StreamController<bool>` manual |
| `MaterialInterpreter` | `ports.MaterialInterpreter` | `GeminiInterpreter` (Vertex AI) | Struct que retorna `InterpretationResult` hardcoded |
| `CorrectionRepository` | `ports.CorrectionRepository` | `PostgresCorrectionRepository` | In-memory map |

---

## Migración y rollout

### Migración de base de datos

- **000006**: Agrega columna `interpretation_json JSONB` a `materials` y crea tabla `material_corrections`. Es aditiva y no bloquea writes existentes.
- **Rollback**: `DROP TABLE material_corrections` + `ALTER TABLE materials DROP COLUMN interpretation_json`. No afecta datos existentes.
- El campo `learning_profile` ya existe en `users` (JSONB); no requiere migración.

### Rollout progresivo

1. **Fase 1 (Staging)**: Desplegar con todos los flags activados. Validar interpretación de PDFs, correcciones, y perfil.
2. **Fase 2 (Producción - flags E3/E4 off)**: Activar bloques A-D y E1-E2. Fondos dinámicos y snapshot de cámara desactivados (COULD).
3. **Fase 3 (Producción - completo)**: Activar E3 y E4 tras validación en staging con múltiples escenarios de fondo y captura de cámara.

### Plan de rollback por bloque

| Bloque | Rollback | Datos afectados |
|--------|----------|-----------------|
| A (Interpretación) | `FF_GEMINI_INTERPRETATION=false` → vuelve a OCR | Interpretaciones ya generadas quedan en DB; se pueden borrar o ignorar |
| B (Correcciones) | `FF_MATERIAL_CORRECTIONS=false` → overrides no se aplican en retrieval | Correcciones quedan en DB pero son inertes |
| C (Perfil) | `FF_LEARNING_PROFILE=false` → no se actualiza post-sesión | Perfil existente queda intacto pero no crece |
| D (PDF Export) | `FF_PDF_EXPORT=false` → botón no visible | Sin efecto en datos |
| E1-E4 | Flags individuales → widget/feature no visible | Sin efecto en datos |

---

## Preguntas abiertas

- [ ] **Asset .riv del avatar**: ¿Lo produce el equipo de diseño en paralelo, o se usa un placeholder genérico (círculo con boca animada) para empezar desarrollo? El código no depende de un diseño final; el `mouthOpen` parameter es genérico.
- [ ] **Modelo exacto de Gemini Live en producción**: El proposal usa `gemini-live-2.5-flash-preview` pero el código actual usa `gemini-2.0-flash-live-001`. Confirmar cuál está disponible y estable al iniciar Sprint 8. El servicio permite configurar el modelo vía constante.
- [ ] **Límite de tamaño de archivo para interpretación con Gemini**: ¿Se impone un máximo adicional (e.g. 10 páginas PDF) para controlar costos de Vertex AI, además del límite de 20 MB existente?
