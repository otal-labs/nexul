# 23 — Push inbox notifications to phones

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** tickets 02, 13

## What to build

Add a nullable `push_token` to `sessions`. `PUT /api/auth/sessions/current/push-token`
(session auth only) sets or clears it for the calling session. When
notifications are created for a user, send one Expo push message per phone
session of that user with a token: title "Nexul", body "New activity in
<workspace name>", data `{notification_id, host}` where host is the instance
URL. Post to `https://exp.host/--/api/v2/push/send` in batches of up to 100,
from the notification event consumer, never inline in the request that made
the notification. Log failures with `slog`; a `DeviceNotRegistered` result
clears that session's token. No retries beyond the consumer's own delivery
semantics. The HTTP client is injectable so tests never reach Expo.

## Acceptance criteria

- [ ] A session can set and clear its push token; a PAT gets 403
- [ ] Creating a notification for a user with one phone session posts exactly one message with the agreed payload (verified against a fake Expo endpoint)
- [ ] A browser session never receives a push
- [ ] `DeviceNotRegistered` clears the token
- [ ] Signing a session out stops its pushes (the row is gone)

## Surfaces

- HTTP route on the sessions API
- Events: consumes the notification events
- No MCP tool

## Read first

`practices/go.md`, `practices/architecture.md` (sections 2 to 6, the event bus), `practices/testing.md`, tickets 02 and 13, `docs/adr/0081-sessions-are-stored-per-device.md`.

## Verification

`go test ./...`, `make lint`, `make coverage`, `make sqlc-check`.

## Files likely touched

- `internal/auth/sessions.go`, `internal/auth/handler.go`
- `internal/workspace/` notification consumer or a new small push sender package
- `internal/platform/storage/migrations/`, `queries/sessions.sql`
- `server/cmd/services.go`

**Size:** M
