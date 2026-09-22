# TASK-2 — Repository layer

## Objective
DB access for every §6 table with correct transactional behavior on answer submission. Spec: `REQUIREMENT.md` §6 (schema).

## Prerequisites
- TASK-1 done (schema migrated, dev Postgres running, module initialized).

## Checklist

### Setup
- [x] Add `pgx` (or `sqlx` over pgx) dependency; pool with sane defaults.
- [x] Decide file layout in `internal/repository/` — one file per table or one aggregate file; keep it boring.

### Queries
- [x] `topics`: list all; create topic (used by `POST /topics` later).
- [x] `quiz_batches`: get by `batch_date`; create (status pending → success/failed); latest successful batch fetch.
- [x] `questions`: bulk insert for a batch; fetch questions by batch (with options JSON intact); omit answer fields in the "serve to user" query path.
- [x] `user_answers` + `user_topic_stats`: single transaction — insert answer AND update stats (accuracy, counts, `last_practiced_at`) atomically; rollback on any failure.
- [x] `generation_logs`: insert entry (status, error, tokens_used); query recent entries (for debugging/fallback checks).

### Robustness
- [x] Wrap errors with context (`failed to insert questions for batch %s`) — no naked `err` returns out of the layer.
- [x] `GetOrCreateBatch(date)` or equivalent is concurrency-safe: relies on unique `batch_date` constraint, INSERT ... ON CONFLICT / retry on conflict rather than check-then-insert.
- [x] Keep `user_id` column handling isolated in this layer (single-user MVP per §6 note; multi-user migration later should only touch here).

### Manual verification (real dev Postgres)
- [x] Seed topics → list returns 3 rows.
- [x] Create batch → insert 5 questions → fetch → all present, fields intact.
- [x] Submit answer in transaction → `user_answers` row + `user_topic_stats` updated consistently (recompute by hand, compare).
- [x] Force a mid-transaction failure → nothing persisted.

## Done when
- [x] All CRUD paths above exercised against the dev compose Postgres.
- [x] Answer submission is provably atomic (`user_answers` + `user_topic_stats` move together).
- [x] Interfaces/types are narrow enough for TASK-7 to mock without heavy scaffolding.
