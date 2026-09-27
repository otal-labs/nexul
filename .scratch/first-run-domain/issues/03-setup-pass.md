# 03 — Setup pass: unlock with the code

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 02

## What to build

`POST /api/setup/unlock {code}` (public, rate-limited) trades a valid setup code for a one-hour bearer. `RequireAuth` accepts it only while no user exists and only on the allowlist in the spec; everywhere else, and after the first user exists, it is a 401. `/api/auth/bootstrap` and `/bootstrap/verify` require the pass. The Cloudflare connector save records `connected_by: setup`. Web: before `configured`, the app shows "Enter the setup code" (prefilled from `#code=` and cleared from the address bar), stores the pass in its own persisted store, and sends it as the bearer.

## Acceptance criteria

- [ ] Wrong code: 400 with `invalid_code`; repeated failures are throttled
- [ ] A pass reaches the allowlisted routes and nothing else
- [ ] Bootstrap without a pass is refused
- [ ] Every pass stops working once a user exists
- [ ] The code screen works at 320/375/414/768px

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/react-guide.md` (F1–F7), `practices/testing.md`, `server/cmd/routes.go`, `internal/auth/middleware.go`, `internal/auth/service.go` (`bootstrapReplaceable`).
