## Change Archived

**Change**: `tutor-course-ux-refactor`  
**Archived on**: 2026-03-11  
**Location**: `openspec/changes/tutor-course-ux-refactor/`

### Resumen

Refactor UX del tutor por curso y robustecimiento de upload: (1) validación MIME/extensión "extensión primero" para eliminar falsos 415, (2) botón global de tutor en detalle de curso y eliminación del botón "Review Summary & Start" por topic, (3) modo zero-material en sesión del tutor con contexto mínimo cuando no hay materiales, (4) endpoints RAG con `has_materials` y `message`; (5) MaterialSummaryScreen deja de ser gate bloqueante.

### Artifacts

| Artifact | Enlace |
|----------|--------|
| Proposal | [proposal.md](./proposal.md) |
| Spec | [spec.md](./spec.md) |
| Design | [design.md](./design.md) |
| Tasks | [tasks.md](./tasks.md) |
| Verify report | [verify-report.md](./verify-report.md) |

### Verificaciones manuales pendientes

Quedan tareas en `tasks.md` sin ejecutar en esta iteración:

- **B3.1**: Reforzar filtro `deleted_at IS NULL` en `GetMaterialsByTopic` (opcional/consistencia).
- **T3.1** – **T3.5**: Smoke tests E2E/manuales (upload con variantes MIME desde Android, flujo tutor sin/con materiales, MaterialSummaryScreen no bloqueante, endpoints de contexto desde Postman).

Cuando se completen, actualizar `tasks.md` marcando esas tareas como realizadas.

### Notas post-archivo

- Verdict de verificación: **PASS WITH WARNINGS**. No hay issues CRITICAL; suites `go test ./...` y `flutter test` en verde.
- Coberturas pendientes no bloqueantes: test explícito del banner zero-material en `TutorSessionScreen` (ZM-AC7), aserción directa de logs `warn` en discrepancias MIME (UPL-AC6).
