# 08 — Personal runner on macOS

**Status:** ready-for-agent

**Blocked by:** 03, 07

Read first: `practices/go.md`, `practices/testing.md`, ADR 0073, `internal/install/host.go`, the spec
(Installing a personal runner).

## What to build

`nexul install computer` on macOS: as the user (refuse root), into `~/Library/Application Support/nexul`,
a LaunchAgent `nexul-computer`, no Docker step, then T3 Code (ticket 07). `computer.sh` handles Darwin.

## Acceptance criteria

- [ ] Unit tests for the plist and paths with the existing macOS fakes.
- [ ] A run on a real Mac (a GitHub macOS runner is enough for install, connect and relay against a
      throwaway server), reported in the PR.
