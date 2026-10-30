# Fullstack Template

A production-ready fullstack starter. Clone it, rename things, and focus on your business logic — the infrastructure is already wired. Ships with a Go + Gin backend, Next.js 16 web app, Android mobile app (Kotlin + Compose), PostgreSQL, Docker Compose, hot reload, integration testing, and a full agentic development setup for AI coding assistants.

## Table of Contents

- [Features](#features)
- [Tech Stack](#tech-stack)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Quick start](#quick-start)
  - [Manual setup](#manual-setup)
- [Project Structure](#project-structure)
- [Environment Variables](#environment-variables)
- [Development commands](#development-commands)
- [Testing](#testing)
- [Working with AI Agents](#working-with-ai-agents)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Go backend** with Gin, structured into clean `cmd/` and `internal/` layers (domain → usecase → infrastructure → transport)
- **Next.js 16 web app** with React 19, TypeScript 5, Tailwind CSS 4, tRPC, and NextAuth v5
- **Android mobile app** with Kotlin 2.2, Jetpack Compose BOM 2026.02, and Material3
- **PostgreSQL 16** managed via Docker Compose with goose migrations
- **Hot reload** on both web (`pnpm dev`) and backend ([Air](https://github.com/air-verse/air))
- **Integration tests** using [Testcontainers](https://testcontainers.com/) — no mocks, real DB
- **Firebase Authentication** across web and Android — Google OAuth + email/password
- **Redis + Asynq** for background job queues (opt-in)
- **pprof** profiling endpoints restricted to loopback/private IPs
- **Prometheus + Grafana** observability stack in Docker Compose
- **Renovate** for automated dependency updates (Monday morning runs, minor/patch automerge)
- **golangci-lint**, **ktlint (Spotless)**, and **ESLint** wired into CI
- **Agentic infrastructure** — AGENTS.md, CLAUDE.md, topic docs, subagents, hooks, and slash commands ready out of the box for all three layers

## Tech Stack

| Layer | Technology |
|---|---|
| Web | Next.js 16, React 19, TypeScript 5, Tailwind CSS 4 |
| Backend | Go 1.26, Gin v1.12 |
| Database | PostgreSQL 16 (via Docker), goose migrations, pgx v5 |
| Auth | Firebase Authentication (web + Android) |
| Background jobs | Redis, Asynq (opt-in) |
| Mobile | Android, Kotlin 2.2, Jetpack Compose BOM 2026.02, Material3 |
| Dev tools | Air (hot reload), pnpm, Docker, Gradle 9.4 |
| Testing | Testcontainers (Go), Vitest + Testing Library (web), JUnit 4 + Compose test rules (Android) |
| Observability | Prometheus, Grafana, Sentry |

## Getting Started

### Prerequisites

| Tool | Min version | Notes |
|---|---|---|
| [Go](https://go.dev/dl/) | 1.26 | |
| [Node.js](https://nodejs.org/) | 22 | |
| [pnpm](https://pnpm.io/installation) | any | `npm i -g pnpm` |
| [Docker Desktop](https://www.docker.com/) | 24 | Required for Postgres and integration tests |
| [Android Studio Meerkat (2024.3+)](https://developer.android.com/studio) | — | For mobile; needs SDK API 36 + JDK 17 |
| [Air](https://github.com/air-verse/air) | any | `go install github.com/air-verse/air@latest` |

### Quick start

```bash
git clone https://github.com/your-username/fullstack-template.git
cd fullstack-template

# First-time setup: installs deps, copies .env.example files, checks prerequisites
./scripts/setup.sh        # macOS / Linux
.\scripts\setup.ps1       # Windows PowerShell

# Start all three services (backend + web + mobile hot-reload) in parallel
./scripts/dev.sh          # macOS / Linux
.\scripts\dev.ps1         # Windows PowerShell
```

### Manual setup

**Backend:**
```bash
cd backend
cp .env.example .env   # fill in your values
go mod download
make docker-run        # start Postgres
make watch             # hot reload via Air → :8080
```

**Web:**
```bash
cd web
pnpm install
cp .env.example .env.local   # fill in Firebase + backend URL
pnpm dev                     # → :3000
```

**Mobile:**

Open `mobile/` in Android Studio. The Gradle wrapper handles all SDK downloads. Copy `google-services.json` from your Firebase project into `mobile/app/`.

```bash
cd mobile && ./gradlew installDebug   # build and install on connected device/emulator
```

## Project Structure

```
fullstack-template/
├── AGENTS.md                    # AI agent instructions (all agents)
├── CLAUDE.md                    # Claude Code workflow and conventions
├── CONTRIBUTING.md              # Contributor guide
├── RUNBOOK.md                   # Deployment and operations guide
├── TEMPLATE_STATUS.md           # Readiness gap tracker
├── docs/
│   └── adr/                     # Architecture Decision Records
├── .github/
│   └── workflows/
│       ├── backend-ci.yml / web-ci.yml / mobile-ci.yml  # Per-layer CI on PRs
│       ├── labeler.yml                # Auto-labels PRs by changed path
│       └── sync-staging.yml           # Auto-syncs `staging` to `main` on every merge
├── scripts/
│   ├── dev.sh / dev.ps1               # Start all services in parallel
│   ├── setup.sh / setup.ps1           # First-run contributor setup
│   ├── deploy-prod.sh / deploy-prod.ps1  # Rebase production onto main and force-push (with lease) — manual
│   └── sync-staging.sh                # Rebase staging onto main and force-push (with lease) — run by CI
├── renovate.json                # Automated dependency updates
├── .claude/
│   ├── agents/                  # Specialized Claude subagents
│   ├── commands/                # Custom slash commands
│   └── hooks/                   # Auto-format + guard hooks
├── backend/
│   ├── docs/                    # Topic docs: routing, testing, migrations, auth, …
│   ├── cmd/
│   │   ├── api/main.go          # Entry point — wires all layers
│   │   └── migrate/main.go      # Migration CLI (goose)
│   ├── internal/
│   │   ├── domain/              # Layer 1: entities (no external deps)
│   │   ├── usecase/             # Layer 2: application logic + interfaces
│   │   ├── infrastructure/
│   │   │   ├── database/postgres/  # Repository implementations + Testcontainers tests
│   │   │   ├── database/migrations/ # SQL migration files (goose)
│   │   │   ├── cache/redis/     # Redis cache implementation
│   │   │   ├── queue/           # Asynq task definitions and worker
│   │   │   └── streams/         # Redis Streams producer/consumer (opt-in)
│   │   ├── transport/
│   │   │   ├── handlers/        # HTTP handlers + routes
│   │   │   └── middleware/      # Logger, auth, rate limiter, pprof guard
│   │   └── server/server.go     # Wires all layers → *http.Server
│   ├── pkg/                     # Shared packages (logger, firebase)
│   ├── .env.example
│   ├── docker-compose.yml       # Postgres + Prometheus + Grafana
│   └── Makefile
├── web/
│   ├── docs/                    # Topic docs: routing, auth, tRPC, data-fetching, …
│   ├── app/                     # Next.js App Router
│   │   ├── (auth)/              # Unauthenticated pages (login, register)
│   │   └── (dashboard)/         # Protected pages
│   ├── components/              # Shared UI components (DataTable, layout, common)
│   ├── features/                # Domain-scoped feature folders
│   ├── lib/                     # Pure utilities and non-React helpers
│   ├── server/                  # tRPC routers and server-side logic
│   └── .env.example
└── mobile/
    ├── docs/                    # Topic docs: compose-conventions, architecture, …
    ├── app/src/main/java/com/company/template/
    │   ├── MainActivity.kt      # Single Activity entry point
    │   ├── navigation/          # AppNavGraph, route constants
    │   ├── ui/state/            # UiState<T> sealed class + UiStateContent composable
    │   ├── ui/theme/            # Color, Theme, Type (Material3)
    │   └── data/network/        # ApiClient, OkHttp interceptors
    └── gradle/libs.versions.toml  # All dependency versions declared here
```

## Environment Variables

Copy the example files and fill in your values. Never commit `.env` or `.env.local`.

```bash
cp backend/.env.example backend/.env
cp web/.env.example web/.env.local
```

Key backend variables (`backend/.env.example` has the full list):

| Variable | Required | Description |
|---|---|---|
| `PORT` | Yes | HTTP server port (default `8080`) |
| `ENV` | Yes | `local` / `staging` / `production` |
| `BLUEPRINT_DB_*` | Yes | PostgreSQL connection (host, port, db, user, password, sslmode) |
| `FIREBASE_PROJECT_ID` | Yes | Firebase project ID |
| `FIREBASE_SERVICE_ACCOUNT_JSON` | Yes | Service account key (single-line JSON) |
| `CORS_ALLOWED_ORIGINS` | Yes | Comma-separated allowed origins |
| `REDIS_URL` | No | Redis URL — omit to disable caching and job queues |
| `SENTRY_DSN` | No | Sentry DSN — omit to disable error tracking |
| `R2_ACCOUNT_ID` | No | Cloudflare R2 — omit to disable object storage |

See `RUNBOOK.md` for the full environment variable reference and production setup guide.

## Development commands

### Backend

```bash
make docker-run      # start Postgres + Prometheus + Grafana
make watch           # hot reload via Air
make run             # run once, no hot reload
make build           # compile binary
make test            # unit + integration tests
make itest           # integration tests only (requires Docker)
make lint            # golangci-lint
make swagger         # regenerate Swagger docs
make docker-down     # stop containers
make clean           # remove compiled binary

# Migrations
make migrate-create name=<slug>   # create timestamped migration file
make migrate-up                   # apply all pending migrations
make migrate-down                 # roll back last migration
make migrate-status               # show applied vs. pending
make migrate-version              # print current schema version
```

### Web

```bash
pnpm dev          # dev server → :3000
pnpm build        # production build + TypeScript check
pnpm start        # serve production build
pnpm lint         # ESLint
pnpm test         # Vitest unit + component tests
pnpm test:watch   # Vitest watch mode (use during TDD)
```

### Mobile

```bash
./gradlew assembleDebug          # compile debug APK
./gradlew installDebug           # build + install on device/emulator
./gradlew lint                   # Android lint
./gradlew test                   # unit tests (JVM, no device)
./gradlew connectedAndroidTest   # instrumented tests (device/emulator required)
./gradlew spotlessCheck          # ktlint formatting check
./gradlew spotlessApply          # auto-fix ktlint formatting
./gradlew clean                  # clean build outputs
```

On Windows outside Git Bash, use `.\gradlew.bat` instead of `./gradlew`.

### Deploy

```bash
make deploy-prod   # rebase the `production` branch onto `main` and force-push (with lease) to trigger a deploy
```

`staging` syncs to `main` automatically on every merge via `.github/workflows/sync-staging.yml` — no manual step needed. `production` only advances when you run `make deploy-prod`.

See [RUNBOOK.md](RUNBOOK.md#triggering-a-deploy) for the full deploy workflow, prerequisites, and platform setup.

## Testing

Backend tests use [Testcontainers](https://testcontainers.com/) to spin up real PostgreSQL and Redis instances — database mocking is prohibited.

```bash
cd backend
make test    # unit + integration tests
make itest   # integration tests only (Docker must be running)
```

Web tests use [Vitest](https://vitest.dev/) with `@testing-library/react`:

```bash
cd web
pnpm test         # run once
pnpm test:watch   # watch mode
```

Mobile has two test tiers:

```bash
cd mobile
./gradlew test                   # unit tests — JVM only, no device needed
./gradlew connectedAndroidTest   # instrumented tests — requires emulator or device
```

## Working with AI Agents

This template ships with a complete agentic development setup so AI assistants have the context they need to work accurately and consistently.

### For any AI coding agent

A layered `AGENTS.md` system follows the [AGENTS.md open standard](https://agents.md). The closest file to the code you are editing takes precedence:

| File | Covers |
|---|---|
| [`AGENTS.md`](AGENTS.md) | Project overview, setup, cross-cutting conventions, security |
| [`backend/AGENTS.md`](backend/AGENTS.md) | Go commands, project structure, links to topic docs |
| [`web/AGENTS.md`](web/AGENTS.md) | pnpm commands, Next.js conventions, links to topic docs |
| [`mobile/AGENTS.md`](mobile/AGENTS.md) | Gradle commands, Android conventions, links to topic docs |

Topic-specific documentation lives in `backend/docs/`, `web/docs/`, and `mobile/docs/`. Each file is kept in sync with the source code it describes and includes `last_verified` metadata so agents can detect when it may be stale.

### For Claude Code

Additional infrastructure in `.claude/` provides a deeper integration:

| Path | Purpose |
|---|---|
| [`CLAUDE.md`](CLAUDE.md) | Feature development workflow, all conventions |
| `.claude/agents/` | Specialized subagents: `backend`, `web`, `mobile`, `reviewer`, `db-explorer`, `docs` |
| `.claude/commands/` | Slash commands: `/project:implement`, `/project:check`, `/project:test`, `/project:new-route` |
| `.claude/hooks/` | Auto-formats Go and TypeScript files on save; blocks dangerous commands |

The recommended workflow for any implementation:

1. Check the relevant topic doc in `backend/docs/`, `web/docs/`, or `mobile/docs/` before writing code
2. Implement against documented patterns rather than general training data
3. Update the doc file after implementation so the next agent session starts with accurate context

### Architecture decisions

Major technology choices are documented in [`docs/adr/`](docs/adr/) as Architecture Decision Records — why each technology was chosen and what trade-offs were accepted.

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

For deployment and operations, see [RUNBOOK.md](RUNBOOK.md).

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
