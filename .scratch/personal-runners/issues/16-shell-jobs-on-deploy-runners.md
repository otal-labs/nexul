# 16 — Shell jobs on runners that are not personal

**Status:** done

**Blocked by:** nothing; ticket 15 shipped the shell job (ADR 0209).

Read first: `practices/go.md`, `practices/mcp.md`, ADRs 0057, 0151, 0209, the spec (Later: shell jobs).

A runner on a machine runs the same shell job a personal runner does. Who may is the machine's business (ADR 0151, which
removed `runners:*`): Run commands, `machines:shell`, is a fifth machine action beside Deploy to, Manage and Delete, held
through roles and the machine's rules. Jobs here follow ticket 15's retention: the record kept forever, the output deleted
after 30 days. Unlike a personal runner, a machine's shell jobs stay off until its operator opts in.

## What to build

- `machines:shell` in the permission table, a machine action; no role gets it on upgrade; scoped tokens need it
  explicitly.
- `nexul install runner --allow-shell`, and `nexul shell on|off <runner>` on the machine; without either the runner refuses
  every job. One Nexul switch per machine, `machines.shell_enabled`, off by default, flipped with `machines:shell`.
- The machine's band on Settings → Runners gets **Commands…** in its menu: the switch, a command for one of its runners,
  and the history; `command_run` takes `runner_id`, and `machine_list` with an id and `commands` reads the record.
- The runners guide says plainly that a shell job on Linux runs as root.

## Acceptance criteria

- [x] A role without `machines:shell` cannot start, list or read jobs; a scoped token needs the bit itself.
- [x] A machine installed without `--allow-shell` refuses every job, whatever Nexul's switch says.
