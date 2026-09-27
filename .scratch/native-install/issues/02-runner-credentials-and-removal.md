# 02 — Runner enrollment, per-runner credentials, removal

**Status:** ready-for-agent
**Type:** task
**Blocked by:** None — can start immediately

## Scope

Everything under "Enrollment and credentials" and the runner half of "HTTP and
MCP" in the spec, plus the runner process side.

- `internal/platform/hostcred`: mint code, mint credential, hash, constant-time
  compare. Reused by ticket 05.
- Schema: runner enrollment codes and runner credentials (with revoked
  tombstones). No production data, so change the schema the way the repo does
  today (check `internal/platform/storage/migrations/` and recent PRs) and add
  sqlc queries.
- Remove the shared runner secret everywhere: `instance_settings.runner_secret`,
  `NEXUL_RUNNER_SECRET`, the `runner-secret` file, `GET /api/runners/install`.
- Enrollment and removal endpoints and MCP tools exactly as the spec's
  contract, with the install commands rendered by the use case (one place for
  web and MCP). Bundled `instance` code files at boot.
- `/ws/runner` Bearer auth; `runner_removed` refusal; the download route on the
  runner credential. Runner id, name and machine come from the record.
- Runner process: new env contract (`NEXUL_SERVER_URL`,
  `NEXUL_CREDENTIAL_FILE`, `NEXUL_RUNNER_NAME`, `NEXUL_STACK_ROOT`,
  `NEXUL_GIT_TOKEN`, `NEXUL_CTL`); a new uninstall frame and the
  `runner_removed` refusal both run `$NEXUL_CTL uninstall runner <name>
  --detach` and exit; the upgrade executor runs `$NEXUL_CTL upgrade --detach
  --version <v>` instead of calling `systemd-run` itself. Drop the
  `/.dockerenv` container path: runners are never containers now.
- Upgrade dispatch targets the connected runner named `instance`.
- Web: Add runner dialog asks name (+ optional machine, git token), calls the
  enrollments endpoint, shows the unix and Windows one-liners. Runners list
  gets Remove with an "are you sure?" confirm. Mirror the existing Runners page
  styling; no new visual design.

## Acceptance

- Handler tests with real SQLite: enroll once, reuse refused, expired refused,
  name mismatch refused; credential connects; removed runner refused with
  `runner_removed`; removing one runner leaves another dispatchable; uninstall
  frame sent to a connected runner.
- Runner tests: removal frame and refusal both invoke the ctl command with the
  exact arguments; upgrade uses the ctl command.
- `make lint`, `make coverage`, `make sqlc-check`, web lint/typecheck/test green.
