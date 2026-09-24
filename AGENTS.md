# Quizme — Agent Guide

Single-user MVP: Go backend + React frontend monorepo. AI-generated daily quizzes, adaptive topic selection, LINE notifications.

## Start here

1. Read `plan/OVERVIEW.md` — task list with checkboxes
2. Read `plan/TASK.md` — task details, dependencies, success criteria
3. Pick the next uncompleted task, read its `plan/TASK-N/TODO.md`
4. Execute the checklist in that TODO.md

**Dependency order matters.** Don't skip ahead. Check `plan/TASK.md` dependency column before starting.

## Architecture

- **Backend:** Go + Gin, `internal/{handler,service,repository,ai,model}`
- **Frontend:** React + Vite, `frontend/`
- **Database:** Postgres, schema in `migrations/` (§6)
- **Scheduler:** GitHub Actions cron `0 23 * * *` UTC
- **Deployment:** Docker Compose + Cloudflare Tunnel on homelab

## Code style

**Ponytail mode (full intensity):**

- Shortest diff that works
- Stdlib before dependencies
- Deletion over addition
- No unrequested abstractions
- No boilerplate "for later"
- One line before fifty

If you're writing more than ~50 lines for a feature, stop and question whether it can be simpler.

## Key constraints

- **Single-user MVP** — no auth, no `user_id` in queries yet (but keep it isolatable in repository layer for future multi-user)
- **No answer leak** — `GET /quiz/today` must never return `correct_option` to the client
- **Fail closed** — AI validator rejects malformed output; retries live in service layer (TASK-4)
- **Idempotent generation** — `batch_date` unique constraint prevents duplicate batches
- **Fire-and-forget notifications** — webhook failure logged, never blocks batch flow

## Testing

- Table-driven Go tests for pure logic (selection, validator, stats math)
- No DB needed for most tests — use interfaces/mocks minimally
- `go test ./...` must pass before marking TASK-7 done

## External prerequisites (user-owned)

Start these early — they block generation testing:

- LINE OA + Messaging API channel (token/secret/userId)
- AI API key
- Homelab Postgres + outside-reachable URL (Cloudflare Tunnel)

## Task execution

Each `plan/TASK-N/TODO.md` is self-contained:

- **Objective:** what this task covers
- **Prerequisites:** what must be done first
- **Checklist:** concrete subtasks
- **Done when:** success criteria
- **Spec refs:** which REQUIREMENT.md sections apply

Read the TODO.md, execute the checklist, verify the "done when" criteria, then mark the task complete in `plan/OVERVIEW.md`.

## Database & containers — ask first

**Never** create, start, or provision a database container; modify database ports, credentials, or volumes; run migrations against a real/shared database; or change database connectivity without first asking the user and receiving explicit approval. Always inspect and use the user's existing database configuration first.

Warn about any port conflict before acting. **Never** run destructive database commands (`docker compose down -v`, drop, reset, `migrate down`, etc.) without explicit user approval.

## Don't

- Don't modify `REQUIREMENT.md`, `plan/OVERVIEW.md`, or `plan/PLAN.md` unless explicitly asked
- Don't add auth/multi-user features (out of MVP scope)
- Don't use LINE Notify (deprecated) — use LINE Messaging API
- Don't commit secrets (`.env` is gitignored)
- Don't over-engineer — ship the lazy version that works
