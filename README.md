# Klyra - Interactive AI Tutoring

Klyra is an innovative, AI-powered mobile application (using Gemini API) that transforms static notes and texts into interactive "masterclasses". It features a Tutor Avatar capable of real-time voice interruptions ("barge-in"), visual background injection (via Graph RAG), and persistent student memory.

## Product Vision
Targeted at middle and higher education students who need personalized learning support, Klyra offers ultra-fast, immersive, real-time explanations directly from their class materials.

## MVP Architecture
* **Frontend:** Flutter (Mobile) with multi-layer rendering support for background injection.
* **Backend:** Go (Modular Monolith with Clean Architecture) hosted on Google Cloud Run.
* **AI Components:** Gemini Live API (barge-in / audio), Imagen (sprite/avatar generation), and Vertex AI Vector Search (Graph RAG).
* **Database:** PostgreSQL for user persistence, learning profiles, and RAG referencing.
* **Connectivity:** WebSockets (WSS) client-server for live asynchronous events (backgrounds), and REST API for CRUD management and file uploads.

## Project Structure
The repository is structured as a monorepo containing both the frontend and the backend.

```
.
├── backend/                # Go Backend (Clean Architecture)
│   ├── cmd/                # Main applications (entry points)
│   ├── internal/           # Private application and library code
│   │   ├── core/           # Use cases, domain models, and interfaces
│   │   ├── handlers/       # HTTP/WebSocket delivery layer
│   │   └── repositories/   # Database and external API integrations
│   ├── pkg/                # Library code that's ok to use by external applications
│   └── migrations/         # PostgreSQL database migrations
│
├── mobile/                 # Flutter application
│   ├── lib/
│   │   ├── core/           # Common utilities, themes, routing
│   │   ├── features/       # Feature-based folder structure (Auth, Courses, LiveSession)
│   │   └── main.dart       # Entry point
│   ├── assets/             # Local images, fonts, animations
│   └── test/               # Flutter tests
│
├── docs/                   # Additional architecture or API documentation
├── .agent/                 # AI assistant project rules and configurations
└── README.md
```

## Git Strategy
We adhere to **GitHub Flow (Simplified Trunk Based Development)** favoring Continuous Integration:
1. The `main` branch is always deployable.
2. Develop new features or fixes on descriptive feature branches (e.g., `feature/auth-oauth2`, `fix/avatar-sync`).
3. Commit messages must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification (`feat:`, `fix:`, `chore:`, etc.).
4. Open a Pull Request against `main` for code review before merging.

## Development Environment Setup

### Prerequisites
* [Go 1.22+](https://go.dev/doc/install)
* [Flutter 3.22+](https://docs.flutter.dev/get-started/install)
* [PostgreSQL 16+](https://www.postgresql.org/download/)
* A [Google Cloud](https://cloud.google.com/) account with Vertex AI and Gemini API enabled.

### Local Installation
1. Clone this repository:
   ```bash
   git clone https://github.com/Unikyri/gemini-live-agent-klyra.git
   cd gemini-live-agent-klyra
   ```
2. Configure environment variables based on `.env.example`.
3. Start your local database and run the corresponding migrations (defined in `/backend/migrations`).
4. Set up the mobile app by running `flutter pub get` inside `/mobile`.

### Platform-Specific Configuration

#### Static File URLs for Mobile Development
When running the backend in local storage mode (`STORAGE_MODE=local`), you can configure the base URL for static files using the `STATIC_BASE_URL` environment variable in `backend/.env`.

**For Android Emulator:**
```bash
STATIC_BASE_URL=http://10.0.2.2:8080/static
```
Android emulators cannot reach `localhost` from the host machine. Use `10.0.2.2` which maps to the host's `localhost`.

**For Web, iOS, or Desktop:**
```bash
STATIC_BASE_URL=http://localhost:8080/static
```
Or leave it empty to use the default value automatically.

**Why this matters:** Avatar images and other static assets are served from the backend. The mobile app uses platform-aware URL resolution to ensure images load correctly across all platforms.

_This repository enforces strict security conventions (see Threat Model) and code quality standards._

## CI/CD (GitHub Actions → Heroku)

This repo uses **GitHub Actions** for CI and deploys the backend to **Heroku** via **Heroku Container Registry**.

### Workflows
- **Backend CI**: `.github/workflows/ci.yml`
  - Runs on PRs and pushes to `main` when `backend/**` changes.
  - Executes `go test ./...` (with race detector) + `go vet`.
- **Deploy to Heroku**: `.github/workflows/deploy-heroku.yml`
  - Runs on `main` **only after** Backend CI succeeds.
  - Builds & pushes two images:
    - `web` from `backend/Dockerfile`
    - `release` from `backend/Dockerfile.release` (runs DB migrations with `RUN_MIGRATIONS_ONLY=true`)
  - Releases both process types (`web` + `release`) to Heroku.

### Required GitHub Secrets
Configure in GitHub → Settings → Secrets and variables → Actions:
- `HEROKU_API_KEY`: your Heroku API key (Account → API Key)
- `HEROKU_APP_NAME`: `klyra-backend-prod`

### Required Heroku Config Vars
These **must live in Heroku**, not in GitHub Secrets:
- `DATABASE_URL` (set automatically by Heroku Postgres addon)
- `ENV=production`, `GIN_MODE=release`
- `JWT_SECRET`, `REFRESH_TOKEN_SECRET`
- `GOOGLE_CLIENT_ID`
- `ALLOWED_ORIGINS`
- `RUN_MIGRATIONS_ON_BOOT=false`
- `STORAGE_MODE=gcs`, `GCP_PROJECT_ID`, `GOOGLE_APPLICATION_CREDENTIALS_JSON` (if using GCS/Vertex)

### Health checks
- `GET /health` → `200 {"status":"ok"}`
- `GET /health?check=db` → verifies DB connectivity
