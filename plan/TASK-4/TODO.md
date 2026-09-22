# TASK-4 — Service layer: generation & adaptive topic selection

## Objective
The brain: adaptive 70/30 topic selection (§8) and the full generation flow with retries, logging, fallback, idempotency (§10). Spec: `REQUIREMENT.md` §8, §10, FR-6, FR-8, FR-9, NFR-5.

## Prerequisites
- TASK-2 done (repository queries incl. transactional answer/stats, batch create, latest-successful fetch, generation_logs).
- TASK-3 done (AI client returns validated questions or typed errors).

## Checklist

### Adaptive topic selection (`internal/service/selection.go`) — §8
- [x] Compute per-topic weakness from `user_topic_stats`; unattempted topics count as weakest (bottom of ranking, per §8 tie-break rule).
- [x] Split 5 questions: ~70% (3) from bottom-ranked topics, ~30% (2) random from remaining topics — deterministic rounding that always sums to n, never 0 weakest.
- [x] Never assign the same topic to all 5 questions.
- [x] Accept injected randomness (`*rand.Rand`) so tests can be deterministic.

### Generation flow (`internal/service/generate.go`) — §10
- [x] Entry: get-or-create pending batch for today (unique `batch_date` makes rerun idempotent — same date never duplicates).
- [x] Select topics → call AI client with topic list.
- [x] Success path: insert questions → batch status = success → `generation_logs` entry (status, tokens_used).
- [x] AI transport/error: retry up to 2 more times (max 3 attempts total, per §10).
- [x] Validation failure: retry once (fresh AI call), per §10.
- [x] Exhausted (AI error or 2nd validation failure): batch status = failed + `generation_logs` entry with error text. Never insert unvalidated questions.
- [x] Webhook notification hook called on terminal state (success or failed) — implemented in TASK-5; here just call the notifier interface if present (nil-safe).
- [x] Pending-batch ambiguity resolved: pending batch with existing questions is finalized as success to avoid duplicate generation.

### Fallback (`internal/service/quiz.go`) — NFR-5
- [x] `GetTodayQuiz`: today's batch exists and is success → serve it.
- [x] Today's batch pending/failed or missing → serve latest successful batch (any date). Pending batch does NOT block serving.
- [x] No successful batch at all → clean `ErrNoQuiz` sentinel, not a crash.

### Failure injection test (manual or scripted)
- [x] AI pointed at dead endpoint / always-error fake → batch failed, log written, `GetTodayQuiz` still serves latest successful batch, process alive.

## Done when
- [x] Selection honors 70/30 across runs; unattempted topics prioritized (verifiable in tests — deterministic seed).
- [x] Failure injection: AI always fails → failed batch + log + fallback serves, no crash.
- [x] Rerunning generation for the same date does not create a second batch.
- [x] Every generate run leaves exactly one `generation_logs` row.
