# 03 — How a runner gets its 1Password token and tells Nexul it can resolve

**Status:** resolved
**Type:** grilling
**Blocked by:** 01, 02

## Question

How does a runner's token get onto its machine (a `nexul` install or runner flag, the runner's env file,
rotation without reinstalling) on systemd, launchd and the Windows service host? How does the runner tell
Nexul which managers it can resolve, and what do Nexul's stack page and `stack_deploy` do when a stack
has references and the runner that deploys it cannot resolve them?

## Answer

Decided 2026-11-12.

- **Who needs a token:** the bundled runner and named runners. Personal (computer) runners are never
  picked for deploys (ADR 0146), so they need none.
- **Setup:** `nexul install runner … --1password-token-file F` at install, and
  `nexul runner 1password <name>` to add, rotate or remove the token later. It reads the token from
  stdin, writes it into the unit's env (the env file on systemd, the plist on launchd, the Windows
  service host's config) and restarts the runner. The token is never a plain flag value, since that
  would land in shell history and `ps`, and `upgrade` keeps it.
- **Bundled runner:** its own token, set the same way, scoped to the stack vaults and separate from the
  server's token (ticket 06).
- **Check at start:** a runner with a token opens its 1Password client when it starts (about 2.6 s, no
  read; ticket 07) and reports `ready` or `token rejected` with its version, os and arch on connect.
  Deploys reuse that client. This replaces ticket 04's "created on the first deploy that needs it".
- **Dispatch:** a deploy with references goes only to a runner on its machine that reported `ready`.
  If such a runner is offline, the deploy queues for it as work already does. If no runner on the
  machine has ever reported `ready`, Deploy is refused straight away with "no runner on <machine> can
  read 1Password references".
- **Display:** the machine's runner list (web and MCP `machine_list`) shows a "1Password" tag, or
  "token rejected". A stack with references whose machine has no ready runner shows a warning in its
  environment section.
