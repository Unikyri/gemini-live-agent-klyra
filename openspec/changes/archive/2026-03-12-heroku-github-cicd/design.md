# Design: CI/CD con GitHub Actions y Heroku Container Registry

## Enfoque Técnico

Implementar dos workflows de GitHub Actions (`ci.yml` y `deploy-heroku.yml`) que automaticen el ciclo completo **test → build → push → release** del backend hacia Heroku Container Registry. El diseño se apoya en tres pilares:

1. **CI gatekeeping**: ningún código llega a producción sin pasar tests.
2. **Container Registry nativo**: `docker build` + `docker push` directo a `registry.heroku.com`, sin depender del CLI de Heroku para el build.
3. **Release phase con imagen dedicada**: una imagen `release` ejecuta migraciones antes de enrutar tráfico al nuevo `web`.

El backend ya implementa el patrón `RUN_MIGRATIONS_ONLY=true` (línea 58 de `main.go`) y el `Dockerfile` ya copia el directorio `migrations/`. El pipeline solo necesita orquestar lo que ya existe.

---

## Decisiones de Arquitectura

### Decisión 1: Docker build/push directo vs `heroku container:push`

**Elección**: Usar `docker build` + `docker push` directo contra `registry.heroku.com` en lugar de `heroku container:push`.

**Alternativas consideradas**:
- *`heroku container:push web --context-path backend`*: Requiere instalar el CLI de Heroku en el runner (~30s extra), añade una dependencia no versionada y el flag `--context-path` ha tenido bugs históricos con monorepos.
- *GitHub Action `akhileshns/heroku-deploy`*: Action de terceros sin mantenimiento garantizado; dependencia de seguridad innecesaria.

**Rationale**: `docker login` + `docker build` + `docker push` usan herramientas ya presentes en los runners `ubuntu-latest` de GitHub Actions. Es más rápido (sin instalación de CLI), más explícito (el Dockerfile controla todo), y elimina una dependencia externa. El login al registry de Heroku se hace con `docker login --username=_ --password=$HEROKU_API_KEY registry.heroku.com`. Luego, `heroku container:release` sí requiere el CLI, pero es un solo comando liviano post-push.

---

### Decisión 2: Imagen `release` separada vs `Procfile` release con misma imagen

**Elección**: **Opción A** — Crear `backend/Dockerfile.release` como imagen separada que sube como process type `release` al Container Registry. Heroku ejecuta `heroku container:release web release`.

**Alternativas consideradas**:
- *Opción B: Usar un `Procfile` con `release:` command*: **No aplica con Container Registry**. Cuando Heroku usa Container Registry (no buildpack), ignora el campo `release:` del `Procfile`. La release phase se activa **solamente** si se pushea una imagen al process type `release` en el registry. Ref: [Heroku Container Registry docs](https://devcenter.heroku.com/articles/container-registry-and-runtime#release-phase).
- *Misma imagen para `web` y `release` con diferente `CMD`*: Posible, pero el Container Registry requiere que el `ENTRYPOINT`/`CMD` de la imagen `release` ejecute directamente el comando de migración. Usar la misma imagen forzaría a cambiar el `CMD` en runtime, lo cual no es posible sin un script wrapper que complicaría la imagen `distroless`.

**Rationale**: Con Container Registry, la release phase **requiere** una imagen separada pusheada como `release`. El `Dockerfile.release` es casi idéntico al `Dockerfile` principal, con la única diferencia de que setea `ENV RUN_MIGRATIONS_ONLY=true`. Al usar la misma base distroless y el mismo binario, la imagen release se beneficia del layer cache de Docker y no añade peso significativo. Esta es la forma canónica documentada por Heroku.

---

### Decisión 3: Workflow único vs workflows separados (CI + Deploy)

**Elección**: Dos workflows separados: `ci.yml` (tests) y `deploy-heroku.yml` (build + deploy).

**Alternativas consideradas**:
- *Workflow único con jobs dependientes (`needs: test`)*: Funciona, pero mezcla concerns. En PRs solo queremos CI; en `main` queremos CI + deploy. Un workflow único requiere condicionales `if:` en cada job, lo cual dificulta la lectura.
- *Reusable workflow (`.github/workflows/ci-reusable.yml`)*: Sobreingeniería para un solo backend con un solo entorno.

**Rationale**: Separar workflows permite:
- `ci.yml` corre en PRs y pushes a `main` (cualquier cambio en `backend/**`).
- `deploy-heroku.yml` corre **solo** en pushes a `main` con cambios en `backend/**` y **solo si CI pasa** (usando `workflow_run` event).
- Cada workflow tiene su propio badge y estado visible en GitHub.
- Los permisos OIDC/secrets se limitan al workflow que los necesita.

---

### Decisión 4: Trigger del deploy — `workflow_run` vs `needs` en mismo workflow

**Elección**: Usar el evento `workflow_run` para que `deploy-heroku.yml` se dispare automáticamente cuando `ci.yml` complete exitosamente en `main`.

**Alternativas consideradas**:
- *`on: push` con `needs: ci-job`*: Requiere que ambos estén en el mismo archivo YAML (vuelve a la decisión anterior de workflow único).
- *`on: push` + verificar manualmente el status check de CI*: Frágil; requiere polling de la API de GitHub.

**Rationale**: `workflow_run` es el mecanismo nativo de GitHub Actions para cadenas de workflows. Permite que el deploy solo se ejecute cuando CI termina con `conclusion == 'success'`. Además, hereda el contexto del commit correcto automáticamente.

---

### Decisión 5: Docker layer caching

**Elección**: Usar `docker/build-push-action` con `cache-from`/`cache-to` de tipo `gha` (GitHub Actions cache).

**Alternativas consideradas**:
- *Sin caching*: Cada build descarga dependencias Go (~1-2 min) y recompila desde cero (~1-2 min).
- *`actions/cache` manual para Go modules*: Solo cachea `go mod download`, no los layers de Docker.
- *Registry-based cache*: Requiere pushear images de cache al registry; más complejo.

**Rationale**: `docker/build-push-action` con el backend de cache `gha` (GitHub Actions cache) es la forma estándar y más eficiente. Cachea todos los layers del multi-stage build, incluyendo `go mod download` y la compilación. Reduce builds subsecuentes de ~3 min a ~30-60s cuando solo cambia código fuente (sin cambios en `go.mod`).

---

### Decisión 6: Instalación del Heroku CLI — solo para `container:release`

**Elección**: Instalar el Heroku CLI **mínimo** (via `npm install -g heroku`) solo en el job de deploy, exclusivamente para ejecutar `heroku container:release`.

**Alternativas consideradas**:
- *Usar la API REST de Heroku directamente*: El endpoint `PATCH /apps/{app}/formation` con el image ID del registry permite hacer release sin CLI. Sin embargo, requiere obtener el `docker_image_id` después del push y construir el request manualmente. Es más frágil y menos documentado.
- *Action pre-built de Heroku*: No existe una action oficial mantenida para Container Registry.

**Rationale**: `heroku container:release web release` es un comando idempotente y bien documentado. El CLI se instala en ~10s con npm. Es el camino con menor riesgo de errores.

---

### Decisión 7: Eliminación de `.buildpacks` y actualización del `Procfile`

**Elección**: Eliminar `.buildpacks` y documentar en el `Procfile` que es legacy (o eliminarlo). Con Container Registry, Heroku no usa buildpacks ni `Procfile`.

**Alternativas consideradas**:
- *Mantener `.buildpacks` y `Procfile` como fallback*: Genera confusión sobre qué método de deploy está activo. Heroku prioriza Container Registry sobre buildpacks si hay imágenes en el registry.

**Rationale**: El repo ya tiene un `Procfile` (con `web: bin/api` que es incorrecto para containers) y `.buildpacks` (apuntando al buildpack Go). Ambos son artefactos del enfoque buildpack que ya no se usa. Eliminar `.buildpacks` y dejar el `Procfile` con un comentario de que es legacy evita confusiones futuras.

---

## Flujo de Datos

### Pipeline CI/CD completo

```
  ┌─────────────────────────────────────────────────────────────┐
  │                    GitHub Repository                         │
  │                                                             │
  │  PR → backend/**   ──→  ci.yml                              │
  │                          ├─ checkout                        │
  │                          ├─ setup-go 1.25                   │
  │                          ├─ go mod download (cached)        │
  │                          ├─ go test ./...                   │
  │                          └─ status check ✓/✗                │
  │                                                             │
  │  push main → backend/** ──→ ci.yml (mismo flujo)            │
  │                               │                             │
  │                          (workflow_run: success)             │
  │                               ▼                             │
  │                         deploy-heroku.yml                   │
  │                          ├─ checkout                        │
  │                          ├─ docker login registry.heroku    │
  │                          ├─ docker build web                │
  │                          ├─ docker build release            │
  │                          ├─ docker push web                 │
  │                          ├─ docker push release             │
  │                          ├─ heroku container:release        │
  │                          └─ verify /health                  │
  └──────────────────────────┬──────────────────────────────────┘
                             │
                             ▼
  ┌─────────────────────────────────────────────────────────────┐
  │                  Heroku (klyra-backend-prod)                 │
  │                                                             │
  │  1. Release phase                                           │
  │     └─ Contenedor "release" arranca                         │
  │        └─ RUN_MIGRATIONS_ONLY=true                          │
  │           └─ main.go detecta flag (línea 58)                │
  │              └─ dbRepo.RunMigrations("./migrations")        │
  │                 ├─ OK → exit(0) → release succeeds          │
  │                 └─ Error → log.Fatalf → exit(1) → ABORT     │
  │                                                             │
  │  2. Web dyno (solo si release phase exitosa)                │
  │     └─ Contenedor "web" arranca                             │
  │        └─ RUN_MIGRATIONS_ON_BOOT=false                      │
  │           └─ Salta migraciones, inicia gin en :$PORT        │
  │                                                             │
  │  3. Health check                                            │
  │     └─ GET /health → 200 OK                                 │
  │     └─ GET /health?check=db → 200 {"db":"connected"}       │
  └─────────────────────────────────────────────────────────────┘
```

### Flujo Docker build (multi-stage, dos imágenes)

```
  backend/Dockerfile (imagen web)           backend/Dockerfile.release (imagen release)
  ┌──────────────────────────┐              ┌──────────────────────────────────┐
  │ FROM golang:1.25-alpine  │              │ FROM golang:1.25-alpine          │
  │   AS builder             │              │   AS builder                     │
  │ ├─ go mod download       │  (mismos     │ ├─ go mod download               │
  │ ├─ COPY . .              │   layers)    │ ├─ COPY . .                      │
  │ └─ go build → binary     │              │ └─ go build → binary             │
  │                          │              │                                  │
  │ FROM distroless          │              │ FROM distroless                  │
  │ ├─ COPY binary           │              │ ├─ COPY binary                   │
  │ ├─ COPY migrations/      │              │ ├─ COPY migrations/              │
  │ └─ ENTRYPOINT [binary]   │              │ ├─ ENV RUN_MIGRATIONS_ONLY=true  │
  │                          │              │ └─ ENTRYPOINT [binary]           │
  │ → registry.heroku.com/   │              │ → registry.heroku.com/           │
  │   klyra-backend-prod/web │              │   klyra-backend-prod/release     │
  └──────────────────────────┘              └──────────────────────────────────┘
```

---

## Cambios por Archivo

| Archivo | Acción | Descripción |
|---------|--------|-------------|
| `.github/workflows/ci.yml` | **Crear** | Workflow de CI: checkout, setup-go, go mod download, go test, con cache de módulos Go. Trigger: PR y push a `main` con cambios en `backend/**`. |
| `.github/workflows/deploy-heroku.yml` | **Crear** | Workflow de deploy: docker build/push de imágenes `web` y `release`, heroku container:release, health check. Trigger: `workflow_run` de ci.yml exitoso en `main`. |
| `backend/Dockerfile.release` | **Crear** | Imagen Docker idéntica al Dockerfile principal con `ENV RUN_MIGRATIONS_ONLY=true`. Heroku la ejecuta en release phase antes de arrancar web. |
| `.buildpacks` | **Eliminar** | Artefacto del enfoque buildpack. Con Container Registry no se usa. |
| `Procfile` | **Modificar** | Añadir comentario indicando que con Container Registry este archivo no controla el runtime. Mantener como documentación legacy o eliminar. |

---

## Estructura de los Workflows YAML

### `.github/workflows/ci.yml`

```yaml
name: Backend CI

on:
  pull_request:
    paths:
      - 'backend/**'
  push:
    branches: [main]
    paths:
      - 'backend/**'

permissions:
  contents: read

defaults:
  run:
    working-directory: backend

jobs:
  test:
    name: Test & Lint
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: backend/go.mod
          cache-dependency-path: backend/go.sum

      - name: Download dependencies
        run: go mod download

      - name: Run tests
        run: go test -race -count=1 ./...

      - name: Run vet
        run: go vet ./...
```

**Notas de diseño:**
- `go-version-file` lee la versión de `go.mod` (actualmente `1.25.6`), evitando hardcodear la versión.
- `cache-dependency-path` habilita el cache nativo de `setup-go` basado en `go.sum`.
- `go test -race` detecta data races; `-count=1` desactiva el cache de tests para CI.
- `go vet` corre análisis estático sin necesidad de instalar herramientas externas.
- `permissions: contents: read` sigue el principio de mínimo privilegio.
- `defaults.run.working-directory: backend` evita repetir `cd backend` en cada step.

### `.github/workflows/deploy-heroku.yml`

```yaml
name: Deploy to Heroku

on:
  workflow_run:
    workflows: ["Backend CI"]
    types: [completed]
    branches: [main]

permissions:
  contents: read

env:
  HEROKU_APP: ${{ secrets.HEROKU_APP_NAME }}
  REGISTRY: registry.heroku.com

jobs:
  deploy:
    name: Build & Deploy
    runs-on: ubuntu-latest
    if: ${{ github.event.workflow_run.conclusion == 'success' }}
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          ref: ${{ github.event.workflow_run.head_sha }}

      - name: Login to Heroku Container Registry
        run: echo "${{ secrets.HEROKU_API_KEY }}" | docker login -u _ --password-stdin ${{ env.REGISTRY }}

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Build and push web image
        uses: docker/build-push-action@v6
        with:
          context: backend
          file: backend/Dockerfile
          push: true
          tags: ${{ env.REGISTRY }}/${{ env.HEROKU_APP }}/web
          cache-from: type=gha,scope=web
          cache-to: type=gha,mode=max,scope=web

      - name: Build and push release image
        uses: docker/build-push-action@v6
        with:
          context: backend
          file: backend/Dockerfile.release
          push: true
          tags: ${{ env.REGISTRY }}/${{ env.HEROKU_APP }}/release
          cache-from: type=gha,scope=release
          cache-to: type=gha,mode=max,scope=release

      - name: Install Heroku CLI
        run: npm install -g heroku

      - name: Release containers
        run: heroku container:release web release --app ${{ env.HEROKU_APP }}
        env:
          HEROKU_API_KEY: ${{ secrets.HEROKU_API_KEY }}

      - name: Verify deployment
        run: |
          sleep 15
          STATUS=$(curl -s -o /dev/null -w "%{http_code}" "https://${{ env.HEROKU_APP }}.herokuapp.com/health")
          if [ "$STATUS" != "200" ]; then
            echo "::error::Health check failed with status $STATUS"
            exit 1
          fi
          echo "Health check passed (HTTP $STATUS)"
```

**Notas de diseño:**
- `workflow_run` + `if: conclusion == 'success'` garantiza que deploy solo ocurre si CI pasó.
- `ref: github.event.workflow_run.head_sha` asegura que se deploya exactamente el commit que pasó CI.
- `docker login` usa `--password-stdin` para evitar que el token aparezca en el historial del shell. GitHub Actions enmascara `secrets.*` en logs automáticamente.
- `docker/build-push-action` con cache `gha` aprovecha el cache de GitHub Actions para acelerar builds.
- `scope=web` y `scope=release` separan los caches de ambas imágenes para evitar invalidaciones cruzadas.
- El health check espera 15s (tiempo suficiente para que la release phase complete y el web dyno arranque).

---

## `backend/Dockerfile.release`

```dockerfile
# Build stage — idéntico al Dockerfile principal
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o klyra-backend ./cmd/api/main.go

# Runtime stage — ejecuta migraciones y sale
FROM gcr.io/distroless/static-debian12 AS runner

WORKDIR /app
COPY --from=builder /app/klyra-backend .
COPY --from=builder /app/migrations ./migrations

ENV RUN_MIGRATIONS_ONLY=true

ENTRYPOINT ["/app/klyra-backend"]
```

**Rationale del diseño:**
- Es idéntico al `Dockerfile` de `web` excepto por `ENV RUN_MIGRATIONS_ONLY=true`.
- El binario detecta esta variable y ejecuta `RunMigrations("./migrations")` → `os.Exit(0)` (línea 58-64 de `main.go`).
- Al compartir los mismos `FROM` y `COPY` layers que el `Dockerfile` web, Docker y el cache de GitHub Actions reutilizan layers, minimizando el tiempo de build adicional.

---

## Interfaces / Contratos

### Secretos de GitHub Repository

| Secreto | Tipo | Descripción |
|---------|------|-------------|
| `HEROKU_API_KEY` | Secret | Token de API de Heroku (Account → API Key). Se usa para `docker login` y `heroku container:release`. |
| `HEROKU_APP_NAME` | Secret | Nombre de la app: `klyra-backend-prod`. Separado para no hardcodear. |

### Variables de entorno del workflow

| Variable | Dónde se define | Valor |
|----------|----------------|-------|
| `REGISTRY` | `env:` en workflow | `registry.heroku.com` |
| `HEROKU_APP` | `env:` en workflow | `${{ secrets.HEROKU_APP_NAME }}` |

### Heroku Config Vars (no se tocan desde CI)

Estas variables persisten entre deploys y se configuran manualmente en Heroku:

| Variable | Valor esperado | Notas |
|----------|---------------|-------|
| `DATABASE_URL` | (auto por addon) | Inyectada por addon PostgreSQL |
| `ENV` | `production` | Activa guards de seguridad |
| `GIN_MODE` | `release` | Desactiva debug output de Gin |
| `RUN_MIGRATIONS_ON_BOOT` | `false` | Las migraciones corren en release phase |
| `JWT_SECRET` | (secreto) | Para tokens de acceso |
| `REFRESH_TOKEN_SECRET` | (secreto) | Para refresh tokens |
| `GOOGLE_CLIENT_ID` | (ID OAuth) | Verificación Google Sign-In |
| `STORAGE_MODE` | `gcs` | Obligatorio en prod (guard en main.go) |
| `GCP_PROJECT_ID` | (proyecto) | Para Vertex AI / GCS |
| `GOOGLE_APPLICATION_CREDENTIALS` | (JSON inline) | SA credentials para GCS + AI |
| `ALLOWED_ORIGINS` | (orígenes) | CORS restrictivo en prod |

### Contrato del Health Check

```
GET /health
→ 200 { "status": "ok" }

GET /health?check=db
→ 200 { "status": "ok", "db": "connected" }
→ 503 { "status": "degraded", "db": "unreachable", "error": "..." }
```

---

## Hardening y Seguridad

### Enmascarado de secretos en logs

- **GitHub Actions**: Toda referencia `${{ secrets.* }}` se enmascara automáticamente en los logs (aparece como `***`).
- **`docker login`**: Se usa `--password-stdin` en lugar de `--password` para evitar que el token aparezca en el historial del proceso.
- **Regla crítica**: Nunca usar `echo ${{ secrets.HEROKU_API_KEY }}` ni similar. El workflow diseñado no hace `echo` de ningún secreto.
- **`HEROKU_API_KEY`** se pasa como env var al step de `heroku container:release`, no como argumento inline.

### Permisos del workflow

- Ambos workflows usan `permissions: contents: read` (mínimo privilegio).
- El deploy workflow no necesita `write` porque no pushea a Git; solo interactúa con el registry de Heroku.
- `HEROKU_API_KEY` tiene scope sobre la app `klyra-backend-prod` únicamente.

### Branch protections recomendadas

| Regla | Valor | Justificación |
|-------|-------|---------------|
| Require status checks to pass | `Backend CI / Test & Lint` | Impide merge a main sin tests verdes |
| Require branches to be up to date | Habilitado | Evita merges que no incluyen el último main |
| Require pull request reviews | ≥1 aprobación | Code review obligatorio (especialmente migraciones) |
| Restrict pushes to `main` | Solo via PR | Evita pushes directos que bypaseen CI |
| Do not allow force pushes | Habilitado | Protege el historial |

### Protección contra leaks

- `DATABASE_URL` **nunca** se pone en GitHub Secrets. Heroku la gestiona automáticamente.
- Las credenciales GCP (`GOOGLE_APPLICATION_CREDENTIALS`) solo viven en Heroku Config Vars.
- El workflow no ejecuta `heroku config` ni ningún comando que pueda exponer config vars en logs.

---

## Estrategia de Testing

| Capa | Qué probar | Enfoque |
|------|-----------|---------|
| Workflow CI | `ci.yml` ejecuta tests y reporta status check | Crear PR con cambio trivial en `backend/`; verificar check verde/rojo en GitHub |
| Workflow CI | Path filter: cambios en `mobile/` no disparan CI de backend | Crear PR solo con cambios en `mobile/`; verificar que `ci.yml` no se ejecuta |
| Workflow Deploy | `deploy-heroku.yml` se dispara tras CI exitoso en main | Merge a main con cambio en backend; verificar que deploy workflow arranca automáticamente |
| Workflow Deploy | Deploy no se ejecuta si CI falla | Introducir test fallido en un branch; verificar que el workflow de deploy no se dispara |
| Docker build | Imagen web arranca correctamente | `docker build -f backend/Dockerfile backend/` local; `docker run -e PORT=8080 <image>` responde en `/health` |
| Docker build | Imagen release ejecuta migraciones y sale | `docker build -f backend/Dockerfile.release backend/` local; `docker run -e DATABASE_URL=... <image>` ejecuta migraciones y sale con code 0 |
| Release phase | Migración exitosa permite deploy | Merge con migración válida; verificar `heroku releases -a klyra-backend-prod` muestra "Succeeded" |
| Release phase | Migración fallida aborta deploy | (Branch de prueba) Migración con SQL inválido; verificar que release falla y dyno anterior sigue vivo |
| Health check | Post-deploy health check pasa | Después del primer deploy, `curl https://klyra-backend-prod.herokuapp.com/health` → 200 |
| Health check | DB connectivity | `curl https://klyra-backend-prod.herokuapp.com/health?check=db` → 200 con `"db":"connected"` |
| Persistencia | Datos existentes persisten tras deploy | Verificar que usuarios, cursos y materiales existentes siguen accesibles después del deploy |

---

## Migración / Rollout

### Fase 1: Crear archivos (este cambio)

1. Crear `.github/workflows/ci.yml`.
2. Crear `.github/workflows/deploy-heroku.yml`.
3. Crear `backend/Dockerfile.release`.
4. Eliminar `.buildpacks`.
5. Actualizar o eliminar `Procfile`.

### Fase 2: Configurar secretos (manual, pre-deploy)

1. En GitHub → Settings → Secrets and Variables → Actions:
   - Crear `HEROKU_API_KEY` con el API key de Heroku.
   - Crear `HEROKU_APP_NAME` con valor `klyra-backend-prod`.
2. En Heroku → Settings → Config Vars:
   - Verificar que `RUN_MIGRATIONS_ON_BOOT=false` está configurado.
   - Verificar que todas las config vars de producción están presentes.

### Fase 3: Activar branch protections (post primer deploy exitoso)

1. En GitHub → Settings → Branches → Branch protection rules para `main`:
   - Require status checks: seleccionar `Backend CI / Test & Lint`.
   - Require PR reviews.
   - Restrict direct pushes.

### Fase 4: Validación

1. Crear PR con cambio trivial en backend → verificar CI.
2. Merge a main → verificar deploy automático.
3. Verificar health check post-deploy.
4. Verificar persistencia de datos.

### Rollback

- **Rollback inmediato**: `heroku rollback -a klyra-backend-prod` revierte al contenedor anterior (~10s).
- **Desactivar pipeline**: Deshabilitar workflows desde GitHub Actions o renombrar archivos YAML.
- **Rollback de migración**: Ejecutar archivo `.down.sql` manualmente via `heroku pg:psql`.
- **Regenerar API key**: Si se sospecha compromiso, regenerar en Heroku Dashboard → Account → API Key y actualizar en GitHub Secrets.

---

## Preguntas Abiertas

- [ ] **Timeout del health check**: El workflow espera 15s antes de verificar `/health`. Si la release phase + boot del dyno tarda más (posible en `essential-0`), se necesitará un retry con backoff. ¿Cuánto tarda típicamente el boot del dyno en `klyra-backend-prod`?
- [ ] **Linter golangci-lint**: El proposal menciona linting opcional. ¿Se incluye en la primera iteración o se añade después? El diseño actual usa `go vet` (built-in) como mínimo viable.
- [ ] **Notificaciones de deploy fallido**: GitHub Actions notifica al autor del commit por defecto. ¿Se necesita integración con Slack u otro canal para el equipo?
