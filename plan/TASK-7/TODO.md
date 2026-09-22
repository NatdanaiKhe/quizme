# TASK-7 — Unit tests

## Objective
Lock in the critical logic: selection distribution, validator, retry/fallback flow, stats math. Table-driven, minimal mocks. Spec coverage: FR-2, FR-6, FR-8.

## Prerequisites
- TASK-3 and TASK-4 code exist (can be written alongside them incrementally).

## Checklist

### Topic selection (§8 / FR-8)
- [x] Table-driven with seeded `rand.Rand`: distribution across many runs ≈ 70/30 split (e.g. over 200 runs of 5, weakest-topic share in expected band).
- [x] Unattempted topics ranked weakest (no stats rows yet).
- [x] Never all 5 questions from one topic.
- [x] Edge: exactly n topics, more topics than n, all topics attempted with equal accuracy.

### Validator (FR-2)
- [x] Valid 5-question payload → pass.
- [x] Malformed JSON → parse error. Markdown-fenced JSON → passes.
- [x] Wrong `correct_option` values (`"e"`, `""`, mismatched id) → fail with message naming the item.
- [x] Missing/empty required fields; wrong option count; duplicate option ids → all fail.

### Generation retry/fallback (§10 / FR-6 / NFR-5)
- [x] AI error → 3 attempts total → batch failed, log written.
- [x] Validation fail → 1 retry → second failure → batch failed.
- [x] Transient failure then success → batch success, questions inserted.
- [x] Failed/pending batch → `GetTodayQuiz` returns latest successful batch.
- [x] No successful batch ever → clean error.
- [x] Mock AI client behind interface; use real (or transactional-mocked) repo where practical — keep mocks minimal, no full framework.

### Stats update math (FR-6)
- [x] First answer for a topic initializes stats correctly (no divide-by-zero, accuracy exact).
- [x] Subsequent answers update accuracy/counts/`last_practiced_at` correctly (hand-computed expected values).

### Run & gate
- [x] `go test ./...` green.
- [x] Failing-branch spot check: temporarily break validator → tests catch it → restore.

## Done when
- [x] `go test ./...` green with FR-2, FR-6, FR-8 critical branches covered.
- [x] Tests are deterministic (seeded randomness), no flaky timing.
