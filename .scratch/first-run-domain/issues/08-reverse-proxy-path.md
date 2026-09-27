# 08 — Reverse-proxy path

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 04, 07

## What to build

The proxy choice asks for the domain, shows the server's public address and the A/AAAA record to create (created automatically when a Cloudflare token is connected), and waits until the name resolves to this server. Then it deploys the gateway Traefik on 80/443 with Let's Encrypt and a route for the domain to Nexul (per 07), and finishes through 04 once https answers.

## Acceptance criteria

- [ ] Fresh Ubuntu server: domain resolves, Traefik gets a real certificate, the domain reaches Nexul
- [ ] A domain that does not resolve here yet says so and keeps checking
- [ ] Ports 80/443 already taken: named, with what holds them

## Read first

`practices/go.md`, `practices/react-guide.md`, the answer in 07, `internal/dns/gateway*.go`, `internal/runner/executor.go` (`runArgs`).
