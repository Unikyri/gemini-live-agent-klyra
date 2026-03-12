## Cambio archivado: klyra-sprint8-mvp-extendido

**Versión**: v0.6  
**Fecha de archivo**: 2026-03-11

### Resumen de alcance
- **Bloques A+B (FASE 1)**: interpretación estructurada de materiales con Gemini 2.0 Flash y chat de corrección con overrides aplicados en retrieval (RAG) sin re-embed.
- **Bloques C+D (FASE 2)**: perfil de aprendizaje invisible actualizado vía Gemini a partir de la transcripción y exportación local a PDF en Flutter con `pdf` + `share_plus`.
- **Bloque E (FASE 3)**: experiencia audiovisual con avatar (Rive), VAD local y barge-in, fondos dinámicos por function calling y snapshot de cámara integrado con Gemini Live.

### Estado de verificación
- **FASE 1**: PASS (con warnings documentados sobre población de `interpretation_json` y mapeo robusto de `chunk_id`).
- **FASE 2**: PASS (perfil de aprendizaje y exportación PDF validados con tests; ver detalles en `verify-report.md`).
- **FASE 3**: PASS (con warnings sobre falta de tests automatizados específicos de Bloque E y limitaciones de la suite TestSprite MCP).

### Notas
- El archivo `tasks.md` mantiene tareas `[ ]` para trabajo futuro (feature flags completos, QA ampliado, assets definitivos de diseño, etc.) que no bloquean el alcance de la versión **v0.6**.
- Warnings y desviaciones menores quedan registrados en `verify-report.md` como referencia para iteraciones posteriores.

