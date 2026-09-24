# Quizme

The goal of this project is to continuously test and strengthen my knowledge across my full-stack development domain.

It is an AI-generated daily quiz system that creates a new set of questions every day based on topics I’m weak in, while also introducing fresh topics to keep the practice varied and well-rounded.


## Features

- Questions auto-generated daily via the AI API
- Adaptive topic selection — weak topics show up more often (70% weak / 30% random)
- Fully automated via a GitHub Actions scheduled workflow — no server-side cron needed
- Webhook notifications (currently wired to LINE) when a new quiz is ready
- Self-hosted on your own infrastructure via Docker

## Tech Stack

| Layer | Tech |
|---|---|
| Backend | Go + [Gin](https://gin-gonic.com/) |
| Database | PostgreSQL |
| Frontend | React (Vite) |
| Scheduler | GitHub Actions (cron) |
| LLM | Any API of your choice |
| Notifications | Webhook → LINE Messaging API |
| Deployment | Docker / docker-compose, self-hosted |

## Architecture

```
GitHub Actions (cron, daily)
        │  POST /internal/generate
        ▼
Go Backend (Gin) ── AI API ── Validator ── Postgres
        │
        ▼
   Webhook → LINE
        ▲
        │
React Frontend (fetches /quiz/today, submits /quiz/answer)
```

See [`REQUIREMENT.md`](./REQUIREMENT.md) for the full requirements and design spec.

## Project Structure

```
quizme/
├── cmd/
│   └── server/main.go           # entrypoint
├── internal/
│   ├── handler/                 # HTTP handlers (Gin)
│   ├── service/                 # business logic (topic selection, generation flow)
│   ├── repository/              # DB access layer (Postgres)
│   ├── ai/                      # ai API client
│   └── model/                   # shared structs
├── migrations/                  # SQL migration files
├── frontend/                    # React app
├── .github/workflows/
│   └── daily-generate.yml       # scheduled trigger
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

## Getting Started

### Prerequisites

- Go 1.26+
- Node.js 18+ (for the frontend)
- Docker & docker-compose
- An AI API key
- A LINE Official Account with a Messaging API channel (Channel Access Token + your own `userId`)
  > Note: LINE Notify was discontinued on March 31, 2025 — this project uses the LINE Messaging API instead.

### Environment Variables

Create a `.env` file in the project root:

```env
DATABASE_URL=postgres://quizme:quizme@localhost:5432/quizme?sslmode=disable
AI_BASE_URL=https://api.openai.com/v1
AI_MODEL=gpt-4o-mini
AI_API_KEY=your_ai_key
INTERNAL_TOKEN=some_random_secret   # protects /internal/generate
WEBHOOK_URL=https://your-relay-or-line-endpoint
PORT=8080
CORS_ORIGIN=http://localhost:5173
```

### Run locally

Using the `Makefile`:

```bash
# Run both backend (:8080) and frontend (:5173 with HMR) concurrently
make dev

# Build the complete single-binary monolith (Go server embedding frontend SPA)
make build
./bin/quizme

# Run backend tests and frontend checks
make test

# Run linters (Go vet/gofmt + oxlint)
make lint

# Build monolith Docker image
make docker-build
```

Using Docker Compose directly:

```bash
# Start Postgres + run migrations + backend
docker compose up -d

# Health check
curl http://localhost:8080/health

# Migrations are run automatically by docker compose.
# To run migrations manually with golang-migrate:
migrate -path migrations -database "$DATABASE_URL" up
```

### Trigger a manual generation (for testing)

```bash
curl -X POST http://localhost:8080/internal/generate \
  -H "Authorization: Bearer $INTERNAL_TOKEN"
```

## Testing

```bash
# Unit tests (pure logic, no database)
go test ./...

# Race-detector run for concurrent service/AI paths
go test -race ./internal/service/ ./internal/ai/
```

Some integration tests require `TEST_DATABASE_URL` and will skip otherwise. **WARNING: those tests `TRUNCATE` all tables — never point `TEST_DATABASE_URL` at a database you care about.**

## API Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/quiz/today` | Fetch today's 5 questions |
| POST | `/quiz/answer` | Submit an answer, get correct/incorrect + explanation |
| GET | `/stats` | Per-topic accuracy stats |
| GET | `/topics` | List all topics |
| POST | `/topics` | Add a new topic |
| POST | `/internal/generate` | Trigger question generation (called by GitHub Actions) |

## Deployment

The backend, frontend, and Postgres run together via `docker-compose` on a self-hosted homelab. A GitHub Actions scheduled workflow calls `/internal/generate` daily at 06:00 ICT; the endpoint is exposed externally via a Cloudflare Tunnel (or similar) so GitHub Actions can reach it.

```yaml
# .github/workflows/daily-generate.yml (excerpt)
on:
  schedule:
    - cron: "0 23 * * *"   # 06:00 ICT
  workflow_dispatch: {}
```

## Operational Runbook

- **Token Rotation**: To rotate `INTERNAL_TOKEN`:
  1. Generate a new secret (`openssl rand -hex 24`).
  2. Update `INTERNAL_TOKEN` in `.env` and restart the backend service.
  3. Update `INTERNAL_TOKEN` in GitHub repository secrets (**Settings → Secrets and variables → Actions**).
- **Manual Migration**: To apply migrations outside Docker Compose:
  ```bash
  migrate -path migrations -database "$DATABASE_URL" up
  ```
- **Fallback Behavior**: If AI generation fails on a given day (e.g. API outage), `/quiz/today` automatically falls back to the most recent successful batch so the user always has a quiz. Review `generation_logs` table in Postgres for failure causes.

## Backlog (Post-MVP)

- **Multi-User Architecture**: Add `user_id` column to `user_answers` and `user_topic_stats` (repository queries are already isolated in anticipation) alongside OAuth/auth middleware.
- **Dynamic Difficulty Tuning**: Pass user proficiency level into AI prompt template to dynamically generate harder or easier questions.
- **Historical Archive**: Allow users to browse and review past completed batches and explanations.
- **Rich Analytics**: Visual graphs of accuracy progression over time.

## Roadmap

- [ ] Multi-user support with authentication
- [ ] Adaptive difficulty tuning
- [ ] Leaderboard
- [ ] Mobile-friendly UI improvements

## License
MIT — see [`LICENSE`](./LICENSE) for details.
