# 03 — Pair, and re-pair before the session ends, through the runner

**Status:** resolved

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

- [x] Integration test: a fake T3 behind a personal runner gets paired with no human input, lands on the
      right harness kind (protocol 1 and protocol 2 fakes), and the bearer is stored encrypted.
- [x] A session ending in 6 days is re-paired on the next facts report; one ending in 8 days is not.
- [x] A protocol-2 computer whose T3 Code answers protocol 1 is still refused (ADR 0113), through the relay.
- [x] A failed mint or exchange leaves the computer as it was and shows the reason on the row.
- [x] Manual check on Linux with a real T3 Code: pairing completes, presence shows Connected, and an
      `@Agent` mention runs a turn through the relay.
- [x] `TestResolveTarget_OfflineComputerIsNeverSwapped`: with two computers, a run aimed at the offline
      one fails at once with "<computer> is offline" and its reason, and nothing runs on the other.
- [x] The contributing docs carry the development recipe.

## Comments

Built: `internal/runner/t3code.go` (both ends: T3 Code detection, the `facts` frame, `t3_pair_token_request` /
`t3_pair_token`, `Handler.PairingToken`), `runner.facts_reported` (ephemeral, members-only, not bridged),
`HandleFactsReported` and `pairThroughRunner` in `internal/pairing/computer_usecase.go`. No migration.

- The relay address is stored: pairing through the runner saves `http://<computer id>.nexul-computer.invalid` as the
  computer's `server_url`, so every caller of `Session()` and the agent's version probe reach it with no lookup;
  `Session()` falls back to it only for a computer that has not paired yet. The port is not on the computer until
  facts are stored (05). `pairing.ComputerHostSuffix` is the one copy of the suffix; `server/cmd` reads it.
- A failed pairing's reason is kept in memory on the pairing service and returned as `pair_error` on
  `GET /api/pairing/computers` and `computer_list`. It is not persisted and pushes no live frame: a server restart
  drops it, and the runner's reconnect reports facts, which tries again and sets it anew. The dialog (06) needs a
  live signal for a failure; today the web refetches on `runner.personal_changed` and `computer.paired` only.
- The facts frame carries the hostname now, so a computer still unnamed when the first report arrives pairs under it.
- `POST /api/pairing/computers/{id}/pair` with no body, and `computer_pair` with only `id`, pair through the runner;
  a computer with a tunnel keeps pairing with a token there until 14. `computer_pair` with `id` and `name` still
  only renames.
- `ReasonOffline` now reads "<computer> is offline: <why>." on every surface (chat reply, play run, MCP); the why
  is "it isn't connected to Nexul" when the runner is down, else "T3 Code isn't answering there". Auto plays still
  requeue on it.
- The runner mints with `t3` on `PATH`, else `$T3CODE_HOME/bin/t3`, else `~/.local/bin/t3`, with `T3CODE_HOME` set
  to its own home, and reads T3 Code's version from the descriptor (`t3rpc.Describe`).
- Manual check on Linux (nightly 0.0.46-nightly.20261010.2935 under a custom `T3CODE_HOME` on :47190, server on
  :8310): within a second of the runner connecting the computer read `kind: t3code-v2`, named after the hostname,
  `server_url` the runner address; `GET /api/pairing/presence` answered `connected` with a live socket open; one
  `@Agent reply with the word ok` was answered `ok` 6 seconds later, every byte over relayed streams.
- For 04+: the protocol-refusal text from the T3 clients names the host, which for a runner computer is
  `<id>.nexul-computer.invalid`; a follow-up could name the computer there. Re-pairing is checked only on a facts
  report (connect and every 6 hours), so a server down across the 7-day window still re-pairs at the runner's
  reconnect.
