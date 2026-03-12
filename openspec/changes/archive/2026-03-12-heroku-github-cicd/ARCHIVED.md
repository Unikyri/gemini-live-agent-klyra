## Cambio archivado: heroku-github-cicd

**Versión**: v0.6.1  
**Fecha de archivo**: 2026-03-12

### Resumen de alcance
- **CI backend (GitHub Actions)**: se añadió workflow `Backend CI` para ejecutar tests y validaciones del backend en PRs y pushes a `main` cuando hay cambios en `backend/**`.
- **Deploy automático a Heroku (Container Registry)**: se añadió workflow `Deploy to Heroku` que, tras CI exitoso en `main`, construye y publica las imágenes `web` y `release` en `registry.heroku.com` y ejecuta `heroku container:release`.
- **Release phase (migraciones)**: se añadió `backend/Dockerfile.release` para ejecutar migraciones vía `RUN_MIGRATIONS_ONLY=true` antes de enrutar tráfico al nuevo `web`.
- **Limpieza de legacy**: se eliminaron artefactos de buildpack (`.buildpacks`) y el `Procfile` legacy para evitar confusión con Container Registry.

### Estado de verificación
- **Evidencia estática**: los artefactos de CI/CD existen en el repo (`.github/workflows/ci.yml`, `.github/workflows/deploy-heroku.yml`, `backend/Dockerfile.release`) y siguen el diseño de `spec.md` / `design.md`.
- **Runtime**: no se incluyó `verify-report.md` específico para este change (la verificación se realiza via GitHub Actions + deploy en Heroku según el plan de `tasks.md`).

### Referencias
- **Tag**: `v0.6.1`
- **Commits relevantes**:
  - `451890fc9738e5c70bc17feeb2efe52aea5f190d` — feat: sprint 8 MVP extendido + deploy production + CI/CD Heroku
  - `c594e5676e9c950c3e4ce53c49f8629d4bd6ba64` — chore(ci): trigger heroku deploy

