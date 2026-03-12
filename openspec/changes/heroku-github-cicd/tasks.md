# Tasks: CI/CD con GitHub Actions y Heroku Container Registry

**Cambio:** `heroku-github-cicd`
**Fecha:** 2026-03-12
**Basado en:** `spec.md`, `design.md`

---

## Orden de dependencias

```
Bloque A (Workflows) ──┐
                        ├──→ Bloque D (Verificación)
Bloque B (Dockerfile)  ─┤
                        ├──→ Bloque E (Documentación)
Bloque C (Limpieza)  ───┘
```

- **A y B** son independientes entre sí; pueden implementarse en paralelo.
- **C** es independiente pero lógicamente se ejecuta junto con A/B.
- **D** requiere que A, B y C estén completos (necesita los artefactos creados).
- **E** se puede escribir en paralelo con D, pero referencia los artefactos de A.

---

## Bloque A: Workflows GitHub Actions

### A.1 — Crear `.github/workflows/ci.yml`

- [ ] **A.1.1** Crear directorio `.github/workflows/` (no existe actualmente en el repositorio).
- [ ] **A.1.2** Crear archivo `.github/workflows/ci.yml` con el siguiente contenido exacto definido en `design.md` (sección "Estructura de los Workflows YAML"):
  - `name: Backend CI`
  - Triggers: `on.pull_request.paths: ['backend/**']` + `on.push.branches: [main]` con `paths: ['backend/**']`.
  - `permissions: contents: read` (mínimo privilegio — AC-18, design.md § Hardening).
  - `defaults.run.working-directory: backend`.
  - Job `test` en `ubuntu-latest` con steps:
    1. `actions/checkout@v4`
    2. `actions/setup-go@v5` con `go-version-file: backend/go.mod` y `cache-dependency-path: backend/go.sum`
    3. `go mod download`
    4. `go test -race -count=1 ./...`
    5. `go vet ./...`
- [ ] **A.1.3** Validar que el YAML es sintácticamente correcto (indentación, uso de comillas en paths con `**`).

**Verificable:** El archivo existe en `.github/workflows/ci.yml`, es YAML válido y coincide con la estructura del design.md líneas 197–238.

### A.2 — Crear `.github/workflows/deploy-heroku.yml`

- [ ] **A.2.1** Crear archivo `.github/workflows/deploy-heroku.yml` con el contenido exacto definido en `design.md` (líneas 250–320):
  - `name: Deploy to Heroku`
  - Trigger: `on.workflow_run.workflows: ["Backend CI"]`, `types: [completed]`, `branches: [main]`.
  - `permissions: contents: read` (mínimo privilegio).
  - Variables de entorno a nivel de workflow:
    - `HEROKU_APP: ${{ secrets.HEROKU_APP_NAME }}`
    - `REGISTRY: registry.heroku.com`
  - Job `deploy` con condición `if: ${{ github.event.workflow_run.conclusion == 'success' }}`.
  - Steps:
    1. `actions/checkout@v4` con `ref: ${{ github.event.workflow_run.head_sha }}`.
    2. Login al registry: `echo "${{ secrets.HEROKU_API_KEY }}" | docker login -u _ --password-stdin ${{ env.REGISTRY }}` (usa `--password-stdin`, nunca `echo` del secret solo — AC-18).
    3. `docker/setup-buildx-action@v3`.
    4. `docker/build-push-action@v6` para imagen `web` (context: `backend`, file: `backend/Dockerfile`, tags: `$REGISTRY/$HEROKU_APP/web`, cache `gha` scope `web`).
    5. `docker/build-push-action@v6` para imagen `release` (context: `backend`, file: `backend/Dockerfile.release`, tags: `$REGISTRY/$HEROKU_APP/release`, cache `gha` scope `release`).
    6. Instalar Heroku CLI: `npm install -g heroku`.
    7. Release: `heroku container:release web release --app ${{ env.HEROKU_APP }}` con env `HEROKU_API_KEY: ${{ secrets.HEROKU_API_KEY }}`.
    8. Health check: `sleep 15` + `curl -s -o /dev/null -w "%{http_code}" https://$HEROKU_APP.herokuapp.com/health` con validación de status 200.

- [ ] **A.2.2** Verificar que `workflow_run.workflows` usa exactamente el string `"Backend CI"` (debe coincidir con el `name:` de `ci.yml`).

**Verificable:** El archivo existe, es YAML válido, referencia los secrets `HEROKU_API_KEY` y `HEROKU_APP_NAME`, y NO contiene `echo` de ningún secret fuera de un pipe a `docker login --password-stdin`.

### A.3 — Seguridad de los workflows

- [ ] **A.3.1** Revisar que ambos workflows tienen `permissions: contents: read` (nunca `write` — design.md § Hardening).
- [ ] **A.3.2** Revisar que `HEROKU_API_KEY` se pasa exclusivamente vía `${{ secrets.HEROKU_API_KEY }}` (GitHub lo enmascara en logs).
- [ ] **A.3.3** Confirmar que NO existe ningún step con `echo ${{ secrets.* }}` ni `run: echo $HEROKU_API_KEY` en ninguno de los dos workflows.
- [ ] **A.3.4** Confirmar que `docker login` usa `--password-stdin` (no `--password` inline).
- [ ] **A.3.5** Confirmar que `deploy-heroku.yml` NO ejecuta `heroku config`, `heroku pg:psql` ni ningún comando que pueda exponer config vars en logs.

**Verificable:** Inspección visual de ambos archivos YAML; ningún secreto aparece en texto plano.

---

## Bloque B: Docker Release Image

- [ ] **B.1** Crear archivo `backend/Dockerfile.release` con el contenido exacto del design.md (líneas 334–355):
  ```dockerfile
  FROM golang:1.25-alpine AS builder
  WORKDIR /app
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 GOOS=linux go build -o klyra-backend ./cmd/api/main.go

  FROM gcr.io/distroless/static-debian12 AS runner
  WORKDIR /app
  COPY --from=builder /app/klyra-backend .
  COPY --from=builder /app/migrations ./migrations
  ENV RUN_MIGRATIONS_ONLY=true
  ENTRYPOINT ["/app/klyra-backend"]
  ```
- [ ] **B.2** Verificar que `backend/Dockerfile.release` es idéntico a `backend/Dockerfile` excepto por:
  - Adición de `ENV RUN_MIGRATIONS_ONLY=true`.
  - Ausencia de `EXPOSE 8080` (no necesario para release phase).
- [ ] **B.3** Verificar que `backend/Dockerfile` existente (`backend/Dockerfile` líneas 1–22) NO requiere cambios (ya copia `migrations/` y usa `distroless`).

**Verificable:** `docker build -f backend/Dockerfile.release backend/` compila sin errores localmente.

---

## Bloque C: Limpieza de artefactos legacy

- [ ] **C.1** Eliminar archivo `.buildpacks` de la raíz del repositorio (actualmente contiene `https://github.com/heroku/heroku-buildpack-go`). Justificación: con Container Registry no se usan buildpacks (spec.md § 8, AC-24).
- [ ] **C.2** Eliminar archivo `Procfile` de la raíz del repositorio (actualmente contiene `web: bin/api` y `release: RUN_MIGRATIONS_ONLY=true bin/api`). Justificación: con Container Registry, Heroku ignora el `Procfile`; los comandos se definen por las imágenes Docker (spec.md § 7.5, AC-23).
- [ ] **C.3** Verificar que no existen referencias a `.buildpacks` ni a `Procfile` en ningún workflow nuevo (`.github/workflows/*.yml`).

**Verificable:** `ls .buildpacks` y `ls Procfile` no existen tras el merge. Ningún workflow los referencia.

---

## Bloque D: Verificación con MCP y herramientas

> **Nota:** Este bloque es de verificación post-implementación. No ejecuta código ni modifica archivos; solo describe las comprobaciones a realizar.

### D.1 — Verificación GitHub (workflows detectados y runs)

- [ ] **D.1.1** Tras push a `main`, verificar vía GitHub (MCP `user-github` o web) que los workflows `Backend CI` y `Deploy to Heroku` aparecen en la pestaña Actions del repositorio.
- [ ] **D.1.2** Verificar que un PR con cambios en `backend/` dispara el workflow `Backend CI` y muestra status check (AC-01).
- [ ] **D.1.3** Verificar que un PR con cambios **solo** en `mobile/` o `openspec/` NO dispara `Backend CI` (AC-02).
- [ ] **D.1.4** Verificar que tras CI exitoso en `main`, el workflow `Deploy to Heroku` se dispara automáticamente (AC-08).
- [ ] **D.1.5** Verificar que si CI falla en `main`, `Deploy to Heroku` NO se ejecuta (AC-10).

### D.2 — Verificación Heroku (release, logs, health)

- [ ] **D.2.1** Verificar con Heroku (MCP `user-heroku` o CLI) que `heroku releases -a klyra-backend-prod` muestra la nueva release con status `succeeded` tras un deploy exitoso (AC-15).
- [ ] **D.2.2** Verificar logs de la release phase: `heroku logs -a klyra-backend-prod --dyno release` confirma ejecución de migraciones (AC-15).
- [ ] **D.2.3** Verificar health check: `GET https://klyra-backend-prod.herokuapp.com/health` responde `200 {"status":"ok"}` (AC-17).
- [ ] **D.2.4** Verificar health check con DB: `GET https://klyra-backend-prod.herokuapp.com/health?check=db` responde `200 {"status":"ok","db":"connected"}`.

### D.3 — Verificación de persistencia de datos

- [ ] **D.3.1** Antes del deploy, anotar un registro existente en la DB (usuario, curso o material).
- [ ] **D.3.2** Tras el deploy, verificar que el mismo registro sigue accesible (AC-21).
- [ ] **D.3.3** Verificar que `heroku addons -a klyra-backend-prod` muestra `heroku-postgresql:essential-0` con estado `created` (addon no fue destruido ni recreado) (AC-22).

### D.4 — Verificación de secretos y configuración

- [ ] **D.4.1** Verificar que en GitHub → Settings → Secrets solo existen `HEROKU_API_KEY` y `HEROKU_APP_NAME` (AC-19: `DATABASE_URL` NO está en GitHub Secrets).
- [ ] **D.4.2** Verificar que `heroku config -a klyra-backend-prod` muestra todas las config vars requeridas (tabla 6.2 de spec.md): `DATABASE_URL`, `ENV`, `GIN_MODE`, `JWT_SECRET`, `REFRESH_TOKEN_SECRET`, `GOOGLE_CLIENT_ID`, `ALLOWED_ORIGINS`, `RUN_MIGRATIONS_ON_BOOT=false`, `STORAGE_MODE=gcs`, `GCP_PROJECT_ID`, `GOOGLE_APPLICATION_CREDENTIALS_JSON` (AC-20).
- [ ] **D.4.3** Inspeccionar logs de los workflow runs y confirmar que `HEROKU_API_KEY` aparece enmascarado como `***` (AC-18).

---

## Bloque E: Documentación de configuración de secrets

- [ ] **E.1** Crear sección en el `README.md` del repositorio (o en `docs/cicd-setup.md` si existe carpeta `docs/`) que documente:
  1. **Secrets de GitHub requeridos**: `HEROKU_API_KEY` (cómo obtenerlo: Heroku Dashboard → Account → API Key) y `HEROKU_APP_NAME` (valor: nombre de la app en Heroku).
  2. **Config Vars de Heroku requeridas**: lista de la tabla 6.2 de spec.md con indicación de cuáles son automáticas (`DATABASE_URL`) y cuáles manuales.
  3. **Cómo verificar configuración pre-deploy**: comandos `heroku config -a <app>` y `heroku addons -a <app>`.
  4. **Cómo hacer rollback**: `heroku rollback -a <app>` y cómo desactivar el pipeline desde GitHub Actions.
  5. **Cómo regenerar el API key**: pasos para regenerar en Heroku y actualizar en GitHub Secrets.

- [ ] **E.2** Añadir advertencia clara: "NUNCA copiar `DATABASE_URL` ni credenciales de runtime (`JWT_SECRET`, `GOOGLE_APPLICATION_CREDENTIALS_JSON`) a GitHub Secrets. Estos valores solo deben existir en Heroku Config Vars."

- [ ] **E.3** Documentar el flujo CI/CD con un diagrama simplificado (el diagrama del spec.md § 12 o el de design.md § Flujo de Datos).

**Verificable:** La documentación existe, es legible y cubre los 5 puntos de E.1.

---

## Resumen de tareas

| Bloque | Tareas | Archivos afectados |
|--------|--------|--------------------|
| A — Workflows | A.1 (3) + A.2 (2) + A.3 (5) = **10** | `.github/workflows/ci.yml`, `.github/workflows/deploy-heroku.yml` |
| B — Dockerfile release | **3** | `backend/Dockerfile.release` |
| C — Limpieza legacy | **3** | `.buildpacks` (eliminar), `Procfile` (eliminar) |
| D — Verificación | D.1 (5) + D.2 (4) + D.3 (3) + D.4 (3) = **15** | Ninguno (solo verificación) |
| E — Documentación | **3** | `README.md` o `docs/cicd-setup.md` |
| **Total** | **34 tareas** | |

---

## Prerrequisitos manuales (antes de la primera ejecución del pipeline)

1. Configurar `HEROKU_API_KEY` en GitHub → Settings → Secrets and Variables → Actions.
2. Configurar `HEROKU_APP_NAME` en GitHub → Settings → Secrets and Variables → Actions.
3. Verificar que `RUN_MIGRATIONS_ON_BOOT=false` está en Heroku Config Vars.
4. Verificar que todas las config vars de la tabla 6.2 (spec.md) están presentes en Heroku.

---

## Siguiente paso

Listo para implementación (`sdd-apply`). El orden recomendado es:
1. Bloques A + B + C en paralelo (creación de artefactos).
2. Bloque E (documentación).
3. Bloque D (verificación post-deploy, tras merge a `main`).
