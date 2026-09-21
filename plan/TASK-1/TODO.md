# TASK-1 — Project skeleton & Phase 0 setup

## Objective
Bootable monorepo: Go backend skeleton with health endpoint, full §6 schema via migrations, seed topics, docker-compose dev environment. Spec: `REQUIREMENT.md` §6 (schema), §11 (layout), Phase 0/1.

## Prerequisites
- None (starting point). User-owned externals to chase early (do not block build on them): LINE OA + Messaging API channel, AI API key, homelab Postgres + outside-reachable URL.

## Checklist

### Repo layout (§11)
- [x] Create directories: `cmd/server/`, `internal/handler/`, `internal/service/`, `internal/repository/`, `internal/ai/`, `internal/model/`, `migrations/`.
- [x] `go mod init` module for the project.
- [x] Minimal `cmd/server/main.go`: loads config, connects (or lazily connects) to Postgres, serves `GET /health` returning 200.
- [x] Placeholder `.gitkeep` or doc.go in empty packages so layout is visible.

### Config
- [x] Env var loading for: `DATABASE_URL`, `AI_API_KEY`, `INTERNAL_TOKEN`, `WEBHOOK_URL`.
- [x] `.env.example` documenting every variable (no real secrets).
- [x] `.gitignore` covers `.env`, binaries, node_modules (frontend dir comes in TASK-8).

### Migrations (§6)
- [x] Pick migration tool: `golang-migrate`; wired into docker-compose via `migrate/migrate` service.
- [x] Write up/down migrations for all §6 tables: `topics`, `quiz_batches`, `questions`, `user_answers`, `user_topic_stats`, `generation_logs` — tables, constraints, defaults exactly per §6 (including unique `batch_date`).
- [x] Seed migration: insert topics `Frontend`, `Backend`, `Infrastructure`.
- [x] Verify `migrate down` then `up` runs clean, twice.

### Dev environment
- [x] `docker-compose.yml`: Postgres 16 service (with volume) + backend service (`go run`), env wired, migrations run automatically.
- [x] `.env` (local, uncommitted) with local Postgres creds.
- [x] README stub: how to `docker compose up`, run migrations, hit `/health`.

## Done when
- [x] `docker compose up` boots Postgres + backend; `GET /health` responds 200.
- [x] Migrations run clean up/down; seed topics present in `topics`.
- [x] Schema diff vs §6: every table, column, constraint, default matches.
- [x] No secrets committed.
