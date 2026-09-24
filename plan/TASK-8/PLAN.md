# TASK-8 PLAN — Frontend (React/Vite)

Scope source: `plan/TASK-8/TODO.md`, `REQUIREMENT.md` §7, actual handler code
(`internal/handler/quiz.go`, `topics.go`). Prerequisite TASK-6 is **done**; CORS
middleware already exists (`CORS_ORIGIN` env, default `http://localhost:5173`
— matches Vite's dev port). `frontend/` is currently empty.

## Verified API contract (from handler code, not just spec)

| Call | Request | Success | Errors |
|---|---|---|---|
| `GET /quiz/today` | — | `200 {batch_date, questions:[{id, topic, prompt, options:[{id,text}]}]}` | `404 {error}` (no quiz / all failed), `500` |
| `POST /quiz/answer` | `{question_id, option}` (option ∈ a–d) | `200 {correct: bool, explanation}` | `400`, `404 {error}` |
| `GET /stats` | — | `200 [{topic_id, correct_count, wrong_count, last_practiced_at?, accuracy_rate}]` | `500` |
| `GET /topics` | — | `200 [{id, name, weight, created_at}]` | `500` |
| `POST /topics` | `{name}` | `201 {id, name, weight, created_at}` | `400` (bad/empty), `409` (duplicate) |

Facts that shape the frontend:
- **No `correct_option` / `explanation` in `/quiz/today`** — explanation only
  arrives per-answer via `/quiz/answer`. Never render pre-submission answers.
- **`GET /stats` has no topic names** — only `topic_id`. The stats page must
  join with `GET /topics` client-side (no backend change needed).
- `last_practiced_at` is omitted when null (unattempted) → show "never".
- `batch_date` is a `YYYY-MM-DD` string → compare to today's local date to
  detect and label a fallback (older) quiz.
- Backend has **no dedup on answers** — resubmitting a question re-counts stats.
  Frontend must track answered question IDs in session state (in-memory only;
  a page reload could re-answer — acceptable single-user MVP behavior, note it).

## Milestones (ordered)

### M1 — Scaffold + env + API client
1. Scaffold Vite + React in `frontend/` (`npm create vite@latest . -- --template react`).
   Plain JS is enough (no TS) — smallest diff. Keep the default Vite gitignore;
   verify `node_modules`/`dist` are ignored at repo root too.
2. `frontend/src/api.js`: one small module. Base URL from `import.meta.env.VITE_API_BASE`
   (default `http://localhost:8080`). Thin wrappers: `getTodayQuiz`, `submitAnswer`,
   `getStats`, `getTopics`, `addTopic`. Each wrapper maps non-2xx responses to
   `{status, error}` so pages can branch on 404 vs 500 vs 409 without parsing.
   Add `frontend/.env.example` with `VITE_API_BASE=http://localhost:8080`.
3. No router dependency. Three views via a tiny nav (useState tab or hash
   `#/quiz /#/stats /#/topics`) — 3 screens don't justify react-router.

**Done when:** `npm run dev` serves the shell; `import.meta.env.VITE_API_BASE`
resolves; `npm run build` green.

### M2 — Quiz page (the core loop)
1. On mount fetch `getTodayQuiz`. Loading spinner while fetching.
2. Render 5 questions one at a time (spec allows either; one-at-a-time is
   simpler for the immediate-feedback flow). Option buttons a–d per question.
3. Click → `submitAnswer(question_id, option)` → disable options → show
   correct/incorrect banner + explanation (FR-4) → "Next" button.
4. After the 5th answer show a done state with a link to stats.
5. Track answered IDs in a `Set` in component state; ignore clicks on already-
   answered questions (prevents double-counted stats).

**Done when:** fetch → render → answer → feedback → done state works against
the live backend; no `correct_option` rendered anywhere (it isn't sent).

### M3 — Stats + Topics pages
1. Stats: fetch `/stats` and `/topics` in parallel, join by `topic_id`, render
   per-topic rows: name, accuracy % as a plain CSS bar, correct/wrong counts,
   last practiced (or "never"). No chart library.
2. Topics: list from `/topics`; add form → `POST /topics`; on `409` show inline
   "topic already exists", on `400` show "name required"; clear + refresh list
   on `201`.

**Done when:** both pages render with live data; empty stats (fresh DB) shows
a sane empty state, not a crash.

### M4 — Error & loading states (no blank screens)
- `GET /quiz/today` → 404: friendly "no quiz available today" screen.
- Fallback quiz: if `batch_date` !== today, render quiz + note "showing a
  previous quiz from {date}".
- Network error / 5xx on any fetch: error message + Retry button (retry = re-run
  the fetch). Kill-backend verification lives in M6.
- Loading state on every fetch (quiz, stats, topics).
- One shared `Error({message, onRetry})` component used everywhere — not one
  per page.

**Done when:** every fetch path has loading / error / empty / success rendered;
no blank screen in any state.

### M5 — Verify the full loop (real commands, per LOOP.md)
1. Start backend (`go run ./cmd/server` with `.env`, or `docker compose up`)
   + `npm run dev` (Vite on :5173).
2. See quiz → answer all 5 → feedback each time → stats page reflects accuracy
   changes (compare before/after).
3. Kill backend mid-session → error state renders, no crash, retry works after
   restarting backend.
4. Empty DB (no batches) → "no quiz available" state.
   ⚠️ Getting an empty DB requires truncating the user's database — **ask first**
   (AGENTS.md DB rule). If refused, verify the 404 branch by temporarily pointing
   `VITE_API_BASE` at a dead port / stopping the DB container the user approves of.
5. Browser console: no errors in the normal flow.
6. `npm run build` green; `go test ./...` still green (backend untouched).

**Test decision:** TODO.md requires no frontend unit tests; the gate is
`npm run build` + the manual loop above. Skip vitest. If the API client's
error-mapping grows beyond ~10 lines of branching, revisit.

### M6 — Commit (LOOP.md step 7)
- Commit only: `frontend/` source (not `node_modules`, not `dist`),
  `frontend/.env.example`, updated `plan/TASK-8/TODO.md` checkboxes,
  `plan/TASK.md` + `plan/OVERVIEW.md` status, `plan/PROGRESS.md` entry.
- Message: `task 8: React/Vite frontend (quiz, stats, topics)`.
- Check `git status` before committing — no lockfile surprises, no `.env` files.

**Done when:** clean tree, commit `task 8: ...` exists.

## Dependencies
- M1 → M2 → M3 → M4 (build up, but M2–M4 are small and can interleave).
- M5 requires a running backend with at least one successful batch.
  If no batch exists (needs `AI_API_KEY`, user-owned), generate one via
  `POST /internal/generate` with `INTERNAL_TOKEN`, or seed questions manually
  with user approval. If neither is possible → fallback-quiz and full-loop
  criteria may force BLOCKED; the 404/error paths are still verifiable.

## Risks
| Risk | Mitigation |
|---|---|
| CORS blocked in dev | Backend middleware already defaults to `http://localhost:5173`. If requests fail, check backend `CORS_ORIGIN` in `.env` before touching frontend code. Prod CORS is avoided entirely by TASK-9's same-origin reverse proxy — don't solve it here. |
| Double-answered stats on page reload | Backend doesn't dedup. Accept as MVP behavior; answered-ID `Set` covers the normal session. Note in PROGRESS.md. |
| Can't reach a successful batch for verification | Needs AI key or manual seed — see M5 step 4; may block full-loop criteria. |
| DB truncation for empty-state test | User approval required (AGENTS.md). Use dead-port trick instead if refused. |
| Vite scaffold noise | Trim template boilerplate (logo, default CSS) to keep the diff small. |

## Explicitly out of scope
- No router/chart/state libraries — plain React + CSS.
- No auth, no multi-user, no persistence of answered state across reloads.
- No backend changes (stats topic-name join is client-side; CORS already done).
- No Docker/nginx for the frontend — TASK-9.
- No database operations of any kind without explicit user approval.
