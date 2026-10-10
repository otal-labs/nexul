# 05 — Computer facts

**Status:** ready-for-agent

**Blocked by:** 03

Read first: `practices/go.md`, `practices/architecture.md` (sections 2, 3), `practices/react-guide.md`
(F1 to F7, The live topic contract), `practices/design-language.md` (shared core, web app),
`practices/mcp.md` (section 7), the spec (Facts).

## What to build

- Migration: `pairing_computers.facts TEXT NOT NULL DEFAULT '{}'` and `facts_at INTEGER`.
- Runner report: OS, architecture, hostname, runner version, T3 Code state, install kind, version, port,
  git `user.name` and `user.email`, free disk in the home folder.
- Server snapshot after pairing and each report: providers with versions and models, projects with
  folders, through the relay.
- `computer.facts_changed` (outbox, members-only, owner audience) only when the stored facts change.
- The computer row's details show them; `computer_list` returns them.

## Acceptance criteria

- [ ] An unchanged report writes nothing and publishes nothing (a guard test counts writes).
- [ ] `TestComputerFacts_NeverReachAnotherUser` over HTTP, MCP and the live socket.
- [ ] The row's details at 768, 1024 and 1440px, screenshots in the PR.
- [ ] `make event-schemas` and `make live-topics` are clean.
