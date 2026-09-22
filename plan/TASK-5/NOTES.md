# TASK-5 — Webhook Notifier Notes

## What changed

- Added `internal/service/notify.go` with `WebhookNotifier`, a stdlib-only HTTP sender that implements the existing `Notifier` seam.
- Added `internal/service/notify_test.go` with `httptest` coverage for success/failure payloads, context detach, timeout, non-2xx, unreachable host, empty URL, default timeout, and service-layer seam integration.
- Updated `.env.example` to clarify that `WEBHOOK_URL` feeds an n8n/LINE relay.
- Marked TASK-5 done in `plan/TASK.md`, `plan/OVERVIEW.md`, and `plan/TASK-5/TODO.md`.

## Payload contract

Success (`status: success`):

```json
{
  "batch_date": "2026-09-22",
  "status": "success",
  "question_count": 5,
  "topics": ["Frontend", "Backend", "Infrastructure"]
}
```

Failure (`status: failed`):

```json
{
  "batch_date": "2026-09-22",
  "status": "failed",
  "error": "AI API: connection refused"
}
```

Rules enforced by `webhookPayload`:

- `batch_date` is `YYYY-MM-DD` from `BatchDate.Format("2006-01-02")`.
- `question_count` and `topics` are omitted on failure.
- `error` is omitted on success.
- No API keys, `WEBHOOK_URL`, tokens, question content, or `correct_option` ever appear.

## Behavior notes

- `Notify` calls `context.WithoutCancel(ctx)` first so a Gin request context expiring does not kill an in-flight POST.
- Empty `WebhookURL` is a no-op (returns `nil`).
- Non-2xx status codes and transport errors return an error and log the URL host only (no query-string secrets).
- Default timeout is 5s when zero/negative.
- The sender is provider-agnostic; LINE delivery is handled by the receiving relay (n8n).

## LINE Messaging API receiver path

This codebase does **not** contain a LINE client. The intended receiver is an n8n workflow (or similar relay):

1. Create a LINE Official Account.
2. Enable the **Messaging API** channel for the OA.
3. In the LINE Developers Console, generate a **Channel Access Token** (long-lived).
4. Add the OA as a friend and obtain your `userId` (e.g., from a webhook event or the LINE console).
5. In n8n: receive the §10 webhook from this backend, then call:
   ```
   POST https://api.line.me/v2/bot/message/push
   Authorization: Bearer {CHANNEL_ACCESS_TOKEN}
   Content-Type: application/json

   {
     "to": "{USER_ID}",
     "messages": [
       {
         "type": "text",
         "text": "Quiz ready: 2026-09-22 (status: success, 5 questions)"
       }
     ]
   }
   ```

> **Do not use LINE Notify.** LINE Notify was discontinued on March 31, 2025.

## Verification

Automated verification (`go test ./internal/service/ -run Notif -v`) covers:

- Exact success payload fields and `Content-Type: application/json`.
- Exact failure payload fields with `error` present and `question_count`/`topics` absent.
- Non-2xx response returns error and logs host + status.
- Timeout fires quickly without hanging.
- Unreachable host returns error, no panic.
- Empty URL is a no-op.
- Canceled context still completes POST (proves `WithoutCancel`).
- Default timeout is 5s.
- Seam integration with `Service.Generate` for both success and failure paths (skipped without `TEST_DATABASE_URL`).

### Manual end-to-end verification status

Full manual verification (triggering actual generation and receiving payloads on a live receiver) is **blocked on external prerequisites**:

- `AI_API_KEY` is not set in `.env`.
- `WEBHOOK_URL` is not set in `.env`.
- `TEST_DATABASE_URL` is not set, so the DB-backed seam integration test skips.

Once those are available:

1. Set `WEBHOOK_URL` to a local receiver (`nc -l 9999`) or webhook.site URL.
2. Set `AI_API_KEY`, `AI_BASE_URL`, and `AI_MODEL`.
3. Start Postgres and run migrations.
4. Trigger generation (TASK-6 endpoint; for now, call `Service.Generate` directly from a small main or test).
5. Capture success payload and confirm fields match §10.
6. Point `AI_BASE_URL` at a dead host to force failure; capture failure payload.
7. Set `WEBHOOK_URL` to an unreachable address; confirm generation still completes and only logs show the failure.

## Note for TASK-6

When wiring `Service` in `cmd/server/main.go`, construct the notifier with:

```go
notifier := service.NewWebhookNotifier(cfg.WebhookURL, 5*time.Second)
svc := service.New(repo, aiClient, notifier)
```

If `cfg.WebhookURL` is empty, the notifier is a no-op and generation works without notifications.
