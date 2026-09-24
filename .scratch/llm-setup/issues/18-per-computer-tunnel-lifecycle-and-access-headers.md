# 18 — Per-computer tunnel lifecycle and Access headers

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 17
**Decided in:** ticket 15, ticket 16

## What to build

Give each paired computer its own tunnel: create the tunnel, create its Access app before the DNS record so the hostname is never public, route `<computer-slug>-<8 random>.<instance domain>` to the local harness port, fetch the connector token, and read status. Tear down in order: rotate the token, delete the tunnel, the DNS record, then the Access app. Store the Access app id beside the tunnel. Wrap the T3 client's HTTP client so the Access headers go only to computer-tunnel hostnames, never to URL-paired machines.

## Acceptance criteria

- [ ] Creating and removing a computer's tunnel leaves nothing behind in Cloudflare (tunnel, CNAME, route, Access app)
- [ ] Status reports tunnel health and a probe of the harness through the hostname
- [ ] Access headers reach tunnel hostnames on HTTP and WebSocket dials and never reach other URLs

## Surfaces

- Events for tunnel created and removed (catalog row plus outbox write)
- Reverse state: removing the computer tears the tunnel down

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`.

## Files likely touched

- `internal/pairing/`
- `internal/dns/cloudflare/tunnel.go`
- `internal/t3client/`
- `server/cmd/services.go`
- `internal/platform/storage/queries/`

**Size:** M
