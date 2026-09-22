# TASK-5 — Webhook notifier

## Objective
POST a summary payload to `WEBHOOK_URL` when a batch reaches terminal state. Sender is generic HTTP; LINE delivery is the receiving relay's job. Spec: `REQUIREMENT.md` §10 (payload), FR-10.

## Prerequisites
- TASK-4 done (batch terminal states exist; notifier hook/interface slot ready).

## Checklist

### Sender (`internal/service/notify.go` or `internal/notifier/`)
- [x] `POST {WEBHOOK_URL}` with JSON payload per §10 (batch date, status, question count or error summary — exact fields per §10).
- [x] Content-Type: application/json; timeout applied (a few seconds, not minutes).
- [x] Fire-and-forget from the generate flow's perspective: notification failure is logged (can reuse `generation_logs` error field or standard logger), NEVER fails the batch flow.
- [x] Hook into both terminal states: success and failed-after-retries.

### LINE path (documentation + verification, not code)
- [x] Confirm receiver side plan: n8n/relay receives webhook → calls LINE Messaging API (OA + Channel Access Token; LINE Notify is deprecated — do not build on it).
- [x] Document required user-side setup in README/plan notes: LINE OA creation, Messaging API channel, token/secret/userId to be wired in n8n (user-owned, external prerequisite from TASK-1).

### Verification
- [x] Local test receiver (webhook.site, `nc -l`, or tiny handler) captures payload on a successful generate run → fields match §10.
- [x] Same on failed generate run (AI endpoint down) → failure payload delivered.
- [x] Point `WEBHOOK_URL` at an unreachable address → generate flow still completes successfully; failure visible in logs only.
- [x] Payload contains no secrets.

## Done when
- [x] Test endpoint receives correct payload on success AND failure paths.
- [x] Notification failure demonstrably does not affect batch outcome.
- [x] LINE Messaging API path documented for the receiver relay.
