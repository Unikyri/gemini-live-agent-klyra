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

_This repository enforces strict security conventions (see Threat Model) and code quality standards._
