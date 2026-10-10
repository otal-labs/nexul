# 08 — Personal runner on macOS

**Status:** ready-for-agent

**Blocked by:** 03, 07, 20

Read first: `practices/go.md`, `practices/testing.md`, ADR 0073, `internal/install/host.go`, the spec
(Installing a personal runner).

## What to build

`nexul install computer` on macOS, after Linux (owner, 2026-10-10), by `curl -fsSL <site>/computer.sh | sudo sh -s -- <token>`:
installs for `$SUDO_USER` and refuses a root login with no `SUDO_USER` (ticket 20's rules), into
`~/Library/Application Support/nexul`, a LaunchDaemon `nexul-computer` in `/Library/LaunchDaemons` with
`UserName` set to that user and `KeepAlive`, no Docker step, then T3 Code run as that user (ticket 07).
`computer.sh` handles Darwin. The removal follows ticket 20's design: a root-owned cleanup a request file triggers (a LaunchDaemon with `WatchPaths` on macOS), no sudo rule.

## Acceptance criteria

- [ ] Unit tests for the plist (`UserName`, `KeepAlive`) and paths with the existing macOS fakes.
- [ ] The daemon runs as that user, starts at boot and survives logout; root login without `SUDO_USER` is
      refused.
- [ ] A run on a real Mac (a GitHub macOS runner is enough for install, connect and relay against a
      throwaway server), reported in the PR.
