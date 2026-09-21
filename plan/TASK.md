# Quizme — Task Index

Single-user MVP: Go backend + React frontend monorepo, AI-generated daily quizzes, LINE notifications.
Spec sections (§) refer to `REQUIREMENT.md` at repo root. Execute tasks in dependency order.
Each `TASK-N/TODO.md` contains the actionable checklist for that task.

| # | Task | Depends on | Spec refs | Status |
|---|------|-----------|-----------|--------|
| 1 | Project skeleton & Phase 0 setup | — (start) | §6, §11 | done |
| 2 | Repository layer | TASK-1 | §6 | todo |
| 3 | AI client + validator | TASK-1 | §9, FR-1, FR-2 | todo |
| 4 | Service layer (generation + adaptive selection) | TASK-2, TASK-3 | §8, §10, FR-6, FR-8, FR-9 | todo |
| 5 | Webhook notifier | TASK-4 | §10, FR-10 | todo |
| 6 | API layer (Gin) | TASK-4, TASK-5 | §7, FR-3, FR-4, FR-7 | todo |
| 7 | Unit tests | TASK-3, TASK-4 (parallel-friendly) | FR-2, FR-6, FR-8 | todo |
| 8 | Frontend (React/Vite) | TASK-6 | §7 (API contract) | todo |
| 9 | Deployment | TASK-1–6, TASK-8 | NFR-1 | todo |
| 10 | Scheduler (GitHub Actions) | TASK-6, TASK-9 | NFR-2 | todo |
| 11 | E2E validation & tuning | all | FR-2, NFR-5 | todo |

---

## TASK-1 — Project skeleton & Phase 0 setup
**Covers:** Repo layout per §11 (`cmd/server`, `internal/{handler,service,repository,ai,model}`, `migrations/`), `go.mod` + bootable health endpoint, env config (`DATABASE_URL`, `AI_API_KEY`, `INTERNAL_TOKEN`, `WEBHOOK_URL`), SQL migrations for §6 schema, seed topics (Frontend/Backend/Infrastructure), docker-compose dev environment (Postgres + backend).
**Dependencies:** none.
**Key files/components:** `cmd/server/main.go`, `internal/model/`, `migrations/*.sql`, `docker-compose.yml`, `.env.example`, `.gitignore`.
**Success criteria:** `docker compose up` boots Postgres + backend, health check responds; migrations run clean up/down; schema matches §6 exactly; seed topics present.
**Risks:** none significant. External prerequisites (LINE OA, AI API key, homelab Postgres/tunnel URL) are user-owned — start early here.

## TASK-2 — Repository layer
**Covers:** DB access (pgx or sqlx) for all §6 tables: topics, quiz_batches, questions, user_answers, user_topic_stats, generation_logs. Key queries: get/create today's batch, insert questions, transactional answer + stats update, fetch stats, latest successful batch.
**Dependencies:** TASK-1.
**Key files/components:** `internal/repository/` (one file per table or one aggregate), transaction helper for answer+stats.
**Success criteria:** All CRUD paths exercised against real dev Postgres; answer submission atomically updates `user_answers` + `user_topic_stats` (accuracy, counts, last_practiced_at).
**Risks:** transaction correctness on stats update — covered by tests in TASK-7. Keep `user_id`-ready (single-user MVP) by isolating answer/stats queries here.

## TASK-3 — AI client + validator
**Covers:** `internal/ai` HTTP client using prompt template §9 (`{n}` questions, `{topics}` list, token limit per NFR-4); JSON parsing tolerant of markdown fences; validator per FR-2 (array of n objects, required fields, 4 options with unique ids a–d, `correct_option` ∈ option ids). Invalid → error (retries live in TASK-4).
**Dependencies:** TASK-1.
**Key files/components:** `internal/ai/client.go`, `internal/ai/prompt.go`, `internal/ai/validate.go`.
**Success criteria:** Mocked AI responses (valid / malformed JSON / wrong `correct_option`) pass/fail as expected; one real API call produces 5 validated questions.
**Risks:** AI format drift — validator fails closed; prompt tuning deferred to TASK-11.

## TASK-4 — Service layer: generation & adaptive topic selection
**Covers:** Adaptive selection per §8 (unattempted = weakest; ~70% of 5 questions from bottom topics, ~30% random from rest; never all-same-topic). Generation flow per §10: create pending batch → select topics → AI call → validate → insert questions → batch success. Retry policy: AI error retry ≤2, validation fail retry once, exhausted → batch failed + log. `generation_logs` entry every run (status, error, tokens_used — FR-9). Fallback: `/quiz/today` serves latest successful batch when today's is failed/pending. Idempotency via unique `batch_date`.
**Dependencies:** TASK-2, TASK-3.
**Key files/components:** `internal/service/generate.go`, `internal/service/selection.go`, `internal/service/quiz.go`.
**Success criteria:** Unit tests: 70/30 mix across runs, unattempted topics prioritized; failure injection (AI always fails) → batch failed, log written, fallback still serves (no crash, NFR-5).
**Risks:** concurrent generate race — resolved by `batch_date` unique constraint; randomness testing needs seed or distribution check.

## TASK-5 — Webhook notifier
**Covers:** Generic HTTP POST to `WEBHOOK_URL` after batch reaches terminal state (success or failed-after-retries), payload per §10. Fire-and-forget with timeout; notification failure logged, never blocks batch flow. LINE delivery is the receiving side's (n8n/relay) job — document/verify LINE Messaging API path (OA + Channel Access Token; LINE Notify is deprecated).
**Dependencies:** TASK-4 (needs batch terminal states).
**Key files/components:** `internal/service/notify.go` (or `internal/notifier/`).
**Success criteria:** Test endpoint (webhook.site or local receiver) receives correct payload on both success and failure paths.
**Risks:** LINE setup is external/user-owned — keep sender generic to decouple.

## TASK-6 — API layer (Gin)
**Covers:** Gin handlers for §7: `GET /quiz/today` (5 questions, **no `correct_option` in response**), `POST /quiz/answer` (immediate correct/incorrect + explanation, updates stats), `GET /stats`, `GET /topics`, `POST /topics`. `POST /internal/generate` guarded by bearer token from `INTERNAL_TOKEN`. Request validation, sane errors (400/401/404/500), service wiring.
**Dependencies:** TASK-4, TASK-5.
**Key files/components:** `internal/handler/quiz.go`, `internal/handler/topics.go`, `internal/handler/internal.go`, router setup in `cmd/server`.
**Success criteria:** Every endpoint manually curl-tested, response shapes match §7; `/quiz/today` provably leaks no answers; `/internal/generate` rejects missing/wrong token.
**Risks:** answer-leak cheating — explicit test asserting `correct_option` absent.

## TASK-7 — Unit tests
**Covers:** Topic selection (70/30, unattempted priority), validator (all malformed cases), generation retry/fallback logic, stats update math. Table-driven Go tests; pure logic without DB, minimal mocks/interfaces for repo.
**Dependencies:** TASK-3, TASK-4 code (can be written alongside).
**Key files/components:** `internal/service/*_test.go`, `internal/ai/*_test.go`.
**Success criteria:** `go test ./...` green; FR-2, FR-6, FR-8 branches covered.
**Risks:** none significant — keep mocks minimal.

## TASK-8 — Frontend (React/Vite)
**Covers:** React + Vite app talking REST. Pages: today's quiz (fetch `/quiz/today`, render, submit `/quiz/answer`, immediate correct/incorrect + explanation), stats (per-topic accuracy, simple chart), topic list. Error states: no quiz today, generation failed (fallback quiz shown), network errors.
**Dependencies:** TASK-6.
**Key files/components:** `frontend/` (Vite root), quiz page, stats page, API client module.
**Success criteria:** Full user loop works: see quiz → answer → feedback → stats reflect answers; error states render gracefully, no blank screens.
**Risks:** CORS across origins — configure backend CORS or serve both behind same reverse proxy (TASK-9).

## TASK-9 — Deployment
**Covers:** Multi-stage Dockerfile (Go backend), frontend Dockerfile (build → nginx serve), docker-compose (backend + frontend + Postgres + reverse proxy Caddy/Nginx), Cloudflare Tunnel for outside reachability, `.env` with all secrets on homelab, never committed.
**Dependencies:** TASK-1–6 (working backend); frontend Dockerfile needs TASK-8.
**Key files/components:** `Dockerfile`, `frontend/Dockerfile`, `docker-compose.prod.yml` (or equivalent), reverse-proxy config, `.env` handling + `.gitignore`.
**Success criteria:** Stack reachable via tunnel URL, all endpoints respond through proxy; no secrets in git history.
**Risks:** tunnel/proxy config on homelab — verify before scheduling TASK-10.

## TASK-10 — Scheduler (GitHub Actions)
**Covers:** `.github/workflows/daily-generate.yml`, cron `0 23 * * *` (06:00 ICT), POST `/internal/generate` with auth header; URL + token in GitHub Actions Secrets; `workflow_dispatch` for manual trigger tested before enabling schedule.
**Dependencies:** TASK-6 (endpoint), TASK-9 (public URL).
**Key files/components:** `.github/workflows/daily-generate.yml`, repo Secrets.
**Success criteria:** Manual dispatch generates today's batch end-to-end; scheduled run fires on time (Actions log + `quiz_batches` row).
**Risks:** endpoint unreachable at cron time — confirm tunnel stability during TASK-11 soak.

## TASK-11 — E2E validation & tuning
**Covers:** 3–5 day soak: generation on time, LINE notifications delivered, no spurious fallback. Verify adaptive behavior in practice (weak topics recur, stats shift over days). Tune prompt template if quality/level off (FR-2 failures are the signal). Record backlog: multi-user, difficulty tuning (out of MVP scope).
**Dependencies:** all tasks.
**Key files/components:** production stack, `generation_logs`, prompt template (`internal/ai/prompt.go`), backlog notes.
**Success criteria:** 3–5 clean daily runs, no missed notification without a logged cause; stats show weak topics surfacing more often than random; backlog documented.
**Risks:** silent AI degradation — watch `generation_logs` failure rate; webhook/LINE rate limits.
