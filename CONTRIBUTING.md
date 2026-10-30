# Contributing to Fullstack Template

Thank you for your interest in contributing! This document covers how to report bugs, suggest features, and submit pull requests.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Features](#suggesting-features)
  - [Submitting Pull Requests](#submitting-pull-requests)
- [Development Setup](#development-setup)
- [Keeping Documentation Current](#keeping-documentation-current)
- [Style Guidelines](#style-guidelines)
- [Commit Messages](#commit-messages)

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold it.

## How Can I Contribute?

### Reporting Bugs

Before submitting a bug report:

1. Search the [existing issues](../../issues) to see if it has already been reported.
2. Make sure you are running the latest version.

When opening a bug report, include:

- A clear, descriptive title
- Steps to reproduce the behavior
- What you expected to happen
- What actually happened (include error messages and stack traces)
- Your environment (OS, Go version, Node version, Docker version)

### Suggesting Features

Feature suggestions are welcome. Open an issue with:

- A clear title prefixed with `[Feature]`
- A description of the problem you are trying to solve
- An explanation of your proposed solution
- Any alternatives you have considered

### Submitting Pull Requests

1. **Fork** the repository and create your branch from `main`:

   ```bash
   git checkout -b feat/my-feature
   # or
   git checkout -b fix/my-bug
   ```

2. **Set up** the development environment following the [README](README.md#getting-started).

3. **Make your changes.** Keep them focused — one logical change per PR.

4. **Add or update tests** when changing behaviour. Integration tests should use Testcontainers, not mocks.

5. **Verify** everything passes before opening the PR:

   ```bash
   # Backend
   cd backend && go vet ./...
   cd backend && make lint      # golangci-lint
   cd backend && make test

   # Web
   cd web && pnpm lint
   cd web && pnpm test
   cd web && pnpm build

   # Mobile
   cd mobile && ./gradlew lint
   cd mobile && ./gradlew spotlessCheck   # ktlint formatting
   cd mobile && ./gradlew test
   ```

6. **Open your pull request** against `main`. Fill in the PR template including:
   - What changed and why
   - Screenshots or recordings for UI changes
   - Related issue numbers (`Fixes #123`)

7. A maintainer will review your PR. Address any requested changes, then the PR will be merged.

## Development Setup

### Prerequisites

| Tool | Minimum version | Check |
|---|---|---|
| Go | 1.26 | `go version` |
| Node.js | 22 | `node --version` |
| pnpm | any | `pnpm --version` |
| Docker Desktop | running | `docker info` |
| JDK | 17 | `java -version` |
| Android SDK | API 35 | `$ANDROID_HOME` set |

### First-time setup

Run the setup script once after cloning. It checks prerequisites, installs dependencies, copies `.env` files, and runs database migrations:

```bash
# macOS / Linux
./scripts/setup.sh

# Windows
./scripts/setup.ps1
```

### Starting all services

```bash
# macOS / Linux — starts Postgres, backend, and web in one terminal
./scripts/dev.sh

# Windows — opens three separate PowerShell windows
./scripts/dev.ps1
```

Or start services individually:

```bash
cd backend && make docker-run   # Postgres
cd backend && make watch        # backend hot reload → :8080
cd web && pnpm dev              # web → :3000

# Mobile — open mobile/ in Android Studio, or:
cd mobile && ./gradlew assembleDebug
```

## Keeping Documentation Current

This project uses topic-based documentation in `backend/docs/` and `web/docs/` to give AI coding agents accurate, up-to-date context. When your changes affect how the project works, update the relevant doc alongside your code — not in a separate PR.

### What to update

| If you change… | Update this doc |
|---|---|
| DB queries, connection setup, or the `Service` interface | `backend/docs/database.md` |
| Routes, handlers, or CORS config | `backend/docs/routing.md` |
| Test setup or testing patterns (backend) | `backend/docs/testing.md` |
| Error handling conventions | `backend/docs/error-handling.md` |
| Environment variables | `backend/docs/environment.md` |
| App Router structure or route files | `web/docs/routing.md` |
| Data fetching or Server Actions | `web/docs/data-fetching.md` |
| Tailwind or CSS conventions | `web/docs/styling.md` |
| Component patterns or TypeScript conventions | `web/docs/components.md` |
| Composable conventions, theme, or Material3 usage | `mobile/docs/compose-conventions.md` |
| Activity setup, lifecycle, or ViewModel pattern | `mobile/docs/architecture.md` |
| UiState<T>, UiStateContent, or loading/error patterns | `mobile/docs/ui-states.md` |
| Test setup or testing patterns (mobile) | `mobile/docs/testing.md` |
| Data table component or TanStack Table usage | `web/docs/data-table.md` |

### How to update a doc

1. Edit the relevant file in `backend/docs/` or `web/docs/`.
2. Update the `last_verified` date in the frontmatter to today's date.
3. Update the `sources` list if you added or removed source files.
4. If you introduce a new topic that isn't covered, create a new doc file and add it to the relevant `_index.md`.

The `AGENTS.md` files at the root, `backend/`, `web/`, and `mobile/` are entry points for AI agents — update them if you change project-level setup commands, tooling, or structure.

## Style Guidelines

### Go (backend)

- Follow standard Go conventions (`gofmt`, `go vet`).
- Keep packages small and focused: business logic in `internal/`, wiring in `cmd/`.
- No exported symbols without a doc comment.
- Use table-driven tests.

### TypeScript/React (web)

- All new files should be TypeScript (`.ts` / `.tsx`), not JavaScript.
- Follow the existing ESLint configuration.
- Prefer server components by default; use client components only when interactivity is required.
- Co-locate component-specific styles with the component.

### Kotlin/Compose (mobile)

- Single Activity — no new Activities or Fragments.
- Keep `@Composable` functions stateless; hoist state to a ViewModel.
- Use Material3 (`androidx.compose.material3`) only.
- Declare all dependency versions in `mobile/gradle/libs.versions.toml`.
- Add `@Preview(showBackground = true)` to every public Composable.

### General

- Keep pull requests small and reviewable — avoid mixing unrelated changes.
- Write meaningful variable and function names; avoid comments that just restate the code.
- Do not commit `.env` files or secrets.

## Commit Messages

Use the [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<scope>): <short description>

[optional body]

[optional footer]
```

| Type       | When to use                                      |
|------------|--------------------------------------------------|
| `feat`     | A new feature                                    |
| `fix`      | A bug fix                                        |
| `docs`     | Documentation changes only                       |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `test`     | Adding or updating tests                         |
| `chore`    | Build process, tooling, or dependency updates    |

Examples:

```
feat(backend): add JWT authentication middleware
fix(web): correct layout shift on mobile viewport
feat(mobile): add profile screen with Material3 card layout
docs: add environment variable table to README
```
