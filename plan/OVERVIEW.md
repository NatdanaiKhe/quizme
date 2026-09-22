# Quizme — Task Overview

Tasks (ordered by dependency):

- [x] **TASK-1** Project skeleton & Phase 0 setup — repo layout per §11, env/config, Docker dev environment, Postgres + migrations + seed topics (§6)
- [x] **TASK-2** Repository layer — DB access for all tables (pgx/sqlx), basic queries tested
- [x] **TASK-3** AI client + validator — prompt template (§9), response parsing, validation per FR-2
- [x] **TASK-4** Service layer — adaptive topic selection 70/30 (§8), generation flow with retries + fallback + logs (§10)
- [ ] **TASK-5** Webhook notifier — POST summary payload after batch completes (§10, LINE Messaging API)
- [ ] **TASK-6** API layer — Gin handlers for all endpoints (§7), internal auth token guard
- [ ] **TASK-7** Unit tests — topic selection, validator, generation retry logic
- [ ] **TASK-8** Frontend (React/Vite) — quiz page, answer submission + feedback, stats page, error states
- [ ] **TASK-9** Deployment — Dockerfiles (multi-stage), docker-compose, reverse proxy + Cloudflare Tunnel, `.env` secrets
- [ ] **TASK-10** Scheduler — GitHub Actions cron workflow (`0 23 * * *`), secrets, manual `workflow_dispatch` test
- [ ] **TASK-11** E2E validation — 3–5 day run, LINE delivery confirmed, adaptive behavior verified, prompt tuning, backlog

External prerequisites (owner: user, start early):
- [ ] LINE OA + Messaging API channel (token/secret/userId)
- [ ] AI API key
- [ ] Homelab Postgres + outside-reachable URL
