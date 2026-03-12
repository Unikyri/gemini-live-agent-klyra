# Proposal: Production Deployment Strategy (Heroku + GCP Ready)

## Intent

Klyra necesita salir a producción de forma inmediata en **Heroku** para validación con usuarios reales, sin sacrificar el soporte de desarrollo local ni cerrar la puerta a una migración futura a **GCP (Cloud Run + Cloud SQL + GCS)**.

Hoy el backend funciona bien localmente, pero tiene fricciones concretas que impiden un deploy directo a PaaS:

1. `initDBRepository()` no consume `DATABASE_URL` (la convención de Heroku Postgres).
2. `STORAGE_MODE=local` escribe a disco efímero, incompatible con dynos de Heroku.
3. El `serverClientId` de Google Sign-In está hardcodeado en el código mobile (`782011204480-...`), lo cual acopla la build a un solo Client ID.
4. No existen artefactos de despliegue (`Procfile`, `app.json`) ni documentación de variables de entorno para producción.
5. Las migraciones se ejecutan en cada boot del proceso web, lo que puede causar race conditions con múltiples dynos o aumentar el cold start.

Este cambio aplica el patrón **12-Factor App + Provider Adapters** al composition root (`main.go`) y genera los artefactos necesarios para un deploy reproducible.

## Scope

### In Scope

- **Soporte `DATABASE_URL`** en el backend con fallback al esquema actual (`DB_MODE` + variables individuales).
- **Migración en release phase** de Heroku (separar migraciones del boot del web dyno).
- **Artefactos Heroku**: `Procfile`, `app.json` (buildpacks, addons, config vars schema), script de release/postdeploy.
- **CORS para producción**: `ALLOWED_ORIGINS` debe aceptar el dominio de Heroku y el deep-link scheme de la app mobile; eliminar el fallback permisivo a `localhost:*` cuando `ENV=production`.
- **Healthcheck robusto**: el endpoint `/health` ya existe; documentar su uso y verificar que retorna status de DB.
- **Logging estructurado** (al menos JSON en producción via `GIN_MODE=release`).
- **Storage mode enforcement**: cuando `ENV=production`, prevenir que `STORAGE_MODE=local` se active accidentalmente (warn o error).
- **Mobile: externalizar `serverClientId`**: mover el Web OAuth Client ID a `--dart-define=GOOGLE_WEB_CLIENT_ID` para manejar dev/prod sin recompilar.
- **Documentación de secretos y variables**: config matrix con valores esperados por entorno.
- **Seguridad de credenciales**: lineamientos para `JWT_SECRET`, `REFRESH_TOKEN_SECRET`, `GOOGLE_CLIENT_ID`, `GOOGLE_APPLICATION_CREDENTIALS` (contenido JSON como config var en Heroku).

### Out of Scope

- **Migración completa a GCP** (Cloud Run, Cloud SQL connector, IAM, VPC). Se deja preparado pero no se ejecuta ahora.
- **Adaptador S3** para storage. Se usa GCS como proveedor unificado; S3 queda como opción futura si se necesita.
- **Web OAuth / Flutter Web**: no se configura login por navegador; solo mobile nativo.
- **CI/CD pipelines** (GitHub Actions, Heroku Pipelines). Se documenta el flujo manual; la automatización es un cambio separado.
- **Custom domain / SSL certificates**: Heroku provee HTTPS por defecto en `*.herokuapp.com`.
- **Monitoreo / APM avanzado** (Datadog, Sentry). Se puede agregar después como addon.

## Approach

### Patrón: 12-Factor + Provider Adapters

Se mantiene el composition root en `main.go` como único punto de decisión de infraestructura. La lógica se resume en: **"si existe la variable estándar del PaaS, úsala; si no, recurre al esquema local/cloud actual"**.

**Base de datos**: `initDBRepository()` gana una rama nueva al inicio: si `DATABASE_URL` está definida, se parsea y se usa directamente (con `?sslmode=require` que Heroku inyecta). Si no existe, el flujo `DB_MODE=local|cloud` sigue intacto. Esto permite que Heroku, Docker Compose y desarrollo local coexistan sin cambios en los archivos `.env` de cada entorno.

**Storage**: en producción se fuerza `STORAGE_MODE=gcs`. El composition root emitirá un warning (o fallo, según se defina) si detecta `ENV=production` con `STORAGE_MODE=local`. GCS funciona tanto desde Heroku (con `GOOGLE_APPLICATION_CREDENTIALS` como JSON en config var) como desde GCP (con Application Default Credentials).

**Autenticación Google**: el backend ya verifica el ID token contra `GOOGLE_CLIENT_ID`. El mobile necesita que su `serverClientId` coincida con ese valor. Se externaliza a `--dart-define` para poder compilar builds de dev y prod contra diferentes Client IDs si fuese necesario.

### Artefactos de Despliegue

Se crean en la raíz del repo (o `backend/` según convenga para el buildpack):

| Artefacto | Propósito |
|-----------|-----------|
| `Procfile` | Define `web:` (binario Go) y `release:` (migraciones) |
| `app.json` | Declara addons (`heroku-postgresql`), buildpacks (`heroku/go`), env schema |
| `scripts/heroku-release.sh` | Ejecuta migraciones en release phase, fuera del boot del dyno web |

### Config Matrix (Variables de Entorno)

| Variable | Local (dev) | Heroku | GCP (Cloud Run) |
|----------|-------------|--------|-----------------|
| `ENV` | `development` | `production` | `production` |
| `PORT` | `8080` | *dinámico* (Heroku lo asigna) | `8080` |
| `GIN_MODE` | `debug` | `release` | `release` |
| `DATABASE_URL` | *(no seteada)* | Auto (addon Heroku Postgres) | *(no seteada, usa Cloud SQL connector)* |
| `DB_MODE` | `local` | *(ignorado si `DATABASE_URL` existe)* | `cloud` |
| `DB_HOST/PORT/NAME/USER/PASSWORD` | valores locales | *(ignorados si `DATABASE_URL` existe)* | *(ignorados, usa socket)* |
| `STORAGE_MODE` | `local` | `gcs` | `gcs` |
| `GCS_BUCKET_NAME` | *(opcional)* | nombre del bucket | nombre del bucket |
| `GOOGLE_APPLICATION_CREDENTIALS` | `./key.json` | JSON content como config var | *ADC automático* |
| `GOOGLE_CLIENT_ID` | Web OAuth Client ID | Web OAuth Client ID | Web OAuth Client ID |
| `JWT_SECRET` | secreto local | secreto fuerte (config var) | Secret Manager |
| `REFRESH_TOKEN_SECRET` | secreto local | secreto fuerte (config var) | Secret Manager |
| `ALLOWED_ORIGINS` | `http://localhost:3000,...` | `https://klyra-api-xyz.herokuapp.com` | `https://klyra.run.app` |
| `GCP_PROJECT_ID` | project-id | project-id (mismo proyecto GCS) | project-id |

## Affected Areas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `backend/cmd/api/main.go` | Modified | `initDBRepository()` soporta `DATABASE_URL`; CORS stricter en prod; storage mode guard; logging config |
| `backend/internal/infrastructure/database/` | Modified | Nueva función/constructor que acepta DSN raw (`DATABASE_URL`) |
| `backend/migrations/` | Modified | Se invocará desde release phase en vez de boot |
| `backend/Procfile` | New | Define procesos `web` y `release` para Heroku |
| `backend/app.json` | New | Manifiesto Heroku: addons, buildpacks, env schema |
| `backend/scripts/heroku-release.sh` | New | Script de release phase para ejecutar migraciones |
| `mobile/lib/features/auth/data/auth_remote_datasource.dart` | Modified | `serverClientId` se lee de `--dart-define` en vez de estar hardcodeado |
| `mobile/lib/core/config/env.dart` | Modified | Nueva constante `googleWebClientId` desde dart-define |
| `docs/deployment.md` o `README` | New/Modified | Documentación de deploy, variables, setup de Google OAuth |

## Risks

| Riesgo | Probabilidad | Mitigación |
|--------|-------------|------------|
| `DATABASE_URL` parsing incorrecto (params SSL, sslmode, encoding) | Media | Usar `net/url` o el parser de `pgx`/`lib/pq` que ya maneja URIs estándar; test con URL real de Heroku |
| Filesystem efímero usado accidentalmente en producción | Baja | Guard en `main.go`: si `ENV=production` y `STORAGE_MODE=local`, log.Fatal con mensaje claro |
| `GOOGLE_APPLICATION_CREDENTIALS` como JSON string en Heroku en vez de archivo | Media | Detectar si el valor es JSON inline (empieza con `{`) y escribir a archivo temporal, o usar `google.CredentialsFromJSON` |
| Race condition en migraciones si hay múltiples dynos arrancando | Baja | Usar `release` phase (solo un proceso lo ejecuta) + advisory lock en la migración |
| CORS demasiado restrictivo bloquea requests legítimos del mobile | Baja | Mobile usa HTTP directo (no browser CORS), pero documentar que CORS aplica a Flutter Web futuro; el mobile nativo no está sujeto a CORS |
| Cold start en Heroku eco/basic tier | Media | Aceptar tradeoff para MVP; documentar upgrade a Standard-1x si latencia es crítica |

## Rollback Plan

1. **Feature flag implícito**: la rama `DATABASE_URL` en `initDBRepository()` solo se activa cuando la variable existe. Si se necesita revertir en Heroku, basta con remover el addon de Postgres (pero esto destruye datos) o apuntar `DATABASE_URL` a otra instancia.
2. **Procfile y release phase**: si la release phase falla, Heroku no promueve el slug → el dyno anterior sigue corriendo. No hay downtime.
3. **Mobile**: el `--dart-define` tiene fallback al Client ID actual hardcodeado, por lo que builds antiguas siguen funcionando.
4. **Migraciones reversibles**: cada migración debe tener su archivo `.down.sql`. En caso de error, ejecutar rollback manual con `migrate down`.
5. **Revert rápido**: todos los cambios son aditivos (nuevos constructores, nueva rama en `init`). Un `git revert` del commit deja el sistema en el estado anterior sin efectos secundarios.

## Dependencies

- **Heroku Account + Heroku Postgres addon** para validar el flujo completo (billing activo).
- **Google Cloud Project** con:
  - OAuth 2.0 Web Client ID configurado (ya existe: `782011204480-...`).
  - GCS bucket creado y accesible por el service account.
  - Vertex AI APIs habilitadas (Embedding, Imagen) si se quiere RAG y generación de imágenes en producción.
- **`GOOGLE_APPLICATION_CREDENTIALS`** del service account con permisos de GCS + Vertex AI, exportado como JSON para Heroku config var.
- **Go buildpack de Heroku** (`heroku/go`) compatible con la versión de Go del proyecto (verificar `go.mod`).

## Rollout Plan

### Fase 1: Preparación del código (este cambio)
1. Implementar soporte `DATABASE_URL` en `initDBRepository()`.
2. Crear `Procfile`, `app.json`, `scripts/heroku-release.sh`.
3. Agregar storage mode guard para producción.
4. Externalizar `serverClientId` en mobile.
5. Documentar variables de entorno y setup.

### Fase 2: Deploy a Heroku (manual, post-merge)
1. Crear app Heroku, agregar addon Postgres.
2. Configurar config vars (ver Config Matrix).
3. `git push heroku main` (o deploy branch).
4. Verificar release phase (migraciones).
5. Ejecutar Test Plan (ver abajo).

### Fase 3: Validación con usuarios
1. Compartir URL de Heroku con testers.
2. Monitorear logs (`heroku logs --tail`).
3. Iterar sobre bugs encontrados.

### Fase 4: (Futura) Migración a GCP
1. Crear Cloud Run service + Cloud SQL instance.
2. Usar el mismo código con `DB_MODE=cloud` y ADC.
3. Cutover DNS / mobile API URL.

## Test Plan

### Smoke Tests (post-deploy)

- [ ] `GET /health` retorna `200 {"status": "ok"}`.
- [ ] `GET /api/v1/` retorna respuesta coherente (puede ser 401 sin token).
- [ ] Logs en Heroku muestran startup correcto sin errores fatales.

### Autenticación Google

- [ ] Desde la app mobile (build con `--dart-define=GOOGLE_WEB_CLIENT_ID=...`), login con Google completa sin errores.
- [ ] El backend verifica el ID token correctamente y retorna JWT.
- [ ] Guest login sigue funcionando.

### Upload de Material

- [ ] Subir un archivo PDF desde mobile a un curso/topic.
- [ ] Verificar que el archivo se almacena en GCS (no en filesystem local).
- [ ] Verificar que la URL de descarga funciona.

### Sesión de Tutor (Gemini Live)

- [ ] Iniciar una sesión de tutor desde mobile.
- [ ] Verificar que la conexión WebSocket/streaming con Gemini Live funciona (esta conexión es mobile → Google, no pasa por el backend, pero la creación de sesión sí).

### RAG (si Vertex AI está configurado)

- [ ] `POST /api/v1/rag/search` retorna resultados relevantes.
- [ ] Si Vertex AI no está configurado, el endpoint responde con error graceful (no crash).

### Regresión Local

- [ ] `docker-compose up` o `go run` local sigue funcionando sin `DATABASE_URL`.
- [ ] `STORAGE_MODE=local` sigue sirviendo archivos en `/static`.

## Success Criteria

- [ ] El backend arranca correctamente en Heroku consumiendo `DATABASE_URL` del addon Postgres.
- [ ] Un usuario puede completar el flujo login → crear curso → subir material → sesión tutor desde la app mobile apuntando a la URL de Heroku.
- [ ] El desarrollo local no sufre regresiones: `go run` y `STORAGE_MODE=local` funcionan igual que antes.
- [ ] Las migraciones se ejecutan en release phase, no en cada boot del web dyno.
