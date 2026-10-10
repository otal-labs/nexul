# 05 — Computer facts

**Status:** resolved

**Blocked by:** 03

Read first: `practices/go.md`, `practices/architecture.md` (sections 2, 3), `practices/react-guide.md`
(F1 to F7, The live topic contract), `practices/design-language.md` (shared core, web app),
`practices/mcp.md` (section 7), the spec (Facts).

## What to build

- Migration: `pairing_computers.facts TEXT NOT NULL DEFAULT '{}'` and `facts_at INTEGER`.
- Runner report: OS, architecture, hostname, runner version, T3 Code state, install kind, version, port,
  the `cloudflared` version when installed, git `user.name` and `user.email`, free disk in the home folder.
- Server snapshot after pairing and each report: providers with versions, models and whether each is signed
  in, projects with folders, through the relay.
- Facts are owner-only: read on the owner's computer row and by agents acting for the owner through
  `computer_list`. Read computer activity does not reach them (spec, Access and privacy, rule 5).
- `computer.facts_changed` (outbox, members-only, owner audience) only when the stored facts change.
- The computer row's details show them; `computer_list` returns them.

## Acceptance criteria

- [x] An unchanged report writes nothing and publishes nothing (a guard test counts writes).
- [x] `TestComputerFacts_NeverReachAnotherUser` over HTTP, MCP and the live socket, for a workspace Owner and
      for a holder of Read computer activity too, once that permission exists (ticket 15).
- [x] The row's details at 768, 1024 and 1440px, screenshots in the PR.
- [x] `make event-schemas` and `make live-topics` are clean.

## Comments

Built: migration `0094_computer_facts.sql`; the runner's report (`internal/runner/t3code.go`, `disk_unix.go`,
`disk_windows.go`); `HandleFactsReported`, `recordFacts` and `readT3Snapshot` in `internal/pairing/computer_usecase.go`;
`computer.facts_changed`; `facts` and `facts_at` on the computer row and `computer_list`; `ComputerFactRows` in the row's
details.

- The runner reports `install` as `service` (the runtime file's `serviceManaged`), `desktop_app` (the `t3` it found is
  `$T3CODE_HOME/bin/t3`) or `command_line`, and the version of `cloudflared` from `cloudflared --version`; git's name and
  email come from `git config --global` as the person the runner runs as.
- Free disk is stored in whole GiB, cut on the server: the exact byte count moved between every two reports, so an
  unchanged computer wrote on each one. Checked live: a runner reconnect with nothing changed wrote no row.
- Sign-in is `harness.Provider.SignIn`, read from T3 Code's provider `auth.status` (`signed_in`, `signed_out`,
  `unknown`). The snapshot keeps slug and name per model, not their options.
- When T3 Code is not answering, or the computer has no live session, the last providers and projects read stay; a
  failed listing keeps the runner's report and is logged, never retried. `POST .../pair` through the runner reads the
  snapshot straight after pairing.
- `computer.facts_changed` carries `computer_id`, `user_id` and `facts_at` only, so no fact sits in the outbox, a dead
  letter or a frame; the web refetches the computer list and that computer's provider picks. Its audience is `ownFrame`.
- T3 Code's `restarted_at` and `restart_error` (ticket 10) are stored with the rest of `t3`; the row shows
  "Restart failed: <why>", else when the runner last restarted it.
- `runOnce` now waits for its heartbeat and facts loops after cancelling them: a facts report still running `git` or
  `cloudflared` used to outlive its connection and log into a closed test's buffer.
- The row hides the relay address (`<id>.nexul-computer.invalid`) and leads with T3 Code's state, port, version and
  install; a closed desktop app reads "Open T3 Code".
- For 06: the row's facts come from `GET /api/pairing/computers`; the dialog can read `facts.t3.state` to say T3 Code is
  missing or not running before pairing is tried. The web `Computer` model still has no `runner` or `pair_error`.
- For 15: `TestComputerFacts_NeverReachAnotherUser` (`server/cmd/computer_facts_privacy_test.go`) gains a holder of
  Read computer activity once the permission exists.
- For 18 and 19: facts are on `Computer` and every computer read is the owner's today; a grantee's view of a shared
  computer must drop `facts` and `facts_at`.

