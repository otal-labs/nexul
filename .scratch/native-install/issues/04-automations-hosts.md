# 04 — Named automations hosts and automation placement

**Status:** resolved
**Type:** task
**Blocked by:** 02 (reuses `internal/platform/hostcred` and its patterns)

## Scope

- Server: automations host records, enrollment codes, credentials with
  tombstones, bundled `instance` code file, the `/api/automation-hosts`
  endpoints, `host_create`/`host_delete` kind `automations` (extend the dispatch ticket 02 built) and automations hosts in `machine_list`, `automations.host_id` (null = the
  `instance` host) settable through the existing update endpoint and MCP tool,
  and the host-scoped automation token accepted by the automation auth path and
  `/ws/automations` only while placement and credential hold. Remove the
  tokens-file seeding (`host_tokens.go`, `NEXUL_AUTOMATIONS_HOST_TOKENS_PATH`).
- Automations host (`automations/`): new env contract, poll
  `/self/assignments` instead of the tokens file, start and stop workers as
  placements change, on the removed refusal run `$NEXUL_CTL uninstall
  automations <name> --detach` and exit. `build:binaries` script compiling the
  five targets with `bun build --compile` including the worker entrypoint;
  prove a compiled binary starts a worker.
- Web: Automations hosts list with Add (name → one-liners) and Remove, and a
  host picker on the automation page. Mirror the Runners page; no new visual
  design.

## Acceptance

- Go handler tests with real SQLite for enroll, assignments, placement moves,
  host-scoped token accepted and refused, removal.
- `automations` typecheck and tests green, including assignment changes and the
  removal path; web lint/typecheck/test green; Go gates green.
