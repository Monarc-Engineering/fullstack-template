# Deployment Runbook

Operational guide for deploying and maintaining the fullstack template in staging and production.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Environment variables](#environment-variables)
- [First-time production setup](#first-time-production-setup)
- [Pre-launch checklist](#pre-launch-checklist)
- [Triggering a deploy](#triggering-a-deploy)
- [Deploying the backend](#deploying-the-backend)
- [Deploying the web app](#deploying-the-web-app)
- [Deploying the mobile app](#deploying-the-mobile-app)
- [Database migrations](#database-migrations)
- [Secrets management](#secrets-management)
- [Rollback procedures](#rollback-procedures)
- [Monitoring and alerting](#monitoring-and-alerting)
- [Common operations](#common-operations)

---

## Prerequisites

| Tool | Purpose | Min version |
|---|---|---|
| Docker | Local Postgres + Prometheus/Grafana | 24 |
| Go | Build the backend binary | 1.26 |
| Node.js | Build the web app | 22 |
| pnpm | Web package manager | any |
| `gh` CLI | GitHub operations | 2 |
| `psql` | Database access / inspection | any |

Production accounts required:
- **Google Firebase** project with Authentication enabled
- **Cloudflare R2** bucket (if file storage is used)
- **Sentry** project for error tracking (optional)
- **Mailjet** account for transactional email (optional)
- A PostgreSQL host: [Supabase](https://supabase.com), [Neon](https://neon.tech), [AWS RDS](https://aws.amazon.com/rds/), or [Railway](https://railway.app)
- A Redis host: [Upstash](https://upstash.com), [Redis Cloud](https://redis.com/redis-enterprise-cloud/), or [Railway](https://railway.app)

---

## Environment variables

### Backend (`backend/.env` → production secrets manager)

| Variable | Required | Description |
|---|---|---|
| `PORT` | Yes | HTTP server port (default `8080`) |
| `ENV` | Yes | `local` / `staging` / `production` |
| `BLUEPRINT_DB_HOST` | Yes | PostgreSQL host |
| `BLUEPRINT_DB_PORT` | Yes | PostgreSQL port (default `5432`) |
| `BLUEPRINT_DB_DATABASE` | Yes | Database name |
| `BLUEPRINT_DB_USERNAME` | Yes | Database user |
| `BLUEPRINT_DB_PASSWORD` | Yes | Database password |
| `BLUEPRINT_DB_SSLMODE` | Yes | `disable` (local) / `require` (production) |
| `FIREBASE_PROJECT_ID` | Yes | Firebase project ID |
| `FIREBASE_SERVICE_ACCOUNT_JSON` | Yes | Service account key (single-line JSON) |
| `REDIS_URL` | No | Redis connection URL — omit to disable cache/queues |
| `CORS_ALLOWED_ORIGINS` | Yes | Comma-separated list, e.g. `https://app.example.com` |
| `SENTRY_DSN` | No | Sentry DSN — omit to disable |
| `RATE_LIMIT_RPS` | No | Requests/sec per IP (default: disabled) |
| `MAILJET_API_KEY` | No | Mailjet API key — omit to disable email |
| `R2_ACCOUNT_ID` | No | Cloudflare R2 account ID — omit to disable storage |

### Web (`web/.env.local` → Vercel environment variables)

| Variable | Required | Description |
|---|---|---|
| `AUTH_SECRET` | Yes | `openssl rand -base64 32` |
| `FIREBASE_PROJECT_ID` | Yes | Same project as backend |
| `FIREBASE_SERVICE_ACCOUNT_JSON` | Yes | Same service account as backend |
| `NEXT_PUBLIC_FIREBASE_API_KEY` | Yes | Firebase client config |
| `NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN` | Yes | Firebase client config |
| `NEXT_PUBLIC_FIREBASE_PROJECT_ID` | Yes | Firebase client config |
| `BACKEND_URL` | Yes | Go backend URL (server-side only) |
| `NEXT_PUBLIC_BACKEND_URL` | Yes | Go backend URL (client-side) |
| `NEXT_PUBLIC_SENTRY_DSN` | No | Sentry DSN |

---

## First-time production setup

### 1. Provision infrastructure

```bash
# PostgreSQL — example using Supabase CLI
supabase init && supabase start   # local reference
# For production: create a Supabase project at supabase.com

# Redis — example using Upstash
# Create a database at upstash.com → copy the Redis URL
```

### 2. Create Firebase project

1. Go to [Firebase Console](https://console.firebase.google.com) → Add project.
2. Enable **Authentication** → Sign-in methods → Google + Email/Password.
3. Add an Android app: package `com.company.template` → download `google-services.json`.
4. Add a web app → copy the client config into `web/.env.local`.
5. Project settings → Service accounts → Generate new private key → download JSON.
6. Minify to a single line: `jq -c . < service-account.json` → set as `FIREBASE_SERVICE_ACCOUNT_JSON`.

### 3. Apply database migrations

```bash
# Point to production DB
export BLUEPRINT_DB_HOST=<prod-host>
export BLUEPRINT_DB_USERNAME=<prod-user>
export BLUEPRINT_DB_PASSWORD=<prod-password>
export BLUEPRINT_DB_DATABASE=<prod-db>
export BLUEPRINT_DB_SSLMODE=require

cd backend && make migrate-up
```

### 4. Deploy backend (first time)

See [Deploying the backend](#deploying-the-backend).

---

## Pre-launch checklist

Run through this before going live with any project based on this template.

### Configuration
- [ ] `ENV` is set to `production` — enables JSON logging and disables `/debug/pprof` and `/admin/queues`
- [ ] `AUTH_SECRET` is a strong random value: `openssl rand -base64 32`
- [ ] `CORS_ALLOWED_ORIGINS` lists your exact domain(s) — never `*` in production
- [ ] `BLUEPRINT_DB_SSLMODE` is `require`
- [ ] `RATE_LIMIT_RPS` is configured to a sensible value for your expected traffic

### Infrastructure
- [ ] Database backups are enabled on your PostgreSQL host (Supabase, Neon, and RDS all have this in their dashboard — turn it on)
- [ ] Run `make migrate-up` against the production database before the first deploy, and on every subsequent deploy that includes migrations
- [ ] If using R2 file uploads, CORS rules on the R2 bucket allow requests from your web domain

### Firebase
- [ ] Firebase Authentication is enabled with only the sign-in methods your app uses
- [ ] Firebase security rules are reviewed (default rules may be too permissive)
- [ ] `FIREBASE_SERVICE_ACCOUNT_JSON` in production is the production project's key, not a dev key

### Observability
- [ ] `SENTRY_DSN` (backend) and `NEXT_PUBLIC_SENTRY_DSN` (web) are set — you want errors reported from day one
- [ ] The `/health` endpoint returns `200` after deploying (confirms DB connectivity)
- [ ] Grafana is accessible and the **Backend Overview** dashboard shows live data (`http://<host>:3001`)

### Mobile (if shipping Android)
- [ ] `google-services.json` in the release build points to the production Firebase project
- [ ] Release APK/AAB is signed with the production keystore (not the debug keystore)
- [ ] Staged rollout is configured in Google Play Console before full release

---

## Triggering a deploy

This template uses two branches as deploy triggers, kept separate from `main`:

| Branch | Advances | Trigger |
|---|---|---|
| `staging` | Automatically, on every merge to `main` | GitHub Actions (`.github/workflows/sync-staging.yml`) |
| `production` | Manually, whenever you choose | `make deploy-prod` |

Configure your hosting platforms (Vercel, Railway, Fly.io, etc.) to auto-deploy from these branches — `staging` for the staging environment, `production` for production — rather than from `main`. That way merges to `main` land safely, staging always reflects the latest `main`, and a production deploy only fires when you explicitly advance `production`.

### Staging (automatic)

`.github/workflows/sync-staging.yml` runs on every push to `main` (i.e. every merge) and calls `scripts/sync-staging.sh`, which:

1. Fetches `origin/main` and `origin/staging`.
2. Checks out (or creates) a local `staging` branch reset to `origin/staging`.
3. Rebases it onto `origin/main`.
4. Force-pushes `staging` with `--force-with-lease`.

There is no confirmation prompt — unlike production, staging is meant to always match `main`. If the rebase hits a conflict (only possible if someone committed directly to `staging`), the job aborts the rebase and fails without pushing; fix it manually with the same commands `scripts/sync-staging.sh` prints on failure, or delete stray commits from `staging` so it stops diverging from `main`.

You can also trigger the workflow manually from the **Actions** tab (`workflow_dispatch`), or run `bash scripts/sync-staging.sh` locally.

**The workflow checks out with a personal access token (PAT), not the default `GITHUB_TOKEN`.** This matters because of GitHub's built-in loop-prevention rule: pushes made with the default `GITHUB_TOKEN` do **not** trigger other `on: push` workflows. If your CD workflow deploys on `push: branches: [staging]`, a `GITHUB_TOKEN`-authenticated push here would update the `staging` ref without ever firing that deploy — it would look like the sync succeeded but nothing would actually deploy. A PAT belonging to a real account avoids that.

To set it up:
1. Generate a PAT with push access to this repo — a fine-grained token scoped to this repo with **Contents: Read and write** is enough; a classic token needs the `repo` scope.
2. Add it as a repository secret named `SYNC_STAGING_PAT` (**Settings → Secrets and variables → Actions → New repository secret**).
3. If `staging` is a protected branch, make sure the PAT's account is allowed to bypass (or is exempt from) those protection rules, otherwise the force-push will still be rejected.

The commit identity used for the sync (`git config user.name`/`user.email`) is resolved from `github.actor` — whoever triggered the workflow (typically whoever merged the PR) — not hardcoded, so this works the same for any fork of this template.

### Production (manual)

Advance `production` to match `main` with:

```bash
make deploy-prod
```

This runs `scripts/deploy-prod.sh` (macOS/Linux) or `scripts/deploy-prod.ps1` (Windows) via the `deploy-prod` Makefile target, which handles both platforms automatically. The script:

1. Aborts if the working tree is dirty — commit, stash, or discard changes first.
2. Fetches `origin/main` and `origin/production`.
3. Checks out (or creates) a local `production` branch reset to `origin/production`.
4. Rebases it onto `origin/main`.
5. Shows the commits about to ship and prompts `Continue? [y/N]` before pushing.
6. Force-pushes `production` with `--force-with-lease` (never a bare `--force`), so the push fails instead of clobbering someone else's work if `production` moved remotely since the last fetch.

Set `CONFIRM=yes` to skip the interactive prompt (e.g. from CI):

```bash
CONFIRM=yes make deploy-prod
```

If the rebase hits conflicts, the script leaves the branch mid-rebase and prints next steps — resolve the conflicts, then run `git rebase --continue` followed by `git push --force-with-lease origin production`, or `git rebase --abort` to back out. The original branch is restored automatically on both success and abort.

`make deploy-prod` only advances the branch — it does not run database migrations or build anything. Apply pending migrations against production **before** running it; see [Database migrations](#database-migrations).

---

## Deploying the backend

The backend compiles to a single binary. Choose one deployment model:

### Option A — Docker (recommended)

```dockerfile
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o /server ./cmd/api

FROM alpine:3.21
COPY --from=build /server /server
EXPOSE 8080
CMD ["/server"]
```

```bash
docker build -t fullstack-backend:latest .
docker push <registry>/fullstack-backend:latest
# Deploy image to Railway, Fly.io, Cloud Run, ECS, etc.
```

### Option B — Binary on a VM

```bash
cd backend
GOOS=linux GOARCH=amd64 go build -o server-linux ./cmd/api
scp server-linux user@host:/opt/app/server
ssh user@host "systemctl restart app"
```

### Environment injection

Inject all backend environment variables as process environment (not a `.env` file) in production. Most PaaS platforms have a secrets/env UI. For self-hosted, use systemd `EnvironmentFile` or a secrets manager (AWS SSM, Doppler, 1Password Secrets Automation).

---

## Deploying the web app

### Vercel (recommended)

```bash
# Install Vercel CLI once
pnpm add -g vercel

cd web
vercel --prod
```

Set all environment variables in the Vercel dashboard under **Settings → Environment Variables**. Variables prefixed `NEXT_PUBLIC_` are embedded at build time.

### Self-hosted (Docker / Railway)

```bash
cd web
pnpm build          # outputs .next/
# Use the official Next.js Docker image or a Dockerfile with `next start`
```

The `BACKEND_URL` must be the internal URL of the backend container (e.g. `http://backend:8080`) when both run in the same Docker network.

---

## Deploying the mobile app

### Debug APK (internal testing)

```bash
cd mobile
./gradlew assembleDebug
# Output: app/build/outputs/apk/debug/app-debug.apk
```

### Release APK / AAB (Google Play)

1. Generate a signing keystore (once):
   ```bash
   keytool -genkey -v -keystore release.jks -alias release -keyalg RSA -keysize 2048 -validity 10000
   ```
2. Add signing config to `mobile/app/build.gradle.kts` (do not commit the keystore).
3. Build signed release bundle:
   ```bash
   cd mobile
   ./gradlew bundleRelease
   # Output: app/build/outputs/bundle/release/app-release.aab
   ```
4. Upload the `.aab` to [Google Play Console](https://play.google.com/console).

### CI signing (GitHub Actions)

Store the keystore as a base64 secret:
```bash
base64 -w 0 release.jks > release.jks.b64
# Add content as GitHub secret KEYSTORE_BASE64
```

In the workflow:
```yaml
- name: Decode keystore
  run: echo "${{ secrets.KEYSTORE_BASE64 }}" | base64 -d > mobile/app/release.jks
```

---

## Database migrations

**Rule: always apply migrations before deploying a new backend version.**

```bash
# Check pending migrations
cd backend && make migrate-status

# Apply all pending migrations
cd backend && make migrate-up

# Apply only the next migration (safer for large schemas)
cd backend && make migrate-up-one

# Roll back the last migration
cd backend && make migrate-down

# View current version
cd backend && make migrate-version
```

**Never edit or delete an applied migration.** If a migration was applied incorrectly, add a new migration to correct it.

### Zero-downtime migrations

For tables with millions of rows, prefer:
1. Additive changes first (add nullable column, add index `CONCURRENTLY`).
2. Deploy code that handles both old and new schema.
3. Backfill data in batches.
4. Add `NOT NULL` constraint in a separate migration after backfill.

---

## Secrets management

### Local development

Copy `.env.example` to `.env` (backend) and `.env.local` (web). Never commit either file.

```bash
cp backend/.env.example backend/.env
cp web/.env.example web/.env.local
```

### Staging / production

| Platform | Recommended approach |
|---|---|
| Vercel | Environment Variables UI → mark secrets as "Production" only |
| Railway | Variables tab per service |
| Fly.io | `fly secrets set KEY=value` |
| AWS | SSM Parameter Store + IAM role |
| Self-hosted | Doppler, 1Password Secrets, or `systemd EnvironmentFile` |

**Rotation procedure:**
1. Generate new secret value.
2. Add new value to secrets manager.
3. Redeploy the service (picks up new value from environment).
4. Revoke the old value.

---

## Rollback procedures

### Backend rollback

```bash
# Redeploy the previous image tag
docker pull <registry>/fullstack-backend:<previous-tag>
# Update the deployment to point to the previous tag
```

If the new version introduced a migration that needs to be undone:
```bash
cd backend && make migrate-down   # rolls back last applied migration
# Then redeploy previous binary
```

### Web rollback

On Vercel: **Deployments** tab → select the previous successful deployment → **Promote to Production**.

On self-hosted: redeploy the previous Docker image or binary.

### Mobile rollback

Mobile apps cannot be forced-rollback for users who have already updated. Options:
- Use a **feature flag** to disable the problematic feature remotely.
- Release a hotfix version to the Play Store (fastest path: ~1-2 hrs for expedited review).
- Halt the staged rollout in Google Play Console before it reaches 100%.

---

## Monitoring and alerting

### Prometheus + Grafana (local / self-hosted)

```bash
cd backend && make docker-run   # starts Postgres + Prometheus + Grafana
```

- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3001` (default credentials: `admin` / `admin`, override with `GRAFANA_ADMIN_USER` / `GRAFANA_ADMIN_PASSWORD`)
- Metrics endpoint: `http://localhost:8080/metrics` (restricted to loopback/private IPs in production)

### Sentry error tracking

Set `SENTRY_DSN` (backend) and `NEXT_PUBLIC_SENTRY_DSN` (web). Errors are captured automatically; the backend uses the Gin middleware, the web uses the Next.js Sentry plugin.

### pprof profiling

Available at `/debug/pprof/` — always restricted to loopback and RFC 1918 addresses:

```bash
# CPU profile (30 s)
go tool pprof http://localhost:8080/debug/pprof/profile

# Heap snapshot
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine dump
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

---

## Common operations

### Restart the backend

```bash
# Docker / Railway / Fly.io
docker restart <container>
# or trigger a redeploy in the platform UI
```

### Access the production database

```bash
psql "postgresql://<user>:<password>@<host>:5432/<database>?sslmode=require"
```

### Inspect job queues (Asynq)

In debug mode, the Asynqmon UI is at `http://localhost:8080/admin/queues`. For production, either:
- Run the backend locally pointed at the production Redis URL, or
- Use the [Asynq CLI](https://github.com/hibiken/asynq): `asynq dash --uri=<redis-url>`

### Clear Redis cache

```bash
redis-cli -u <REDIS_URL> FLUSHDB   # clears the current database only
```

### Scale the backend horizontally

The backend is stateless — multiple instances can run behind a load balancer. Ensure:
- `REDIS_URL` is set (rate limiter uses Redis for distributed counting).
- Database connection pool size (`BLUEPRINT_DB_*`) is tuned per instance.
- The load balancer forwards `X-Forwarded-For` and is listed in trusted proxies if needed.
