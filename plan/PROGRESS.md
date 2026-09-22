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

### Notes for TASK-3
- `GetOrCreateBatch` returns `(model.QuizBatch, bool, error)`; use the bool to decide whether to start generation.
- Nullable model fields are pointers; service layer should handle `nil` `LastPracticedAt` as unattempted/weakest topic.
- `SubmitAnswer` is fully atomic; service/handler only needs question ID, selected option, and correctness.

### Blockers
- None.

## 2026-09-22 — TASK-3 done

Implemented AI client + strict fail-closed validator per `plan/TASK-3/PLAN.md`, `REQUIREMENT.md` §9/FR-1/FR-2/NFR-4.

### Changed
- `internal/config/config.go`: added `AIBaseURL` and `AIModel` env vars (`AI_BASE_URL`, `AI_MODEL`).
- `.env.example`: added placeholder `AI_BASE_URL` and `AI_MODEL`.
- `internal/ai/prompt.go`: §9 prompt template as const; `buildPrompt(n, topics)` via `strings.ReplaceAll`.
- `internal/ai/validate.go`: `GeneratedQuestion` (reuses `model.Option`), `Parse(raw, n)` with markdown-fence stripping, `Validate(qs, n, topics)` fail-closed with aggregated `*ValidationError`.
- `internal/ai/client.go`: OpenAI-compatible `POST /chat/completions`; `Client.Generate(ctx, n, topics)` pipeline; `*AIError` for transport/non-200/malformed envelope; 60s timeout, `max_tokens=2000`; no API key logging.
- `internal/ai/*_test.go`: table-driven validator tests, httptest client tests, prompt test, env-guarded `TestRealSmoke`.
- Deleted `internal/ai/doc.go`.
- Updated `plan/TASK-3/TODO.md`, `plan/TASK.md`, `plan/OVERVIEW.md`; created `plan/TASK-3/NOTES.md`.

### Verified
- `gofmt -w .` clean.
- `go build ./...` green.
- `go vet ./...` green.
- `go test ./...` green.
- `go test ./internal/ai/ -v` passes all 27 sub-tests.
- Grep confirmed no API key value outside `Authorization: Bearer ...` construction; no `correct_option` HTTP serialization path in `internal/ai`.

### Notes for TASK-4
- Consume `ai.Client.Generate(ctx, n, topics)` → `([]GeneratedQuestion, tokensUsed, err)`.
- Use `errors.As(err, &ai.AIError{})` for AI failures (retry ≤2) and `errors.As(err, &ai.ValidationError{})` for validation failures (retry once).
- `Validate` checks topic membership against requested topic names (case-insensitive) so AI-invented topics fail before DB insert.

### Blockers
- Real API smoke test skipped: `AI_BASE_URL`/`AI_API_KEY`/`AI_MODEL` are not set in `.env`. To unblock, populate `.env` and run `AI_SMOKE_TEST=true go test ./internal/ai/ -run TestRealSmoke -v`, then record `tokens_used` here.

## 2026-09-22 — TASK-4 done

Implemented service layer per `plan/TASK-4/TODO.md`, `PLAN.md`, `REQUIREMENT.md` §8/§10, FR-6, FR-8, FR-9, NFR-5.

### Changed
- Added `internal/repository/repo.go::GetBatchByDate` read-only batch lookup (GAP-1) and `Pool()` test accessor.
- Created `internal/service/selection.go`: deterministic adaptive selection, unattempted topics ranked weakest, 70/30 split (`n*7/10` = 3 weak for n=5), injected `*rand.Rand`.
- Created `internal/service/generate.go`: `Service.Generate` with idempotent `GetOrCreateBatch`, typed retry budgets (3 AI-error attempts, 1 validation retry), topic-name→ID mapping, terminal status updates, one `generation_logs` row per run, nil-safe fire-and-forget notifier seam.
- Created `internal/service/quiz.go`: `GetTodayQuiz` with fallback to latest successful batch and `ErrNoQuiz` sentinel; pending batches never block reads.
- Added pure tests (`selection_test.go`, retry-loop tests in `generate_test.go`) and DB-gated integration tests (`generate_test.go`, `quiz_test.go`) covering success, idempotency, AI failure, validation retry/failure, fallback, and `ErrNoQuiz`.
- Deleted placeholder `internal/service/doc.go`.
- Updated `plan/TASK-4/TODO.md`, `plan/TASK.md`, `plan/OVERVIEW.md`.

### Verified
- `gofmt -w .` clean.
- `go build ./...` green.
- `go vet ./...` green.
- `go test ./...` green (integration tests skip without `TEST_DATABASE_URL`).
- Live Postgres integration: `go test -p 1 ./...` with `TEST_DATABASE_URL=postgres://quizme:quizme@localhost:5432/quizme?sslmode=disable` passes all service and repository tests.

### Notes for TASK-5
- Notifier interface is defined in `internal/service/generate.go`; implement `Notifier.Notify(ctx, GenerateResult)` and inject it into `Service.Notifier`.

### Blockers
- Rootless Docker port publishing/DNS is flaky in this environment; integration tests were run via `--network container:quizme-db-1`. This is an environment issue, not a code issue.

## 2026-09-22 — TASK-5 done

Implemented webhook notifier per `plan/TASK-5/PLAN.md`, `REQUIREMENT.md` §10, FR-10.

### Changed
- Created `internal/service/notify.go`: stdlib-only `WebhookNotifier` implementing the existing `Notifier` seam.
  - Exact success/failure JSON payloads (`batch_date`, `status`, `question_count`+`topics` or `error`).
  - POST JSON with `Content-Type: application/json`.
  - 5s default timeout; caller-configurable.
  - `context.WithoutCancel(ctx)` so request-context cancellation does not abort in-flight notifications.
  - Empty `WEBHOOK_URL` is a no-op.
  - Errors logged with URL host only (no query-string secrets); error values returned but discarded by the service's fire-and-forget goroutine.
- Created `internal/service/notify_test.go`: comprehensive `httptest` coverage including success/failure payloads, context detach, timeout, non-2xx, unreachable host, empty URL, default timeout, and `Service` seam integration (DB-gated).
- Updated `.env.example` to clarify `WEBHOOK_URL` feeds an n8n/LINE relay.
- Created `plan/TASK-5/NOTES.md` with LINE Messaging API receiver-side setup and verification notes.
- Marked TASK-5 done in `plan/TASK.md`, `plan/OVERVIEW.md`, and `plan/TASK-5/TODO.md`.

### Verified
- `gofmt -w .` clean.
- `go build ./...` green.
- `go vet ./...` green.
- `go test ./...` green.
- `go test ./internal/service/ -run Notif -v` passes all 9 sub-tests (seam integration skips without `TEST_DATABASE_URL`).
- Confirmed log lines expose only host and status, never `WEBHOOK_URL`, query params, API keys, or question content.

### Notes for TASK-6
- When wiring `Service` in `cmd/server/main.go`, construct the notifier with:
  ```go
  notifier := service.NewWebhookNotifier(cfg.WebhookURL, 5*time.Second)
  svc := service.New(repo, aiClient, notifier)
  ```
- If `cfg.WebhookURL` is empty, the notifier is a no-op and generation works without notifications.

### Blockers
- Full manual end-to-end verification (live AI + Postgres + webhook receiver) is blocked on external/user-owned prerequisites:
  - `AI_API_KEY` is not set in `.env`.
  - `WEBHOOK_URL` is not set in `.env`.
  - `TEST_DATABASE_URL` is not set, so the DB-backed seam integration test skips.
- These do not block the implementation; automated httptest coverage exercises the code paths.

## 2026-09-22 — TASK-6 done

Implemented API layer (Gin) per `plan/TASK-6/PLAN.md`, `TODO.md`, `REQUIREMENT.md` §7, FR-3, FR-4, FR-7.

### Changed
- Added `gin` and `pgerrcode` dependencies.
- `internal/repository/repo.go`: added `GetQuestion(ctx, id)` for answer submission.
- `internal/service/generate.go`: extended `Repository` interface with `GetQuestion`, `CreateTopic`, `SubmitAnswer`.
- `internal/service/quiz.go`: added `Answer(ctx, questionID, option)` (fetches question, compares option, records answer transactionally), `ErrNotFound` sentinel, and thin `ListTopics`/`CreateTopic`/`GetStats` delegators.
- `internal/service/answer_test.go`: table-driven service tests for correct/wrong/not-found answer paths.
- `internal/handler/quiz.go`: `GET /quiz/today` with answer-safe `publicQuestion` response struct and topic-name lookup; `POST /quiz/answer` with option validation; `GET /stats`.
- `internal/handler/topics.go`: `GET /topics`, `POST /topics` (duplicate → 409, empty name → 400).
- `internal/handler/internal.go`: constant-time bearer token middleware and `POST /internal/generate` handler.
- `internal/handler/handler_test.go`: comprehensive httptest coverage for all six endpoints plus auth/CORS scenarios.
- `cmd/server/main.go`: Gin router, stdlib CORS middleware, health endpoint, route registration, service wiring.
- Updated `plan/TASK-6/TODO.md`, `plan/TASK.md`, `plan/OVERVIEW.md`; created `plan/TASK-6/NOTES.md`.

### Verified
- `gofmt -w .` clean.
- `go build ./...` green.
- `go vet ./...` green.
- `go test ./...` green (handler + service unit tests).
- `go test -p 1 ./...` with `TEST_DATABASE_URL` from `.env` green (repository + service integration tests).
- Manual curl verification of all §7 endpoints: shapes/status codes match spec; `/quiz/today` leaks neither `correct_option` nor `explanation`; `/internal/generate` rejects missing/wrong tokens; `/quiz/answer` updates stats.

### Notes for TASK-7
- Selection, validator, and generation retry logic already have tests; TASK-7 may add more coverage or tidy existing tests.

### Blockers
- Full live-AI end-to-end of `/internal/generate` is blocked on `AI_API_KEY` (user-owned). Idempotency/auth paths are verified.

## 2026-09-22 — TASK-7 done

Implemented TASK-7 backend hardening and test coverage per `plan/TASK-7/PLAN.md`, `TODO.md`, `REQUIREMENT.md` FR-2/FR-6/FR-8.

### Changed
- `internal/service/selection_test.go`: added distribution-band, never-all-one-topic, more-topics-than-n, and equal-accuracy tie-break tests.
- `internal/ai/validate_test.go`: added `correct_option: ""`, missing `topic` key, and `q[i]` naming assertions.
- `internal/service/generate_test.go`: added AI-error-then-success retry test, `toQuestions` topic-name normalization test, pending-with-questions finalize test, and failed-batch reset+retry test.
- `internal/repository/repo_test.go`: added hand-computed 3-answer sequence test (correct/wrong/correct → 2/3 → 66.67).
- `internal/handler/handler_test.go`: added empty/unset `INTERNAL_TOKEN` regression test.
- `internal/handler/internal.go`: reject empty `wantToken` and trim supplied bearer token before constant-time compare.
- `internal/service/generate.go`: normalize AI topic names before mapping to DB topic IDs.
- `README.md`: added Testing section, missing env vars, fixed typos.
- `plan/TASK-7/TODO.md`: all checkboxes ticked.
- `plan/TASK.md` and `plan/OVERVIEW.md`: TASK-7 marked done.
- Created `plan/TASK-7/NOTES.md` with baseline gates, read-only migration verification, security/reliability review, backlog, and failing-branch spot-check evidence.

### Verified
- `gofmt -w .` clean.
- `go build ./...` green.
- `go vet ./...` green.
- `go test ./...` green (DB-gated tests skip cleanly without `TEST_DATABASE_URL`).
- `go test -race ./internal/service/ ./internal/ai/` green.
- Read-only migration verification against the running Postgres container: `schema_migrations` version 7, dirty false, all §6 tables present, expected constraints and FKs, seed topics present.
- Failing-branch spot check: temporarily accepted option id `"e"`; `go test ./internal/ai/` failed as expected; restored strict check.

### Warnings
- **Never point `TEST_DATABASE_URL` at the user's real database.** DB-gated tests `TRUNCATE` all tables. They were left to skip in this task.
- `.env` `DATABASE_URL` (`postgres/postgres`) does not match the running Docker Compose DB (`quizme/quizme`). Align `.env` with `.env.example` before live generation.

### Notes for downstream tasks
- **TASK-8**: API contract unchanged; handlers are stable for frontend work.
- **TASK-9**: Backlog item — add explicit `http.Server` `ReadHeaderTimeout`/`WriteTimeout`.
- **TASK-10**: Worst-case `/internal/generate` latency is 3 × 60s AI timeout = ~180s; do not set a shorter Actions job timeout than the default 360s.
