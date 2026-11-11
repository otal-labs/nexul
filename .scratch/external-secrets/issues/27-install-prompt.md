# 27 — An interactive install offers to take the 1Password tokens

**Status:** ready-for-agent

**Blocked by:** 15, 21

Read first: `practices/go.md`, `practices/architecture.md`, `practices/testing.md`, ADR 0020, the spec (Install).

## What to build

An interactive `nexul install` asks once for the server's token and once for the bundled runner's
(paste, or Enter to skip). It never prompts without a terminal. The closing summary names the
`nexul … 1password` commands in one line.

## Acceptance criteria

- [ ] Install tests: a terminal with a pasted token stores it, Enter skips, and no terminal never prompts.
- [ ] The pasted token never appears in the install's output or log.

Verify: `make lint`, `go test ./internal/install/...`.
