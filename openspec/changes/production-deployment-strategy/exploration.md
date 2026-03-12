## Exploration: Production Deployment Strategy (Heroku & GCP)

### Current State
El backend de Klyra está desarrollado en Go (Gin) y utiliza una arquitectura de puertos y adaptadores. Actualmente soporta:
- **Base de Datos**: PostgreSQL local (vía TCP) y GCP Cloud SQL (vía Unix sockets/pgx).
- **Almacenamiento**: Local filesystem y Google Cloud Storage (GCS).
- **Autenticación**: Google OAuth (verificación de ID Token) y Guest mode.
- **Configuración**: Variables de entorno vía `.env` (godotenv).

#### Observaciones concretas del codebase (2026-03)
- **Composition root**: `backend/cmd/api/main.go` centraliza DI, CORS, rate limiting, `DB_MODE`, `STORAGE_MODE` y el `GOOGLE_CLIENT_ID`.
- **DB init**: `initDBRepository()` solo soporta:
  - `DB_MODE=cloud` usando `DB_INSTANCE_CONNECTION_NAME|INSTANCE_CONNECTION_NAME` (unix socket /cloudsql) o `DB_HOST` fallback.
  - `DB_MODE=local` armando DSN por piezas (`DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSL_MODE`).
  - **No** consume `DATABASE_URL`, que es la convención estándar en Heroku.
- **Migrations**: se ejecutan en arranque (`dbRepo.RunMigrations("./migrations")`), lo cual funciona pero puede aumentar el tiempo de boot en PaaS.
- **Storage**:
  - `STORAGE_MODE=local` escribe a disco (`./storage`) y expone `/static` por Gin.
  - En Heroku el filesystem es **efímero**, por lo que `local` no es opción en producción.
  - Ya existe `STORAGE_MODE=gcs` (`repositories.NewGCSStorageService()`), ideal para “multi-cloud” (Heroku ahora, GCP después).
- **Google Sign-In**:
  - Backend verifica el `aud` del ID token con `GOOGLE_CLIENT_ID` (`repositories.NewGoogleVerifier(mustEnv("GOOGLE_CLIENT_ID"))`).
  - Mobile usa `google_sign_in` con `serverClientId` (web client) hardcodeado en `mobile/lib/features/auth/data/auth_remote_datasource.dart`.
  - En producción, **`GOOGLE_CLIENT_ID` del backend debe igualar** ese `serverClientId` para que la verificación pase.

### Affected Areas
- `backend/internal/infrastructure/database/postgresql_repository.go`: Necesita soportar conexión vía DSN (`DATABASE_URL`) para Heroku.
- `backend/cmd/api/main.go`: Refactorizar la lógica de inicialización para ser más flexible según el proveedor.
- `backend/internal/repositories/storage_service.go`: (Opcional) Añadir soporte para S3 si se prefiere sobre GCS en Heroku.
- `Procfile`: Nuevo archivo para despliegue en Heroku.
- `app.json` (Heroku manifest): Declarar add-ons, buildpacks y env vars.

### Approaches

1. **Estrategia Multi-Cloud Adaptativa (Recomendada)**
   - Utilizar el patrón de **Ports & Adapters** existente.
   - Refactorizar la configuración para que sea 100% 12-factor (config por env), **priorizando `DATABASE_URL`** si está presente (Heroku).
   - Usar GCS como almacenamiento unificado para producción (tanto Heroku como GCP) para simplificar la migración, o implementar un adaptador de S3.
   - **Pros**: Mínimo acoplamiento, fácil migración entre Heroku y GCP, soporte local intacto.
   - **Cons**: Requiere una pequeña refactorización en el punto de entrada (`main.go`).
   - **Effort**: Low/Medium.

2. **Despliegue Directo en GCP (Cloud Run)**
   - Ignorar Heroku e ir directamente a GCP para aprovechar la integración nativa de Cloud SQL y GCS.
   - **Pros**: Mejor rendimiento y seguridad (IAM, VPC).
   - **Cons**: Mayor complejidad inicial de configuración comparado con Heroku. Menor agilidad para prototipos rápidos.
   - **Effort**: Medium.

### Configuration Matrix

| Variable | Local (Docker) | Heroku | GCP (Cloud Run) |
| :--- | :--- | :--- | :--- |
| `ENV` | `development` | `production` | `production` |
| `PORT` | `8080` | Dinámico (asignado por Heroku) | `8080` |
| `DB_MODE` | `local` | `local` (pero usando `DATABASE_URL`) | `cloud` |
| `DATABASE_URL` | (opcional) | Automático (Heroku Postgres) | (opcional si usas TCP/connector) |
| `STORAGE_MODE` | `local` | `gcs` (o `s3`) | `gcs` |
| `GOOGLE_APPLICATION_CREDENTIALS` | `./key.json` | Contenido JSON en Env Var | Automático (ADC) |
| `ALLOWED_ORIGINS` | `http://localhost:*` | URL de Heroku + App Mobile | URL de Cloud Run + App Mobile |
| `GOOGLE_CLIENT_ID` | Web OAuth client ID | Web OAuth client ID | Web OAuth client ID |
| `JWT_SECRET` / `REFRESH_TOKEN_SECRET` | secrets locales | config vars (secrets) | Secret Manager / env |

### Google Login Flow
1. **Mobile App**: Obtiene el `id_token` de Google.
2. **Backend**: Recibe el `id_token`, lo verifica usando la librería oficial de Google con el `GOOGLE_CLIENT_ID`.
3. **Producción**:
   - Asegurar que el **Web OAuth Client** (el `serverClientId` del mobile) es el mismo que valida el backend (`GOOGLE_CLIENT_ID`).
   - Mantener la app detrás de HTTPS (Heroku y Cloud Run lo proveen por defecto).
   - Si luego habilitamos web login (Flutter web), configurar "Authorized JavaScript origins" y "Authorized redirect URIs" en Google Cloud Console.
   - El backend debe estar detrás de HTTPS (Heroku y GCP lo proveen por defecto).

### Heroku Deployment Steps
- **Build**:
  - Usar `heroku/go` buildpack y apuntar a `backend/cmd/api`.
  - Alternativa: container registry (Docker) si quieres homogeneidad con Cloud Run.
- **Procfile**:
  - `web: <comando que ejecute el binario del backend>`
  - Recomendación: incluir un `release:` para migraciones (evita migrar en cada boot de web dyno).
- **Add-ons**:
  - `heroku-postgresql` (crea `DATABASE_URL`).
- **Config Vars**:
  - `DB_MODE=local` pero **consumir `DATABASE_URL`** (cambio a implementar).
  - `STORAGE_MODE=gcs` + bucket/credenciales (o `s3` si agregamos adaptador).
  - `GOOGLE_CLIENT_ID`, `JWT_SECRET`, `REFRESH_TOKEN_SECRET`, `ALLOWED_ORIGINS`, etc.
- **Health checks**: `GET /health` ya existe.
- **Nota websockets/timeouts**:
  - Si en el futuro el backend termina “proxying” WebSockets (p.ej. Gemini Live), incluir heartbeats y tolerancia a reconexión. Hoy el WS crítico está en el mobile → Gemini Live, no necesariamente en backend.

### GCP Deployment Steps
- **Artifact Registry**: Build y push de imagen Docker.
- **Cloud Run**: Desplegar imagen.
- **Cloud SQL**: Conectar usando el `Cloud SQL Connector` (Unix sockets).
- **IAM**: Asignar Service Account con permisos en GCS, Cloud SQL y Vertex AI.

### Recommendation
Implementar la **Estrategia Multi-Cloud Adaptativa**. Es la que mejor cumple con los requisitos de flexibilidad y bajo acoplamiento. La clave es que el código no sepa en qué nube está, solo qué interfaces utilizar basándose en las variables de entorno.

### Proposed Pattern (para /sdd-propose)
Aplicar explícitamente un patrón **Provider Adapter** para infraestructura, manteniendo el DI en `main.go` como “composition root”:
- **Config (12-factor)**:
  - `DATABASE_URL` tiene precedencia (Heroku).
  - Si no existe `DATABASE_URL`, usar `DB_MODE` + variables por piezas (local y Cloud SQL).
- **DBRepository**:
  - Agregar una ruta `database.NewPostgreSQLRepositoryFromURL(databaseURL string)` o equivalente.
  - `initDBRepository()`:
    - Si `DATABASE_URL` está seteado: usarla.
    - Else: usar el flujo actual (local/cloud).
- **StorageService**:
  - En Heroku y GCP usar `STORAGE_MODE=gcs` para no duplicar proveedores.
  - Opcional: añadir `STORAGE_MODE=s3` si prefieres Heroku+S3 y migración posterior.
- **Auth (Google)**:
  - Back: `GOOGLE_CLIENT_ID` en config var/secret.
  - Mobile: mover `serverClientId` a `--dart-define=GOOGLE_WEB_CLIENT_ID=...` para no hardcodear y poder manejar dev/prod.

### Risks
- **Heroku Ephemeral Filesystem**: El código actual ya tiene `STORAGE_MODE=local`, pero en Heroku se perderían los archivos. Se debe forzar `gcs` o `s3` en producción.
- **Migrations en boot**: ejecutar migraciones al iniciar el web dyno puede aumentar tiempos de arranque. Ideal: `release` phase.
- **CORS / ALLOWED_ORIGINS**: en producción hay que enumerar explícitamente origins (web y/o dominios de Cloud Run/Heroku). El fallback “localhost:*” solo aplica en dev.
- **Cold Starts**: Tanto Heroku (free tier/eco) como Cloud Run tienen cold starts.

### Ready for Proposal
Yes. La arquitectura actual facilita mucho esta transición. Solo faltan ajustes menores en la inicialización y la documentación del proceso.
