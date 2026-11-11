# 10 — The remaining scope, build and docs calls

**Status:** resolved
**Type:** grilling
**Blocked by:** None — can start immediately

## Question

The small calls the other tickets left: edition, a host restart loop's read cost, the guide, tests, the
ADRs, who hears about a rejected token, and whether `nexul install` asks for tokens.

## Answer

Decided 2026-11-12.

- **Edition:** open source (AGPL), where the README already lists it.
- **Batching on hosts:** in v1, the supervisor collects the hellos that arrive within about a second,
  per workspace, and resolves them in one call, so a host restart costs one read per workspace.
- **Guide:** one new "External secrets" page (a service account per runner, host and server,
  references, the token commands, the check, errors), with a one-line link from the stack,
  automation, connector and runner pages.
- **Tests:** against an in-memory fake behind the `Resolver` interface that can fail in every typed
  way. There is no real-token test; ticket 07 measured the real behaviour.
- **ADRs:** three. Secrets are resolved where they are used, with credentials that never reach
  Nexul. An automations host holds one token per workspace, in its supervisor. The bootstrap secret
  stays local.
- **Rejected token:** once, when a state turns `token rejected`, an inbox notification goes to the
  people who can Manage that machine, or to the Owner for the server.
- **Install:** an interactive `nexul install` offers an optional prompt for the server's and the
  bundled runner's tokens (paste one, or Enter to skip). Without a terminal it never prompts, so
  nothing is required at install (ADR 0020). The `--1password-token-file` flags and the
  `nexul … 1password` commands work as decided.
