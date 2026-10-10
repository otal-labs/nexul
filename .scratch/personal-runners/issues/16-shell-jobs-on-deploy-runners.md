# 16 — Shell jobs on runners that are not personal

**Status:** ready-for-agent

**Blocked by:** 15, permission-overrides 01

Read first: `practices/go.md`, `practices/mcp.md`, ADRs 0057, 0087, 0088, the spec (Later: shell jobs).

This ticket adds an instance-area permission, so it lands in the evaluator that names sources (permission-overrides
01); once permission-overrides 13 ships, `runners:shell` is part of the Runners and machines entity the instance
Owner can hand to a role or a person. Jobs here follow ticket 15's
retention: the record kept forever, the output deleted after 30 days. Unlike a personal runner, a machine's
shell jobs stay off until its operator opts in.

## What to build

- `runners:shell` in the permission table, instance area; no role gets it on upgrade; scoped tokens need it
  explicitly.
- `nexul install runner --allow-shell` and the same refusal without it; `runners.shell_enabled` flipped
  with `runners:shell`.
- The runner row on Settings → Runners gets the switch and the history; `command_run` takes `runner_id`.
- The runners guide says plainly that a shell job on Linux runs as root.

## Acceptance criteria

- [ ] A role without `runners:shell` cannot start, list or read jobs; a scoped token needs the bit itself.
- [ ] A machine installed without `--allow-shell` refuses every job, whatever Nexul's switch says.
