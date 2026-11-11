# 15 — Set, rotate and remove a runner's 1Password token from the command line

**Status:** ready-for-agent

**Blocked by:** None — can start immediately

Read first: `practices/go.md`, `practices/architecture.md`, `practices/testing.md`, ADRs 0073, 0074, 0181, the spec (Runners, Setup).

## What to build

- `nexul install runner … --1password-token-file F` stores the token in the runner unit's env.
- `nexul runner 1password <name>` reads a token from stdin (`--remove` removes it), writes it into the
  unit's env (the env file on systemd, the 0600 plist on launchd, the Windows service host's config) and
  restarts the runner. A token is never accepted as a plain flag value.
- `upgrade` keeps the token; `status` shows whether one is set, never its value.
- The token lands in the runner unit's env as `OP_SERVICE_ACCOUNT_TOKEN`.

## Acceptance criteria

- [ ] Unit tests with `internal/install`'s fakes for systemd, launchd and the Windows service host:
      set, rotate, remove, and `upgrade` keeping it.
- [ ] No argument list in any recorded exec call contains the token.
- [ ] `nexul help` lists the command.

Verify: `make lint`, `go test ./internal/install/...`.
