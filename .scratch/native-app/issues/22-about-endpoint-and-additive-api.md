# 22 — Public server version and the additive API rule

**Type:** implementation
**Status:** done
**Blocked by:** None — can start immediately
**Decided in:** ticket 09

## What to build

Add a public, unauthenticated `GET /api/about` returning `{"product": "nexul",
"version": "<version.Version>"}` and nothing else. Write the ADR "The HTTP API
only grows" (next free number): a route or JSON field that a released app may
call is never removed or renamed; a replacement ships beside it. Link the ADR
from `practices/architecture.md` where the HTTP gateway is described.

## Acceptance criteria

- [ ] `/api/about` answers without a token and carries only product and version
- [ ] It appears in the OpenAPI spec like the other routes
- [ ] The ADR is in `docs/adr/` and the practices file points at it

## Surfaces

- HTTP gateway route (public)
- Docs: new ADR, `practices/architecture.md`

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, ticket 09.

## Verification

`go test ./...`, `make lint`, `make coverage`.

## Files likely touched

- `server/cmd/routes.go`, `server/cmd/version_http.go`
- `docs/adr/`
- `practices/architecture.md`

**Size:** S
