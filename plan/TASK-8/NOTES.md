# TASK-8 Notes — Frontend (React/Vite)

## Decisions

- **No client router dependency:** Routing uses hash-based tabs (`#/quiz`, `#/stats`, `#/topics`) in `App.jsx`, avoiding extra bundle weight.
- **Client-side topic join for stats:** The backend `GET /stats` returns `topic_id` without topic names. The stats view fetches `GET /stats` and `GET /topics` concurrently via `Promise.all` and joins them client-side by `topic_id`.
- **Session-level answer tracking:** Answered question IDs are tracked in a `Set` in component state to prevent double-answering and duplicate stats updates during the quiz session.
- **Answer-leak protection:** Frontend never expects or checks `correct_option` from `GET /quiz/today`. Options only show feedback after `POST /quiz/answer` returns `correct` and `explanation`.
- **CSS-only visual styling:** Stats accuracy bars use pure CSS fill percentages (`width: accuracy_rate%`), avoiding heavyweight charting libraries.
- **Universal error/loading states:** Shared `Error` and `Loading` components handle empty/loading/error states across Quiz, Stats, and Topics screens with retry affordances.

## Verification

- `npm run lint`: 0 warnings, 0 errors.
- `npm run build`: built clean in <400ms (`dist/` generated).
- `go test ./...`: all backend tests remain green.
- Live backend verification:
  - CORS preflight and headers verified against `http://localhost:5173`.
  - Quiz flow: `GET /quiz/today` → renders question & options → `POST /quiz/answer` → returns correctness + explanation → completion screen links to stats.
  - Stats view: displays total answered, correct, wrong, and per-topic accuracy bars and last practiced date.
  - Topics view: lists topics, adds new topic, and displays inline error on 409 conflict or empty input.
  - Network error: graceful error banner with Retry button when backend is unreachable.

## Notes for downstream tasks

- **TASK-9**: Frontend Dockerfile can use multi-stage build: `node:alpine` (`npm run build`) → `nginx:alpine` to serve static assets from `/app/dist`. Reverse proxy will route `/api/` or same-origin requests to the Go backend.
