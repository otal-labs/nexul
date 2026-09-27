# 01 — Install on port 5123 and say where HTTPS comes from

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately

## What to build

`nexul install` defaults the web port to 5123 instead of 80 (a fresh install; a re-run keeps the stored port). The summary drops "point a domain at this server and put HTTPS in front of it" and says instead: open the setup page, enter the setup code, and the domain, HTTPS and ports 80/443 are set up from there by the reverse proxy or a tunnel, not by the installer. `ProvisionReverseProxy`'s `http://localhost:80/` health default goes with it.

## Acceptance criteria

- [ ] A fresh non-interactive install listens on 5123 and prints `http://<ip>:5123/`
- [ ] An existing install on 80 keeps 80 on re-run and upgrade
- [ ] The summary names where 80/443 get set up
- [ ] Install tests updated; the install guide shows 5123

## Read first

`practices/go.md`, `practices/testing.md`, `internal/install/install.go` (`defaultPort`, `printSummary`, `siteURL`).
