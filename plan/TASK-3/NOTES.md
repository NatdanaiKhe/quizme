# TASK-3 Notes

## Blocker

Real API smoke test was not run because `AI_BASE_URL`, `AI_API_KEY`, and `AI_MODEL` are not configured in the local `.env`.

```bash
set -a && source .env && set +a
if [ -n "$AI_BASE_URL" ] && [ -n "$AI_API_KEY" ] && [ -n "$AI_MODEL" ]; then
  AI_SMOKE_TEST=true go test ./internal/ai/ -run TestRealSmoke -v
fi
```

The above check failed, so `TestRealSmoke` skipped. To unblock:
1. Add real values to `.env` (gitignored):
   - `AI_BASE_URL=https://api.openai.com/v1` (or OpenRouter/Groq/local equivalent)
   - `AI_MODEL=gpt-4o-mini` (or chosen model)
   - `AI_API_KEY=sk-...`
2. Re-run: `AI_SMOKE_TEST=true go test ./internal/ai/ -run TestRealSmoke -v`
3. Record observed `tokens_used` in `plan/PROGRESS.md`.

All unit tests pass without env setup. The smoke test is intentionally env-guarded so CI/`go test ./...` stays green.
