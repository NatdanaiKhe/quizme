# TASK-7 — Notes

## M1 — Baseline + read-only migration verification

### Gates (current HEAD, before changes)
- `gofmt -l .`: clean
- `go build ./...`: ok
- `go vet ./...`: ok
- `go test ./...`: ok (DB-gated tests skip without `TEST_DATABASE_URL`)

### Migration state
Read-only checks against the running Postgres container (`quizme-db-1`):

| Check | Result |
|---|---|
| `schema_migrations.version` | 7 |
| `schema_migrations.dirty` | false |
| §6 tables present | generation_logs, questions, quiz_batches, topics, user_answers, user_topic_stats |
| `quiz_batches.batch_date` UNIQUE | ✓ |
| `user_topic_stats` PK on `topic_id` | ✓ |
| FKs | questions.batch_id, questions.topic_id, generation_logs.batch_id |
| Seed topics | Backend, Frontend, Infrastructure |

**Note:** `.env`'s `DATABASE_URL` is `postgres://postgres:postgres@localhost:5432/quizme?sslmode=disable`, but the running Docker Compose DB was initialized with user/password `quizme`/`quizme`. Verification used the actual container credentials; the `.env` credential mismatch should be aligned with `.env.example` before any real generate run.

Zero writes or destructive commands were executed against the database.

## M2 — Coverage gaps closed

### Added tests

| File | New coverage |
|---|---|
| `internal/service/selection_test.go` | Distribution band over 200 seeds; never-all-one-topic; 8 topics with n=5; equal-accuracy tie-break |
| `internal/ai/validate_test.go` | `correct_option: ""`; missing `topic` key; `q[i]` naming assertion in `ValidationError.Reasons` |
| `internal/service/generate_test.go` | AI error→then-success retry; `toQuestions` topic-name normalization; pending-with-questions finalize; failed→pending reset+retry |
| `internal/repository/repo_test.go` | Hand-computed 3-answer sequence (correct/wrong/correct → 2/3 → 66.67) |
| `internal/handler/handler_test.go` | Empty/unset `INTERNAL_TOKEN` regression cases |

### Code fixes
- `internal/handler/internal.go`: reject empty `wantToken` so an unset `INTERNAL_TOKEN` never accepts any request; trim supplied bearer token before constant-time compare.
- `internal/service/generate.go`: normalize AI topic names (`strings.ToLower` + `TrimSpace`) before mapping to topic IDs in `toQuestions`, matching the validator's case-insensitive topic check.

### Verification
- `go test ./...` green (DB-gated tests skip cleanly).
- `go test -race ./internal/service/ ./internal/ai/` green.

## M3 — Security & reliability review

| Item | Status | Note |
|---|---|---|
| Empty `INTERNAL_TOKEN` bypass fixed + tested | ✓ | `InternalAuth("")` returns 401 for all cases |
| Answer leak: `/quiz/today` only returns `publicQuestion` | ✓ | Handler test asserts no `correct_option`/`explanation` |
| Validator fails closed | ✓ | M2 tests cover malformed JSON, wrong count, bad ids, empty fields, unknown topic |
| All SQL parameterized | ✓ | `InsertQuestions` builds placeholders only; user input never concatenated |
| `.env` gitignored | ✓ | `git check-ignore .env` confirms |
| AI key only in `Authorization` header | ✓ | `internal/ai/client.go` |
| Webhook logs host only | ✓ | `hostOnly` in `internal/service/notify.go` |
| Generate error log does not print secrets | ✓ | Logs `err.Error()` from AI/repo, not request bodies |
| Fire-and-forget notifier | ✓ | Goroutine + panic recovery, `context.WithoutCancel`, 5s timeout |
| Idempotent generation | ✓ | `batch_date` UNIQUE; success rerun is no-op; failed resets to pending |
| Pending-with-questions finalize | ✓ | `TestGenerateIntegrationPendingWithQuestionsFinalizes` |
| Log row on every terminal state | ✓ | `failBatch`/`succeedBatch` both call `log` |
| Fallback serves latest success | ✓ | `TestGetTodayQuizFallback`, `TestGetTodayQuizErrNoQuiz` |
| Worst-case generate latency | ✓ | 3 × 60s = 180s; note for TASK-10 to keep Actions timeout ≥ default 360s |

### Backlog for later tasks
- **TASK-9**: `http.Server` lacks explicit `ReadHeaderTimeout`/`WriteTimeout`.
- **TASK-9/TASK-10**: `POST /topics` is unauthenticated and rate-unbounded (acceptable for single-user MVP).
- **TASK-11**: Repeat answers to the same question re-inflate stats (acceptable for MVP; revisit with multi-user auth).
- **TASK-11**: Failed webhook payload omits an `error` key when `GenerateResult.Error` is nil; for human receivers it may be useful to include `"error": null`.

## M4 — Docs & spot check

### README changes
- Added **Testing** section with `go test ./...`, `-race` command, and `TEST_DATABASE_URL` / `TRUNCATE` warning.
- Added missing env vars (`AI_BASE_URL`, `AI_MODEL`, `PORT`, `CORS_ORIGIN`) to the `.env` block.
- Fixed typos: "chioce" → "choice", "quizeme" → "quizme".

### Failing-branch spot check
Command: temporarily changed `validate.go` to accept option id `"e"`, then ran `go test ./internal/ai/`.

Observed failure:
```
--- FAIL: TestParseAndValidate
    --- FAIL: TestParseAndValidate/missing_option_id_e
        validate_test.go:220: expected *ValidationError, got <nil>: <nil>
FAIL
```

Restored the original strict check; `go test ./internal/ai/` green.

## Blockers / next-task notes
- DB-gated integration tests were **not** run against the real database (they `TRUNCATE` tables). They compile and skip cleanly; a disposable `TEST_DATABASE_URL` is required to execute them.
- `.env` `DATABASE_URL` credentials (`postgres/postgres`) do not match the running Docker Compose DB (`quizme/quizme`). Align `.env` with `.env.example` before running live generation.
- No changes were made to handlers or API contract; TASK-8 can proceed against the current §7 endpoints.

