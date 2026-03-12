# Proposal: CI/CD con GitHub Actions y Heroku Container Registry

## Intent

El proyecto `gemini-live-agent-klyra` carece de un pipeline de integración y despliegue continuo. Actualmente, cualquier deploy a `klyra-backend-prod` requiere intervención manual. Esto introduce riesgo de errores humanos, dificulta la trazabilidad de qué código está en producción y ralentiza el ciclo de entrega.

Se necesita un pipeline automatizado que:
1. Ejecute tests y validaciones en cada cambio al backend.
2. Despliegue automáticamente a Heroku cuando los tests pasen en `main`.
3. Garantice que las migraciones de base de datos se ejecuten de forma segura antes de que el nuevo código reciba tráfico.

## Scope

### In Scope
- Crear workflow de CI (`.github/workflows/ci.yml`) que ejecute `go test ./...` y opcionalmente lints en PRs y pushes al backend.
- Crear workflow de deploy (`.github/workflows/deploy-heroku.yml`) que construya la imagen Docker, la suba al Heroku Container Registry y haga release tras pasar CI en `main`.
- Crear `backend/Dockerfile.release` para la release phase (migraciones vía `RUN_MIGRATIONS_ONLY=true`).
- Documentar los secretos necesarios en GitHub y las config vars que deben permanecer en Heroku.
- Garantizar que la base de datos PostgreSQL (addon `essential-0`) persista entre deploys sin reset.

### Out of Scope
- Heroku Review Apps para PRs (mejora futura).
- Pipeline multi-entorno (staging → producción); se asume un solo entorno `klyra-backend-prod`.
- Migración a otro proveedor de hosting (Fly.io, Railway, etc.).
- CI para el directorio `mobile/` (frontend Flutter).
- Configuración de dominio personalizado o SSL adicional.

## Approach

### Opción elegida: GitHub Actions + Heroku Container Registry

Se usará GitHub Actions como motor de CI/CD con deploy vía `heroku container:push` / `heroku container:release`. Esta opción fue recomendada en la exploración por las siguientes razones:

- **Control total del pipeline**: se pueden ejecutar tests de Go, lints (`golangci-lint`) y cualquier validación antes de tocar Heroku.
- **Manejo nativo de monorepos**: el workflow filtra por `paths: backend/**` y usa `--context-path backend` o `-f backend/Dockerfile` sin buildpacks adicionales.
- **Imagen Docker determinista**: el `backend/Dockerfile` ya existe y produce una imagen reproducible basada en `distroless`.
- **Release phase segura**: Heroku ejecuta el contenedor `release` antes de enrutar tráfico al nuevo `web`. Si las migraciones fallan, el deploy se aborta y la versión anterior sigue viva.

### Alternativa descartada: Heroku GitHub Integration (dashboard)

La integración nativa de Heroku con GitHub (auto-deploy desde dashboard) es más sencilla de configurar, pero presenta limitaciones críticas para este proyecto:

- En monorepos requiere `heroku-buildpack-monorepo` y la variable `PROJECT_PATH=backend`, lo cual añade complejidad implícita.
- No permite ejecutar tests de Go previos al deploy (solo "wait for CI" genérico).
- Menor control sobre el proceso de build y release.
- Menos visibilidad y trazabilidad que un workflow YAML versionado en el repo.

### Arquitectura del pipeline

```
PR → backend/**  ──→  ci.yml (test + lint)  ──→  ✓/✗ status check
                                                       │
push main → backend/** ──→  ci.yml (test + lint)       │
                               │ (si pasa)             │
                               ▼                       │
                         deploy-heroku.yml             │
                           ├─ docker build web         │
                           ├─ docker build release     │
                           ├─ docker push (registry)   │
                           ├─ heroku container:release  │
                           │    └─ release phase:      │
                           │       migraciones DB      │
                           └─ verify /health           │
```

### Detalle de los workflows

**`ci.yml`** — Se ejecuta en PRs y pushes a `main` cuando hay cambios en `backend/**`:
1. Checkout del código.
2. Setup Go 1.25.
3. Descarga de dependencias (`go mod download`).
4. Ejecución de tests: `go test ./...`.
5. (Opcional) Ejecución de linter: `golangci-lint run`.

**`deploy-heroku.yml`** — Se ejecuta solo en pushes a `main` cuando hay cambios en `backend/**`, y solo si CI pasa:
1. Checkout del código.
2. Login al Heroku Container Registry (`heroku container:login` usando `HEROKU_API_KEY`).
3. Build y push de la imagen `web`: `docker build -t registry.heroku.com/$APP/web -f backend/Dockerfile backend/`.
4. Build y push de la imagen `release`: `docker build -t registry.heroku.com/$APP/release -f backend/Dockerfile.release backend/`.
5. Push de ambas imágenes al registry.
6. Release: `heroku container:release web release -a $APP`.
7. Verificación post-deploy: `curl https://klyra-backend-prod.herokuapp.com/health`.

### Gestión de migraciones (release phase)

Se creará `backend/Dockerfile.release`, idéntico al `Dockerfile` principal pero con `ENV RUN_MIGRATIONS_ONLY=true`. Cuando Heroku ejecuta la release phase:
1. Arranca el contenedor `release`.
2. El binario detecta `RUN_MIGRATIONS_ONLY=true` (línea 58 de `main.go`).
3. Ejecuta las migraciones SQL desde `./migrations`.
4. Sale con `os.Exit(0)` si todo va bien, o `log.Fatalf` si falla.
5. Si falla (exit code ≠ 0), Heroku aborta el deploy y mantiene la versión anterior.

### Gestión de secretos

**GitHub Repository Secrets** (configurar manualmente en Settings → Secrets):

| Secreto | Descripción |
|---------|-------------|
| `HEROKU_API_KEY` | Token de la cuenta Heroku con acceso a `klyra-backend-prod` |
| `HEROKU_APP_NAME` | `klyra-backend-prod` (o variable de entorno en el workflow) |

**Heroku Config Vars** (persisten entre deploys, no se tocan desde CI):

| Variable | Origen | Descripción |
|----------|--------|-------------|
| `DATABASE_URL` | Addon Postgres | Inyectada automáticamente por Heroku; el backend la detecta con precedencia (línea 333 de `main.go`) |
| `ENV` | Manual | `production` |
| `GIN_MODE` | Manual | `release` |
| `JWT_SECRET` | Manual | Secreto para tokens JWT |
| `REFRESH_TOKEN_SECRET` | Manual | Secreto para refresh tokens |
| `GOOGLE_CLIENT_ID` | Manual | ID de cliente OAuth de Google |
| `ALLOWED_ORIGINS` | Manual | Orígenes permitidos (CORS) |
| `RUN_MIGRATIONS_ON_BOOT` | Manual | `false` en producción (las migraciones corren en release phase) |
| `STORAGE_MODE` | Manual | `gcs` (obligatorio en producción) |
| `GCP_PROJECT_ID` | Manual | Proyecto de GCP para Vertex AI / GCS |
| `GOOGLE_APPLICATION_CREDENTIALS_JSON` | Manual | Credenciales de servicio GCP |

**Regla crítica**: `DATABASE_URL` nunca se pone en GitHub Secrets. Heroku la gestiona automáticamente a través del addon PostgreSQL.

### Persistencia de la base de datos

La DB **no se resetea** en deploys. Garantías:
1. El addon `heroku-postgresql:essential-0` ya está provisionado (estado `created`). Solo se destruye con `heroku addons:destroy` explícito.
2. `heroku container:release` reemplaza la imagen del dyno, no toca la DB.
3. Las migraciones en release phase son incrementales (archivos `000001_...up.sql`, `000002_...up.sql`, etc.) y solo aplican los que no se han ejecutado.
4. En producción, `RUN_MIGRATIONS_ON_BOOT=false` evita que el proceso `web` ejecute migraciones al arrancar.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `.github/workflows/ci.yml` | New | Workflow de CI: tests y lints para backend |
| `.github/workflows/deploy-heroku.yml` | New | Workflow de deploy a Heroku Container Registry |
| `backend/Dockerfile.release` | New | Imagen Docker para release phase (migraciones) |
| `backend/Dockerfile` | No change | Ya existe y es compatible; no requiere modificación |
| `Procfile` | Modified | Actualizar rutas para alinearse con el binario del contenedor (`/app/klyra-backend`) o marcar como legacy |
| `.buildpacks` | Removed | Ya no se necesita el buildpack de Go; se usa container registry |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| **Leak del `HEROKU_API_KEY`** si se expone en logs del workflow | Low | Usar `${{ secrets.HEROKU_API_KEY }}` que GitHub enmascara automáticamente. Nunca hacer `echo` del token. |
| **Build lento** en GitHub Actions (imagen Docker multi-stage) | Medium | Usar Docker layer caching (`actions/cache` o `docker/build-push-action` con cache). Build actual toma ~2-3 min para Go. |
| **Falso positivo en path filter**: un cambio solo en `mobile/` dispara deploy | Low | El filtro `paths: backend/**` en el trigger del workflow previene esto. Verificar en el primer PR. |
| **Migración destructiva** rompe la DB en release phase | Medium | Revisar migraciones en PR (code review). Siempre escribir migraciones reversibles con archivos `.down.sql`. Probar localmente antes de merge. |
| **Rate limit de Heroku API** durante pushes frecuentes | Low | El rate limit de Heroku es generoso (~4500 req/h). Un pipeline normal hace <10 llamadas. Monitorear si se hacen muchos merges seguidos. |
| **Divergencia Procfile vs Container**: confusión sobre qué ejecuta Heroku | Medium | Documentar claramente que con container registry el `Procfile` no se usa para el comando `web`. Mantener `Procfile` como referencia legacy o eliminarlo. |

## Rollback Plan

1. **Rollback inmediato (< 1 min)**: Usar `heroku releases -a klyra-backend-prod` para identificar la release anterior y ejecutar `heroku rollback -a klyra-backend-prod` para revertir al contenedor previo.
2. **Desactivar CI/CD**: Si el pipeline tiene problemas sistémicos, desactivar el workflow desde GitHub (Settings → Actions → disable workflow) o eliminar/renombrar el archivo YAML. Esto no afecta la app en Heroku.
3. **Rollback de migraciones**: Si una migración causó problemas, ejecutar el archivo `.down.sql` correspondiente manualmente: `heroku run -a klyra-backend-prod -- /app/klyra-backend` con el flag apropiado, o conectarse a la DB con `heroku pg:psql`.
4. **Regenerar `HEROKU_API_KEY`**: Si se sospecha compromiso del token, regenerar en Heroku Dashboard → Account → API Key y actualizar el secreto en GitHub.

## Dependencies

- **`HEROKU_API_KEY`** debe generarse y agregarse como secreto en el repositorio de GitHub antes del primer deploy.
- **`HEROKU_APP_NAME`** (`klyra-backend-prod`) debe configurarse como secreto o variable de entorno en el workflow.
- **Heroku CLI** debe estar disponible en el runner de GitHub Actions (se instala vía `heroku/cli` o `npm install -g heroku`).
- **Config vars de producción** deben estar configuradas en Heroku antes del primer deploy automatizado (especialmente `ENV=production`, `RUN_MIGRATIONS_ON_BOOT=false`, `JWT_SECRET`, etc.).
- El `backend/Dockerfile` actual ya es compatible y no requiere cambios.

## Test Plan

1. **Validar CI en PR**: Crear un PR con un cambio trivial en `backend/` y verificar que `ci.yml` ejecuta tests y reporta status check.
2. **Validar filtro de paths**: Crear un PR con un cambio solo en `mobile/` y verificar que los workflows de backend NO se disparan.
3. **Primer deploy**: Hacer merge a `main` con un cambio en backend y verificar:
   - El workflow `deploy-heroku.yml` se dispara.
   - La imagen se construye y sube al registry.
   - La release phase ejecuta migraciones (verificar en `heroku releases -a klyra-backend-prod` y `heroku logs --tail -a klyra-backend-prod`).
   - El endpoint `/health` responde `200 OK`.
   - El endpoint `/health?check=db` confirma conexión a la DB.
4. **Validar persistencia de datos**: Verificar que los datos existentes en la DB siguen intactos después del deploy.
5. **Simular fallo de migración**: (En un branch de prueba) Introducir una migración inválida y verificar que la release phase falla y el deploy se aborta.

## Observabilidad y Monitoreo

- **GitHub Actions**: Logs detallados de cada step en la pestaña Actions del repo.
- **Heroku Release Logs**: `heroku releases -a klyra-backend-prod` muestra el historial de releases con estado (succeeded/failed).
- **Heroku App Logs**: `heroku logs --tail -a klyra-backend-prod` para ver logs en tiempo real, incluyendo output de migraciones.
- **Health Check**: `GET /health` (liveness) y `GET /health?check=db` (readiness con verificación de DB).
- **Alertas**: Configurar notificación en GitHub Actions para builds fallidos (por defecto, GitHub notifica al autor del commit).

## Success Criteria

- [ ] Push a `main` con cambios en `backend/**` dispara automáticamente build + test + deploy sin intervención manual.
- [ ] Si los tests fallan, el deploy NO se ejecuta y el PR/push muestra status check rojo.
- [ ] Las migraciones se ejecutan en release phase antes de que el nuevo código reciba tráfico; si fallan, el deploy se aborta.
- [ ] La base de datos persiste todos los datos entre deploys sucesivos (no hay reset ni pérdida de datos).
- [ ] El endpoint `/health` responde `200` después de cada deploy exitoso.
