# 10 — Desktop installs stay on localhost

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 03

## What to build

macOS and Windows installs are marked local (an optional server setting the installer writes, never required). First run there skips the domain step, sets the instance URL to `http://localhost:<port>`, and goes straight to the GitHub step with the code.

## Acceptance criteria

- [ ] Desktop install reaches GitHub sign-in on localhost with the code, no domain step
- [ ] Server installs never show the local shortcut

## Read first

`practices/go.md`, `internal/install/desktop.go`, ADR 0020 (no required env vars).
