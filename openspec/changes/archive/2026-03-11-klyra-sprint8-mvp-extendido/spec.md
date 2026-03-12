# Spec: Sprint 8 – Klyra MVP Extendido

> Generada a partir de `openspec/changes/klyra-sprint8-mvp-extendido/proposal.md` y `exploration.md`.
> Objetivo: definir contratos de backend (Go), WebSocket (Gemini Live), Flutter (mobile), feature flags y criterios de aceptación con detalle suficiente para que `sdd-design` y `sdd-tasks` puedan trabajar sin ambigüedad.

---

## Tabla de contenidos

1. [Bloque A — Interpretación de materiales](#bloque-a--interpretación-de-materiales)
2. [Bloque B — Chat de corrección](#bloque-b--chat-de-corrección)
3. [Bloque C — Perfil de aprendizaje invisible](#bloque-c--perfil-de-aprendizaje-invisible)
4. [Bloque D — Exportación PDF](#bloque-d--exportación-pdf)
5. [Bloque E — Experiencia audiovisual](#bloque-e--experiencia-audiovisual)
6. [Feature flags y configuración](#feature-flags-y-configuración)
7. [Contratos de Gemini Live (WebSocket)](#contratos-de-gemini-live-websocket)
8. [Criterios de aceptación](#criterios-de-aceptación)
9. [Pendientes y decisiones diferidas](#pendientes-y-decisiones-diferidas)

---

## Bloque A — Interpretación de materiales

### A.1 Modelo de dominio: `MaterialInterpretation`

Nuevo struct en `backend/internal/core/domain/`:

```go
type MaterialInterpretation struct {
    ID         string    `json:"id"`          // UUID
    MaterialID string    `json:"material_id"` // FK → materials.id
    TopicID    string    `json:"topic_id"`    // FK → topics.id (denormalizado para queries)
    Status     string    `json:"status"`      // pending | processing | completed | failed
    Content    JSONB     `json:"content"`     // JSON estructurado (ver schema abajo)
    ModelUsed  string    `json:"model_used"`  // ej. "gemini-2.0-flash"
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}
```

#### Schema JSON de `Content` (response_schema de Gemini)

```json
{
  "type": "object",
  "properties": {
    "blocks": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "index":       { "type": "integer", "description": "Orden secuencial del bloque en el material" },
          "block_type":  { "type": "string", "enum": ["text", "equation", "figure", "transcription"] },
          "content":     { "type": "string", "description": "Texto plano, LaTeX (sin delimitadores $$), descripción semántica de figura, o transcripción de audio" },
          "confidence":  { "type": "number", "description": "0.0–1.0, confianza del modelo en la extracción" },
          "page":        { "type": "integer", "description": "Número de página de origen (1-indexed), null si no aplica" }
        },
        "required": ["index", "block_type", "content"]
      }
    },
    "summary": { "type": "string", "description": "Resumen breve del material completo (1–3 oraciones)" },
    "language": { "type": "string", "description": "Idioma detectado del material (ISO 639-1)" }
  },
  "required": ["blocks"]
}
```

> **Nota:** La estructura exacta del `response_schema` (nombres de campos, niveles de anidamiento, manejo de bloques mixtos texto+ecuación en un mismo párrafo) queda como **"Definido en design"**. El schema anterior es el contrato mínimo; design puede extenderlo pero no reducirlo.

### A.2 Migración de base de datos

Nueva migración `000006_create_material_interpretations.up.sql`:

```sql
CREATE TABLE material_interpretations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    material_id UUID NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
    topic_id    UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    content     JSONB,
    model_used  TEXT NOT NULL DEFAULT 'gemini-2.0-flash',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_interpretations_material ON material_interpretations(material_id);
CREATE INDEX idx_interpretations_topic    ON material_interpretations(topic_id);
CREATE UNIQUE INDEX idx_interpretations_material_unique ON material_interpretations(material_id);
```

Restricción: **una interpretación por material** (índice único en `material_id`). Si se re-interpreta, se hace UPSERT.

### A.3 Endpoint: interpretar material

| Campo | Valor |
|-------|-------|
| **Método** | `POST` |
| **Path** | `/api/v1/courses/:course_id/topics/:topic_id/materials/:material_id/interpret` |
| **Auth** | Bearer JWT. Ownership: el curso debe pertenecer al `user_id` del JWT. |
| **Path params** | `course_id` (UUID), `topic_id` (UUID), `material_id` (UUID) |
| **Body** | Vacío. El material ya está en GCS (`materials.storage_url`). |

**Comportamiento:**

1. Verificar que el material existe, pertenece al topic/course y tiene `status = validated`.
2. Si ya existe una interpretación con `status = completed`, retornar 200 con la interpretación existente.
3. Si no existe o `status = failed`, crear/actualizar registro con `status = processing`.
4. **Si `FF_GEMINI_VISION = true`**: invocar Gemini 2.0 Flash vía Vertex AI:
   - Endpoint: `POST https://{GCP_REGION}-aiplatform.googleapis.com/v1/projects/{GCP_PROJECT_ID}/locations/{GCP_REGION}/publishers/google/models/gemini-2.0-flash:generateContent`
   - Body: `fileData` con GCS URI del material + `response_mime_type: "application/json"` + `response_schema` (sección A.1).
   - System instruction: prompt de extracción estructurada (indicar al modelo que extraiga texto, ecuaciones en LaTeX, descripciones de figuras).
5. **Si `FF_GEMINI_VISION = false`**: fallback al extractor actual (`PlainTextExtractor`) y generar un JSON con un solo bloque `text`.
6. Validar el JSON de respuesta contra el schema. Si falla validación, `status = failed` con error descriptivo.
7. Persistir en `material_interpretations` con `status = completed`.
8. Retornar la interpretación.

**Respuesta exitosa (200 / 201)**

```json
{
  "interpretation": {
    "id": "<uuid>",
    "material_id": "<uuid>",
    "status": "completed",
    "content": {
      "blocks": [
        { "index": 0, "block_type": "text", "content": "La integral definida..." },
        { "index": 1, "block_type": "equation", "content": "\\int_a^b f(x)\\,dx = F(b) - F(a)" },
        { "index": 2, "block_type": "figure", "content": "Gráfico de la función f(x) = x² mostrando el área bajo la curva entre x=1 y x=3" }
      ],
      "summary": "Apuntes de cálculo sobre integrales definidas.",
      "language": "es"
    },
    "model_used": "gemini-2.0-flash",
    "created_at": "2026-03-15T10:30:00Z",
    "updated_at": "2026-03-15T10:30:05Z"
  }
}
```

**Errores**

| Código | Condición |
|--------|-----------|
| 401 | JWT ausente o inválido |
| 403 | El curso no pertenece al usuario |
| 404 | Material, topic o course no encontrado |
| 409 | Interpretación en progreso (`status = processing`) |
| 422 | Material no está en estado `validated` |
| 500 | Error de Gemini, GCS o base de datos |

### A.4 Endpoint: obtener interpretación de un material

| Campo | Valor |
|-------|-------|
| **Método** | `GET` |
| **Path** | `/api/v1/courses/:course_id/topics/:topic_id/materials/:material_id/interpretation` |
| **Auth** | Bearer JWT + ownership |

**Respuesta exitosa (200):** Mismo formato que A.3.

**Errores:** 401, 403, 404 (material o interpretación no encontrada).

### A.5 Procesamiento asíncrono

Para PDFs grandes (>5 páginas), la interpretación puede tardar varios segundos. El flujo es:

1. `POST .../interpret` retorna **202 Accepted** con `status: "processing"` y el `id` de la interpretación.
2. El cliente hace polling con `GET .../interpretation` hasta que `status` sea `completed` o `failed`.
3. **Decisión en design:** si se prefiere notificación push en lugar de polling (ej. evento por WebSocket), documentar el mecanismo.

### A.6 Contrato Flutter — `material_review_screen`

Nueva pantalla: `mobile/lib/features/course/presentation/screens/material_review_screen.dart`.

**Navegación:** Desde `material_list_view.dart` o `material_summary_screen.dart`, al tocar un material con interpretación disponible → navegar a `/courses/:courseId/topics/:topicId/materials/:materialId/review`.

**Modelo de datos en Flutter:**

```dart
@freezed
class MaterialInterpretation with _$MaterialInterpretation {
  const factory MaterialInterpretation({
    required String id,
    required String materialId,
    required String status, // pending | processing | completed | failed
    required InterpretationContent? content,
    required String modelUsed,
    required DateTime createdAt,
  }) = _MaterialInterpretation;
}

@freezed
class InterpretationContent with _$InterpretationContent {
  const factory InterpretationContent({
    required List<InterpretationBlock> blocks,
    String? summary,
    String? language,
  }) = _InterpretationContent;
}

@freezed
class InterpretationBlock with _$InterpretationBlock {
  const factory InterpretationBlock({
    required int index,
    required String blockType, // text | equation | figure | transcription
    required String content,
    double? confidence,
    int? page,
  }) = _InterpretationBlock;
}
```

**Renderizado por tipo de bloque:**

| `block_type` | Widget | Librería |
|-------------|--------|----------|
| `text` | `MarkdownBody` | `flutter_markdown` (ya integrado) |
| `equation` | `Math.tex(content)` | `flutter_math_fork` (ya integrado) |
| `figure` | `Text` con estilo descriptivo (itálica, icono de imagen) | Nativo |
| `transcription` | `Text` con estilo de transcripción (comillas, icono de audio) | Nativo |

**Estados de la pantalla:**

- **Cargando**: indicador de progreso mientras `status = processing`.
- **Completado**: lista scrollable de bloques renderizados + resumen arriba.
- **Error**: mensaje con opción de reintentar (`POST .../interpret` de nuevo).
- **Sin interpretación**: botón "Interpretar material" que dispara el POST.

---

## Bloque B — Chat de corrección

### B.1 Modelo de dominio: `MaterialCorrection`

Nuevo struct en `backend/internal/core/domain/`:

```go
type MaterialCorrection struct {
    ID               string    `json:"id"`                // UUID
    MaterialID       string    `json:"material_id"`       // FK → materials.id
    InterpretationID string    `json:"interpretation_id"` // FK → material_interpretations.id
    BlockIndex       int       `json:"block_index"`       // Índice del bloque corregido en el array blocks[]
    OriginalContent  string    `json:"original_content"`  // Contenido original del bloque
    CorrectedContent string    `json:"corrected_content"` // Contenido corregido por el usuario
    CorrectedType    *string   `json:"corrected_type"`    // Si el tipo de bloque cambió (ej. text → equation), nullable
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

### B.2 Migración de base de datos

Nueva migración `000007_create_material_corrections.up.sql`:

```sql
CREATE TABLE material_corrections (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    material_id       UUID NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
    interpretation_id UUID NOT NULL REFERENCES material_interpretations(id) ON DELETE CASCADE,
    block_index       INTEGER NOT NULL,
    original_content  TEXT NOT NULL,
    corrected_content TEXT NOT NULL,
    corrected_type    TEXT, -- nullable; solo si cambió el block_type
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(interpretation_id, block_index)
);

CREATE INDEX idx_corrections_material ON material_corrections(material_id);
CREATE INDEX idx_corrections_interpretation ON material_corrections(interpretation_id);
```

Restricción: **una corrección por bloque por interpretación** (UNIQUE en `interpretation_id + block_index`). Al corregir de nuevo, se hace UPSERT.

### B.3 Endpoint: crear o actualizar corrección

| Campo | Valor |
|-------|-------|
| **Método** | `POST` |
| **Path** | `/api/v1/courses/:course_id/topics/:topic_id/materials/:material_id/corrections` |
| **Auth** | Bearer JWT + ownership |

**Body (JSON):**

```json
{
  "block_index": 1,
  "corrected_content": "\\int_a^b f(x)\\,dx = F(b) - F(a)",
  "corrected_type": "equation"
}
```

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `block_index` | int | Sí | Índice del bloque en `interpretation.content.blocks[]` |
| `corrected_content` | string | Sí | Contenido corregido (texto plano, LaTeX, etc.) |
| `corrected_type` | string | No | Nuevo `block_type` si el usuario cambió el tipo (ej. "text" → "equation") |

**Comportamiento:**

1. Verificar que existe una interpretación `completed` para el material.
2. Verificar que `block_index` es válido (dentro del rango de `blocks[]`).
3. Si ya existe corrección para ese `block_index`, hacer UPDATE (UPSERT).
4. Si no existe, INSERT con el `original_content` tomado de la interpretación.
5. **Inyección en RAG** (decisión `re-embed` vs `merge en retrieval`):
   - **Opción A — Re-embed:** Al guardar la corrección, regenerar el chunk asociado al bloque corregido y actualizar su embedding en `material_chunks`. El texto del chunk cambia del original al corregido.
   - **Opción B — Merge en retrieval:** Al montar el contexto para el tutor (`RAGUseCase.GetTopicContext`), después del similarity search, recorrer los chunks devueltos y reemplazar el contenido con el override si existe una corrección.
   - **Decisión en design:** cuál opción adoptar. La spec soporta ambas.

**Respuesta exitosa (200 / 201)**

```json
{
  "correction": {
    "id": "<uuid>",
    "material_id": "<uuid>",
    "interpretation_id": "<uuid>",
    "block_index": 1,
    "original_content": "∫ab f(x)dx = F(b) – F(a)",
    "corrected_content": "\\int_a^b f(x)\\,dx = F(b) - F(a)",
    "corrected_type": "equation",
    "created_at": "2026-03-15T11:00:00Z"
  }
}
```

**Errores**

| Código | Condición |
|--------|-----------|
| 400 | `block_index` fuera de rango o `corrected_content` vacío |
| 404 | Material o interpretación no encontrada |
| 422 | Interpretación no está en estado `completed` |

### B.4 Endpoint: listar correcciones de un material

| Campo | Valor |
|-------|-------|
| **Método** | `GET` |
| **Path** | `/api/v1/courses/:course_id/topics/:topic_id/materials/:material_id/corrections` |
| **Auth** | Bearer JWT + ownership |

**Respuesta exitosa (200)**

```json
{
  "corrections": [
    {
      "id": "<uuid>",
      "block_index": 1,
      "original_content": "...",
      "corrected_content": "...",
      "corrected_type": "equation",
      "created_at": "2026-03-15T11:00:00Z"
    }
  ]
}
```

### B.5 Endpoint: eliminar corrección

| Campo | Valor |
|-------|-------|
| **Método** | `DELETE` |
| **Path** | `/api/v1/courses/:course_id/topics/:topic_id/materials/:material_id/corrections/:correction_id` |
| **Auth** | Bearer JWT + ownership |

**Respuesta:** 204 No Content. Si se usa re-embed, restaurar el chunk original y re-generar embedding.

### B.6 Contrato Flutter — Chat de corrección

Embebido en `material_review_screen` como un bottom sheet o panel lateral.

**Flujo de interacción:**

1. El usuario toca un bloque de la interpretación.
2. Se abre un panel/modal con el contenido original del bloque.
3. El usuario edita el contenido (campo de texto editable).
4. Opcionalmente cambia el tipo de bloque (dropdown: text/equation/figure/transcription).
5. Al pulsar "Guardar corrección" → `POST .../corrections`.
6. El bloque en la lista se marca visualmente como "corregido" (borde o badge).

**Modelo de datos Flutter:**

```dart
@freezed
class MaterialCorrection with _$MaterialCorrection {
  const factory MaterialCorrection({
    required String id,
    required String materialId,
    required int blockIndex,
    required String originalContent,
    required String correctedContent,
    String? correctedType,
    required DateTime createdAt,
  }) = _MaterialCorrection;
}
```

---

## Bloque C — Perfil de aprendizaje invisible

### C.1 Estructura del JSONB `learning_profile`

El campo `learning_profile` ya existe en la tabla `users` (migración `000001`). Su contenido JSONB sigue esta estructura:

```json
{
  "style": {
    "visual":   0.0,
    "auditory": 0.0,
    "reading":  0.0
  },
  "difficult_topics": [
    {
      "topic_name": "Integrales por partes",
      "course_id": "<uuid>",
      "confusion_count": 3,
      "last_seen": "2026-03-15T10:00:00Z"
    }
  ],
  "total_minutes": 0,
  "sessions_count": 0,
  "last_updated": "2026-03-15T10:00:00Z"
}
```

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `style.visual` | float (0.0–1.0) | Peso del estilo visual (inferido del tipo de preguntas y materiales) |
| `style.auditory` | float (0.0–1.0) | Peso del estilo auditivo |
| `style.reading` | float (0.0–1.0) | Peso del estilo de lectura |
| `difficult_topics` | array | Lista de conceptos donde el estudiante mostró confusión recurrente |
| `difficult_topics[].topic_name` | string | Nombre del concepto (inferido por Gemini) |
| `difficult_topics[].confusion_count` | int | Veces que se detectó confusión en ese tema |
| `total_minutes` | int | Minutos acumulados de tutoría |
| `sessions_count` | int | Número total de sesiones completadas |
| `last_updated` | string (ISO 8601) | Última actualización del perfil |

> **Nota:** La taxonomía exacta de estilos y las dimensiones pueden refinarse en design. La spec fija los 3 ejes mínimos.

### C.2 Endpoint: actualizar perfil de aprendizaje

| Campo | Valor |
|-------|-------|
| **Método** | `PATCH` |
| **Path** | `/api/v1/users/me/learning-profile` |
| **Auth** | Bearer JWT (el `user_id` se extrae del token) |

**Body (JSON):**

```json
{
  "session_summary": {
    "course_id": "<uuid>",
    "topic_id": "<uuid>",
    "duration_minutes": 25,
    "message_count": 42,
    "transcript_excerpt": "Últimos 2000 tokens de la conversación..."
  }
}
```

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `session_summary.course_id` | UUID | Sí | Curso de la sesión |
| `session_summary.topic_id` | UUID | No | Tema específico (null si fue sesión a nivel curso) |
| `session_summary.duration_minutes` | int | Sí | Duración de la sesión en minutos |
| `session_summary.message_count` | int | Sí | Número de intercambios en la sesión |
| `session_summary.transcript_excerpt` | string | Sí | Extracto de la conversación (limitado a max N tokens) |

**Comportamiento:**

1. Recibir el resumen de sesión.
2. **Si `FF_LEARNING_PROFILE = true`**: enviar a Gemini (2.0 Flash, no Live) un prompt de análisis:
   - Input: el `transcript_excerpt` + el `learning_profile` actual del usuario.
   - Instrucción: "Analiza esta conversación de tutoría y actualiza el perfil de aprendizaje. Devuelve el perfil actualizado en formato JSON."
   - `response_schema`: mismo schema que la estructura de `learning_profile`.
3. Fusionar la respuesta de Gemini con el perfil existente:
   - `style`: promedio ponderado con el histórico (peso mayor al histórico para estabilidad).
   - `difficult_topics`: agregar nuevos o incrementar `confusion_count` de existentes.
   - `total_minutes`: sumar `duration_minutes`.
   - `sessions_count`: incrementar en 1.
4. Persistir el JSONB actualizado en `users.learning_profile`.
5. **Si `FF_LEARNING_PROFILE = false`**: solo actualizar `total_minutes` y `sessions_count` (sin invocación a Gemini).

**Mitigación de tokens (del proposal):**

- **Límite de `transcript_excerpt`:** El cliente envía como máximo los últimos **2000 tokens** (~1500 palabras) de la conversación. Si la sesión es más larga, enviar un resumen pre-agregado del cliente: `"tema: {topic}, preguntas: {n}, duración: {min}"`.
- **Resumen incremental (decisión en design):** Opcionalmente, el backend puede actualizar el perfil cada N mensajes en lugar de solo al cierre. Si se adopta, se necesita un endpoint adicional o un mecanismo interno. N sugerido: 10–15 intercambios.

**Respuesta exitosa (200)**

```json
{
  "learning_profile": {
    "style": { "visual": 0.6, "auditory": 0.3, "reading": 0.1 },
    "difficult_topics": [...],
    "total_minutes": 150,
    "sessions_count": 7,
    "last_updated": "2026-03-15T12:00:00Z"
  }
}
```

**Errores**

| Código | Condición |
|--------|-----------|
| 400 | Body inválido o `duration_minutes` <= 0 |
| 401 | JWT ausente o inválido |
| 500 | Error de Gemini o base de datos |

### C.3 Inyección del perfil en el tutor

Al construir el system instruction de Gemini Live (en `gemini_live_service.dart` o en el backend si hace proxy), concatenar el perfil del usuario como contexto adicional:

```
[Perfil del estudiante]
Estilo de aprendizaje: visual (60%), auditivo (30%), lectura (10%).
Temas difíciles: integrales por partes (3 confusiones), límites al infinito (2 confusiones).
Sesiones acumuladas: 7, tiempo total: 150 min.
Adapta tus explicaciones priorizando ejemplos visuales y analogías gráficas.
```

La construcción de este texto a partir del JSONB es responsabilidad del backend (nuevo método en el use case) o del cliente al enviar `clientContent`. **Decisión en design.**

---

## Bloque D — Exportación PDF

### D.1 Generación en Flutter (lado cliente)

No se requiere endpoint de backend. La generación se hace localmente en el dispositivo.

**Flujo:**

1. Desde `material_review_screen`, el usuario pulsa botón "Exportar PDF".
2. La app recopila:
   - Bloques de `MaterialInterpretation.content.blocks[]`.
   - Correcciones (`MaterialCorrection[]`) para superponer sobre los bloques originales.
   - Resumen (`summary`) de la interpretación.
3. Genera un PDF con la librería `pdf` de Flutter.
4. Guarda el PDF en un directorio temporal (`getTemporaryDirectory()`).
5. Lanza la hoja de compartir nativa con `share_plus` → `Share.shareXFiles([XFile(path)])`.

**Contenido del PDF:**

| Sección | Contenido |
|---------|-----------|
| **Encabezado** | Nombre del material, nombre del curso, fecha de generación |
| **Resumen** | `interpretation.content.summary` |
| **Bloques** | Cada bloque renderizado según tipo (ver tabla abajo) |
| **Correcciones** | Bloques corregidos marcados con indicador visual (ej. texto tachado + corrección) |

**Renderizado de bloques en PDF:**

| `block_type` | Renderizado en PDF |
|-------------|-------------------|
| `text` | Texto con formato (párrafos) |
| `equation` | **Decisión en design:** renderizar LaTeX a imagen vía `flutter_math_fork` + `RepaintBoundary.toImage()` y embeber como imagen en el PDF, o usar un engine de renderizado LaTeX en Dart. |
| `figure` | Descripción textual en itálica (no se embebe imagen original en MVP) |
| `transcription` | Texto en bloque con estilo de cita |

### D.2 Permisos

**No se requieren permisos de almacenamiento.** El uso de `share_plus` con un archivo temporal evita las restricciones de Android 13+ e iOS 15+. El usuario decide el destino (Drive, WhatsApp, Archivos, etc.) desde la hoja de compartir nativa.

### D.3 Dependencia Flutter

Agregar a `pubspec.yaml`:

```yaml
pdf: ^3.11.0       # Generación de PDF
share_plus: ^10.0.0 # Share sheet nativa
```

Versiones exactas a confirmar en implementación con las últimas disponibles.

---

## Bloque E — Experiencia audiovisual

### E.1 Avatar animado con lip-sync (Rive)

#### Contrato del archivo `.riv`

| Propiedad | Valor |
|-----------|-------|
| **Artboard** | `TutorAvatar` (o nombre definido por diseño) |
| **State Machine** | `MainStateMachine` con estados: `idle`, `speaking`, `listening`, `thinking`, `reconnecting` |
| **Data binding** | Propiedad numérica `mouthOpen` (rango 0.0–1.0) vinculada via ViewModel |
| **Triggers** | `onBargeIn` → transición a `listening` (mano a la oreja); `onReconnecting` → transición a `thinking` |

#### Widget Flutter

Nuevo archivo: `mobile/lib/features/tutor/presentation/widgets/tutor_avatar_widget.dart`.

```dart
class TutorAvatarWidget extends StatefulWidget {
  final Stream<double> amplitudeStream; // 0.0–1.0 normalizado
  final TutorAvatarState avatarState;   // idle | speaking | listening | thinking | reconnecting
}
```

**Pipeline de amplitud RMS:**

1. `gemini_live_service.audioOutputStream` emite chunks PCM (bytes).
2. Calcular amplitud RMS del chunk: `sqrt(sum(sample²) / n)` sobre las muestras int16.
3. Normalizar al rango 0.0–1.0 (dividir por 32768 u otro valor de pico calibrado).
4. Alimentar `mouthOpen` del ViewModel de Rive a 15–30 fps.

> **Decisión en design:** Librería exacta para el cálculo de amplitud y frecuencia de actualización del parámetro Rive. Opciones: procesamiento directo en Dart sobre los bytes PCM, o uso de un paquete nativo de análisis de audio.

#### Feature flag

Si `FF_AVATAR = false`, la pantalla de tutoría muestra la UI actual (solo audio/transcripción, sin widget Rive).

### E.2 Barge-in / Interrupción natural

#### Flujo de barge-in

```
[Estudiante habla] → VAD detecta actividad vocal
  → (1) Feedback visual INMEDIATO: avatar → estado "listening" (mano a oreja) + ícono micrófono resaltado
  → (2) AudioPlayer.stop() — detener reproducción del tutor
  → (3) Enviar señal al WebSocket para cortar turno del servidor
  → (4) Comenzar a enviar audio del estudiante como realtimeInput
```

**Latencia del feedback visual:** < 100 ms desde la detección del VAD. El feedback es local (no depende de la red).

#### VAD local

| Aspecto | Contrato |
|---------|----------|
| **Activación** | Solo cuando `sessionState == speaking` (el tutor está hablando) |
| **Detección** | Umbral configurable de amplitud RMS sobre el stream del micrófono |
| **Duración mínima** | 200–300 ms de actividad vocal continua para evitar falsos positivos |
| **Anti-ruido** | Threshold calibrado en design; opcionalmente `flutter_silero_vad` para mayor precisión |
| **Feature flag** | Si `FF_BARGE_IN = false`, el estudiante debe esperar a que el tutor termine (`turnComplete`) |

> **Decisión en design:** Paquete de VAD (`flutter_silero_vad`, threshold simple, o solución nativa). Umbral de sensibilidad y estrategia anti-ruido ambiente.

#### Señal WebSocket de corte de turno

Al activarse el barge-in, el cliente envía por el WebSocket un mensaje que indica fin del stream de audio del servidor e inicio de un nuevo turno del cliente. El formato exacto depende de la API de Gemini Live:

- Detener el envío de audio del servidor (el cliente deja de reproducir lo que le llega).
- Empezar a enviar `realtimeInput.mediaChunks` con el audio del estudiante.
- La API de Gemini Live interpreta el nuevo input como interrupción natural y cambia el turno.

> **Nota:** Si la API requiere un mensaje explícito de `audioStreamEnd` o similar, documentar en design.

#### Buffer de audio y AEC

| Problema | Mitigación |
|----------|-----------|
| Buffer residual tras `player.stop()` (200–500 ms de audio del tutor sigue sonando) | Usar reproductor de **baja latencia** que limpie el buffer al detener. **Decisión en design:** evaluar `just_audio` con `AudioPlayer.stop()` + `AudioPlayer.seek(Duration.zero)`, o solución nativa con AudioTrack (Android) / AVAudioEngine (iOS). |
| Audio residual captado por micrófono abierto → retroalimentación a Gemini | Aplicar **AEC (Acoustic Echo Cancellation)** en la captura del micrófono. En Android: `AudioRecord` con `VOICE_COMMUNICATION` como fuente de audio activa AEC por hardware. En iOS: `AVAudioSession` con categoría `.playAndRecord` y modo `.voiceChat`. **Decisión en implementación:** verificar disponibilidad por plataforma y paquete de grabación. |

### E.3 Fondos dinámicos

#### Function calling de Gemini Live

Declarar una tool en la configuración de setup del WebSocket:

```json
{
  "tools": [{
    "functionDeclarations": [{
      "name": "change_background",
      "description": "Cambia el fondo visual de la pantalla del tutor según el tema de conversación",
      "parameters": {
        "type": "object",
        "properties": {
          "context_type": {
            "type": "string",
            "enum": ["math", "science", "history", "default"],
            "description": "Tipo de contexto temático para el fondo"
          }
        },
        "required": ["context_type"]
      }
    }]
  }]
}
```

**Manejo en Flutter:**

1. Al recibir una `toolCall` en el stream del WebSocket con `name = "change_background"`, extraer `context_type`.
2. Mapear `context_type` a un asset de fondo: `assets/backgrounds/{context_type}.png` (o `.riv`).
3. Transición animada (fade de 300 ms) al nuevo fondo.
4. Enviar `toolResponse` confirmando la ejecución.

**Assets:** 4 fondos estáticos pre-diseñados (math, science, history, default).

**Feature flag:** Si `FF_DYNAMIC_BACKGROUNDS = false`, fondo estático `default` siempre.

### E.4 Snapshot de cámara

#### Flujo

1. Botón de cámara visible en la pantalla de tutoría (icono de cámara en toolbar o FAB secundario).
2. Al pulsar → abrir cámara trasera con `camera` o `image_picker`.
3. Capturar foto → comprimir a JPEG (calidad 80%, max 1024px de lado largo).
4. Enviar como `inlineData` en el siguiente mensaje del WebSocket:

```json
{
  "clientContent": {
    "turns": [{
      "role": "user",
      "parts": [{
        "inlineData": {
          "mimeType": "image/jpeg",
          "data": "<base64>"
        }
      }, {
        "text": "Mira mis apuntes y explícame lo que ves"
      }]
    }],
    "turnComplete": true
  }
}
```

5. El tutor responde contextualmente sobre la imagen.

**Permisos requeridos:**

| Plataforma | Permiso | Rationale (texto para la tienda) |
|-----------|---------|----------------------------------|
| Android | `android.permission.CAMERA` | "Klyra necesita la cámara para que puedas mostrar tus apuntes escritos a mano al tutor interactivo y recibir ayuda visual inmediata." |
| iOS | `NSCameraUsageDescription` | "Klyra necesita la cámara para que puedas mostrar tus apuntes escritos a mano al tutor interactivo y recibir ayuda visual inmediata." |

**Feature flag:** Si `FF_CAMERA_SNAPSHOT = false`, el botón de cámara no se muestra.

### E.5 Permisos consolidados (AndroidManifest e Info.plist)

#### `android/app/src/main/AndroidManifest.xml`

```xml
<!-- Ya existente -->
<uses-permission android:name="android.permission.RECORD_AUDIO"/>
<uses-permission android:name="android.permission.INTERNET"/>

<!-- Nuevo para Sprint 8 -->
<uses-permission android:name="android.permission.CAMERA"/>
```

**Rationales exactos (para Google Play):**

| Permiso | Rationale |
|---------|-----------|
| `RECORD_AUDIO` | "Klyra necesita el micrófono para que el estudiante pueda hablar con el tutor interactivo de inteligencia artificial en sesiones de estudio por voz en tiempo real." |
| `CAMERA` | "Klyra necesita la cámara para que el estudiante pueda mostrar sus apuntes escritos a mano al tutor interactivo y recibir ayuda visual inmediata." |

#### `ios/Runner/Info.plist`

```xml
<!-- Ya existente -->
<key>NSMicrophoneUsageDescription</key>
<string>Klyra necesita el micrófono para que puedas hablar con tu tutor interactivo de inteligencia artificial en sesiones de estudio por voz en tiempo real.</string>

<!-- Nuevo para Sprint 8 -->
<key>NSCameraUsageDescription</key>
<string>Klyra necesita la cámara para que puedas mostrar tus apuntes escritos a mano al tutor interactivo y recibir ayuda visual inmediata.</string>
```

---

## Feature flags y configuración

### Variables de entorno (backend)

| Variable | Tipo | Default | Bloque | Descripción |
|----------|------|---------|--------|-------------|
| `FF_GEMINI_VISION` | bool | `false` | A | Usar Gemini 2.0 Flash para interpretación. Si `false`, fallback a OCR/extractor actual. |
| `FF_CORRECTIONS` | bool | `false` | B | Habilitar endpoints de correcciones y lógica de override en RAG. |
| `FF_LEARNING_PROFILE` | bool | `false` | C | Habilitar análisis de sesión con Gemini para actualizar perfil. Si `false`, solo contadores. |
| `FF_AVATAR` | bool | `false` | E1 | Habilitar widget de avatar Rive en el cliente (flag expuesto vía `/config`). |
| `FF_BARGE_IN` | bool | `false` | E2 | Habilitar VAD y lógica de barge-in en el cliente (flag expuesto vía `/config`). |
| `FF_DYNAMIC_BACKGROUNDS` | bool | `false` | E3 | Habilitar fondos dinámicos (flag expuesto vía `/config`). |
| `FF_CAMERA_SNAPSHOT` | bool | `false` | E4 | Habilitar botón de snapshot de cámara (flag expuesto vía `/config`). |
| `FF_PDF_EXPORT` | bool | `false` | D | Habilitar exportación PDF (flag expuesto vía `/config`). |

Lectura: `os.Getenv("FF_GEMINI_VISION") == "true"`.

### Endpoint: configuración de cliente

| Campo | Valor |
|-------|-------|
| **Método** | `GET` |
| **Path** | `/api/v1/config` |
| **Auth** | Bearer JWT |

**Respuesta (200):**

```json
{
  "features": {
    "gemini_vision": true,
    "corrections": true,
    "learning_profile": true,
    "avatar": true,
    "barge_in": true,
    "dynamic_backgrounds": false,
    "camera_snapshot": false,
    "pdf_export": true
  }
}
```

El handler lee las variables de entorno `FF_*` y las expone como JSON. Flutter consulta este endpoint al inicio de sesión y cachea los flags localmente.

---

## Contratos de Gemini Live (WebSocket)

### Formato de audio

| Dirección | Formato | MIME | Notas |
|-----------|---------|------|-------|
| Cliente → Gemini | PCM mono 16 kHz, 16 bits signed LE | `audio/pcm;rate=16000` | **Obligatorio.** Los micrófonos de smartphones graban a 44.1/48 kHz por defecto. |
| Gemini → Cliente | PCM mono 24 kHz, 16 bits signed LE | `audio/pcm;rate=24000` | Según API actual; verificar en implementación. |

### Downsampling

Si el paquete de grabación (`record`) no permite forzar `sampleRate: 16000` en el hardware (verificar por plataforma), se debe implementar un **downsampler** antes de enviar:

- **Opción A (preferida):** Configurar `RecordConfig(sampleRate: 16000)` — el paquete `record` ya lo soporta y lo usa actualmente. Verificar que funcione en todos los dispositivos objetivo.
- **Opción B (fallback):** Grabar a la tasa nativa del hardware y aplicar downsampling en Dart: filtro anti-aliasing + decimación (ej. de 48 kHz a 16 kHz = factor 3). Impacto en CPU a evaluar.

> **Decisión en design/implementación:** Confirmar que `record: ^6.0.0` con `sampleRate: 16000` funciona sin artefactos en Android e iOS. Si no, diseñar el downsampler.

### Reconexión WebSocket

| Aspecto | Contrato |
|---------|----------|
| **Detección de desconexión** | `WebSocketChannel.stream.done` o `closeCode != null` |
| **Backoff exponencial** | Reintentos a 1s, 2s, 5s, 10s, 30s (max). Máximo 5 reintentos. |
| **Estado UI durante reconexión** | Avatar → estado `reconnecting` (animación de "pensando"). Deshabilitar input de voz. Mostrar indicador visual de reconexión. |
| **Preservación de contexto** | Al reconectar, reenviar el `clientContent` con el contexto RAG y el historial reciente de turnos (almacenado en memoria en `TutorSessionController`). |
| **Session resumption** | Si la API de Gemini Live soporta `session_id` para reanudar, utilizarlo. **Decisión en implementación.** |
| **Fallo definitivo** | Tras agotar reintentos, mostrar error y opción de "Reconectar manualmente" o "Volver al curso". |

### Mensajes WebSocket (resumen actualizado para Sprint 8)

| Tipo | Dirección | Cambio en Sprint 8 |
|------|-----------|---------------------|
| `setup` | C→S | Añadir `tools` con declaración de `change_background` (E3) |
| `realtimeInput.mediaChunks` | C→S | Sin cambios (PCM 16 kHz base64) |
| `clientContent` | C→S | Añadir `inlineData` para snapshot de cámara (E4) |
| `serverContent.modelTurn.parts[].inlineData` | S→C | Audio PCM del tutor (sin cambios) |
| `serverContent.modelTurn.parts[].text` | S→C | Transcripción (sin cambios) |
| `serverContent.modelTurn.parts[].functionCall` | S→C | **Nuevo:** llamada a `change_background` (E3) |
| `toolResponse` | C→S | **Nuevo:** confirmación de ejecución de `change_background` |
| `turnComplete` | S→C | Sin cambios |

---

## Criterios de aceptación

### Bloque A — Interpretación de materiales

| # | Criterio | Verificación |
|---|----------|--------------|
| A1 | Al subir un PDF/imagen, el estudiante puede disparar la interpretación con un botón en la UI. | Botón "Interpretar" visible en `material_review_screen`. POST retorna 200/201/202. |
| A2 | La interpretación devuelve JSON estructurado con bloques de tipo `text`, `equation`, `figure` y/o `transcription`. | Validar schema del JSON de respuesta. |
| A3 | Las ecuaciones LaTeX se renderizan correctamente en la pantalla de interpretación. | `flutter_math_fork` muestra la ecuación sin errores de parsing. |
| A4 | Las figuras se muestran como descripción semántica textual. | Bloque `figure` renderizado en itálica con icono. |
| A5 | Si `FF_GEMINI_VISION = false`, la interpretación usa el extractor actual y devuelve un solo bloque `text`. | Verificar fallback con flag desactivado. |
| A6 | PDFs grandes (>5 páginas) no bloquean la UI; se muestra indicador de progreso. | Verificar procesamiento asíncrono con polling o notificación. |

### Bloque B — Chat de corrección

| # | Criterio | Verificación |
|---|----------|--------------|
| B1 | El estudiante puede tocar un bloque de la interpretación y editar su contenido. | Panel/modal de edición se abre con contenido pre-llenado. |
| B2 | Las correcciones se persisten en `material_corrections` y se pueden listar por material. | `GET .../corrections` retorna las correcciones guardadas. |
| B3 | Los bloques corregidos se marcan visualmente como "corregido" en la pantalla. | Badge o borde visual diferenciado. |
| B4 | Las correcciones se inyectan en el contexto del tutor (re-embed o merge). | En una sesión de tutoría posterior, el contexto incluye el texto corregido en lugar del original. |
| B5 | El estudiante puede eliminar una corrección y restaurar el contenido original. | `DELETE .../corrections/:id` retorna 204. El bloque vuelve al original. |

### Bloque C — Perfil de aprendizaje

| # | Criterio | Verificación |
|---|----------|--------------|
| C1 | Al cerrar una sesión de tutoría, el cliente envía `PATCH /users/me/learning-profile` con el resumen. | Verificar llamada HTTP al cerrar sesión. |
| C2 | Tras ≥3 sesiones, `learning_profile` contiene `style`, `difficult_topics`, `total_minutes` y `sessions_count` actualizados. | Consultar campo JSONB del usuario en la DB. |
| C3 | El extracto de transcripción enviado no excede 2000 tokens. | Verificar truncamiento en el cliente. |
| C4 | Si `FF_LEARNING_PROFILE = false`, solo se actualizan contadores (`total_minutes`, `sessions_count`) sin invocar a Gemini. | Verificar que no hay llamada a Gemini con flag desactivado. |
| C5 | El perfil del estudiante se inyecta como contexto adicional en el system instruction del tutor. | Verificar que el system prompt incluye datos del perfil. |

### Bloque D — Exportación PDF

| # | Criterio | Verificación |
|---|----------|--------------|
| D1 | El estudiante puede exportar un PDF desde la pantalla de interpretación. | Botón "Exportar PDF" visible y funcional. |
| D2 | El PDF contiene: encabezado (material, curso, fecha), resumen, bloques interpretados y correcciones marcadas. | Abrir el PDF generado y verificar contenido. |
| D3 | El PDF se genera en directorio temporal y se lanza con `share_plus` (hoja de compartir nativa). | No se piden permisos de almacenamiento; el usuario elige destino. |
| D4 | No se requieren permisos de almacenamiento en Android 13+ ni iOS 15+. | Verificar que la app no solicita `WRITE_EXTERNAL_STORAGE`. |

### Bloque E — Experiencia audiovisual

| # | Criterio | Verificación |
|---|----------|--------------|
| E1 | El avatar Rive se muestra en la pantalla de tutoría con lip-sync sincronizado al audio del tutor. | La boca del avatar se mueve en tiempo real con la amplitud del audio. |
| E2 | El avatar tiene estados visuales diferenciados: idle, speaking, listening, thinking, reconnecting. | Verificar transiciones de state machine al cambiar de estado. |
| E3 | Al detectar barge-in (VAD), la UI muestra feedback visual inmediato (< 100 ms): avatar escuchando y/o ícono de micrófono resaltado. | Medir latencia entre detección VAD y cambio visual. |
| E4 | Tras barge-in, el audio del tutor se detiene en < 500 ms. | Medir desde detección VAD hasta silencio del altavoz. |
| E5 | El VAD requiere ≥200 ms de actividad vocal continua para evitar falsos positivos por ruido ambiente. | Probar con ruidos cortos (< 200 ms); no deben disparar barge-in. |
| E6 | Los fondos cambian según el contexto temático cuando Gemini emite `change_background`. | Probar con conversación de matemáticas → fondo "math"; cambiar a historia → fondo "history". |
| E7 | El estudiante puede capturar una foto y el tutor responde contextualmente sobre la imagen. | Enviar foto de apuntes escritos a mano y verificar respuesta relevante. |
| E8 | Los rationales de permisos (`RECORD_AUDIO`, `CAMERA`) están correctamente configurados en AndroidManifest e Info.plist. | Inspeccionar archivos de configuración. |
| E9 | Audio del micrófono se envía en PCM mono 16 kHz. | Verificar formato con herramienta de análisis de audio o log del WebSocket. |
| E10 | Al perder conexión WebSocket, la UI muestra estado de reconexión y reintenta con backoff exponencial. | Simular desconexión de red y verificar comportamiento. |
| E11 | Tras reconexión exitosa, el contexto de la conversación se preserva. | Verificar que el tutor recuerda el tema de la conversación tras reconectar. |

---

## Pendientes y decisiones diferidas

| # | Pendiente | Bloque | Resolución |
|---|-----------|--------|------------|
| P1 | Estructura exacta del `response_schema` de interpretación (campos, anidamiento, bloques mixtos) | A | Definido en design. La spec fija el contrato mínimo (sección A.1). |
| P2 | Re-embed vs merge en retrieval para correcciones | B | Definido en design. La spec soporta ambas opciones. |
| P3 | Taxonomía completa del `learning_profile` (ejes adicionales, rangos, pesos) | C | Definido en design. La spec fija 3 ejes mínimos. |
| P4 | Resumen incremental (cada N mensajes) vs solo al cierre de sesión | C | Definido en design. La spec describe ambas opciones y sugiere N=10–15. |
| P5 | Renderizado de LaTeX a imagen para embeber en PDF | D | Definido en design. Opciones: `RepaintBoundary.toImage()` en Flutter o engine server-side. |
| P6 | Librería de VAD en Flutter (`flutter_silero_vad`, threshold simple, solución nativa) | E2 | Definido en design. |
| P7 | Librería de audio de baja latencia para reproducción y stop limpio | E2 | Definido en design. Opciones: `just_audio`, `audioplayers` con flush, solución nativa. |
| P8 | AEC (Acoustic Echo Cancellation) por plataforma | E2 | Decisión en implementación. Depende de disponibilidad por paquete de grabación. |
| P9 | Pipeline de amplitud RMS para lip-sync (librería, frecuencia de actualización) | E1 | Definido en design. |
| P10 | Confirmación de `record: ^6.0.0` con `sampleRate: 16000` en todos los dispositivos | E | Decisión en implementación. Fallback: downsampler manual en Dart. |
| P11 | Session resumption en Gemini Live (soporte de `session_id` o equivalente) | E | Decisión en implementación. Depende de la API. |
| P12 | Procesamiento asíncrono de interpretación: polling vs notificación push | A | Definido en design. La spec describe polling como baseline. |
| P13 | Modelo de datos del avatar Rive (nombre del artboard, parámetros, estados) | E1 | Definido en design, coordinado con equipo de diseño del .riv. |
| P14 | Versión exacta del modelo Gemini Live en producción (`gemini-live-2.5-flash-preview` u otro) | E | Decisión en implementación según disponibilidad. |
