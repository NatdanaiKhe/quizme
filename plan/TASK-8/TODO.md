# TASK-8 — Frontend (React/Vite)

## Objective
React + Vite SPA talking to the §7 API: today's quiz with immediate feedback, stats view, topic list, graceful error states. Spec: `REQUIREMENT.md` §7 (API contract).

## Prerequisites
- TASK-6 done (all endpoints live, CORS configured for dev origins).

## Checklist

### Scaffold
- [x] Vite + React app in `frontend/` (separate from Go backend; talks REST).
- [x] Small API client module (`frontend/src/api.js|ts`): base URL from env (`VITE_API_BASE`), typed/thin wrappers for all §7 endpoints.
- [x] `.gitignore` covers `node_modules`, `dist`.

### Today's quiz page
- [x] Fetch `GET /quiz/today` on load; render 5 questions with options (a–d).
- [x] Submit each answer to `POST /quiz/answer`; immediately show correct/incorrect + explanation (FR-3 behavior) before advancing.
- [x] Track completion; after 5th answer show a done state with link to stats.
- [x] Do not attempt to display correct answers pre-submission — API doesn't send them.

### Stats page
- [x] `GET /stats` → per-topic accuracy, attempts, last practiced.
- [x] Simple chart or bar representation (plain CSS bars fine — no chart library unless already trivially available).

### Topics
- [x] List topics (`GET /topics`); add-topic form (`POST /topics`) per spec; show inline error on duplicate/invalid.

### Error states (no blank screens)
- [x] No quiz today / all-batches-failed → friendly "no quiz available" screen.
- [x] Fallback quiz served (older batch) → still renders; optionally note it's a previous quiz if the API exposes the date.
- [x] Network error / 5xx → retry affordance + message.
- [x] Loading states for every fetch.

### Verification (full user loop)
- [x] Start backend + frontend dev servers → see quiz → answer all 5 → feedback shown each time → stats page reflects the answers (accuracy changes).
- [x] Kill backend mid-session → error state renders, no crash.
- [x] Empty DB (no batches) → "no quiz" state.

## Done when
- [x] Full loop works: quiz → answer → feedback → stats reflect answers.
- [x] All error states render gracefully.
- [x] No console errors in normal flow.
