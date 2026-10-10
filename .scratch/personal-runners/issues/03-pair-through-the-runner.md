# 03 — Pair, and re-pair before the session ends, through the runner

**Status:** ready-for-agent

**Blocked by:** 02

Read first: `practices/go.md`, `practices/architecture.md` (sections 2, 3, 5), `practices/mcp.md`,
`practices/testing.md`, ADRs 0054, 0063, 0113, 0142, the spec (Pairing over the relay, Finding T3 Code).

## What to build

- Runner: the T3 Code detection in the spec (command, port, probe, state), sent in a `facts` frame on
  connect and every 6 hours with at least the T3 Code state, port and version (ticket 05 adds the rest).
  `t3_pair_token_request {id}` runs `t3 auth pairing create --json --ttl 5m --label Nexul` and answers
  `t3_pair_token {id, token}` or an error. The token is never logged.
- Server: `runner.facts_reported` on the bus (ephemeral, members-only, not bridged). A pairing consumer
  pairs the computer when it is unpaired, or its session ends within 7 days, and its T3 Code is answering:
  it asks for a token through a runner seam and calls the existing `pair` with the relay address
  (`internal/pairing/usecase.go:219`). `Computer.Session()` returns the relay address for a computer with a
  runner.
- `POST /api/pairing/computers/{id}/pair` and `computer_pair` take only the id and pair now through the
  runner (the token and URL fields stay accepted only for old computers until ticket 14).
- A development recipe: run a personal runner against the debug stack or a local server, in the
  contributing docs, replacing Pair by URL in local and end-to-end recipes. Point `T3CODE_HOME` at the
  throwaway T3 Code's base directory: a runner with a custom home never falls back to 3773, so once the
  throwaway stops its dials are refused instead of reaching the developer's own T3 Code.

## Acceptance criteria

- [ ] Integration test: a fake T3 behind a personal runner gets paired with no human input, lands on the
      right harness kind (protocol 1 and protocol 2 fakes), and the bearer is stored encrypted.
- [ ] A session ending in 6 days is re-paired on the next facts report; one ending in 8 days is not.
- [ ] A protocol-2 computer whose T3 Code answers protocol 1 is still refused (ADR 0113), through the relay.
- [ ] A failed mint or exchange leaves the computer as it was and shows the reason on the row.
- [ ] Manual check on Linux with a real T3 Code: pairing completes, presence shows Connected, and an
      `@Agent` mention runs a turn through the relay.
- [ ] `TestResolveTarget_OfflineComputerIsNeverSwapped`: with two computers, a run aimed at the offline
      one fails at once with "<computer> is offline" and its reason, and nothing runs on the other.
- [ ] The contributing docs carry the development recipe.
