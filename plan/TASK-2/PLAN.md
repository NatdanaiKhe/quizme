# TASK-2 — Repository Layer: Execution Plan

Status: scoped, not implemented. Executor follows `plan/TASK-2/TODO.md` + this plan.
Spec refs: `REQUIREMENT.md` §6 (schema), §8 (stats feed selection), §10 (batch lifecycle).
Depends on: TASK-1 (done — schema migrated, `pgxpool`-ready module, dev compose Postgres).

## ⚠ Pre-existing defect to fix first (from prior quality review)

`internal/model/models.go` types nullable §6 columns as plain values:

| Struct field | Schema | Problem | Fix |
|---|---|---|---|
| `Question.Explanation` | `explanation TEXT` (nullable) | pgx scan errors on NULL; nil vs "" indistinguishable | `*string` |
| `UserTopicStats.LastPracticedAt` | `TIMESTAMPTZ` (nullable, no default) | scan errors on NULL — **every fresh topic row hits this** | `*time.Time` |
| `GenerationLog.ErrorMessage` | `error_message TEXT` (nullable) | same | `*string` |
| `GenerationLog.TokensUsed` | `tokens_used INT` (nullable, no default) | same | `*int` |
| `GenerationLog.BatchID` | `batch_id INT` (nullable FK) | same | `*int` |

`LastPracticedAt` is the blocking one: TASK-4's selection reads `user_topic_stats` for
unattempted topics, which are exactly the rows with NULL there. Fix models before any
query code — it's a TASK-2 file touch (models are the repo's row types), not a schema change.

## Decisions (fixed, don't relitigate)

- **Driver:** `pgx/v5` + `pgxpool` (no sqlx layer — struct tags already exist but pgx `Scan`/`CollectRows` suffices; stdlib-adjacent, one dependency as TODO allows).
- **File layout:** ONE file `internal/repository/repo.go` + one `repo_test.go`. Six tables, ~15 queries — per-table files would be boilerplate. `doc.go` gets deleted (redundant).
- **Constructor:** `repository.New(pool)` returning a single `*Repo` struct with exported methods. No interface definition in-repo — TASK-7 mocks the method set it needs (Go interfaces are satisfied implicitly).
- **user_id readiness:** zero `user_id` anywhere; answer/stats mutation confined to one `SubmitAnswer` method so the multi-user migration stays in this layer.

## Ordered steps

### Step 1 — Fix model nullability
- [ ] Apply the pointer-type fixes in `internal/model/models.go` (table above).
- [ ] `go build ./...` green.
**Done when:** models compile; no behavior change elsewhere (no other code imports these yet).

### Step 2 — Pool + Repo skeleton
- [ ] Add `pgx/v5` to `go.mod`.
- [ ] `New(ctx, databaseURL) (*Repo, error)` — `pgxpool.New`, sane defaults (pool max ~5; single user, NFR-3 <500ms is trivial at this scale).
- [ ] `Close()`.
**Done when:** `New` connects to the dev compose Postgres with `DATABASE_URL` from `.env`.

### Step 3 — Queries (in this order)
- [ ] `ListTopics()` / `CreateTopic(name, weight int)` — FR-7.
- [ ] `GetOrCreateBatch(batchDate)` — `INSERT ... ON CONFLICT (batch_date) DO NOTHING` + `SELECT` on conflict (never check-then-insert; unique constraint is the lock). Returns `(batch, created bool)`.
- [ ] `UpdateBatchStatus(id, status)`.
- [ ] `LatestSuccessfulBatch()` — §10 fallback source.
- [ ] `InsertQuestions(batchID, []model.Question)` — single multi-row INSERT; `options` marshaled to JSONB.
- [ ] `GetQuestionsByBatch(batchID)` — full rows; answer-field stripping happens in the service/handler layer, not here (repo serves rows, §6 keeps `correct_option` server-side anyway).
- [ ] `SubmitAnswer(questionID, selected, isCorrect)` — **one transaction**: insert `user_answers`, upsert `user_topic_stats` (`INSERT ... ON CONFLICT (topic_id) DO UPDATE` bumping `correct_count`/`wrong_count`/`last_practiced_at`/`accuracy_rate` recomputed from counts), commit. Any error → rollback. Needs the question's `topic_id` — either join in the upsert or one prior SELECT inside the same tx.
- [ ] `GetStats()` — all rows of `user_topic_stats` (joined with topic name for `GET /stats` later).
- [ ] `InsertGenerationLog(log)` / `RecentGenerationLogs(n)`.
- [ ] All errors wrapped: `fmt.Errorf("insert questions for batch %d: %w", ...)`. No naked returns.
**Done when:** all methods compile and are covered by Step 4.

### Step 4 — Verification (real dev Postgres, one test file)
- [ ] `repo_test.go`, table-driven where it matters. **Skips when `TEST_DATABASE_URL` is unset** (keeps `go test ./...` green everywhere; set it in dev to run for real).
- [ ] Test: seed topics → `ListTopics` = 3.
- [ ] Test: batch round-trip — create, insert 5 questions, fetch back, options JSON intact.
- [ ] Test: `SubmitAnswer` atomicity — hand-recompute expected counts/accuracy/`last_practiced_at`, compare. Cover both first-answer (INSERT path) and repeat-answer (ON CONFLICT path).
- [ ] Test: force mid-tx failure (e.g. bad question FK) → zero rows in `user_answers`, stats unchanged.
- [ ] Test: `GetOrCreateBatch` idempotency — two calls, one row.
- [ ] Run: `TEST_DATABASE_URL=$(grep DATABASE_URL .env | cut -d= -f2-) go test ./internal/repository/ -v`.

## Success criteria (whole task)

- [ ] Every §6 table has its needed read/write path exercised against the dev compose Postgres.
- [ ] `SubmitAnswer` is provably atomic — answer row + stats move together, failure rolls back both.
- [ ] `GetOrCreateBatch` relies on the unique constraint (no duplicate-batch race for TASK-4).
- [ ] Nullable columns scan without error (fresh topic row with NULL `last_practiced_at` round-trips).
- [ ] `go test ./...` and `go build ./...` green; `gofmt` clean.
- [ ] Method set is narrow enough for TASK-7 to mock with an anonymous struct.

## Verification checklist

1. `go build ./... && go test ./...`
2. `docker compose up -d postgres` (if not running) → run Step 4 test command with `TEST_DATABASE_URL` set.
3. Read actual test output — no "should work" passes.

## Dependencies / risks

- **Blocked by:** nothing (TASK-1 done).
- **Blocks:** TASK-4 (service layer calls all of these), TASK-7 (mocks the method set).
- **Risk:** stats-transaction correctness — mitigated by the hand-recompute test; deeper coverage lands in TASK-7 per plan.
- **Risk:** `accuracy_rate` NUMERIC(5,2) vs float64 rounding — recompute as `round(correct::numeric / attempts * 100, 2)` in SQL so DB and app never disagree.

## Commit step (per LOOP.md §7, after verification passes)

1. Tick all boxes in `plan/TASK-2/TODO.md`.
2. Set TASK-2 `done` in `plan/TASK.md`.
3. Append run summary to `plan/PROGRESS.md` (date, what changed, note for TASK-4: `GetOrCreateBatch` returns `(batch, created)`; nullable model fields are pointers).
4. Commit everything as: `task 2: repository layer with transactional answer+stats updates`

One commit, one task per run, then stop.
