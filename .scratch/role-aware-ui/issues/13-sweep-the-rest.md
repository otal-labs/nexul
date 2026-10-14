# 13: Sweep: docs, memories, plays, automations, chat, team and instance settings

Type: task
Status: resolved
Blocked by: 10

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

The same for every remaining area in research 01. Per-play Run on the play controls is done (#654).

## Acceptance criteria

- [ ] As ticket 11. After this ticket, research 01 lists no ungated action on the web.

## Answer

Swept docs lists and folders, memories (list, detail, versions, interview sources and suggestions, interview template), plays (settings list, trail answer and continue, decisions check), auto plays, automations (list, detail, token, secrets, versions, hosts), chat (sidebar, channel rows and settings, bots), team (invitations, account actions), roles, workspace danger zone, instance settings (address, version, sign-in, connectors, archived workspaces), template reset links and attachments. Every write there is an action component, `ActionSwitch`, `GatedSaveBar` or a `RowMenu` gate.

Left hidden-only, on purpose: server-sent per-item flags (`can_close`, `can_manage_members`, `can_detach`, a play's `can_run`), the run's starter-or-write rule, row menu Permissions..., the command palette, workspace creation (`workspaces:create` is held anywhere but is not in the instance action list), read-only editor and form modes, and sections a role can't open (Roles, General, Mentions, instance Templates). Integrations, dead letters and audit have no web control.

