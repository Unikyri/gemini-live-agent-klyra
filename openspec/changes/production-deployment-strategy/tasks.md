# Tasks: Production Deployment Strategy (Heroku + GCP Ready)

## Bloque 1: Backend DB — Soporte `DATABASE_URL`

> **Dependencia**: Completar 1.1 antes de 1.2. El constructor `NewPostgreSQLRepositoryFromURL` debe existir antes de que `initDBRepository()` lo invoque.

- [ ] 1.1 Crear función `NewPostgreSQLRepositoryFromURL(databaseURL string) (ports.DBRepository, error)` en `backend/internal/infrastructure/database/postgresql_repository.go`. Abre GORM con `postgres.Open(databaseURL)` directamente (el driver `pgx` acepta URIs `postgres://...`). Retorna `*PostgreSQLRepository` con `config` vacío (no aplica campos individuales).
- [ ] 1.2 Modificar `initDBRepository()` en `backend/cmd/api/main.go`: añadir rama al inicio que lea `os.Getenv("DATABASE_URL")`. Si no está vacía, loguear `"Database mode: url (DATABASE_URL detected)"` y llamar a `database.NewPostgreSQLRepositoryFromURL(databaseURL)`. Si está vacía, continuar con el flujo actual `DB_MODE=local|cloud` sin cambios.
- [ ] 1.3 Verificar que el import de `"os"` ya está en `main.go` (sí lo está). No se requiere acción extra, solo validar que compila con `go build ./cmd/api/...`.

## Bloque 2: Migraciones en Release Phase — `RUN_MIGRATIONS_ONLY`

> **Dependencia**: Requiere bloque 1 completo (la conexión DB debe funcionar con `DATABASE_URL` antes de ejecutar migraciones).

- [ ] 2.1 Modificar `main()` en `backend/cmd/api/main.go`: después de `dbRepo.Ping()` y **antes** de cualquier wiring de usecases, añadir bloque condicional: si `os.Getenv("RUN_MIGRATIONS_ONLY")` es `"true"` (case-insensitive con `strings.EqualFold`), ejecutar `dbRepo.RunMigrations("./migrations")`, loguear resultado y llamar `os.Exit(0)`.
- [ ] 2.2 Eliminar la llamada actual a `dbRepo.RunMigrations("./migrations")` de la línea ~56 de `main()` (la que se ejecuta siempre en boot). Reemplazarla con un bloque condicional: si `ENV != production` (o `RUN_MIGRATIONS_ONLY` no está definido y no estamos en producción), ejecutar migraciones en boot como antes (retrocompatibilidad local). En producción, las migraciones solo corren vía release phase.
- [ ] 2.3 Verificar que `strings` ya está importado en `main.go` (sí lo está). Confirmar que compila.

## Bloque 3: Storage Guard + Credenciales GCS JSON Inline

> **Dependencia**: 3.1 y 3.2 son independientes entre sí, pero 3.3 depende de 3.2.

- [ ] 3.1 Añadir storage guard en `backend/cmd/api/main.go`, justo antes de la llamada a `initStorageService()` (~línea 99): definir `isProduction := strings.EqualFold(getEnv("ENV", "development"), "production")`. Si `isProduction && strings.EqualFold(getEnv("STORAGE_MODE", "gcs"), "local")`, llamar `log.Fatalf(...)` con mensaje explicativo sobre filesystem efímero de Heroku.
- [ ] 3.2 Crear archivo `backend/internal/repositories/gcp_credentials_helper.go` con función `ResolveGoogleCredentials(credsValue string, scopes ...string) (option.ClientOption, error)`. Lógica: si `credsValue` vacío → retorna `nil, nil` (ADC automático); si empieza con `{` → usa `google.CredentialsFromJSON`; si no → `option.WithCredentialsFile(credsValue)`. Imports necesarios: `context`, `fmt`, `strings`, `golang.org/x/oauth2/google`, `google.golang.org/api/option`.
- [ ] 3.3 Refactorizar `NewGCSStorageService()` en `backend/internal/repositories/gcs_storage_service.go`: modificar `UploadFile` para usar `ResolveGoogleCredentials(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"), "https://www.googleapis.com/auth/cloud-platform")` al crear el `storage.NewClient`. Si retorna `nil` ClientOption, omitir la option (ADC automático). Si retorna un ClientOption, pasarlo a `storage.NewClient(ctx, opt)`.

## Bloque 4: CORS Production-Safe

> **Dependencia**: Requiere que `isProduction` (de 3.1) esté definido antes de la configuración CORS. Reorganizar si es necesario para que `isProduction` sea una variable accesible en todo `main()`.

- [ ] 4.1 Modificar `allowedOriginFunc` en `backend/cmd/api/main.go` (~líneas 168-180): envolver el bloque de fallback a `localhost:*` con la condición `if !isProduction { ... }`. En producción, solo los origins explícitos de `ALLOWED_ORIGINS` son aceptados. Si `isProduction` es `true`, la función retorna `false` para cualquier origin no listado.
- [ ] 4.2 Verificar que el default de `ALLOWED_ORIGINS` solo aplica cuando la variable no está definida. El fallback actual a puertos de desarrollo (`3000,5173,...`) es correcto para local; en Heroku la variable estará explícitamente seteada.

## Bloque 5: Healthcheck con Verificación de DB (`?check=db`)

> **Dependencia**: Requiere bloque 1 (conexión DB funcional). El endpoint `/health` ya existe en línea ~191.

- [ ] 5.1 Modificar el handler `/health` en `backend/cmd/api/main.go`: si el query param `check` contiene `db` (e.g. `GET /health?check=db`), ejecutar `dbRepo.Ping()` y retornar `{"status":"ok","db":"connected"}` o `500 {"status":"degraded","db":"error: ..."}`. Sin `?check=db`, retornar el `200 {"status":"ok"}` actual (fast path para load balancers).
- [ ] 5.2 Asegurar que la variable `dbRepo` es accesible en el closure del handler (ya lo es, se define antes en `main()`).

## Bloque 6: Artefactos Heroku — `Procfile`, `app.json`, Dockerfile

> **Dependencia**: Independiente del código Go. Puede hacerse en paralelo con bloques 1-5. El `Procfile` referencia el binario compilado por el buildpack.

- [ ] 6.1 Crear `Procfile` en la **raíz del repo** (`gemini-live-agent-klyra/Procfile`) con dos procesos:
  ```
  web: backend/bin/klyra-backend
  release: RUN_MIGRATIONS_ONLY=true backend/bin/klyra-backend
  ```
  Nota: la ruta exacta del binario depende de `GO_INSTALL_PACKAGE_SPEC`; ajustar según el buildpack.
- [ ] 6.2 Crear `app.json` en la **raíz del repo** con: `name`, `description`, `buildpacks` (`heroku/go`), `addons` (`heroku-postgresql:essential-0`), `env` schema (variables requeridas: `GOOGLE_CLIENT_ID`, `JWT_SECRET`, `REFRESH_TOKEN_SECRET`, `GCS_BUCKET_NAME`, `GCP_PROJECT_ID`, `GOOGLE_APPLICATION_CREDENTIALS`, `ALLOWED_ORIGINS`; variables con default: `ENV=production`, `GIN_MODE=release`, `GOMODPATH=backend`, `GO_INSTALL_PACKAGE_SPEC=./cmd/api`, `STORAGE_MODE=gcs`), `formation` (`web: quantity 1, size basic`).
- [ ] 6.3 Actualizar `backend/Dockerfile`: cambiar `golang:1.22-alpine` → `golang:1.25-alpine` (match con `go.mod`). Añadir `COPY --from=builder /app/migrations ./migrations` en el stage `runner` para que el binario Docker tenga acceso a las migraciones en release phase.

## Bloque 7: Mobile `--dart-define` — `GOOGLE_WEB_CLIENT_ID`, `API_BASE_URL`

> **Dependencia**: Independiente del backend. Puede hacerse en paralelo.

- [ ] 7.1 Modificar `mobile/lib/features/auth/data/auth_remote_datasource.dart` línea 20: reemplazar el string literal `'782011204480-0eejl4shc1f9n360mln5secbeng6k5gb.apps.googleusercontent.com'` por una constante que use `String.fromEnvironment('GOOGLE_WEB_CLIENT_ID', defaultValue: '782011204480-0eejl4shc1f9n360mln5secbeng6k5gb.apps.googleusercontent.com')`. Esto permite override en build con `--dart-define=GOOGLE_WEB_CLIENT_ID=...`.
- [ ] 7.2 Modificar `mobile/lib/core/config/env.dart` — clase `EnvInfo`: añadir campo estático `static const _apiBaseUrlOverride = String.fromEnvironment('API_BASE_URL');`. En el getter `backendBaseUrl`, si `_apiBaseUrlOverride.isNotEmpty`, retornar `_apiBaseUrlOverride` inmediatamente (antes de la detección de plataforma actual). Esto permite builds de producción con `--dart-define=API_BASE_URL=https://klyra-backend.herokuapp.com/api/v1`.
- [ ] 7.3 (Opcional) Añadir constante `static const googleWebClientId` en `env.dart` también, centralizando la lectura de `String.fromEnvironment('GOOGLE_WEB_CLIENT_ID')` en un solo sitio en vez de en `auth_remote_datasource.dart`.

## Bloque 8: Tests Unitarios

> **Dependencia**: Los tests se escriben **después** de implementar la funcionalidad correspondiente de cada bloque.

### Tests Go

- [ ] 8.1 Añadir test en `backend/internal/infrastructure/database/postgresql_repository_test.go`: `TestNewPostgreSQLRepositoryFromURL_InvalidURL` — verificar que pasar un DSN malformado retorna error (no nil). No requiere DB real.
- [ ] 8.2 Añadir test de integración (con tag o skip si DB no disponible): `TestNewPostgreSQLRepositoryFromURL_ConnectsWithDSN` — usar `DATABASE_URL=postgres://klyra_user:klyra_pass@localhost:5433/klyra_db?sslmode=disable`, verificar que `Ping()` no falla.
- [ ] 8.3 Crear test en `backend/cmd/api/main_test.go` (o archivo nuevo `backend/cmd/api/init_test.go`): testear que `initDBRepository()` prioriza `DATABASE_URL` sobre `DB_MODE`. Setear `os.Setenv("DATABASE_URL", ...)` y verificar la rama correcta (puede requerir refactorizar `initDBRepository` a función testable que acepte un getter de env, o testear indirectamente verificando logs).
- [ ] 8.4 Crear test en `backend/internal/repositories/gcp_credentials_helper_test.go`: testear las 3 ramas de `ResolveGoogleCredentials`:
  - Cadena vacía → retorna `nil, nil`.
  - JSON inline (e.g. `{"type":"service_account",...}`) → retorna `option.WithCredentials(...)` sin error (mock JSON mínimo).
  - Ruta de archivo (e.g. `/tmp/key.json`) → retorna `option.WithCredentialsFile(...)`.
- [ ] 8.5 Crear test para CORS production-safe: verificar que `allowedOriginFunc` con `isProduction=true` rechaza `http://localhost:3000` y acepta un origin explícito configurado. Puede requerir extraer la lógica CORS a función testable.
- [ ] 8.6 Crear test para storage guard: verificar que cuando `ENV=production` y `STORAGE_MODE=local`, el sistema emite un error fatal. Refactorizar el guard a una función pura `validateStorageMode(env, storageMode string) error` para hacerlo testable sin `log.Fatalf`.

### Tests Flutter (Dart)

- [ ] 8.7 Crear test simple en `mobile/test/features/auth/env_defines_test.dart`: verificar que `EnvInfo.backendBaseUrl` retorna el override cuando `API_BASE_URL` está definido. Nota: `String.fromEnvironment` es compile-time, así que el test debe compilarse con `--dart-define=API_BASE_URL=https://test.example.com/api/v1` o verificar el comportamiento por defecto (fallback a detección de plataforma).

## Bloque 9: Documentación

- [ ] 9.1 Crear o actualizar `docs/deployment.md` (o sección en `README.md` raíz) con:
  - Config Matrix completa: variable → valor local / Heroku / GCP Cloud Run.
  - Instrucciones paso a paso para deploy en Heroku (crear app, addons, config vars, push).
  - Instrucciones de build mobile con `--dart-define` para producción.
  - Sección de troubleshooting: `DATABASE_URL` con SSL, credenciales GCS inline, storage mode guard.
- [ ] 9.2 Documentar en `docs/deployment.md` la sección de seguridad de secretos: lineamientos para generar `JWT_SECRET` y `REFRESH_TOKEN_SECRET` (mínimo 32 caracteres, `openssl rand -base64 32`), manejo de `GOOGLE_APPLICATION_CREDENTIALS` como JSON inline en Heroku (nunca commitear al repo).
- [ ] 9.3 Documentar el rollback plan: cómo revertir migraciones (`migrate down`), cómo funciona el release phase de Heroku (si falla, slug anterior se mantiene), y que todos los cambios son aditivos (revertibles con `git revert`).

## Bloque 10: Verificación Manual (Smoke Tests)

> **No ejecutar realmente**: estas son tareas de verificación post-deploy para validar el sistema en Heroku. Se listan como checklist para el equipo.

- [ ] 10.1 **Healthcheck básico**: `curl https://<app>.herokuapp.com/health` → esperar `200 {"status":"ok"}`.
- [ ] 10.2 **Healthcheck con DB**: `curl https://<app>.herokuapp.com/health?check=db` → esperar `200 {"status":"ok","db":"connected"}`.
- [ ] 10.3 **Release phase**: verificar que las migraciones corrieron en release phase via `heroku releases:output` — buscar logs `"RUN_MIGRATIONS_ONLY=true"` y `"Migrations completed successfully"`.
- [ ] 10.4 **Auth Google desde mobile**: build mobile con `--dart-define=API_BASE_URL=https://<app>.herokuapp.com/api/v1 --dart-define=GOOGLE_WEB_CLIENT_ID=782011204480-...`, completar login OAuth → esperar JWT válido.
- [ ] 10.5 **Guest login**: desde mobile apuntando a Heroku, login como guest → esperar JWT válido y usuario con `auth_provider=guest`.
- [ ] 10.6 **Upload a GCS**: subir un PDF a un topic → verificar que la URL retornada es `https://storage.googleapis.com/<bucket>/...` y es accesible.
- [ ] 10.7 **Storage guard**: verificar que si se setea `STORAGE_MODE=local` en Heroku config, el dyno falla al arrancar con mensaje claro en logs.
- [ ] 10.8 **CORS en producción**: desde un browser, hacer un fetch a la API desde un origin no listado en `ALLOWED_ORIGINS` → verificar que el preflight CORS falla (403 o sin cabecera `Access-Control-Allow-Origin`).
- [ ] 10.9 **Regresión local**: ejecutar `go run ./cmd/api/main.go` localmente sin `DATABASE_URL` ni `RUN_MIGRATIONS_ONLY` → verificar que el flujo local funciona igual que antes (migraciones en boot, `STORAGE_MODE=local` permitido).
- [ ] 10.10 **Logs estructurados**: verificar en `heroku logs --tail` que con `GIN_MODE=release` no aparece debug output de Gin.
