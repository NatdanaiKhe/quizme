# Quizme System — Requirements & Spec

## 1. Overview

A system that automatically generates and delivers daily technical quizzes for practice. It uses AI to auto-generate questions based on topics the user is weak in, mixed with newly rotated topics, forming an adaptive learning loop.

**Goals**
- Have a fresh quiz ready every day without manual prep
- The system learns which topics the user is weak in and surfaces those topics more often
- Easy to deploy, low maintenance — suited for a personal project

---

## 2. Scope

### In Scope (MVP)
- Generate multiple-choice questions daily via the AI API
- Store questions, answers, and user stats in Postgres
- API to fetch today's quiz / submit answers
- Daily scheduled job via GitHub Actions
- Adaptive topic selection (70% weak topics / 30% new/random)

### Out of Scope (Future Phase)
- Full multi-user / authentication system (MVP is single-user)
- Leaderboard / social features
- Fine-grained adaptive difficulty tuning

### Decisions (Finalized)
- **Frontend**: A separate app from the backend (NextJS, communicating via REST API)
- **User model**: Single-user for now, no auth in MVP
- **Initial topics**: Frontend, Backend, Infrastructure (more can be added later)
- **Notification**: Uses a generic webhook (not tied to a specific provider), currently wired to **LINE**

---

## 3. Functional Requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-1 | The system must generate 5 questions/day automatically via the AI API | High |
| FR-2 | The system must validate the AI response before saving (valid JSON, `correct_option` matches one of `options`) | High |
| FR-3 | Users must be able to fetch today's quiz via API | High |
| FR-4 | Users must be able to submit an answer and get immediate feedback (correct/incorrect + explanation) | High |
| FR-5 | The system must record per-topic stats (accuracy, attempt count, last practiced date) | High |
| FR-6 | The system must select next-day topics using a weighted approach (favoring weak topics) | Medium |
| FR-7 | The system must support adding/editing the topic list (via DB or config) | Medium |
| FR-8 | If generation fails (AI error, failed validation), the system must retry or fall back | Medium |
| FR-9 | The system must log each day's generation run (success/failure, question count) | Low |
| FR-10 | After a batch is generated successfully, the system must fire a webhook (POST to a URL set via env var) with a summary payload, so a downstream service (LINE/Discord/Slack/etc.) can forward it | Medium |

---

## 4. Non-Functional Requirements

| ID | Requirement |
|---|---|
| NFR-1 | Must be deployable on the homelab via Docker or serverless |
| NFR-2 | Scheduling runs via GitHub Actions cron (no separate server needed just for scheduling) |
| NFR-3 | API response time < 500ms (excluding generation, which runs as a background job) |
| NFR-4 | AI API cost must stay low (controlled via short prompts, token limits) |
| NFR-5 | The system must keep running even if a given day's generation fails (no full-system crash) |

---

## 5. System Architecture

![Flow](https://bucket.natdanai.dev/quizeme-flow.drawio.png)

---

## 6. Database Schema

```sql
CREATE TABLE topics (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    weight INT DEFAULT 1,       -- initial importance weight
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE quiz_batches (
    id SERIAL PRIMARY KEY,
    batch_date DATE NOT NULL UNIQUE,   -- one batch per day
    status TEXT NOT NULL DEFAULT 'pending', -- pending / success / failed
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE questions (
    id SERIAL PRIMARY KEY,
    batch_id INT REFERENCES quiz_batches(id),
    topic_id INT REFERENCES topics(id),
    prompt TEXT NOT NULL,
    options JSONB NOT NULL,        -- [{"id":"a","text":"..."}, ...]
    correct_option TEXT NOT NULL,
    explanation TEXT,
    source TEXT DEFAULT 'ai_generated',
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE user_answers (
    id SERIAL PRIMARY KEY,
    question_id INT REFERENCES questions(id),
    selected_option TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL,
    answered_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE user_topic_stats (
    topic_id INT PRIMARY KEY REFERENCES topics(id),
    correct_count INT DEFAULT 0,
    wrong_count INT DEFAULT 0,
    last_practiced_at TIMESTAMPTZ,
    accuracy_rate NUMERIC(5,2) DEFAULT 0
);

CREATE TABLE generation_logs (
    id SERIAL PRIMARY KEY,
    batch_id INT REFERENCES quiz_batches(id),
    status TEXT NOT NULL,          -- success / failed
    error_message TEXT,
    tokens_used INT,
    created_at TIMESTAMPTZ DEFAULT now()
);
```

> Note: this schema is designed for single-user (MVP). To extend to multi-user later, add `user_id` to `user_answers` and `user_topic_stats`, and change the unique constraint to `(user_id, topic_id)`.

**Seed data (initial topics):**
```sql
INSERT INTO topics (name, weight) VALUES
  ('Frontend', 1),
  ('Backend', 1),
  ('Infrastructure', 1);
```

---

## 7. API Endpoints

| Method | Path | Purpose |
|---|---|---|
| GET | `/quiz/today` | Fetch today's question set (5 questions) |
| POST | `/quiz/answer` | Submit an answer for one question → get correct/incorrect + explanation |
| GET | `/stats` | Fetch per-topic accuracy stats |
| GET | `/topics` | List all topics |
| POST | `/topics` | Add a new topic |
| POST | `/internal/generate` | (called by GitHub Actions) triggers today's question generation — should be protected by an auth token to prevent outside callers |

### Example Response: `GET /quiz/today`
```json
{
  "batch_date": "2026-09-14",
  "questions": [
    {
      "id": 101,
      "topic": "Database Indexing",
      "prompt": "...",
      "options": [{"id": "a", "text": "..."}, {"id": "b", "text": "..."}]
    }
  ]
}
```
(Note: the response sent to the frontend **must not include `correct_option`** — keep that server-side only, to prevent clients from cheating.)

---

## 8. Topic Selection Logic (Adaptive)

1. Fetch all `user_topic_stats`
2. Sort topics by `accuracy_rate` ascending (a topic never attempted counts as weakest / highest priority)
3. Pick 70% of today's questions from the weakest topic group (e.g. 3–4 questions from the bottom 3 topics)
4. Pick the remaining 30% randomly from all other topics (to avoid always repeating the same topics)
5. Pass the selected topic list as input to the question generator

---

## 9. AI Prompt Template (Draft)

```
System: You are writing exam questions for a full-stack developer.
Generate {n} multiple-choice questions on the following topics: {topics}
Respond with a JSON array only, no other text, in this format:
[
  {
    "topic": "...",
    "prompt": "...",
    "options": [{"id":"a","text":"..."}, {"id":"b","text":"..."}, {"id":"c","text":"..."}, {"id":"d","text":"..."}],
    "correct_option": "b",
    "explanation": "..."
  }
]
```

---

## 10. Generation Flow (Error Handling)

```
1. GitHub Actions cron trigger → POST /internal/generate
2. Backend creates a quiz_batches row (status: pending)
3. Select topics (adaptive logic)
4. Call the AI API
   ├─ Success → validate JSON
   │    ├─ Passes → insert questions, set batch status = success
   │    └─ Fails → retry once → if still failing → status = failed, log error
   └─ Failure (timeout/API error) → retry (max 2 times) → if still failing → status = failed
5. Record a generation_logs entry every time (success or failure)
```

**Fallback:** if today's batch status = failed and a user opens `/quiz/today` → the backend serves questions from the most recent batch with status = success instead (so the user never sees a blank page).

### Webhook Notification

After a batch's status becomes `success` (or `failed` after exhausting retries), the system POSTs to `WEBHOOK_URL` (env var) with this payload:

```json
{
  "batch_date": "2026-09-14",
  "status": "success",
  "question_count": 5,
  "topics": ["Frontend", "Backend", "Infrastructure"]
}
```

It's not tied to a specific provider — the receiving side can turn this payload into a LINE message, Discord embed, Slack message, etc. (e.g. via a small relay/middleware service, or a tool like n8n/Zapier). Currently wired to **LINE**, via the **LINE Messaging API** (see note below).

> **Important note:** LINE Notify was discontinued as of March 31, 2025 and can no longer be used. Use the **LINE Messaging API** instead — push messages through a LINE Official Account, which requires setting up an OA and obtaining a Channel Access Token.

---

## 11. Project Structure (Go, proposed)

```
quizme/
├── cmd/
│   └── server/main.go
├── internal/
│   ├── handler/          # HTTP handlers (Gin)
│   ├── service/          # business logic (topic selection, generation)
│   ├── repository/       # DB access (Postgres)
│   ├── ai/               # AI API client
│   └── model/            # structs
├── migrations/           # SQL migration files
├── .github/
│   └── workflows/
│       └── daily-generate.yml
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

---

## 12. Deployment Plan

- Backend + Postgres run on the homelab via `docker-compose`
- GitHub Actions workflow (cron `0 23 * * *` UTC = 06:00 ICT) sends an HTTP request to `/internal/generate` (the endpoint must be reachable from outside, e.g. via a Cloudflare Tunnel)
- Secrets (AI API key, internal auth token, LINE Channel Access Token) are stored in GitHub Actions Secrets + a `.env` file on the homelab

---

## 13. Open Questions

None remaining — all resolved (see Decisions above): frontend = React, notification destination = LINE.

---

## 14. Next Steps (Detailed)

### Phase 0 — Basic Setup
- [ ] Create the `quizME` repo (separate backend/frontend repos or a monorepo — monorepo recommended for a personal project)
- [ ] Create a LINE Official Account (free, via LINE Developers Console)
- [ ] Enable a Messaging API channel → obtain a **Channel Access Token** + **Channel Secret**
- [ ] Add the LINE OA as a friend on your own account (required before you can push messages to yourself) → get your own `userId` for push messaging
- [ ] Sign up for the AI API, get an API key
- [ ] Provision Postgres on the homelab (Docker container or use an existing instance)

### Phase 1 — Database
- [ ] Write SQL migration files per the schema in section 6 (use a tool like `golang-migrate` or `goose`)
- [ ] Run migrations + seed initial topics (Frontend, Backend, Infrastructure)
- [ ] Test basic queries (insert/select) to confirm the schema works as expected

### Phase 2 — Backend Core (Go + Gin)
- [ ] Set up the project structure per section 11 (`cmd/`, `internal/handler`, `internal/service`, `internal/repository`, `internal/ai`, `internal/model`)
- [ ] Write `internal/repository` — the DB access layer (using `pgx` or `sqlx`) for every table
- [ ] Write `internal/ai` — an HTTP client that calls the AI API with the prompt template (section 9) and parses the response
- [ ] Write a validator — checks JSON structure, confirms `correct_option` matches one of the real `options`
- [ ] Write `internal/service` — the adaptive topic selection logic (70/30, section 8) and the generation flow with retries (section 10)
- [ ] Write the webhook sender — POSTs the payload to the LINE Messaging API after a batch completes (section 10.1)

### Phase 3 — API Layer
- [ ] Write Gin handlers for every endpoint in section 7 (`/quiz/today`, `/quiz/answer`, `/stats`, `/topics`, `/internal/generate`)
- [ ] Add an auth token guard for `/internal/generate` (to block outside callers)
- [ ] Write unit tests for the service layer (topic selection, validator), at least for the critical logic
- [ ] Manually test every endpoint with Postman/curl

### Phase 4 — Frontend (React)
- [ ] Set up the React project (Vite recommended — light and fast)
- [ ] Today's quiz page — fetches from `/quiz/today`, renders one question at a time or all at once
- [ ] Submit answers to `/quiz/answer` → show correct/incorrect + explanation immediately
- [ ] Stats page — shows per-topic accuracy (a simple chart works fine)
- [ ] Handle error states (e.g. no quiz yet today, generation failed)

### Phase 5 — Infrastructure & Deployment
- [ ] Write a `Dockerfile` for the Go backend (multi-stage build to keep the image small)
- [ ] Write a `Dockerfile` for the React frontend (build, then serve via nginx or static hosting)
- [ ] Write a `docker-compose.yml` that ties together backend + frontend + Postgres
- [ ] Deploy to the homelab, set up a reverse proxy (e.g. Caddy/Nginx) + expose access from outside the home network via a Cloudflare Tunnel (needed because GitHub Actions must reach `/internal/generate` over HTTP)
- [ ] Store secrets (AI API key, LINE token, internal auth token, DB credentials) in a `.env` file on the homelab (never commit them to the repo)

### Phase 6 — Scheduler (GitHub Actions)
- [ ] Write `.github/workflows/daily-generate.yml` — cron `0 23 * * *` (23:00 UTC = 06:00 ICT)
- [ ] The workflow sends a `POST` to `/internal/generate` with an auth header
- [ ] Store the URL + auth token in GitHub Actions Secrets
- [ ] Test triggering the workflow manually (`workflow_dispatch`) before letting it run on schedule

### Phase 7 — End-to-End Testing & Tuning
- [ ] Run the full system for 3–5 consecutive days, confirm generation runs on time and the webhook actually reaches LINE
- [ ] Confirm adaptive topic selection behaves as designed (weak topics really do show up more often)
- [ ] Tune the prompt template if the generated questions aren't at the right level
- [ ] Log a backlog for the next phase (multi-user, difficulty tuning, etc.)
