# Role-aware UI: every screen shows a person only what their roles let them do

**Status:** ready-for-agent

## Problem

Of 152 actions across the web and phone apps, 74 are shown to people who can't take them and are refused only after
the click. Another 8 are gated on a different permission than the server checks (`research/01`). Fifteen things can
be entered but not left, or made but never undone (`research/02`). AGENTS.md calls a one-way door a bug, and a
refused click teaches nothing.

## Decisions

From the map and tickets 01 to 06.

1. **Hidden or locked.** An action the viewer can't take is hidden. A viewer who holds any role with **Show locked
   actions** on sees it as a Locked action instead: dimmed, with the reason. The reason is "Your role can't <verb>"
   or, when a rule on that one thing refused it, the rule's source sentence. A thing they can't read stays not found
   (ADR 0148).
2. **The switch** is a field on the role. The template roles ship it off for Client and on for Junior Developer,
   Senior Developer, DevOps and Administrator. Owner can do everything, so it doesn't apply.
3. **One shared piece draws every action** (ticket 05). The state comes from one function in `client-core/`, and the
   web and phone each have components. Nobody writes a check by hand.
4. **The guard** (ticket 05).
   - Every request that writes declares its permission where it is defined.
   - `make route-permissions` generates the server's route → permission file.
   - A test fails when a request's permission is missing or differs from its route's.
5. **Single-item answers** (ticket 05).
   - A stack, doc, play or project page reads its item's answers once through `GET /api/permissions/explain`.
   - Lists stay role-wide, and a refusal on click shows the rule's sentence.
   - explain covers stacks, machines and instance areas once the rework's tickets 19, 23 and 24 land.
6. **The look** (ticket 04): "Quiet lock and tooltip", with the values in the ticket. It uses `aria-disabled`, never
   `disabled`; on the phone it is tap for the reason.
7. **The doors** (ticket 06): as listed there.
   - Machine delete is refused while stacks deploy to it.
   - Tunnel delete is refused while hostnames use it.
   - Both refusals name what is in the way and say what to do.
   - Retry rebuilds the same commit with current settings.
   - Secrets are masked for readers.
   - A workspace archives and restores.

## Architecture

### `client-core/`

`actionState({permission, me, workspaceId, projectId?, entityAnswer?}) → {state: "live" | "locked" | "hidden",
reason?}` reads `/me`, which already carries the roles' union, the project's actions and the switch. The switch shows
up as a new `/me` field, `show_locked_actions`, which is true when any held role has it on. An `entityAnswer` from
explain overrides the role-wide answer and supplies the source sentence. "Your role can't <verb>" is built from the
permission catalog's label.

### Web and phone components

- Web: `ActionButton`, `ActionMenuItem`, `ActionInlineEdit` and `ActionDragHandle` in `web/src/components/action/`.
  Each takes the mutation (which carries its permission) or a permission, plus the usual props.
- Phone: the matching pieces in `native/src/components/action/`.
- Both render ticket 04's look and accessibility.

### Declared permissions and the route file

- **Requests**: each mutation hook is defined with its permission (`permission: "deploys:write"`).
- **The route file**: `make route-permissions` writes `client-core/routePermissions.json`, mapping method and path to
  action, from the router's permission table. The audit test already classifies every route by method, so the same
  walk can emit it.
- **The guard test**: `client-core/routePermissions.test.ts` checks every declared mutation against the file.
- **The phone**: its mutations use the same declaration.

### Single-item answers

One `useEntityActions(kind, id)` per detail page calls explain once for the actions its page shows. The answer is
cached until a `permissions.changed` frame names the item. It follows the live topic the rework already pushes for
rule changes; if no frame names the item, the build adds an id beside the existing fields.

### The switch

- `roles.show_locked_actions INTEGER NOT NULL DEFAULT 0`, in a forward-only migration that leaves existing roles off.
- It appears on role create, update and clone, in `role_update` and `workspace_list`, and in a switch in the role
  editor's Workspace tab.
- Template roles carry it: the welcome wizard effort's ticket 13 seeds the five with the defaults above.

### Doors

| Door | Server | Web and phone |
|---|---|---|
| Ticket delete | exists | a Delete item in the ticket menu, with a confirm |
| DNS records | exists | Delete on each record the web made |
| Archived docs | exists | an Archived filter in the docs list, and Restore |
| Tunnel delete | refuse while hostnames use it, naming them; remove its CNAME | a Delete with a confirm; the refusal shown as a toast that names the hostnames |
| Stack environment | mask secret values for readers; `PATCH` already edits | an Environment section on the stack page, "Applies on the next deploy", Deploy now, reveal for writers |
| Deploy retry | record the deploy's commit; Retry rebuilds it with current settings | Retry on a failed deploy; Rollback hidden without an image |
| Machine delete | `DELETE /api/machines/{id}`, refused while stacks target it, naming them; MCP `machine_delete` | Delete on the machine card; the refusal shown as a toast |
| Branch and PR unlink | unlink in the ticket use-case; `ticket_update` gains `unlink_branch` and `unlink_pr` | Unlink on each linked branch and PR |
| Enrollment codes | list open codes and revoke one | open codes under Runners and Computers, each with Revoke |
| Notification unread | mark unread; `notification_update` takes it | "Mark unread" on a notification |
| Phone | — | message edit and delete; deploy cancel |
| Workspace | `archived_at` and archive and restore use-cases; archived ones leave every list; MCP `workspace_update` takes `archived` | Archive in the workspace Danger zone; Restore in instance settings |
| Copy | — | the automations host's "Add it again", Reactivate's confirm, the stack's "Attach a repository … from here", "Change it later from the DNS page", and the Danger zone's double negative |

## Hit every surface

- **UI**: every screen in research 01, at 768, 1024 and 1440, and the phone's writes.
- **HTTP**: the new and changed routes above, each classified for audit.
- **MCP**: `machine_delete`; `ticket_update` unlinks; `notification_update` unread; `workspace_update` archived;
  `role_update` takes the switch; an enrollment code revoke is added only inside the tool budget
  (`practices/mcp.md`).
- **Events**: `machine.deleted`, `workspace.archived`, `workspace.restored` and `deploy.retried`, each with payload
  types, `make event-schemas` and outbox writes. The switch rides `role.updated`.
- **Live**: the new topics get audience rules and followers. A machine's and a workspace's removal reach the lists
  that show them.
- **Both apps**: the rule and the route file live in `client-core/`.
- **Permissions**: this effort is about them. The guard test enforces the match.
- **Reverse states**: this effort is about them too.
- **Docs**:
  - An ADR for the hidden-or-locked rule and the switch.
  - An ADR for the two refusing deletes, a deliberate exception to signals over gates.
  - `practices/react-guide.md` and `practices/native.md` say every write goes through the action components.
  - The guard goes in AGENTS.md's enforcement table.
- **User guide**: `people-and-access.md` (the switch and locked actions), `runners.md` (machine delete, codes),
  `topology-and-dns.md` (tunnel and record delete), `stacks-and-deploys.md` (environment, retry),
  `docs-tickets-and-board.md` (ticket delete, unlink, archived docs), and `phone-app.md`.

## Out of scope

- "My tickets" and a PR list.
- Threads that can't be deleted (ADR 0094).
- Instance downgrade.
- Permanent workspace delete.
- MCP refusals, which already carry the source sentence.

## Risks

- **The sweep touches about 80 call sites.** Each sweep ticket covers one group of areas, so a pull request stays
  reviewable, and the guard test lands first so the sweep can't miss one.
- **Masking secrets changes what `GET /api/stacks/{id}` returns.** Any client reading values as a stack reader breaks.
  The automations SDK and the phone are checked in the stack environment ticket.
- **explain's cost on a busy detail page** is one call per page. The build adds a statement guard for it.

## Build order

Tickets 08 to 23 in `issues/`. 08, then 09, then 10. 11 to 15 follow 10 and can run in parallel. The door tickets, 16
to 22, can start at any time, in parallel with the rest. 23 comes last.
