# TASK-6 — API layer (Gin)

## Objective
Expose the §7 endpoints with correct response shapes, no answer leakage, and token-guarded internal trigger. Spec: `REQUIREMENT.md` §7 (contract + examples), FR-3, FR-4, FR-7.

## Prerequisites
- TASK-4 done (services: generation, selection, fallback quiz).
- TASK-5 done (notifier wired into generate flow).

## Checklist

### Handlers (`internal/handler/`)
- [x] `GET /quiz/today` — 5 questions via service fallback logic (§10). Response: question text, options (id + text), topic. **`correct_option` and explanation MUST NOT be serialized** — build a dedicated response struct, do not marshal the domain model.
- [x] `POST /quiz/answer` — body: question id + chosen option id; returns immediate correct/incorrect + explanation; triggers transactional answer + stats update (TASK-2 repo). Validate option id ∈ {a,b,c,d}; unknown question id → 404.
- [x] `GET /stats` — per-topic stats from `user_topic_stats` (accuracy, attempts, last_practiced_at) matching §7 example shape.
- [x] `GET /topics` — list topics.
- [x] `POST /topics` — create topic; duplicate name → 400 or 409 per §7.
- [x] `POST /internal/generate` — runs the generation service for today; guarded by `Authorization: Bearer {INTERNAL_TOKEN}`; missing/wrong token → 401.

### Wiring & middleware
- [x] Gin router in `cmd/server`; handlers receive service interfaces (not concrete structs) where it aids testing.
- [x] Request validation on bodies → 400 with useful message; unexpected errors → 500, logged, no stack traces to client.
- [x] CORS: allow the frontend origin (needed by TASK-8) — or defer to same-origin reverse proxy in TASK-9, but configure it now for local dev.

### Manual curl verification (all of §7)
- [x] Every endpoint exercised with curl; request/response shape matches §7 examples field-by-field.
- [x] `/quiz/today`: assert in response JSON that `correct_option` (and explanation) are absent — explicit check, this is the cheating vector.
- [x] `/internal/generate`: no token → 401; wrong token → 401; correct token → generation runs, 200 (or accepted-status per §7).
- [x] `/quiz/answer`: correct answer → correct:true + explanation; wrong → correct:false; stats row updated after call.

## Done when
- [x] All §7 endpoints respond with spec-conformant shapes and status codes.
- [x] `/quiz/today` provably leaks no answers (checked, not assumed).
- [x] `/internal/generate` rejects missing/wrong token.
- [x] Error responses are sane (400/401/404/409/500) — no 200-wrapped errors.
