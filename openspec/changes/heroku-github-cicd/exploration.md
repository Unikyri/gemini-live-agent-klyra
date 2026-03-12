## Exploración: Integración CI/CD con GitHub y Heroku

### Estado Actual
El proyecto `gemini-live-agent-klyra` es un monorepo con el backend en el directorio `/backend`. Actualmente, existe una aplicación en Heroku llamada `klyra-backend-prod` que tiene un addon de PostgreSQL provisionado (`postgresql-curved-48489`). Existe un archivo `Procfile` en la raíz que intenta ejecutar `bin/api` y tiene una fase de `release` configurada para ejecutar migraciones, aunque el binario parece estar mal referenciado si consideramos que el código vive en `/backend`.

#### Verificación con Heroku MCP (estado real)
- **App**: `klyra-backend-prod`
- **DB**: `heroku-postgresql:essential-0` → **State: `created`**
  - Implicación: la DB **ya está provisionada** y **no se recrea** ni se “limpia” en cada despliegue.
  - Solo se pierde o reinicia si haces acciones explícitas como `pg:reset` / destrucción del addon.

#### Verificación con GitHub MCP (estado real)
- **Repo**: `Unikyri/gemini-live-agent-klyra`
- **Default branch**: `main`
- **Workflows**: no existe aún ningún `.github/workflows/*.yml` (hay que crearlo).

### Áreas Afectadas
- `backend/Dockerfile` — Define cómo se construye la imagen para despliegue por contenedores.
- `Procfile` — Define los procesos (web y release) para Heroku.
- `.github/workflows/deploy.yml` — (Por crear) Si se elige la opción de GitHub Actions.
- `app.json` — (Por ajustar) Si se usa Heroku GitHub Integration con review apps.

### Approaches

1. **Heroku GitHub Integration (Auto-deploy)**
   - **Descripción**: Conectar Heroku directamente al repositorio de GitHub para que cada commit a `main` dispare un despliegue.
   - **Pros**:
     - Configuración extremadamente sencilla desde el dashboard de Heroku.
     - Review Apps automáticas para Pull Requests.
     - No requiere gestionar tokens externos de larga duración.
   - **Cons**:
     - Menos control sobre el pipeline (no es fácil insertar pasos previos como linter o tests complejos).
     - En un monorepo, requiere el uso de `heroku-buildpack-monorepo` y la variable `PROJECT_PATH=backend`.
   - **Effort**: Low

2. **GitHub Actions + Heroku CLI**
   - **Descripción**: Usar un workflow de GitHub Actions que construya la imagen (vía Docker) y la pushee al Container Registry de Heroku.
   - **Pros**:
     - Control total: podemos ejecutar tests de Go, linter y validaciones antes de tocar Heroku.
     - Manejo nativo de monorepos: simplemente se hace `cd backend` antes de construir.
     - Despliegues deterministas basados en la imagen Docker ya existente en el repo.
   - **Cons**:
     - Requiere configurar un `HEROKU_API_KEY` como secreto en GitHub.
     - Configuración inicial de YAML un poco más compleja.
   - **Effort**: Medium

### Recomendación
**Opción 2: GitHub Actions + Heroku CLI**. Dado que:
- ya tenemos un `backend/Dockerfile`,
- el proyecto es monorepo (`backend/`),
- y necesitamos meter checks previos (tests) antes de desplegar,

GitHub Actions da el control necesario para un pipeline **CI completo** (Test → Build → Deploy).

### Flujo Recomendado Paso a Paso
1. **Preparación**:
   - Obtener el `HEROKU_API_KEY` desde la cuenta de Heroku.
   - Guardarlo como `HEROKU_API_KEY` en "GitHub Repository Secrets".
   - (Recomendado) Guardar también `HEROKU_APP_NAME=klyra-backend-prod` en GitHub Secrets o en `env:` del workflow.
2. **Workflow**:
   - Trigger en `push` a `main` y en cambios dentro de `/backend/**`.
   - Paso 1: Checkout del código.
   - Paso 2: Login al Container Registry (`heroku container:login`).
   - Paso 3: Construir y pushear la imagen:
     - `heroku container:push web -a klyra-backend-prod --context-path backend`
   - Paso 4: Liberar la imagen:
     - `heroku container:release web -a klyra-backend-prod`
   - Referencia oficial (CLI): `container:login`, `container:push`, `container:release`.
3. **Migraciones**:
   - Usar la **Release Phase** de Heroku configurada en el `Procfile`.
   - En nuestro caso, el `Procfile` debe ejecutar el binario del contenedor (`/app/klyra-backend`) o usar el mecanismo que ya implementamos (`RUN_MIGRATIONS_ONLY=true`).

### Gestión de Base de Datos
- **Provisionamiento**: La base de datos Heroku Postgres se provisiona una sola vez (ya está hecho).
- **Persistencia**: Heroku NO borra la DB en cada deploy. Los datos persisten a menos que se ejecute un comando manual de `pg:reset`.
- **Migraciones Seguras**: La fase de `release` garantiza que las migraciones se completen ANTES de que los nuevos dynos de la aplicación comiencen a recibir tráfico. Si la fase de `release` falla, el despliegue se cancela y la versión anterior sigue viva.

### Seguridad / Secretos (qué va dónde)
- **GitHub Secrets**:
  - `HEROKU_API_KEY` (obligatorio para GitHub Actions).
  - (Opcional) `HEROKU_APP_NAME`, `HEROKU_EMAIL` (si lo necesitas).
  - No guardar `DATABASE_URL` en GitHub: eso vive en Heroku como config var gestionada por el addon.
- **Heroku Config Vars** (persisten entre deploys):
  - `DATABASE_URL` (lo pone el addon).
  - `ENV`, `GIN_MODE`, `ALLOWED_ORIGINS`, `JWT_SECRET`, `REFRESH_TOKEN_SECRET`, etc.
  - Credenciales GCP/GCS (si aplican) como `GOOGLE_APPLICATION_CREDENTIALS_JSON`.

### Comportamiento en PRs vs Main
- **PRs**: Se recomienda ejecutar tests y linter en cada PR, pero NO desplegar a `klyra-backend-prod`. Opcionalmente, se pueden usar Heroku Review Apps si se desea una URL temporal.
- **Main**: Despliegue automático a producción tras pasar todos los tests.

### Checklist de Verificación con MCP
- **Heroku**
  - [x] Confirmar addon Postgres existe y está listo: `list_addons` → **State: `created`**.
  - [ ] Confirmar config vars clave: `get_app_info` / `config` (si se expone) → `DATABASE_URL`, `RUN_MIGRATIONS_ON_BOOT=false`.
  - [ ] Verificar release phase / logs después del primer deploy CI.
- **GitHub**
  - [ ] Confirmar que el workflow YAML existe en `main` (`.github/workflows/deploy-heroku.yml`).
  - [ ] Confirmar que el workflow corre en `push` a `main`.
  - [ ] Confirmar que el job de deploy solo corre si pasan los tests.

### Ready for Proposal
Yes. El equipo puede proceder a crear el workflow de GitHub Actions y ajustar el `Procfile` para alinearse con la estructura del contenedor.

### Notas importantes (monorepo + buildpack vs container)
- Si usas **Heroku GitHub Integration** (dashboard), para monorepo normalmente terminas metiendo buildpacks extra (monorepo buildpack) y `PROJECT_PATH=backend`.
- Con **GitHub Actions + container**, evitas esa complejidad y el deploy es consistente: el contexto del build es `backend/` y Heroku ejecuta la imagen.

