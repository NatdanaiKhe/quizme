# TASK-3 — AI client + validator

## Objective
Call the AI API with the §9 prompt, parse the response tolerantly, validate strictly per FR-2, fail closed. Spec: `REQUIREMENT.md` §9 (prompt template), FR-1, FR-2, NFR-4.

## Prerequisites
- TASK-1 done (module, config with `AI_API_KEY`).
- User-provided: AI API key (from `.env`); endpoint + model name per REQUIREMENT.md.

## Checklist

### Client (`internal/ai/client.go`)
- [x] HTTP client with timeout; POST to AI endpoint with auth header from config.
- [x] Prompt template from §9 with `{n}` (question count) and `{topics}` substitution.
- [x] Request `max_tokens` sized to keep cost low (NFR-4) — but large enough for 5 questions; document the number chosen.
- [x] Return raw response text + token usage (for `generation_logs.tokens_used` later).
- [x] Non-200 / transport error → typed error (distinguishing "AI error" from "validation error" matters for TASK-4 retry policy).

### Parsing (`internal/ai/parse.go` or in client)
- [x] Strip markdown fences (```json ... ```) before unmarshal.
- [x] Unmarshal to a slice of question structs matching §9 fields.

### Validator (`internal/ai/validate.go`) — fail closed
- [x] Exactly n items.
- [x] Each item: all required fields present and non-empty (question text, options, `correct_option`, explanation if §9 requires).
- [x] Exactly 4 options, ids unique and exactly `a`, `b`, `c`, `d`.
- [x] `correct_option` ∈ {a,b,c,d}.
- [x] Any violation → single validation error listing what failed (error text goes into `generation_logs`).

### Verification
- [x] Table of mocked responses: valid 5-question JSON → pass; malformed JSON → parse error; fenced JSON → pass; wrong `correct_option` (e.g. `"e"` or mismatched option id) → validation error; missing explanation → validation error (if required).
- [ ] One real API call (manual, local) → 5 validated questions; record tokens_used observed.
- [x] Validator is pure function — no IO — so TASK-7 can test it directly.

## Done when
- [x] Valid response → []Question. Any malformed response → clear error, never partial/garbage questions.
- [ ] Real API call produces 5 validated questions end-to-end.
- [x] Error types let callers distinguish AI-failure vs validation-failure.
