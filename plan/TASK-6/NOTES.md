# TASK-6 Notes — API Layer (Gin)

## Decisions

- Duplicate topic name returns **409 Conflict** (not 400). This gives the client a clearer signal and is allowed by §7's "400 or 409" wording.
- CORS is implemented as a ~15-line stdlib-style Gin middleware. It reads `CORS_ORIGIN` (default `http://localhost:5173`) and handles `OPTIONS` preflight with `204 No Content`.
- `/internal/generate` remains synchronous. The service already handles idempotency, retries, and fire-and-forget notifications. Making it async is over-engineering for the MVP scheduler (GitHub Actions tolerates the request duration).
- Explanation is always returned on `/quiz/answer`, whether the answer is correct or wrong, matching the frontend's immediate-feedback requirement.

## Manual curl verification (captured)

Server booted with `.env` values; DB seeded with one successful batch and one question via a temporary Go helper.

```text
GET /health → 200 {"status":"ok"}

GET /quiz/today → 200
{"batch_date":"2026-09-22","questions":[{"id":1,"topic":"Frontend",...}]}
Leak check (jq): correct_option absent ✓, explanation absent ✓

POST /quiz/answer {"question_id":1,"option":"b"} → 200
{"correct":true,"explanation":"Beta is correct"}

POST /quiz/answer {"question_id":1,"option":"a"} → 200
{"correct":false,"explanation":"Beta is correct"}

POST /quiz/answer {"question_id":1,"option":"e"} → 400
{"error":"option must be a, b, c, or d"}

POST /quiz/answer {"question_id":999,"option":"a"} → 404
{"error":"question not found"}

POST /quiz/answer not-json → 400
{"error":"invalid request body"}

GET /stats → 200 (after one correct + one wrong answer)
[{"topic_id":1,"correct_count":1,"wrong_count":1,"accuracy_rate":50,...}]

GET /topics → 200 (3 seeded topics)

POST /topics {"name":"Security"} → 201
POST /topics {"name":"Security"} → 409 {"error":"topic already exists"}
POST /topics {"name":"  "} → 400 {"error":"name is required"}

POST /internal/generate (no token) → 401
POST /internal/generate Authorization: Bearer wrong → 401
POST /internal/generate Authorization: Bearer dev-token → 200
  (idempotent: today's batch already success, so no AI call)

OPTIONS /quiz/today Origin:http://localhost:5173 → 204
  Access-Control-Allow-Origin: http://localhost:5173
  Access-Control-Allow-Methods: GET, POST, OPTIONS
  Access-Control-Allow-Headers: Content-Type, Authorization
```

## Blockers

- Full end-to-end `/internal/generate` through a live AI call is blocked on the user-owned `AI_API_KEY` (and `AI_BASE_URL`/`AI_MODEL`). With the current empty `AI_API_KEY`, the endpoint returns 401 for bad tokens and 200 for idempotent/success batches, but a fresh date would hit the AI and fail.

## Next task

TASK-7 — unit tests for selection, validator, generation retry logic.
