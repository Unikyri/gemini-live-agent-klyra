# Proposal: Sprint 8 – Klyra MVP Extendido

## Resumen ejecutivo

Klyra deja de ser "un chat que lee PDFs" para convertirse en un tutor que **entiende de verdad** el material del estudiante y le devuelve un producto de estudio tangible. En este sprint el estudiante podrá ver cómo la IA interpreta sus apuntes (ecuaciones, gráficos, texto), corregirla antes de que esa información alimente al tutor, y exportar un libro de apuntes propio. Además, se sienta la primera piedra de la experiencia inmersiva: un avatar animado que habla, se interrumpe de forma natural, y reacciona visualmente al tema de conversación.

## Objetivos de negocio y UX

1. **Confianza en el contenido** — El estudiante necesita ver y validar lo que la IA "entendió" de su material antes de confiar en el tutor. Sin esta transparencia, cualquier error de extracción erosiona la credibilidad del producto completo.
2. **Inmersión conversacional** — Un avatar animado con lip-sync, barge-in y fondos contextuales transforma una sesión de estudio monótona en una experiencia que retiene atención, especialmente en móvil.
3. **Personalización silenciosa** — Capturar un perfil de aprendizaje invisible (sin formularios ni encuestas) permite que el tutor mejore sesión tras sesión, generando un efecto de retención a largo plazo.
4. **Producto final tangible** — La exportación a PDF/libro de apuntes cierra el ciclo de estudio: el estudiante no solo aprende, sino que se lleva algo concreto.

## Alcance (MoSCoW)

| Feature | User Story | Alcance Sprint 8 | Prioridad |
|---------|-----------|-------------------|-----------|
| Interpretación de materiales | US1 | Texto + LaTeX (ecuaciones) vía Vertex Gemini 2.0 Flash. Gráficos como descripción textual semántica. | **MUST** |
| Chat de corrección | US2 | Chat simple para corregir texto extraído antes de guardarlo al RAG. Correcciones como "overrides" en DB. | **MUST** |
| Perfil de aprendizaje invisible | US8 | Persistencia de métricas básicas en `learning_profile` JSONB: estilo (visual/auditivo), temas difíciles, tiempo total. | **MUST** |
| Avatar animado (Rive) | US4 | Animación 2D básica con lip-sync por amplitud de audio. Un personaje, sin variantes. | **SHOULD** |
| Interrupción natural (barge-in) | US5 | VAD local básico para detectar voz del estudiante e interrumpir al tutor. | **SHOULD** |
| Exportación PDF | US9 | PDF simple con texto interpretado, fórmulas LaTeX renderizadas y correcciones. Sin fondos decorativos. | **SHOULD** |
| Fondos dinámicos | US6 | Cambio entre 3-4 fondos pre-definidos mediante triggers de texto (function calling). | **COULD** |
| Snapshot de cámara | US7 | Foto manual de apuntes enviada al tutor (no streaming continuo de video). | **COULD** |

### No-alcance explícito

- **Graph RAG avanzado (US3)**: La estructuración de capítulos con relaciones complejas entre tópicos queda fuera de este sprint. Se mantiene el RAG vectorial existente (pgvector + IVFFlat).
- **Video streaming continuo (US7 completo)**: El envío de frames de cámara en tiempo real (~1-2 fps) por WebSocket queda diferido. Solo se implementa el modo "snapshot" manual.
- **Múltiples avatares o personalización visual del tutor**: Un solo personaje Rive para este sprint.
- **Lip-sync por fonemas**: Se usa amplitud de audio como proxy; el análisis fonético queda para iteraciones futuras.

## Bloques funcionales

### Bloque A — Interpretación de materiales (US1)

**Objetivo**: Migrar la extracción de contenido de OCR tradicional a Vertex AI Gemini 2.0 Flash multimodal, obteniendo JSON estructurado con ecuaciones, figuras y texto.

**Stack técnico**:
- **Modelo**: `gemini-2.0-flash` vía Vertex AI.
- **Endpoint**: `POST https://{REGION}-aiplatform.googleapis.com/v1/projects/{PROJECT}/locations/{REGION}/publishers/google/models/gemini-2.0-flash:generateContent`.
- **Input**: `fileData` con el PDF/imagen/audio subido (desde GCS URI o inline base64).
- **Output estructurado**: Se usa `response_mime_type: "application/json"` junto con `response_schema` para que el modelo devuelva un JSON con campos tipados (bloques de texto, ecuaciones LaTeX, descripciones de figuras, transcripciones de audio).
- **Renderizado en Flutter**: `flutter_math_fork` para ecuaciones LaTeX; Markdown nativo para texto.

**Flujo**:
1. El estudiante sube material → pipeline existente lo almacena en GCS.
2. Backend invoca Gemini 2.0 Flash con el archivo y un schema de respuesta.
3. El JSON estructurado se persiste como "interpretación" asociada al material.
4. Flutter presenta la interpretación en una nueva pantalla (`material_review_screen`) con LaTeX renderizado.

### Bloque B — Chat de corrección (US2)

**Objetivo**: Permitir al estudiante corregir errores de interpretación antes de que el contenido alimente al tutor.

**Enfoque**:
- Interfaz de chat embebida en la pantalla de interpretación (Bloque A).
- El estudiante señala un bloque incorrecto y escribe la corrección.
- Las correcciones se guardan como "overrides" en una tabla dedicada (`material_corrections`), vinculadas al chunk original.
- Al construir el contexto para el tutor, los overrides se inyectan como "Verdades verificadas por el usuario", con prioridad sobre el texto original.

**Dependencia**: Bloque A debe completarse primero (se necesita la interpretación para corregirla).

### Bloque C — Perfil de aprendizaje invisible (US8)

**Objetivo**: Capturar métricas de aprendizaje sin fricción para personalizar al tutor progresivamente.

**Enfoque**:
- Nuevo campo `learning_profile JSONB` en la tabla `users` (o tabla dedicada `user_learning_profiles`).
- Al finalizar cada sesión de tutoría, el backend envía un prompt de resumen a Gemini que analiza la conversación y actualiza tres métricas iniciales:
  - **Estilo predominante**: visual vs auditivo vs lectura (inferido del tipo de preguntas).
  - **Temas difíciles**: lista de conceptos donde el estudiante mostró confusión recurrente.
  - **Tiempo acumulado**: total de minutos de tutoría.
- El perfil se inyecta como contexto adicional en el system prompt del tutor en sesiones futuras.

**Esfuerzo**: Bajo-medio. No requiere UI nueva; la captura es transparente.

### Bloque D — Exportación PDF (US9)

**Objetivo**: Generar un libro de apuntes descargable que combine el material interpretado, las correcciones del estudiante y resúmenes clave.

**Enfoque**:
- Generación de PDF en el móvil (Flutter) usando una librería como `pdf` o `printing`.
- Contenido del PDF: fotos originales del material, texto interpretado con LaTeX renderizado (como imágenes o texto), y correcciones del estudiante destacadas.
- Primera versión simple: layout lineal, sin fondos decorativos ni diseño editorial complejo.
- **Permisos y guardado**: En Android 13+ e iOS 15+ las restricciones de almacenamiento hacen inviable pedir permiso para escribir en "Descargas". La mitigación estándar es usar el paquete **`share_plus`**: el PDF se genera en un directorio temporal de la app y se lanza la **hoja de compartir nativa** (Share Intent / UIActivityViewController) para que el usuario elija dónde guardarlo o enviarlo (WhatsApp, Drive, archivos). Así se evita lidiar con permisos de almacenamiento y se cumple con las políticas de las tiendas.

**Pendiente de decisión técnica**: Evaluar si el renderizado de LaTeX a PDF se hace en Flutter (renderizar a imagen y embeber) o en backend (generar PDF server-side con un engine como WeasyPrint o similar). Decisión a tomar en fase de diseño.

### Bloque E — Experiencia audiovisual: primera iteración (US4 + US5 + US6 + US7)

Este bloque agrupa las features de inmersión con alcance limitado para Sprint 8.

#### E1 — Avatar animado con lip-sync (US4)

**Stack técnico**:
- **Runtime**: Paquete `rive` (Rive Flutter) con `RiveWidgetController`.
- **Renderer**: Rive Renderer nativo (C++) para rendimiento óptimo en móvil. Se prioriza `useArtboardSize: true` y el renderer por defecto del paquete Rive Flutter.
- **Archivo**: Un `.riv` pre-diseñado con un personaje tutor.
- **Data binding**: Parámetro numérico expuesto (por ejemplo `mouthOpen: 0.0–1.0`) controlado vía `ViewModelInstanceNumber`. Flutter actualiza este valor en tiempo real mapeando la amplitud RMS del audio PCM de salida de Gemini Live al rango del parámetro.
- **Animación**: State machine en Rive con transiciones idle ↔ speaking. El lip-sync por amplitud no es perfecto pero es suficiente para MVP.

**Pendiente de decisión técnica**: Definir el pipeline exacto de cálculo de amplitud RMS desde el stream de audio PCM en Flutter (librería de audio a usar, frecuencia de actualización del parámetro Rive).

#### E2 — Barge-in / Interrupción natural (US5)

**Requisito de UX (criterio de éxito)**: Cuando el VAD detecte que el estudiante está hablando (interrupción), la UI debe dar **feedback visual inmediato** para que el usuario sepa que la máquina lo escuchó — por ejemplo: el avatar lleva la mano a la oreja, o un ícono de micrófono se ilumina/resalta. Así se evita la sensación de que la app se trabó mientras la red tarda en responder.

**Stack técnico**:
- **Gemini Live API**: Modelo `gemini-live-2.5-flash-preview` (o el disponible en producción) vía WebSocket `wss://generativelanguage.googleapis.com/v1beta/models/{model}:bidiGenerateContent`.
- **Audio**: PCM mono 16 kHz (`audio/pcm;rate=16000`), enviado como `realtimeInput.mediaChunks` por el WebSocket.
- **VAD local**: Detección de actividad vocal en el móvil. Al detectar que el estudiante empieza a hablar mientras el tutor reproduce audio:
  1. Se detiene inmediatamente el `AudioPlayer`.
  2. Se pausa la animación del avatar (transición a idle).
  3. Se envía señal de corte de turno al WebSocket (cierre del stream de audio del servidor / inicio de nuevo turno del cliente).

**Pendiente de decisión técnica**: Librería de VAD en Flutter (opciones: `flutter_silero_vad`, threshold simple sobre amplitud, o solución nativa por plataforma). Definir umbral de sensibilidad y estrategia anti-ruido ambiente. Decisión a tomar en fase de diseño.

#### E3 — Fondos dinámicos (US6)

**Enfoque**:
- Usar **function calling** de Gemini Live para que el modelo emita llamadas tipo `change_background(context_type: "math" | "history" | "science" | "default")`.
- Flutter escucha estas tool calls en el stream del WebSocket y cambia el asset de fondo (imagen estática o animación Rive simple).
- Sprint 8: 3-4 fondos pre-diseñados. No se generan fondos on-the-fly.

#### E4 — Snapshot de cámara (US7)

**Enfoque**:
- Botón manual en la pantalla de tutoría para capturar una foto con la cámara trasera.
- La imagen se envía como `inlineData` (base64 JPEG) en el siguiente mensaje del WebSocket de Gemini Live, o como input multimodal en un request puntual a Gemini 2.0 Flash.
- El tutor analiza la imagen y responde sobre lo que "ve" (apuntes escritos a mano, un problema en la pizarra, etc.).
- **No** se implementa streaming continuo de frames.

## Enfoque general y secuenciación

Se adopta la **Alternativa 1 (Confianza primero, luego inmersión)** de la exploración:

| Fase | Bloques | Justificación |
|------|---------|---------------|
| **Fase 1** | A (Interpretación) + B (Corrección) | Base de todo: si el tutor no entiende bien el material, el resto no importa. |
| **Fase 2** | C (Perfil) + D (Exportación) | Cierra el ciclo de estudio con valor tangible y personalización. |
| **Fase 3** | E1 (Avatar) + E2 (Barge-in) + E3 (Fondos) + E4 (Snapshot) | Inmersión visual/auditiva sobre una base de conocimiento ya sólida. |

Klyra es una herramienta educativa antes que un juguete visual. Asegurar que el estudiante confíe en lo que la IA "lee" es crítico para la retención y el valor diferencial del producto.

## Tecnologías y modelos (referencia rápida)

| Tecnología | Modelo / Versión | Uso en Sprint 8 | Referencia |
|-----------|-----------------|-----------------|------------|
| **Vertex AI Gemini 2.0 Flash** | `gemini-2.0-flash` | Interpretación multimodal de materiales (PDF, imagen, audio). JSON estructurado con `response_schema`. | Endpoint: `POST .../models/gemini-2.0-flash:generateContent` |
| **Gemini Live API** | `gemini-live-2.5-flash-preview` | Tutoría por voz en tiempo real. WebSocket bidireccional `bidiGenerateContent`. Audio PCM mono 16 kHz. | Endpoint: `wss://generativelanguage.googleapis.com/v1beta/models/{model}:bidiGenerateContent` |
| **Rive Flutter** | Paquete `rive` (runtime) | Avatar 2D con lip-sync por amplitud. Data binding numérico vía `ViewModelInstanceNumber`. Rive Renderer C++ para rendimiento. | `RiveWidgetController`, state machines, data binding |
| **flutter_math_fork** | (ya integrado) | Renderizado de ecuaciones LaTeX en la vista de interpretación. | — |
| **pgvector + IVFFlat** | (ya integrado) | RAG vectorial existente. Sin cambios en este sprint. | — |

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `backend/internal/repositories/text_extractor.go` | Modified | Migrar extracción a Gemini 2.0 Flash multimodal con `response_schema` JSON. |
| `backend/internal/core/domain/` | New | Modelos para `material_corrections`, `learning_profile`, schema de interpretación estructurada. |
| `backend/internal/core/usecases/rag_usecase.go` | Modified | Lógica para aplicar correcciones (overrides) al contexto del tutor. |
| `backend/internal/core/usecases/` | New | Use case para actualización de `learning_profile` post-sesión. |
| `backend/internal/handlers/http/` | Modified | Endpoints para correcciones y perfil de aprendizaje. |
| `mobile/lib/features/course/presentation/screens/material_review_screen.dart` | New | Pantalla de interpretación de materiales (US1) con chat de corrección (US2). |
| `mobile/lib/features/tutor/data/gemini_live_service.dart` | Modified | Soporte para barge-in (VAD + corte de turno), function calling (fondos), y envío de snapshot de cámara. |
| `mobile/lib/features/tutor/presentation/tutor_session_controller.dart` | Modified | Coordinación de avatar Rive, audio, fondos dinámicos y estado de barge-in. |
| `mobile/lib/features/tutor/presentation/widgets/` | New | Widget de avatar Rive con data binding de lip-sync. Widget de fondos dinámicos. |
| `mobile/lib/features/export/` | New | Feature de exportación PDF con contenido interpretado + correcciones. |
| `assets/rive/` | New | Archivo `.riv` del personaje tutor con state machine y parámetro `mouthOpen`. |
| `assets/backgrounds/` | New | 3-4 imágenes de fondo temáticas (matemáticas, ciencias, historia, default). |
| `android/app/src/main/AndroidManifest.xml` | Modified | Incluir permisos `RECORD_AUDIO` y `CAMERA` con **rationales** (justificación para la tienda). Ejemplo: `android:name="android.permission.RECORD_AUDIO"` con descripción en strings o en el manifest explicando que Klyra necesita el micrófono para que el estudiante hable con el tutor interactivo. |
| `ios/Runner/Info.plist` | Modified | Incluir `NSMicrophoneUsageDescription` y `NSCameraUsageDescription` con los textos de justificación exactos (ej. "Klyra necesita el micrófono para que puedas hablar con tu tutor interactivo"; "Klyra necesita la cámara para que puedas mostrar tus apuntes al tutor"). Las tiendas rechazan actualizaciones si no hay justificación clara. |

## Riesgos y supuestos

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|-------------|---------|------------|
| **Costos de Gemini Vision**: Procesar cada archivo con Gemini 2.0 Flash es significativamente más caro que OCR tradicional | Alta | Medio | Procesar solo una vez al subir; cachear resultado. Limitar tamaño de archivo. Monitorizar costos en dashboard de Vertex. |
| **Latencia de interpretación**: PDFs grandes pueden tardar varios segundos en procesarse con Gemini 2.0 Flash | Media | Medio | Procesamiento asíncrono con indicador de progreso. Chunking de PDFs grandes en páginas individuales. |
| **Permisos de cámara/micrófono**: Usuarios deniegan permisos y la experiencia inmersiva queda rota | Media | Alto | Degradación elegante: sin micrófono → chat de texto; sin cámara → sin snapshot. Explicar claramente por qué se piden. |
| **Rendimiento móvil del avatar Rive**: Dispositivos de gama baja pueden sufrir con animación + audio + WebSocket simultáneos | Media | Medio | Usar Rive Renderer C++ (mejor rendimiento). Perfil de calidad reducida en gama baja. El avatar es SHOULD, no MUST. |
| **Falsos positivos del VAD**: Ruido ambiente dispara barge-in inadvertidamente | Media | Bajo | Threshold configurable. Requerir duración mínima de actividad vocal (200-300ms). Iteración en fase de testing. |
| **Lip-sync por amplitud insatisfactorio**: La boca se mueve de forma poco natural | Baja | Bajo | Aceptable para MVP. El lip-sync por fonemas es mejora futura documentada. |
| **Complejidad de UI en pantalla de tutoría**: Avatar + fondos + transcripción + botón de cámara en una sola pantalla móvil | Media | Medio | Diseño UX iterativo. Las features COULD (fondos, snapshot) se ocultan tras menú si la pantalla se satura. |
| **`response_schema` de Gemini no cubre todos los casos**: Algunos materiales producen JSON mal formado o incompleto | Baja | Medio | Validación estricta del JSON en backend. Fallback a texto plano si el schema falla. Chat de corrección (US2) como red de seguridad. |
| **RAG + overrides**: Si el chunk original no se re-embedea tras una corrección, la búsqueda vectorial puede devolver contexto incorrecto o ignorar el override. | Alta | Alto | Ver sección *Mitigaciones* (re-embed de chunks corregidos o merge de overrides en tiempo de retrieval). |
| **Alto consumo de tokens por resumen de sesión**: Enviar la transcripción completa al cerrar sesión dispara costes en sesiones largas (ej. 40 min). | Alta | Alto | Ver sección *Mitigaciones* (resumen incremental o solo últimos intercambios). |
| **Desconexión del WebSocket a mitad de explicación**: Redes móviles inestables (4G/5G, ascensor, cambio Wi‑Fi) pueden caer el socket. | Alta | Alto | Ver sección *Mitigaciones* (reconexión automática + estado UI). |
| **Latencia tras barge-in sin feedback visual**: Si la red tarda ~1 s en responder tras interrumpir, el usuario puede creer que la app se trabó. | Media | Alto | Ver *Criterios de éxito*: feedback visual inmediato al activarse el VAD. |
| **Buffer de audio en barge-in (E2)**: Tras `player.stop()`, muchos reproductores (audioplayers, just_audio) tienen buffer interno; el audio puede seguir sonando 200–500 ms hasta vaciarlo. Ese sonido residual puede captarse por el micrófono abierto y retroalimentar a Gemini (loop o respuestas confundidas). | Media | Alto | Ver sección *Mitigaciones*: audio de baja latencia y/o cancelación de eco acústico (AEC) en la captura del micrófono. |
| **Permisos y exportación PDF (D)**: En Android 13+ e iOS 15+ guardar directamente en "Descargas" exige permisos de almacenamiento que las tiendas revisan con severidad. | Alta | Medio | Usar `share_plus`: generar PDF en temporal y lanzar la hoja de compartir nativa para que el usuario elija dónde guardar; no se piden permisos de almacenamiento. |

### Mitigaciones (detalle)

- **RAG + overrides**: Al aplicar una corrección del estudiante, se debe **re-embedear el chunk** con el texto corregido y actualizar el vector en pgvector, de modo que la búsqueda por similitud devuelva el mismo bloque pero con el contenido ya corregido. Alternativa: en tiempo de retrieval, al montar el contexto para el tutor, fusionar los overrides sobre los chunks devueltos (el chunk se busca por ID o posición y se reemplaza el texto por el override). La decisión (re-embed vs merge en retrieval) se toma en diseño.
- **Resumen de sesión (perfil de aprendizaje)**: No enviar la transcripción completa al cerrar sesión. Estrategias:
  - **Resumen incremental**: Actualizar el perfil cada N mensajes (ej. cada 10 intercambios) con un resumen corto de ese segmento y fusionarlo con el perfil existente.
  - **Límite por tokens**: Si la sesión supera X tokens (ej. 8k), enviar a Gemini solo los **últimos** intercambios (ej. últimos 2k tokens) para el resumen de cierre, o un resumen pre-agregado en el cliente (ej. "tema: derivadas; preguntas: 5; duración: 25 min").
  - Definir N y X en fase de diseño para equilibrar coste y calidad del perfil.
- **WebSocket desconectado**: En Flutter implementar **reconexión automática** con backoff (ej. reintentos a 1s, 2s, 5s). Mientras se reconecta, la UI debe mostrar un estado claro: el avatar en animación de "pensando" o "reconectando" (state en Rive o ícono overlay), y no permitir nuevo input de voz hasta que la sesión esté de nuevo activa. El **contexto de la conversación** (historial de turnos) se preserva en el cliente y, si la API lo permite, se puede reanudar la sesión con el mismo `session_id` o enviando el historial reciente al reconectar. Detalle en diseño.
- **Feedback visual al activarse el VAD**: Ver *Criterios de éxito* y Bloque E2 más abajo.
- **Buffer de audio en barge-in**: Usar reproductores de **baja latencia** o asegurar que al llamar a stop se limpie el buffer de salida. En la captura del micrófono, aplicar **Echo Cancellation (AEC)** cuando esté disponible en la plataforma, para que el audio que aún sale por el altavoz no se envíe de vuelta a Gemini y cause bucles o respuestas incoherentes.
- **Exportación PDF y permisos**: Ver Bloque D — uso de `share_plus` para evitar permisos de almacenamiento.

### Supuestos clave

- El proyecto ya tiene acceso configurado a Vertex AI y las APIs de Gemini (credenciales, proyecto GCP, billing activo).
- `gemini-live-2.5-flash-preview` estará disponible y estable durante el desarrollo del sprint (es un modelo en preview).
- El diseño del personaje Rive (.riv) se produce en paralelo por el equipo de diseño o se usa un placeholder.
- Los 3-4 fondos temáticos se diseñan como assets estáticos; no se generan con IA en este sprint.

## Pendientes de decisión técnica

Estas decisiones deben resolverse durante la fase de `sdd-spec` o `sdd-design`, opcionalmente con apoyo de Context7 para consultar documentación actualizada:

1. **Librería de VAD en Flutter**: Evaluar `flutter_silero_vad` vs threshold simple de amplitud vs solución nativa por plataforma. Definir umbral y estrategia anti-ruido.
2. **Pipeline de audio PCM → amplitud RMS en Flutter**: Qué librería usar para capturar y procesar el stream de audio de salida de Gemini Live para alimentar el lip-sync del avatar.
3. **Generación de PDF con LaTeX**: ¿Renderizar LaTeX a imagen en Flutter y embeber en PDF, o generar PDF server-side con un engine (WeasyPrint, Typst, etc.)?
4. **Estructura exacta del `response_schema`** para Gemini 2.0 Flash: campos, tipos, manejo de bloques mixtos (texto + ecuación + figura).
5. **Modelo de datos de `material_corrections`**: ¿Vinculadas a chunk, a posición en el texto, o a bloque del JSON estructurado?
6. **Taxonomía del `learning_profile`**: Definir las dimensiones y rangos de las métricas de estilo de aprendizaje.
7. **Estrategia de feature flags**: Un flag por bloque (A–E); se adoptan feature flags para este sprint (ver sección Configuración).
8. **Versión exacta del modelo Gemini Live** a usar en producción (el modelo preview puede cambiar de nombre).
9. **PCM a 16 kHz (Bloque E – Gemini Live)**: Los micrófonos de smartphones graban por defecto a 44,1 kHz o 48 kHz; Gemini Live espera PCM mono 16 kHz. Enviar audio crudo produce errores de formato o respuestas distorsionadas. Definir qué paquete de Flutter (`record`, `flutter_sound`, `mic_stream` u otro) permite forzar sample rate a 16000 Hz en el hardware, o si se debe implementar un **downsampler manual** en Dart (reducir la frecuencia del buffer antes de enviarlo por WebSocket). Decisión en diseño.

## Estrategia de QA y pruebas

Para no depender de WebSockets reales, VAD en dispositivo y créditos de Gemini en cada compilación:

- **WebSocket en tests Flutter**: Falsear (mock) el cliente del Live API: inyectar un `GeminiLiveService` que implemente la misma interfaz pero use un stream local o un stub que emita mensajes de prueba (ej. audio PCM sintético, respuestas de texto predefinidas). Los tests unitarios y de widgets no abren conexión real; solo validan la lógica de reconexión, el manejo de mensajes y el estado de la UI (p. ej. "reconectando").
- **VAD en emulador**: Para probar el barge-in sin hablar al micrófono en cada run, inyectar **audio de prueba** (archivo PCM o WAV de unos segundos de voz) que el pipeline de audio del emulador reproduzca como si fuera entrada del micrófono, o simular el evento "VAD activado" desde un test para comprobar que el avatar/ícono de feedback se actualiza al instante.
- **Resumen**: Definir en diseño los puntos de inyección (interfaces) para sustituir el servicio real por mocks en tests, y documentar cómo ejecutar una suite de "smoke" con Gemini real (opcional, bajo demanda) para validar contratos de API.

## Configuración y variables de entorno

El backend ya utiliza un `.env` con las variables necesarias para Vertex AI, GCS, base de datos y JWT. Para Sprint 8 se asume que **el móvil siempre habla con Gemini Live a través del backend** (no se expone API key de Gemini en la app).

### Lo que ya está en `.env` (referencia)

- `GCP_PROJECT_ID`, `GCP_REGION` — Proyecto y región de Google Cloud.
- `GEMINI_API_KEY` — Usado por el backend para llamadas a Gemini (incluido Live si el backend hace de proxy).
- `GOOGLE_APPLICATION_CREDENTIALS` — Ruta al JSON de cuenta de servicio para Vertex AI / GCS.
- `GCS_BUCKET_NAME` — Bucket para materiales (PDFs, imágenes, etc.) que consumirá Gemini 2.0 Flash.
- `EMBEDDING_LOCATION`, `EMBEDDING_MODEL_ID` — Para RAG (embeddings); si se re-embedean chunks corregidos, se usa el mismo servicio.
- `DB_*`, `STORAGE_*`, `JWT_*`, `GOOGLE_CLIENT_ID`, `ALLOWED_ORIGINS` — Sin cambios para Sprint 8.

### Qué puede faltar o añadirse para Sprint 8

- **Vertex AI Gemini 2.0 Flash (interpretación)**: Si el backend usa el endpoint de Vertex (`.../models/gemini-2.0-flash:generateContent`), suele bastar con `GOOGLE_APPLICATION_CREDENTIALS` y `GCP_PROJECT_ID`/región; no es obligatorio un `GEMINI_API_KEY` separado para Vertex. Si en cambio se usa la API de Gemini (Generative Language) con la misma key, mantener `GEMINI_API_KEY`. Decisión en implementación según cómo esté montado el cliente de Vertex en Go.
- **Gemini Live (proxy por backend)**: Si el backend abre el WebSocket a `bidiGenerateContent` y el móvil se conecta al backend (no a Google), el backend necesitará la misma autenticación (API key o ADC) para hablar con Gemini Live; lo que ya tienes suele ser suficiente.
- **Variables opcionales para límites**: Por ejemplo `MAX_INTERPRETATION_FILE_SIZE_MB`, `MAX_SESSION_SUMMARY_TOKENS`, o `LEARNING_PROFILE_UPDATE_INTERVAL_MESSAGES` para las mitigaciones de coste y resumen incremental. Se pueden añadir cuando se implementen los use cases.

### Feature flags (qué son y para qué sirven)

Un **feature flag** es una configuración (variable de entorno o valor en base de datos) que activa o desactiva una funcionalidad sin cambiar código ni desplegar de nuevo. Ejemplo: `USE_GEMINI_VISION=true` en el backend hace que la extracción de materiales use Gemini 2.0 Flash; si se pone `false`, se usa el OCR anterior. Sirve para:

- **Rollback rápido**: Si algo falla en producción, se desactiva el flag y se vuelve al comportamiento anterior.
- **Despliegue progresivo**: Activar la feature solo para un grupo de usuarios o entornos (staging primero, luego producción).
- **Desarrollo**: Tener la feature implementada pero no visible hasta que esté lista.

En este sprint **sí se usarán feature flags**: un flag por bloque (A–E) como mínimo. No hace falta una plataforma externa; basta con leer variables de entorno en el backend (ej. `os.Getenv("USE_GEMINI_VISION")`) y en Flutter, si se necesitan flags por cliente, un endpoint tipo `GET /config` que devuelva `{ "avatarEnabled": true, "bargeInEnabled": true }` según configuración del servidor.

### Avatar: Vertex Imagen vs Rive

Vertex AI (Imagen) puede generar **imágenes estáticas** de un tutor; Rive sirve para **animación 2D** (lip-sync, estados). Son complementarios:

- **Opción A**: Usar la imagen generada por Vertex Imagen como **textura o arte de referencia** del personaje que luego se lleva a Rive (diseñador crea el .riv basándose en esa imagen). La animación (boca, gestos) se controla en Rive con data binding.
- **Opción B**: Avatar 100 % Rive con un personaje genérico o placeholder (sin Imagen). Cuando haya arte final del tutor, se sustituye el .riv.

Para Sprint 8 se asume que el avatar es un **archivo .riv** (creado a partir de Imagen o placeholder). Si Vertex Imagen ya genera una imagen de tutor que quieras usar, se puede definir en diseño cómo integrarla (por ejemplo como fondo del widget Rive o como asset estático junto al Rive). La decisión concreta (A o B) queda en diseño según disponibilidad de arte.

## Plan de rollback

- **Bloques A + B (Interpretación + Corrección)**: La migración de OCR a Gemini 2.0 Flash se controla con un feature flag `useGeminiVision = true/false` en el backend. Si se desactiva, se vuelve al extractor OCR existente. Las correcciones (overrides) son aditivas y no modifican el flujo existente del RAG.
- **Bloque C (Perfil)**: El campo `learning_profile` JSONB es aditivo. Si se desactiva la actualización post-sesión, el perfil simplemente no se rellena y el tutor opera sin personalización (como hasta ahora).
- **Bloque D (Exportación)**: Feature completamente nueva y aislada. Desactivar el botón de exportación en la UI es suficiente.
- **Bloque E (Experiencia audiovisual)**: Cada sub-feature tiene rollback independiente:
  - Avatar: ocultar el widget Rive y mostrar UI de audio puro (estado actual).
  - Barge-in: desactivar VAD; el estudiante espera a que el tutor termine (estado actual).
  - Fondos: mantener fondo estático por defecto.
  - Snapshot: ocultar botón de cámara.
- **Estrategia general**: Ningún bloque modifica destructivamente funcionalidad existente. Todo es aditivo o reemplazable vía flags.

## Dependencias

- **Bloques A → B**: La corrección (B) requiere que la interpretación (A) exista.
- **Bloques A+B → D**: La exportación (D) requiere contenido interpretado y corregido.
- **Bloque E2 (barge-in) depende de E1 (avatar)**: El barge-in debe pausar la animación del avatar, por lo que necesita que el avatar esté integrado (o al menos tener un stub).
- **Bloque C es independiente**: Puede desarrollarse en paralelo con cualquier otro bloque.
- **Asset del personaje Rive (.riv)**: Debe estar disponible antes de empezar E1. Si no lo está, se puede usar un placeholder geométrico simple.
- **Acceso a Vertex AI Gemini 2.0 Flash**: Prerequisito para Bloque A. Ya configurado en el proyecto.
- **Acceso a Gemini Live API**: Prerequisito para Bloque E. Ya funcional (integración existente en `gemini_live_service.dart`).
- **Cambio anterior `tutor-course-upload-crud`**: Debe estar completado (upload funcional, CRUD, contexto por curso). Es la base sobre la que opera este sprint.

## Criterios de éxito

- [ ] El estudiante puede ver la interpretación estructurada de un PDF/imagen subido, incluyendo ecuaciones LaTeX renderizadas correctamente en la app móvil.
- [ ] El estudiante puede corregir errores de interpretación mediante el chat de corrección, y las correcciones se reflejan en el contexto que recibe el tutor en sesiones futuras.
- [ ] Tras al menos 3 sesiones de tutoría, el campo `learning_profile` del usuario contiene métricas actualizadas (estilo, temas difíciles, tiempo).
- [ ] El estudiante puede exportar un PDF con el contenido interpretado y corregido de sus materiales.
- [ ] El avatar Rive se muestra en la pantalla de tutoría con lip-sync básico sincronizado al audio del tutor (en dispositivos que soporten Rive Renderer).
- [ ] El estudiante puede interrumpir al tutor hablando (barge-in) y el tutor detiene su respuesta en menos de 500ms.
- [ ] **Feedback visual inmediato al activarse el VAD**: Al detectar que el estudiante habló (barge-in), se muestra de forma instantánea un indicador en la UI (avatar con mano a la oreja, o ícono de micrófono resaltado) para que el usuario sepa que la app lo escuchó, aunque la respuesta del tutor tarde ~1 s por latencia de red.
- [ ] Los fondos de la pantalla de tutoría cambian según el contexto temático de la conversación (al menos en 2 escenarios distintos).
- [ ] El estudiante puede enviar una foto de sus apuntes al tutor durante la sesión y recibir una respuesta contextual sobre la imagen.
