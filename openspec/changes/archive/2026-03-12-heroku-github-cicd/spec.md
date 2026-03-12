# Spec: CI/CD con GitHub Actions y Heroku Container Registry

**Cambio:** `heroku-github-cicd`
**Fecha:** 2026-03-12
**Basado en:** `exploration.md`, `proposal.md`

---

## 1. Resumen

Implementar un pipeline CI/CD automatizado usando GitHub Actions que:

1. Ejecute tests y validaciones del backend en cada PR y push a `main`.
2. Despliegue automáticamente a Heroku Container Registry cuando los tests pasen en `main`.
3. Ejecute migraciones de base de datos en la release phase antes de enrutar tráfico al nuevo código.

---

## 2. Artefactos a crear / modificar

| Artefacto | Acción | Descripción |
|-----------|--------|-------------|
| `.github/workflows/ci.yml` | **Crear** | Workflow de integración continua (tests + lint) |
| `.github/workflows/deploy-heroku.yml` | **Crear** | Workflow de despliegue a Heroku |
| `backend/Dockerfile.release` | **Crear** | Imagen Docker para release phase (migraciones) |
| `Procfile` | **Eliminar o vaciar** | No se usa con container registry; evitar confusión |
| `.buildpacks` | **Eliminar** | No se necesita con container registry |

---

## 3. Workflow CI — `.github/workflows/ci.yml`

### 3.1 Triggers

```yaml
on:
  pull_request:
    paths:
      - "backend/**"
  push:
    branches: [main]
    paths:
      - "backend/**"
```

- Se ejecuta en PRs que toquen `backend/**`.
- Se ejecuta en push a `main` que toque `backend/**`.
- NO se ejecuta en cambios fuera de `backend/` (e.g., `mobile/`, `openspec/`, `README.md`).

### 3.2 Job: `test`

| Paso | Acción / Comando | Detalle |
|------|------------------|---------|
| 1 | `actions/checkout@v4` | Checkout completo del repositorio |
| 2 | `actions/setup-go@v5` con `go-version: '1.25'` | Debe coincidir con `go.mod` (`go 1.25.6`) |
| 3 | `go mod download` | Descarga de dependencias (working-directory: `backend`) |
| 4 | `go test ./...` | Ejecución de todos los tests (working-directory: `backend`) |

### 3.3 Entorno de ejecución

- **Runner:** `ubuntu-latest`
- **Go version:** `1.25` (resuelve a la última patch de la serie 1.25.x, actualmente `1.25.6`)
- **Timeout:** 10 minutos por job (valor por defecto, suficiente para el backend actual)

### 3.4 Comportamiento esperado

| Escenario | Resultado |
|-----------|-----------|
| PR con cambios en `backend/`, tests pasan | Check verde en el PR |
| PR con cambios en `backend/`, tests fallan | Check rojo en el PR; merge bloqueado si se activa branch protection |
| PR con cambios solo en `mobile/` | Workflow NO se dispara |
| Push a `main` con cambios en `backend/` | Workflow se ejecuta; si pasa, habilita el deploy |
| Push a `main` con cambios solo en `mobile/` | Workflow NO se dispara |

---

## 4. Workflow Deploy — `.github/workflows/deploy-heroku.yml`

### 4.1 Triggers y condición

```yaml
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
    branches: [main]
```

- Se dispara **solo** cuando el workflow CI (`ci.yml`) completa en la rama `main`.
- El job de deploy tiene la condición: `if: github.event.workflow_run.conclusion == 'success'`.
- **NO se ejecuta en PRs**, solo en push a `main` que haya pasado CI.

### 4.2 Job: `deploy`

| Paso | Acción / Comando | Detalle |
|------|------------------|---------|
| 1 | `actions/checkout@v4` | Checkout del código |
| 2 | Instalar Heroku CLI | `curl https://cli-assets.heroku.com/install.sh \| sh` o usar `npm install -g heroku` |
| 3 | Login al Container Registry | `heroku container:login` usando `HEROKU_API_KEY` en el env |
| 4 | Build imagen `web` | `docker build -t registry.heroku.com/$HEROKU_APP_NAME/web -f backend/Dockerfile backend/` |
| 5 | Build imagen `release` | `docker build -t registry.heroku.com/$HEROKU_APP_NAME/release -f backend/Dockerfile.release backend/` |
| 6 | Push imagen `web` | `docker push registry.heroku.com/$HEROKU_APP_NAME/web` |
| 7 | Push imagen `release` | `docker push registry.heroku.com/$HEROKU_APP_NAME/release` |
| 8 | Release | `heroku container:release web release -a $HEROKU_APP_NAME` |
| 9 | Health check post-deploy | `curl -sf https://$HEROKU_APP_NAME.herokuapp.com/health` con reintentos (3 intentos, 10s entre cada uno) |

### 4.3 Variables de entorno del workflow

```yaml
env:
  HEROKU_API_KEY: ${{ secrets.HEROKU_API_KEY }}
  HEROKU_APP_NAME: ${{ secrets.HEROKU_APP_NAME }}
```

### 4.4 Comportamiento esperado

| Escenario | Resultado |
|-----------|-----------|
| CI pasa en `main` con cambios en `backend/` | Deploy se ejecuta automáticamente |
| CI falla en `main` | Deploy NO se ejecuta (`conclusion != 'success'`) |
| PR mergeada a `main` (CI pasa) | Deploy se ejecuta |
| Push a `main` sin cambios en `backend/` | CI no se dispara → deploy no se dispara |
| Release phase falla (migración inválida) | Heroku aborta el deploy; versión anterior sigue viva |
| Health check falla post-deploy | El step falla y se marca el workflow como fallido (alerta) |

---

## 5. Contrato de secretos

### 5.1 GitHub Repository Secrets (obligatorios)

| Secreto | Valor esperado | Propósito |
|---------|----------------|-----------|
| `HEROKU_API_KEY` | Token de API de la cuenta Heroku | Autenticación con Heroku CLI y Container Registry |
| `HEROKU_APP_NAME` | `klyra-backend-prod` | Nombre de la app destino en Heroku |

### 5.2 Lo que NO va en GitHub Secrets

| Variable | Razón |
|----------|-------|
| `DATABASE_URL` | La gestiona Heroku automáticamente vía el addon Postgres; el valor rota y nunca debe copiarse fuera |
| `JWT_SECRET` | Secreto de aplicación; vive en Heroku Config Vars |
| `REFRESH_TOKEN_SECRET` | Secreto de aplicación; vive en Heroku Config Vars |
| `GOOGLE_APPLICATION_CREDENTIALS_JSON` | Credenciales GCP; vive en Heroku Config Vars |
| Cualquier config var de runtime | GitHub Actions no necesita acceso al runtime de la app |

### 5.3 Seguridad del `HEROKU_API_KEY`

- GitHub enmascara automáticamente los secrets en los logs del workflow.
- El workflow NUNCA debe hacer `echo $HEROKU_API_KEY` ni exponer el valor en logs.
- Si se sospecha compromiso: regenerar en Heroku Dashboard → Account → API Key y actualizar en GitHub.

---

## 6. Contrato Heroku

### 6.1 Addon PostgreSQL

- **Addon:** `heroku-postgresql:essential-0` (ya provisionado, slug `postgresql-curved-48489`).
- **Estado requerido:** `created` (operativo).
- **Regla:** el addon NO se crea ni se destruye en cada deploy. Persiste entre deploys.
- **`DATABASE_URL`:** inyectada automáticamente por Heroku como config var. El backend la detecta con precedencia sobre `DB_MODE` (línea 333 de `main.go`).
- **Reset:** solo ocurre con `heroku pg:reset` o `heroku addons:destroy` explícito. Ningún paso del CI/CD ejecuta estos comandos.

### 6.2 Config Vars mínimas requeridas en Heroku

Estas variables deben existir en Heroku **antes** del primer deploy automatizado:

| Variable | Valor | Origen |
|----------|-------|--------|
| `DATABASE_URL` | (automática) | Addon Postgres |
| `ENV` | `production` | Manual |
| `GIN_MODE` | `release` | Manual |
| `PORT` | `8080` (o el que asigne Heroku) | Heroku / Manual |
| `JWT_SECRET` | (secreto) | Manual |
| `REFRESH_TOKEN_SECRET` | (secreto) | Manual |
| `GOOGLE_CLIENT_ID` | ID de cliente OAuth Google | Manual |
| `ALLOWED_ORIGINS` | Lista de orígenes permitidos (CORS) | Manual |
| `RUN_MIGRATIONS_ON_BOOT` | `false` | Manual |
| `STORAGE_MODE` | `gcs` | Manual |
| `GCP_PROJECT_ID` | ID del proyecto GCP | Manual |
| `GOOGLE_APPLICATION_CREDENTIALS_JSON` | JSON de service account GCP | Manual |

### 6.3 Validación pre-deploy

Antes del primer deploy automatizado, el equipo debe verificar manualmente:

1. `heroku config -a klyra-backend-prod` muestra todas las variables de la tabla 6.2.
2. `heroku addons -a klyra-backend-prod` muestra `heroku-postgresql:essential-0` con estado `created`.
3. `HEROKU_API_KEY` y `HEROKU_APP_NAME` están configurados en GitHub Repository Secrets.

---

## 7. Migraciones — Release Phase

### 7.1 Mecanismo elegido: `Dockerfile.release`

Se crea un `backend/Dockerfile.release` que produce una imagen dedicada para la release phase. Heroku ejecuta esta imagen antes de enrutar tráfico al nuevo dyno `web`.

### 7.2 Contenido de `backend/Dockerfile.release`

```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o klyra-backend ./cmd/api/main.go

FROM gcr.io/distroless/static-debian12

WORKDIR /app
COPY --from=builder /app/klyra-backend .
COPY --from=builder /app/migrations ./migrations

ENV RUN_MIGRATIONS_ONLY=true
ENTRYPOINT ["/app/klyra-backend"]
```

- Idéntico al `Dockerfile` principal, excepto por `ENV RUN_MIGRATIONS_ONLY=true`.
- El binario detecta `RUN_MIGRATIONS_ONLY=true` (línea 58 de `main.go`), ejecuta migraciones y sale con `os.Exit(0)`.

### 7.3 Flujo de ejecución de la release phase

1. Heroku inicia el contenedor `release` con las config vars del app (incluyendo `DATABASE_URL`).
2. El binario se conecta a la base de datos usando `DATABASE_URL`.
3. Ejecuta `dbRepo.RunMigrations("./migrations")` — aplica solo migraciones pendientes (archivos `000001_...up.sql` hasta `000006_...up.sql`, etc.).
4. Si las migraciones completan: sale con código `0`. Heroku continúa con el deploy del dyno `web`.
5. Si las migraciones fallan: `log.Fatalf` causa exit con código `!= 0`. Heroku **aborta el deploy** y mantiene la versión anterior.

### 7.4 Comportamiento esperado

| Escenario | Resultado |
|-----------|-----------|
| Migraciones pendientes existen y son válidas | Release phase completa (exit 0); deploy continúa |
| No hay migraciones pendientes | Release phase completa rápidamente (exit 0); deploy continúa |
| Migración contiene SQL inválido | `log.Fatalf` → exit 1; deploy **abortado**; versión anterior activa |
| `DATABASE_URL` no está configurada | `log.Fatalf` al conectar → exit 1; deploy **abortado** |
| Release phase excede 1 hora (timeout Heroku) | Heroku mata el proceso; deploy **abortado** |

### 7.5 Por qué NO Procfile

Con container registry, Heroku ignora el `Procfile` para determinar los comandos de los dynos. El comando `web` y `release` se definen por las imágenes Docker pusheadas. El `Procfile` actual (`web: bin/api`, `release: RUN_MIGRATIONS_ONLY=true bin/api`) solo aplica a deploys por buildpack, que se abandonan con este cambio.

- El `Procfile` se **elimina** (o se vacía) para evitar confusión.
- El `.buildpacks` se **elimina** por la misma razón.

---

## 8. Eliminación de artefactos legacy

| Artefacto | Acción | Justificación |
|-----------|--------|---------------|
| `Procfile` | Eliminar | No se usa con container registry; genera confusión sobre qué ejecuta Heroku |
| `.buildpacks` | Eliminar | El deploy es por contenedor, no por buildpack de Go |

---

## 9. Criterios de aceptación

### CI (`ci.yml`)

| # | Criterio | Verificación |
|---|----------|--------------|
| AC-01 | CI se ejecuta en PRs con cambios en `backend/**` | Crear PR con cambio en `backend/`; verificar que aparece check de CI |
| AC-02 | CI NO se ejecuta en PRs con cambios solo fuera de `backend/` | Crear PR con cambio en `mobile/`; verificar que CI NO se dispara |
| AC-03 | CI ejecuta `go test ./...` en el directorio `backend/` | Inspeccionar logs del step de tests en GitHub Actions |
| AC-04 | CI usa Go 1.25 (serie 1.25.x) | Verificar en logs: `Setup Go 1.25.x` |
| AC-05 | CI falla si algún test falla | Introducir test roto en un PR; verificar check rojo |
| AC-06 | CI se ejecuta en push a `main` con cambios en `backend/**` | Hacer merge a `main`; verificar ejecución de CI |
| AC-07 | CI completa en menos de 10 minutos en condiciones normales | Observar duración del workflow en las primeras ejecuciones |

### Deploy (`deploy-heroku.yml`)

| # | Criterio | Verificación |
|---|----------|--------------|
| AC-08 | Deploy se ejecuta automáticamente cuando CI pasa en `main` | Verificar que `deploy-heroku.yml` se dispara tras CI exitoso |
| AC-09 | Deploy NO se ejecuta en PRs (solo en `main`) | Verificar que el deploy no aparece en checks de PR |
| AC-10 | Deploy NO se ejecuta si CI falla | Forzar fallo de CI en `main`; verificar que deploy no se dispara |
| AC-11 | La imagen `web` se construye desde `backend/Dockerfile` | Inspeccionar logs del step de build en GitHub Actions |
| AC-12 | La imagen `release` se construye desde `backend/Dockerfile.release` | Inspeccionar logs del step de build en GitHub Actions |
| AC-13 | Ambas imágenes se pushean al Heroku Container Registry | Verificar `docker push registry.heroku.com/...` exitoso en logs |
| AC-14 | `heroku container:release web release` se ejecuta sin errores | Verificar en logs del workflow |
| AC-15 | Release phase ejecuta migraciones antes de enrutar tráfico | Verificar en `heroku releases -a klyra-backend-prod` que release phase completó |
| AC-16 | Si la release phase falla, el deploy se aborta y la versión anterior sigue activa | Introducir migración inválida; verificar que la release falla y la app sigue en la versión previa |
| AC-17 | Health check post-deploy responde `200 OK` en `/health` | Verificar step de health check exitoso en logs del workflow |

### Secretos y configuración

| # | Criterio | Verificación |
|---|----------|--------------|
| AC-18 | `HEROKU_API_KEY` nunca aparece en texto plano en los logs | Inspeccionar logs del workflow; GitHub lo enmascara como `***` |
| AC-19 | `DATABASE_URL` NO está en GitHub Secrets | Verificar en GitHub → Settings → Secrets que solo están `HEROKU_API_KEY` y `HEROKU_APP_NAME` |
| AC-20 | Las config vars de Heroku (tabla 6.2) están configuradas antes del primer deploy | Ejecutar `heroku config -a klyra-backend-prod` y verificar presencia de todas |

### Persistencia y datos

| # | Criterio | Verificación |
|---|----------|--------------|
| AC-21 | La base de datos NO se resetea entre deploys | Insertar dato de prueba; hacer deploy; verificar que el dato persiste |
| AC-22 | Las migraciones son incrementales (solo aplican las pendientes) | Hacer dos deploys sucesivos sin nuevas migraciones; verificar que la release phase completa rápidamente sin errores |

### Artefactos legacy

| # | Criterio | Verificación |
|---|----------|--------------|
| AC-23 | `Procfile` eliminado del repositorio | `ls Procfile` no existe tras el merge |
| AC-24 | `.buildpacks` eliminado del repositorio | `ls .buildpacks` no existe tras el merge |

### Observabilidad

| # | Criterio | Verificación |
|---|----------|--------------|
| AC-25 | Los logs del workflow de CI son accesibles en GitHub → Actions | Navegar a la pestaña Actions del repo |
| AC-26 | Los logs del workflow de deploy son accesibles en GitHub → Actions | Navegar a la pestaña Actions del repo |
| AC-27 | Los logs de la release phase son accesibles en Heroku | Ejecutar `heroku releases -a klyra-backend-prod` y `heroku logs -a klyra-backend-prod --dyno release` |

---

## 10. Observabilidad

### 10.1 GitHub Actions

| Qué observar | Dónde |
|---------------|-------|
| Resultado de CI (tests) | GitHub → repo → Actions → workflow "CI" |
| Resultado de deploy | GitHub → repo → Actions → workflow "Deploy Heroku" |
| Logs detallados de cada step | Click en el run → expandir steps |
| Notificaciones de fallo | GitHub notifica al autor del commit por email (por defecto) |

### 10.2 Heroku

| Qué observar | Comando / Ubicación |
|---------------|---------------------|
| Historial de releases | `heroku releases -a klyra-backend-prod` |
| Estado de la última release | Columna `status` en el output de releases (`succeeded` / `failed`) |
| Logs de la release phase (migraciones) | `heroku logs -a klyra-backend-prod --dyno release` |
| Logs de la app en tiempo real | `heroku logs --tail -a klyra-backend-prod` |
| Health check liveness | `GET https://klyra-backend-prod.herokuapp.com/health` → `{"status":"ok"}` |
| Health check readiness (DB) | `GET https://klyra-backend-prod.herokuapp.com/health?check=db` → `{"status":"ok","db":"connected"}` |
| Dashboard de la app | `https://dashboard.heroku.com/apps/klyra-backend-prod` |

---

## 11. Fuera de alcance

| Tema | Razón |
|------|-------|
| Review Apps para PRs | Mejora futura; requiere `app.json` y configuración adicional |
| Pipeline multi-entorno (staging → producción) | Se asume un solo entorno `klyra-backend-prod` por ahora |
| CI/CD para `mobile/` (Flutter) | Pipeline independiente; no forma parte de este cambio |
| Dominio personalizado o SSL | Configuración de infraestructura separada |
| Docker layer caching en GitHub Actions | Optimización futura; el build actual (~2-3 min) es aceptable |
| Branch protection rules en GitHub | Recomendado pero se configura manualmente fuera de este cambio |

---

## 12. Diagrama de flujo completo

```
  PR → backend/**
       │
       ▼
  ┌──────────┐
  │  ci.yml  │  (test + lint)
  └────┬─────┘
       │
       ├── ✗ → Check rojo en PR (merge bloqueado)
       │
       └── ✓ → Check verde en PR
                    │
                    │ (merge a main)
                    ▼
  push main → backend/**
       │
       ▼
  ┌──────────┐
  │  ci.yml  │  (test en main)
  └────┬─────┘
       │
       ├── ✗ → Deploy NO se ejecuta
       │
       └── ✓ (workflow_run completed + success)
                    │
                    ▼
  ┌──────────────────────┐
  │  deploy-heroku.yml   │
  ├──────────────────────┤
  │ 1. checkout          │
  │ 2. install heroku    │
  │ 3. container:login   │
  │ 4. build web         │
  │ 5. build release     │
  │ 6. push web          │
  │ 7. push release      │
  │ 8. container:release │
  │    └─ release phase: │
  │       migraciones DB │
  │       ├── ✗ → abort  │
  │       └── ✓ → swap   │
  │ 9. health check      │
  └──────────────────────┘
```

---

## 13. Rollback

| Acción | Comando | Tiempo estimado |
|--------|---------|-----------------|
| Revertir al release anterior | `heroku rollback -a klyra-backend-prod` | < 1 minuto |
| Desactivar pipeline temporalmente | Deshabilitar workflow en GitHub → Actions → Disable | Inmediato |
| Rollback de migración destructiva | Ejecutar `.down.sql` manualmente vía `heroku pg:psql -a klyra-backend-prod` | Variable |
| Regenerar API key comprometida | Heroku Dashboard → Account → API Key + actualizar GitHub Secret | ~2 minutos |
