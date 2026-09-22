# Quizme — Progress Log

## 2026-09-21 — TASK-1 done

Implemented project skeleton and Phase 0 setup per `plan/TASK-1/TODO.md` and `REQUIREMENT.md` §6/§11.

### Changed
- Initialized Go module `github.com/natdanai/quizme`.
- Created repo layout: `cmd/server/`, `internal/{handler,service,repository,ai,model}/`, `migrations/`, `frontend/`.
- `cmd/server/main.go`: stdlib HTTP server, `GET /health` returns `{"status":"ok"}` on `:8080`.
- `internal/config/config.go`: env loading for `DATABASE_URL`, `AI_API_KEY`, `INTERNAL_TOKEN`, `WEBHOOK_URL`, `PORT`.
- `internal/model/models.go`: structs for all §6 tables.
- `internal/{handler,service,repository,ai}/doc.go`: package placeholders.
- Migrations 000001–000006 for `topics`, `quiz_batches`, `questions`, `user_answers`, `user_topic_stats`, `generation_logs` matching §6 exactly (constraints, defaults, FKs).
- Migration 000007 seeds `Frontend`, `Backend`, `Infrastructure`.
- `docker-compose.yml`: Postgres 16 + auto-migrate + backend (`go run`), env wired.
- `.env.example`, `.env` (local), `.gitignore`.
- Updated `README.md` for current env vars, commands, and migration path.

### Verified
- `go test ./...` passes.
- `gofmt` clean.
- Health endpoint responds 200 locally and inside docker compose.
- `migrate up` / `down -all` / `up` runs clean twice against Postgres.
- `pg_dump --schema-only` confirms schema matches §6.
- Seed topics present after `docker compose up`.
- No secrets committed (`.env` is gitignored).

### Blockers / Notes
- Docker daemon was not running initially; installed/started rootless Docker (`dockerd-rootless`) to verify `docker compose up`. Podman was also available as fallback.
- Next task: TASK-2 repository layer.

## 2026-09-22 — TASK-2 done

Implemented repository layer per `plan/TASK-2/TODO.md` and `PLAN.md`.

### Changed
- Fixed nullable columns in `internal/model/models.go`: `Question.Explanation`, `UserTopicStats.LastPracticedAt`, `GenerationLog.BatchID/ErrorMessage/TokensUsed` are now pointers.
- Added `pgx/v5` + `pgxpool` dependency.
- Created `internal/repository/repo.go` with aggregate `Repo` and methods:
  - `ListTopics`, `CreateTopic`
  - `GetOrCreateBatch` (insert-on-conflict, returns `(batch, created bool)`)
  - `UpdateBatchStatus`, `LatestSuccessfulBatch`
  - `InsertQuestions`, `GetQuestionsByBatch` (JSONB round-trip via `json.Marshal`/`Unmarshal`)
  - `SubmitAnswer` (single tx: insert `user_answers` + upsert `user_topic_stats` with SQL accuracy recompute; rollback on any failure)
  - `GetStats`
  - `InsertGenerationLog`, `RecentGenerationLogs`
- Deleted redundant `internal/repository/doc.go`.
- Added `internal/repository/repo_test.go`: skips when `TEST_DATABASE_URL` is unset; covers topics, batch idempotency, question JSONB round-trip, answer stats math, mid-tx rollback, generation logs.
- Updated `plan/TASK-2/TODO.md`, `plan/TASK.md`, and `plan/OVERVIEW.md` to mark TASK-2 done.

### Verified
- `gofmt -w .` clean.
- `go build ./...` green.
- `go vet ./...` green.
- `go test ./...` green (repository tests skip without `TEST_DATABASE_URL`).
- `TEST_DATABASE_URL=... go test ./internal/repository/ -v` passes all 8 integration tests against local Postgres.

### Notes for TASK-4
- `GetOrCreateBatch` returns `(model.QuizBatch, bool, error)`; use the bool to decide whether to start generation.
- Nullable model fields are pointers; service layer should handle `nil` `LastPracticedAt` as unattempted/weakest topic.
- `SubmitAnswer` is fully atomic; service/handler only needs question ID, selected option, and correctness.

### Blockers
- None.
