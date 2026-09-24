# 17 — Cloudflare Access client and connector permissions

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** ticket 12, ticket 15, ticket 16

## What to build

Add a Cloudflare Access client beside the existing tunnel client: create and delete a self-hosted Access application for one hostname, a service-auth policy admitting one service token, and create, rotate, and delete that token (one per instance, stored encrypted). Add the two Access permissions to the Cloudflare connector's requested and verified list, with a verification that never mints a real token. Detect when Zero Trust is not enabled and report it as a distinct, explainable error.

## Acceptance criteria

- [ ] Access apps, policies, and the instance service token can be created and deleted against a recorded Cloudflare fixture
- [ ] The connector asks for and verifies Access: Apps and Policies Edit and Access: Service Tokens Edit
- [ ] A Zero Trust-disabled account returns a specific error the UI can explain

## Surfaces

- Connector settings show the new permissions
- Docs: the Cloudflare connector page lists them

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`.

## Files likely touched

- `internal/dns/cloudflare/ (new access.go)`
- `internal/connectors/registry.go, verify.go`
- `website/src/content/docs/docs/guide/topology-and-dns.md`

**Size:** M
