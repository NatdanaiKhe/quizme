# TASK-7 — Plan: Final backend hardening & integration

Scopes `plan/TASK-7/TODO.md` (unit tests) plus the TASK-7 completion pass: coverage gaps,
security/reliability review, docs/config, migration verification against the user-approved
existing `DATABASE_URL`, and completion criteria. **No infrastructure changes. No new deps.**

## Current state (verified, not assumed)

- TASK-1..6 done. Test files already exist: `internal/ai/{validate,client,prompt}_test.go`,
  `internal/service/{selection,generate,quiz,answer,notify,service,repomock}_test.go`,
  `internal/repository/repo_test.go`, `internal/handler/handler_test.go`.
- DB-gated tests (`repo_test.go`, `generate_test.go`, `quiz_test.go`, `notify_test.go` seam)
  skip without `TEST_DATABASE_URL` and **TRUNCATE all tables** when they run.
- `.env` has the user-approved `DATABASE_URL` → `postgres://postgres:postgres@localhost:5432/quizme`
  (this is the dev/homelab DB — treat as real data).
- Gates at TASK-6 close: `gofmt`/`go build`/`go vet`/`go test ./...` reported green.

### Real gaps vs TODO.md checklist

| TODO.md item | Status today | Action |
|---|---|---|
| 70/30 distribution across many seeded runs | Missing — no distribution-band test | Add |
| Never all 5 from one topic | Only trivial single-topic case | Add explicit multi-topic assertion |
| More topics than n | Missing (only exactly-n in `TestSelectionSplitBounds`) | Add |
| All topics equal accuracy (tie-break) | Missing | Add |
| `correct_option` = `""` fails | Missing (only `"e"`, `"B"` covered) | Add table case |
| Error names the failing item (`q[i]`) | Missing assertion | Add |
| Transient AI error → success → batch success | Missing (validation-retry variant exists) | Add |
| Pending batch (with/without questions) paths | Pending-with-questions finalize path (generate.go:90–99) untested | Add |
| Stats: hand-computed multi-answer sequence | Only 1-correct-then-1-wrong (100%→50%) | Add 2/3-rounding + longer sequence |
| Failing-branch spot check | Not done | Do once, document |
| Everything else on the checklist | Covered | Verify by re-reading tests, tick |

## Milestones

### M1 — Baseline + migration verification (read-only, existing DATABASE_URL only)

Dependencies: none (runs first so later milestones build on a known-good DB).

1. Run gates on current HEAD: `gofmt -l .`, `go build ./...`, `go vet ./...`,
   `go test ./...` (unit tests; DB-gated ones skip by design). Record results in NOTES.md.
2. Migration verification against `DATABASE_URL` from `.env` — **read-only queries only**:
   - `schema_migrations` → version `7`, `dirty = false`.
   - All 6 §6 tables exist (`information_schema.tables`) with expected columns/constraints:
     `quiz_batches.batch_date` UNIQUE, `user_topic_stats` PK on `topic_id`, FKs on
     `questions.batch_id/topic_id`, `generation_logs.batch_id`.
   - Seed topics: exactly `Frontend`, `Backend`, `Infrastructure` present.
   - Mechanism: `psql "$DATABASE_URL" -c ...` if psql is installed; otherwise
     `docker compose exec <db-service> psql ...`. Never create/modify containers; if neither
     path works, log the blocker and stop (do not stand up anything new).
3. **If `schema_migrations` is not at version 7 or is dirty: STOP, record in NOTES.md, ask the user.**
   The only permitted write is a plain `migrate ... up` for missing additive versions, and
   only after the user confirms. No `down`, no `drop`, no reset, ever.

Success criteria:
- [ ] Gates run on HEAD, results recorded.
- [ ] Read-only migration report in `plan/TASK-7/NOTES.md`: version=7, tables ✓, seeds ✓.
- [ ] Zero writes executed against the user's database.

### M2 — Close the test-coverage gaps (pure tests first, DB-gated second)

Dependencies: M1 green.

Keep table-driven, stdlib-only, seeded randomness. Extend existing files — no new packages.

1. `internal/service/selection_test.go` (pure):
   - Distribution: over ≥200 seeded runs of 5, assert the 3 weak slots always come from the
     weakest-ranked topics (unattempted first) and the 2 remaining slots from the rest; the
     split is `n*7/10` by construction — assert the observed weak:rest share stays in the
     3/5 band. (Implementation is deterministic per seed; the test guards against regressions,
     not flaky sampling.)
   - Never-all-one-topic: with ≥2 topics, no seeded run returns 5× the same name.
   - More topics than n (e.g. 8 topics, n=5).
   - Equal-accuracy tie-break: all attempted at 50% → selection is deterministic
     (ID order) and stable across seeds.
2. `internal/ai/validate_test.go` (pure):
   - Table cases: `correct_option: ""`; missing `topic` key; per TODO "wrong correct_option
     values fail with message naming the item" — assert `ValidationError.Reasons` contains
     `q[<i>]` for the mutated question.
3. `internal/service/generate_test.go`:
   - Pure: AI error on attempt 1, success on attempt 2 → 3-call budget not exhausted,
     questions returned (transient-then-success).
   - DB-gated: pending batch with questions → `Generate` finalizes to success without an AI
     call; failed batch → status reset to pending, regeneration proceeds (reuse existing
     truncating helper; only with `TEST_DATABASE_URL`).
4. `internal/repository` stats math — extend `TestSubmitAnswer` style coverage (DB-gated):
   - Sequence with hand-computed values: e.g. correct, wrong, correct → 2/3 → accuracy
     `66.67` (exercises NUMERIC(5,2) rounding); init row has counts 1/0, accuracy exact,
     `last_practiced_at` set (divide-by-zero path guarded by `NULLIF`).

Success criteria:
- [ ] All four test files extended per the gap table; `go test ./...` green.
- [ ] DB-gated additions still skip cleanly with `TEST_DATABASE_URL` unset.
- [ ] No new dependencies; tests deterministic (no `time.Now()` inside pure assertions, seeds fixed).

### M3 — Security & reliability review (walkthrough + minimal fixes)

Dependencies: M2 (so fixes land against green tests).

Walk every trust boundary and failure path; write findings to `plan/TASK-7/NOTES.md`.
Fix only what is small, in-scope, and security-critical; everything else becomes backlog notes.

**Known finding — must fix (trust boundary):**
- `InternalAuth` accepts `Authorization: Bearer ` (empty token) when `INTERNAL_TOKEN` is
  unset — `subtle.ConstantTimeCompare("", "") == 1`. Fix: reject empty `wantToken` in the
  middleware (or fail-fast at startup). Smallest correct diff: early-return 401 when
  `wantToken == ""` or the trimmed token is empty. Add a test case: unset/empty token → 401.

**Verify-only checklist (each line gets a ✓ or a finding in NOTES.md):**
- [ ] Answer leak: `publicQuestion` is the only serialized question shape; no
      `correct_option`/`explanation` in any HTTP response (grep handler package; handler
      test already asserts).
- [ ] Validator fails closed: unknown topic, wrong count, bad option ids, non-array all fail
      before DB insert (covered by M2 tests).
- [ ] All SQL parameterized — no string-built queries with user input
      (`InsertQuestions` builds placeholders only).
- [ ] Secrets: `.env` gitignored (`git check-ignore .env`); AI key appears only in the
      `Authorization` header; webhook logs URL host only; generate error log doesn't print
      request secrets.
- [ ] Fire-and-forget notifier: goroutine + panic recovery, `context.WithoutCancel`, 5s
      timeout; failure never blocks batch flow (tests exist — confirm they cover the seam).
- [ ] Generation reliability: idempotent via `batch_date` unique; failed→pending reset;
      pending-with-questions finalize (M2 test); log row on every terminal state (FR-9);
      fallback serves latest success (NFR-5).
- [ ] Worst-case latency of `POST /internal/generate`: 3 attempts × 60s AI timeout ≈ 3+ min —
      within GitHub Actions' default 360s timeout; note for TASK-10 to not set a shorter one.
- [ ] Backlog-only (do NOT fix in MVP): server lacks `http.Server` ReadHeader/Write timeouts
      (note for TASK-9); `/topics` POST is unauthenticated rate-unbounded (single-user
      acceptable); repeat answering the same question re-inflates stats (accept for MVP);
      notify failure payload omits `error` key when `GenerateResult.Error` is nil.

Success criteria:
- [ ] Empty-token 401 fix + test merged; full suite green.
- [ ] NOTES.md has the review table with ✓/finding per line and a short backlog section.

### M4 — Docs & config

Dependencies: M3 (document the state as fixed).

1. `README.md`:
   - Add a **Testing** section: `go test ./...`; explain `TEST_DATABASE_URL` gates DB tests
     and add the warning: DB-gated tests **TRUNCATE all tables — never point it at a DB you
     care about**; unit tests cover everything else.
   - Add `AI_BASE_URL`, `AI_MODEL`, `PORT`, `CORS_ORIGIN` to the env block (currently missing;
     `.env.example` is already correct — cross-check, don't diverge).
   - Fix typos ("chioce" → "choice"; "quizeme" → "quizme" in the structure tree).
2. `.env.example` — verify only; no edits expected.
3. Failing-branch spot check (TODO.md item): temporarily break `internal/ai/validate.go`
   (e.g. accept option id "e"), run `go test ./internal/ai/`, confirm failure, restore,
   re-run green. Record the observed failure text in NOTES.md.

Success criteria:
- [ ] README Testing section + env vars + typo fixes committed.
- [ ] Spot-check evidence (command + failure + restore) in NOTES.md.
- [ ] `.env`, `REQUIREMENT.md`, `plan/PLAN.md` untouched.

### M5 — Final gates + bookkeeping (LOOP step 5–7)

Dependencies: M1–M4.

1. Gates: `gofmt -w .` clean → `go build ./...` → `go vet ./...` → `go test ./...` →
   `go test -race ./internal/service/ ./internal/ai/` (race check for the notifier goroutine).
2. Tick every checkbox in `plan/TASK-7/TODO.md` (only ones actually verified).
3. Update `plan/TASK.md` row 7 → `done`; tick TASK-7 in `plan/OVERVIEW.md`
   (status updates are part of the loop convention per LOOP.md steps 7 and AGENTS.md).
4. Append `plan/PROGRESS.md`: date, what changed, test counts, review findings resolved,
   **explicit warning that TEST_DATABASE_URL must never be the user's real DB (TRUNCATE)**,
   and notes for TASK-8 (API contract stable, no changes made to handlers) and TASK-9
   (server timeouts backlog item) and TASK-10 (3-min worst-case generate latency).
5. Commit: `task 7: unit test coverage gaps, internal-token hardening, docs, migration verification`.

Success criteria:
- [ ] All gates green including `-race`.
- [ ] TODO.md boxes ticked against real verification; TASK.md/OVERVIEW.md updated;
      PROGRESS.md appended; single commit made.

## Dependencies & ordering

```
M1 (baseline + read-only DB verification)
 └─> M2 (coverage gaps)
      └─> M3 (security review + token fix)
           └─> M4 (docs + spot check)
                └─> M5 (gates + bookkeeping + commit)
```

External/user-owned (start asking early, do not block on): AI_API_KEY for the real-AI smoke
test (`TestRealSmoke` stays skipped — not a TASK-7 criterion); a disposable Postgres if the
user ever wants the DB-gated suites to run against anything.

## Risks

| Risk | Mitigation |
|---|---|
| Integration test helpers `TRUNCATE ... CASCADE` — pointing `TEST_DATABASE_URL` at the user's real DB wipes all quiz data | **Hard rule:** leave `TEST_DATABASE_URL` unset in this task. DB-gated tests skip by design. If the user later provides a disposable DB, set it only there. Never use `.env`'s `DATABASE_URL` for test runs. |
| Migration state dirty or behind on the user DB | M1 stops and asks; only additive `migrate up` with explicit user approval. No down/reset regardless. |
| Rootless Docker flakiness (seen in TASK-4) | Prefer direct `psql "$DATABASE_URL"` for read-only checks; docker exec only as fallback. |
| `rnd.Shuffle` in selection could make distribution tests flaky if written as sampling asserts | Assert per-seed deterministic outcomes (weak slots ⊆ weakest set), not stochastic thresholds — seeded ⇒ zero flake. |
| Scope creep into TASK-9/TASK-10 territory (server timeouts, Actions timeout) | Review-only; findings go to NOTES.md backlog, no code beyond the empty-token fix. |
| Editing `REQUIREMENT.md`/`plan/PLAN.md` is forbidden | Plan targets `plan/TASK-7/PLAN.md` (this file), NOTES.md, README, TODO.md, TASK.md/OVERVIEW.md status columns only. |

## Overall completion criteria for TASK-7

1. `gofmt`, `go build`, `go vet`, `go test ./...`, `go test -race` all green.
2. Every checkbox in `plan/TASK-7/TODO.md` ticked with real verification evidence; the gap
   table in this plan fully closed.
3. Empty-`INTERNAL_TOKEN` trust-boundary hole fixed with a regression test.
4. Security/reliability review recorded in `plan/TASK-7/NOTES.md` with ✓/finding per line
   and a TASK-9/TASK-10 backlog section.
5. Migration state verified read-only against the user-approved `DATABASE_URL`: version 7,
   clean, §6 schema + seeds present; zero writes/destructive commands against that DB.
6. Docs updated (README testing + env), statuses updated, PROGRESS.md appended, committed
   as `task 7: ...`.
