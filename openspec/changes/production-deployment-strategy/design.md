# Design: Production Deployment Strategy (Heroku + GCP Ready)

## Enfoque Técnico

Aplicar el patrón **12-Factor App + Provider Adapters** al composition root existente (`backend/cmd/api/main.go`) para que el mismo binario Go funcione en **desarrollo local**, **Heroku** y **GCP Cloud Run** sin cambios de código entre entornos. Toda decisión de infraestructura se resuelve por variables de entorno con precedencia bien definida.

El diseño mantiene la arquitectura hexagonal actual (puertos y adaptadores) y solo añade:

1. Una nueva ruta de conexión a DB vía `DATABASE_URL` (Heroku Postgres).
2. Un comando `migrate` separado del proceso `web` para release phase.
3. Artefactos de despliegue Heroku (`Procfile`, `app.json`).
4. Guards de producción para storage y CORS.
5. Externalización del `serverClientId` de Google Sign-In en el mobile.

---

## Decisiones de Arquitectura

### Decisión 1: `DATABASE_URL` como fuente primaria de conexión

**Elección**: Si `DATABASE_URL` está definida, usarla directamente como DSN para `gorm.io/driver/postgres`. Si no, recurrir al flujo actual `DB_MODE=local|cloud`.

**Alternativas consideradas**:
- *Parsear `DATABASE_URL` con `net/url` y reconstruir DSN key/value*: Innecesario; el driver `pgx` (subyacente a `gorm.io/driver/postgres`) acepta URIs de PostgreSQL nativamente (`postgres://user:pass@host:port/db?sslmode=require`). Heroku inyecta exactamente ese formato.
- *Nuevo `DB_MODE=heroku`*: Añade complejidad sin beneficio. La presencia/ausencia de `DATABASE_URL` es suficiente para discriminar.

**Rationale**: El driver `pgx` (usado por `gorm.io/driver/postgres`) ya parsea URIs estándar de PostgreSQL. Heroku Postgres inyecta `DATABASE_URL` en formato `postgres://user:pass@host:port/dbname?sslmode=require`, que es directamente consumible. No necesitamos parser manual ni conversión key/value. Esto es el enfoque más simple y menos propenso a errores.

---

### Decisión 2: Constructor `NewPostgreSQLRepositoryFromURL` separado

**Elección**: Crear una nueva función constructora `NewPostgreSQLRepositoryFromURL(databaseURL string)` en el paquete `database`, que acepte un DSN raw y abra GORM directamente con él.

**Alternativas consideradas**:
- *Modificar `NewPostgreSQLRepository` para aceptar opcionalmente una URL*: Rompe la firma actual y complica el API.
- *Parsear la URL y llamar al constructor existente con campos individuales*: Doble trabajo; el driver ya maneja URIs.

**Rationale**: Sigue el principio Open/Closed: el constructor existente no cambia, se añade uno nuevo. Ambos retornan `ports.DBRepository`, el caller (main.go) decide cuál usar basándose en env vars.

---

### Decisión 3: Migraciones en release phase (no en boot del web dyno)

**Elección**: **Opción B** — Separar la ejecución de migraciones en un comando `migrate` que Heroku ejecuta en su release phase. En desarrollo local, las migraciones siguen corriendo en boot (comportamiento actual).

**Alternativas consideradas**:
- *Opción A: Mantener migraciones en boot*: Funciona pero tiene problemas en multi-dyno (race conditions) y aumenta el cold start en Heroku.
- *Usar `golang-migrate` CLI externo*: Añade una dependencia; el sistema de migraciones propio ya funciona bien.

**Rationale**: Heroku ejecuta el `release` command **una sola vez** antes de arrancar los dynos. Esto elimina race conditions cuando hay múltiples dynos y reduce el cold start del proceso web. El mismo binario Go soportará un subcomando `migrate` (o se controla con `RUN_MIGRATIONS_ONLY=true`) que ejecuta `RunMigrations` y sale sin iniciar el servidor HTTP.

---

### Decisión 4: Storage mode guard en producción

**Elección**: Si `ENV=production` y `STORAGE_MODE=local`, el backend falla con `log.Fatalf` y mensaje claro.

**Alternativas consideradas**:
- *Warning y continuar*: Riesgo de perder datos en filesystem efímero de Heroku sin que nadie lo note.
- *Forzar override a `gcs`*: Comportamiento mágico; viola el principio de menor sorpresa.

**Rationale**: Fail-fast es preferible en producción. Un error explícito en arranque es fácil de diagnosticar y fuerza la configuración correcta.

---

### Decisión 5: Credenciales GCS en Heroku vía `google.CredentialsFromJSON`

**Elección**: En Heroku, `GOOGLE_APPLICATION_CREDENTIALS` contiene el JSON del service account inline (no una ruta de archivo). El backend detecta si el valor empieza con `{` y usa `google.CredentialsFromJSON` para crear el client; si no, trata el valor como ruta de archivo (comportamiento actual ADC).

**Alternativas consideradas**:
- *Escribir JSON a archivo temporal y setear `GOOGLE_APPLICATION_CREDENTIALS`*: Funciona pero deja secretos en disco (aunque efímero).
- *Usar `GOOGLE_CREDENTIALS_JSON` como variable separada*: Añade otra variable; mejor reutilizar la existente con detección automática.

**Rationale**: La detección `startsWith("{")` es simple y no ambigua (una ruta de archivo nunca empieza con `{`). Las librerías de Google Cloud (`cloud.google.com/go`) soportan `google.CredentialsFromJSON` nativamente. Se aplica tanto para GCS como para Vertex AI Embedding.

---

### Decisión 6: Externalizar `serverClientId` via `--dart-define`

**Elección**: Mover el Web OAuth Client ID a `--dart-define=GOOGLE_WEB_CLIENT_ID=<value>` y leerlo con `String.fromEnvironment` en Dart. Mantener el valor actual como fallback en desarrollo.

**Alternativas consideradas**:
- *Archivo de configuración por entorno (dev.json, prod.json)*: Más complejo; `--dart-define` es el mecanismo estándar en Flutter para build-time config.
- *Variable de runtime (API call al backend)*: Añade latencia al startup; el Client ID es estático.

**Rationale**: `--dart-define` es compile-time, no añade latencia, y permite builds de dev/prod con diferentes Client IDs sin cambiar código.

---

### Decisión 7: Heroku buildpack con `go.mod` en subdirectorio

**Elección**: Usar la variable `GOMODPATH=backend` (o `GO_INSTALL_PACKAGE_SPEC=./cmd/api`) en Heroku para indicar que `go.mod` está en `backend/`, no en la raíz del repo. El `Procfile` va en la raíz del repo.

**Alternativas consideradas**:
- *Mover `go.mod` a la raíz*: Reestructuración innecesaria del monorepo.
- *Heroku Container Registry (Docker)*: Más flexible pero más complejo para MVP. Se deja como opción futura.

**Rationale**: El buildpack `heroku/go` soporta `GOMODPATH` o `GO_INSTALL_PACKAGE_SPEC` para monorepos. El `Procfile` y `app.json` en la raíz es el estándar de Heroku.

---

### Decisión 8: CORS restrictivo en producción

**Elección**: Eliminar el fallback permisivo a `localhost:*` cuando `ENV=production`. En producción, solo se aceptan los origins explícitos de `ALLOWED_ORIGINS`.

**Alternativas consideradas**:
- *Mantener fallback siempre*: Brecha de seguridad en producción (cualquier localhost podría acceder).
- *Deshabilitar CORS completamente en producción*: Las apps mobile nativas no usan CORS, pero cerrar la puerta a Flutter Web futuro.

**Rationale**: Las apps mobile nativas (Android/iOS) no están sujetas a CORS del browser. Pero si en el futuro se habilita Flutter Web, un CORS permisivo en prod sería una vulnerabilidad. El cambio es mínimo (un `if` en el `allowedOriginFunc`).

---

### Decisión 9: Entrypoint dual (web + migrate) vía variable de entorno

**Elección**: Un solo binario Go. Si `RUN_MIGRATIONS_ONLY=true`, ejecuta migraciones y sale con `os.Exit(0)`. Si no (default), inicia el servidor HTTP sin ejecutar migraciones.

**Alternativas consideradas**:
- *Subcomando (`klyra-backend migrate`)*: Requiere usar `cobra` o `flag` para parsing; overhead para un solo flag.
- *Script bash wrapper*: Heroku release phase ejecuta un comando; un script añade complejidad.
- *Binary separado*: Duplica el build; innecesario.

**Rationale**: Una variable de entorno es la forma más simple. El `Procfile` de Heroku puede setear `RUN_MIGRATIONS_ONLY=true` solo para el comando `release`. En local, el comportamiento actual (migraciones en boot) se mantiene si `RUN_MIGRATIONS_ONLY` no está definida.

---

## Flujo de Datos

### Arranque del backend (composition root)

```
                         main.go (Composition Root)
                                   │
          ┌────────────────────────┼────────────────────────┐
          ▼                        ▼                        ▼
   RUN_MIGRATIONS_ONLY?      initDBRepository()      initStorageService()
          │                        │                        │
     ┌────┴────┐          ┌────────┴────────┐         ┌────┴────┐
     │  true   │          │ DATABASE_URL?   │         │ENV=prod?│
     ▼         │          ▼                 ▼         ▼         ▼
  RunMigrate   │    FromURL(dsn)    DB_MODE switch   Guard:    GCS/Local
  + os.Exit(0) │    (Heroku)       local│cloud       no local
               │                                     en prod
               ▼
          initCORS()
               │
          ┌────┴─────┐
          │ENV=prod?  │
          ▼           ▼
     Solo ALLOWED    + localhost
     _ORIGINS       fallback (dev)
               │
               ▼
         gin.Run(:PORT)
```

### Flujo de conexión a DB (precedencia)

```
  ┌─────────────────────────┐
  │ ¿DATABASE_URL definida? │
  └──────┬──────────────────┘
         │
    ┌────┴────┐
    │   Sí    │──────▶ NewPostgreSQLRepositoryFromURL(DATABASE_URL)
    └─────────┘        (Heroku Postgres / cualquier PaaS)
         │
    ┌────┴────┐
    │   No    │──────▶ ¿DB_MODE?
    └─────────┘           │
                    ┌─────┴──────┐
                    │            │
                 "cloud"      "local" (default)
                    │            │
                    ▼            ▼
          NewCloudSQLRepo   NewPostgreSQLRepo
          (GCP unix sock)   (host/port/user/pass)
```

### Flujo de credenciales GCS en Heroku

```
  GOOGLE_APPLICATION_CREDENTIALS
              │
     ┌────────┴─────────┐
     │ startsWith("{")? │
     ▼                  ▼
    Sí                 No
     │                  │
     ▼                  ▼
  CredentialsFromJSON   ADC estándar
  (inline JSON)         (ruta a archivo)
     │                  │
     └────────┬─────────┘
              ▼
    storage.NewClient(ctx, opt)
    (GCS, Vertex AI Embedding)
```

---

## Cambios por Archivo

| Archivo | Acción | Descripción |
|---------|--------|-------------|
| `backend/cmd/api/main.go` | Modificar | Añadir rama `DATABASE_URL` en `initDBRepository()`; guard de storage en prod; CORS sin fallback localhost en prod; flag `RUN_MIGRATIONS_ONLY`; separar migraciones del boot normal |
| `backend/internal/infrastructure/database/postgresql_repository.go` | Modificar | Añadir `NewPostgreSQLRepositoryFromURL(databaseURL string)` que abre GORM con el DSN raw |
| `backend/internal/repositories/gcs_storage_service.go` | Modificar | Refactorizar `NewGCSStorageService()` para aceptar JSON inline de credenciales vía helper `resolveGoogleCredentials()` |
| `backend/internal/repositories/gcp_credentials_helper.go` | Crear | Helper `resolveGoogleCredentials()` que detecta JSON inline vs ruta y retorna `option.ClientOption` apropiado |
| `mobile/lib/features/auth/data/auth_remote_datasource.dart` | Modificar | Reemplazar `serverClientId` hardcodeado con `String.fromEnvironment('GOOGLE_WEB_CLIENT_ID', defaultValue: '782011204480-...')` |
| `mobile/lib/core/config/env.dart` | Modificar | Añadir `static const googleWebClientId = String.fromEnvironment(...)` y `backendBaseUrl` configurable vía `--dart-define=API_BASE_URL` |
| `Procfile` (raíz del repo) | Crear | Define `web:` y `release:` commands para Heroku |
| `app.json` (raíz del repo) | Crear | Manifiesto Heroku: addons, buildpacks, env schema |
| `backend/Dockerfile` | Modificar | Actualizar versión de Go a 1.25; copiar directorio `migrations/` al stage de runtime |

---

## Interfaces / Contratos

### Nuevo constructor de DB (Go)

```go
// backend/internal/infrastructure/database/postgresql_repository.go

// NewPostgreSQLRepositoryFromURL abre una conexión GORM usando un DSN de PostgreSQL
// en formato URI (postgres://user:pass@host:port/db?sslmode=require).
// Pensado para PaaS como Heroku que inyectan DATABASE_URL.
func NewPostgreSQLRepositoryFromURL(databaseURL string) (ports.DBRepository, error) {
    db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
    if err != nil {
        return nil, fmt.Errorf("failed to connect via DATABASE_URL: %w", err)
    }
    return &PostgreSQLRepository{db: db}, nil
}
```

### Helper de credenciales GCP (Go)

```go
// backend/internal/repositories/gcp_credentials_helper.go

package repositories

import (
    "context"
    "strings"

    "golang.org/x/oauth2/google"
    "google.golang.org/api/option"
)

// ResolveGoogleCredentials retorna un option.ClientOption adecuado según
// el valor de GOOGLE_APPLICATION_CREDENTIALS:
//   - Si empieza con "{": JSON inline → google.CredentialsFromJSON
//   - Si no: ruta de archivo → ADC estándar (option.WithCredentialsFile)
//   - Si vacío: ADC automático (Cloud Run, etc.)
func ResolveGoogleCredentials(credsValue string, scopes ...string) (option.ClientOption, error) {
    if strings.TrimSpace(credsValue) == "" {
        return nil, nil // ADC automático
    }

    if strings.HasPrefix(strings.TrimSpace(credsValue), "{") {
        creds, err := google.CredentialsFromJSON(
            context.Background(),
            []byte(credsValue),
            scopes...,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to parse inline credentials JSON: %w", err)
        }
        return option.WithCredentials(creds), nil
    }

    return option.WithCredentialsFile(credsValue), nil
}
```

### initDBRepository refactorizado (Go)

```go
// backend/cmd/api/main.go (fragmento)

func initDBRepository() (ports.DBRepository, error) {
    // Precedencia 1: DATABASE_URL (Heroku, Railway, Render, etc.)
    if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
        log.Println("Database mode: url (DATABASE_URL detected)")
        return database.NewPostgreSQLRepositoryFromURL(databaseURL)
    }

    // Precedencia 2: DB_MODE (local | cloud)
    dbMode := strings.ToLower(getEnv("DB_MODE", "local"))
    log.Printf("Database mode: %s", dbMode)

    if dbMode == "cloud" {
        return database.NewCloudSQLRepository(
            getEnv("DB_INSTANCE_CONNECTION_NAME", getEnv("INSTANCE_CONNECTION_NAME", "")),
            mustEnv("DB_NAME"),
            mustEnv("DB_USER"),
            mustEnv("DB_PASSWORD"),
            getEnv("DB_SSL_MODE", "disable"),
        )
    }

    return database.NewPostgreSQLRepository(
        getEnv("DB_HOST", "localhost"),
        getEnv("DB_PORT", "5432"),
        mustEnv("DB_NAME"),
        mustEnv("DB_USER"),
        mustEnv("DB_PASSWORD"),
        getEnv("DB_SSL_MODE", "disable"),
    )
}
```

### Entrypoint con flag de migraciones (Go)

```go
// backend/cmd/api/main.go (fragmento — inicio de main())

func main() {
    // ... carga de .env ...

    dbRepo, err := initDBRepository()
    if err != nil {
        log.Fatalf("Failed to initialize database repository: %v", err)
    }
    defer dbRepo.Close()

    if err := dbRepo.Ping(); err != nil {
        log.Fatalf("Database ping failed: %v", err)
    }

    // Release phase: solo migraciones, sin servidor
    if strings.EqualFold(os.Getenv("RUN_MIGRATIONS_ONLY"), "true") {
        log.Println("RUN_MIGRATIONS_ONLY=true — executing migrations and exiting")
        if err := dbRepo.RunMigrations("./migrations"); err != nil {
            log.Fatalf("Migration failed: %v", err)
        }
        log.Println("Migrations completed successfully")
        os.Exit(0)
    }

    // ... resto del wiring e inicio del servidor (sin RunMigrations) ...
}
```

### CORS production-safe (Go)

```go
// backend/cmd/api/main.go (fragmento)

isProduction := strings.EqualFold(getEnv("ENV", "development"), "production")

allowedOriginFunc := func(origin string) bool {
    for _, configured := range allowedOrigins {
        if strings.EqualFold(strings.TrimSpace(configured), strings.TrimSpace(origin)) {
            return true
        }
    }
    // Solo en desarrollo: aceptar localhost como fallback
    if !isProduction {
        originLower := strings.ToLower(strings.TrimSpace(origin))
        return strings.HasPrefix(originLower, "http://localhost:") ||
            strings.HasPrefix(originLower, "http://127.0.0.1:") ||
            strings.HasPrefix(originLower, "https://localhost:") ||
            strings.HasPrefix(originLower, "https://127.0.0.1:")
    }
    return false
}
```

### Storage guard (Go)

```go
// backend/cmd/api/main.go (fragmento, antes de initStorageService)

if isProduction && strings.EqualFold(getEnv("STORAGE_MODE", "gcs"), "local") {
    log.Fatalf("FATAL: STORAGE_MODE=local is not allowed in production (ENV=production). " +
        "Heroku uses ephemeral filesystem. Set STORAGE_MODE=gcs and configure GCS_BUCKET_NAME.")
}
```

### Dart: serverClientId externalizado

```dart
// mobile/lib/features/auth/data/auth_remote_datasource.dart

const webServerClientId = String.fromEnvironment(
  'GOOGLE_WEB_CLIENT_ID',
  defaultValue: '782011204480-0eejl4shc1f9n360mln5secbeng6k5gb.apps.googleusercontent.com',
);
googleSignIn = GoogleSignIn(
  serverClientId: webServerClientId,
  scopes: ['email', 'profile'],
);
```

### Dart: API base URL externalizado

```dart
// mobile/lib/core/config/env.dart

class EnvInfo {
  static const _apiBaseUrlOverride = String.fromEnvironment('API_BASE_URL');

  static String get backendBaseUrl {
    if (_apiBaseUrlOverride.isNotEmpty) {
      return _apiBaseUrlOverride;
    }
    // ... lógica actual de detección de plataforma ...
  }
}
```

---

## Artefactos Heroku

### Procfile (raíz del repo)

```
web: backend/bin/klyra-backend
release: RUN_MIGRATIONS_ONLY=true backend/bin/klyra-backend
```

> Nota: El buildpack `heroku/go` compila y coloca el binario automáticamente. La ruta exacta depende de `GO_INSTALL_PACKAGE_SPEC`. Si se usa Docker (`heroku.yml`), se usa el `Dockerfile` directamente.

### app.json (raíz del repo)

```json
{
  "name": "klyra-backend",
  "description": "Klyra AI Tutor — Backend API",
  "buildpacks": [
    { "url": "heroku/go" }
  ],
  "addons": [
    {
      "plan": "heroku-postgresql:essential-0",
      "as": "DATABASE"
    }
  ],
  "env": {
    "ENV": { "value": "production" },
    "GIN_MODE": { "value": "release" },
    "GOMODPATH": { "value": "backend" },
    "GO_INSTALL_PACKAGE_SPEC": { "value": "./cmd/api" },
    "GOOGLE_CLIENT_ID": { "required": true, "description": "Google OAuth Web Client ID" },
    "JWT_SECRET": { "required": true, "description": "Secret for JWT access tokens (min 32 chars)" },
    "REFRESH_TOKEN_SECRET": { "required": true, "description": "Secret for JWT refresh tokens (min 32 chars)" },
    "STORAGE_MODE": { "value": "gcs" },
    "GCS_BUCKET_NAME": { "required": true, "description": "GCS bucket for file uploads" },
    "GCP_PROJECT_ID": { "required": true, "description": "Google Cloud project ID" },
    "GOOGLE_APPLICATION_CREDENTIALS": { "required": true, "description": "Service account JSON (inline) for GCS + Vertex AI" },
    "ALLOWED_ORIGINS": { "required": true, "description": "Comma-separated allowed CORS origins" }
  },
  "formation": {
    "web": { "quantity": 1, "size": "basic" }
  }
}
```

### Dockerfile actualizado

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
EXPOSE 8080
ENTRYPOINT ["/app/klyra-backend"]
```

---

## Estrategia de Testing

| Capa | Qué probar | Enfoque |
|------|-----------|---------|
| Unit | `NewPostgreSQLRepositoryFromURL` parsea DSN correctamente | Test con URL mock; verificar que GORM recibe el DSN sin transformación |
| Unit | `ResolveGoogleCredentials` con JSON inline, ruta, y vacío | Test de las 3 ramas; mock no necesario (solo string parsing) |
| Unit | `initDBRepository` elige el constructor correcto según env | Set/unset `DATABASE_URL` y `DB_MODE`; verificar rama seleccionada |
| Unit | CORS `allowedOriginFunc` bloquea localhost en prod | Test con `ENV=production` y origins no listados |
| Unit | Storage guard: `log.Fatalf` si `ENV=production` + `STORAGE_MODE=local` | Test que verifica el panic/exit (o refactorizar a función testable) |
| Unit (Dart) | `EnvInfo.backendBaseUrl` respeta `API_BASE_URL` override | Test con `--dart-define=API_BASE_URL=https://test.herokuapp.com/api/v1` |
| Integración | Conexión a Heroku Postgres con `DATABASE_URL` real | Test manual post-deploy; `GET /health` retorna 200 |
| Integración | Migraciones en release phase ejecutan sin error | Verificar logs de `heroku releases:output` |
| Integración | Upload a GCS desde Heroku con credenciales JSON inline | Upload de archivo PDF y verificar URL de GCS accesible |
| E2E | Login Google desde mobile → backend en Heroku | Build mobile con `--dart-define` apuntando a Heroku; completar flujo OAuth |
| E2E | Flujo completo: login → crear curso → subir material → sesión tutor | Smoke test manual con la app mobile contra la URL de Heroku |

---

## Migración / Rollout

### Fase 1: Cambios de código (este cambio)

1. `NewPostgreSQLRepositoryFromURL` en `postgresql_repository.go`.
2. `ResolveGoogleCredentials` helper.
3. Refactorizar `main.go`: `DATABASE_URL` precedence, `RUN_MIGRATIONS_ONLY`, storage guard, CORS prod.
4. Externalizar `serverClientId` y `API_BASE_URL` en mobile.
5. Crear `Procfile` y `app.json`.
6. Actualizar `Dockerfile` (Go version + copiar migrations).

### Fase 2: Deploy a Heroku (post-merge)

1. `heroku create klyra-backend` (o nombre elegido).
2. `heroku addons:create heroku-postgresql:essential-0`.
3. Configurar config vars según la matrix de la propuesta.
4. `git push heroku main`.
5. Verificar release phase en logs: `heroku releases:output`.
6. `curl https://klyra-backend.herokuapp.com/health`.

### Fase 3: Validación

1. Build mobile: `flutter run --dart-define=API_BASE_URL=https://klyra-backend.herokuapp.com/api/v1 --dart-define=GOOGLE_WEB_CLIENT_ID=782011204480-...`.
2. Probar login Google, upload, sesión tutor.
3. Monitorear `heroku logs --tail`.

### Rollback

- Heroku no promueve el slug si la release phase falla → dyno anterior sigue activo.
- Todos los cambios son aditivos (`DATABASE_URL` solo se activa si existe; `RUN_MIGRATIONS_ONLY` solo si está seteada).
- `git revert` del commit restaura el estado anterior sin efectos secundarios.

---

## Consideraciones de PORT, Proxies y Trusted Proxies

- **PORT**: Heroku asigna `$PORT` dinámicamente. El código actual ya usa `getEnv("PORT", "8080")`, no hay cambio necesario.
- **Trusted Proxies**: El código actual usa `router.SetTrustedProxies(nil)` (no confía en ninguno). En Heroku, los requests pasan por un router/proxy que setea `X-Forwarded-For`. Si se necesita `ClientIP()` real, cambiar a `router.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12"})` (rangos internos de Heroku). **Para MVP**, el comportamiento actual es seguro; el rate limiter podría ver la IP del router en vez del cliente, pero es aceptable.
- **`GIN_MODE`**: Setear `GIN_MODE=release` en Heroku para desactivar debug output.

---

## Preguntas Abiertas

- [x] ¿Usar Go buildpack o Container Registry (Docker) en Heroku? → **Decisión: Go buildpack para MVP** por simplicidad. Docker queda como alternativa si el buildpack no soporta bien el monorepo.
- [ ] ¿Heroku plan `essential-0` o `basic` para Postgres? → Depende de presupuesto. `essential-0` tiene 1GB de storage y 20 conexiones; suficiente para MVP.
- [ ] ¿Configurar `pgvector` en Heroku Postgres? → Heroku Postgres `essential-0` no soporta extensiones custom. Si RAG requiere pgvector, evaluar: (a) usar `essential-1` o superior, (b) Neon/Supabase externo, o (c) deshabilitar RAG en Heroku MVP y habilitarlo en GCP.
- [ ] ¿Nombre de la app Heroku? → Por definir (`klyra-backend`, `klyra-api`, etc.).
