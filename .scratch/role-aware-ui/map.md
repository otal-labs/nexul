# Wayfinder map: every screen shows a person only what their roles let them do

Charted 2026-10-13 with the owner, from the persona walks of the welcome wizard effort
(`.scratch/onboarding/research/persona-gaps.md`). About a dozen screens show buttons a role can't use and refuse
them only after the click: project settings, deploy, rollback, delete stack, gateways, exposures, "Link branch/PR",
topology "Add a node", and Runners without `machines:read`. Four things can be made but never undone from the web:
a machine, a Cloudflare tunnel, a stack's environment after launch, and a failed deploy.

## Destination

A spec in `.scratch/role-aware-ui/spec.md`, ready to slice into build tickets. It covers:
- the rule for how an action the viewer can't take shows
- the sweep that applies it to every web and phone screen
- the one-way doors
- the guard that keeps a new button from slipping back

## Notes

- Skills every session consults: `grilling` and `domain-modeling`, and `CONTEXT.md` (Locked action, Source,
  Permission overwrite). Look tickets run in `auto-design-mode`: the agent picks and reports the runner-up.
- The permission rework (`.scratch/permission-overrides/`) already decides two things:
  - A thing you can't read is not found (its rule 7).
  - A refused action answers with its source sentence (rule 8).

  Locked pills already show "You don't have access". Stack rules (its ticket 19) and instance-area rules (23, 24)
  are still to come, so a role-wide check can't always answer for one stack or machine.
- The owner's call: **everything here builds after the `permission-switch` branch merges**, the one-way doors
  included.
- Decided while charting:
  - **Hidden or locked.** An action the viewer can't take is hidden, unless they hold a role with **Show locked
    actions** on. Then it shows as a Locked action: dimmed, with the reason. The reason is "Your role can't <do
    it>", or the source sentence when a rule on that one thing refused it.
  - **The switch.** It lives on the role, one is enough among several roles, and it is edited in the role editor.
    The template roles ship it off for Client and on for Junior Developer, Senior Developer, DevOps and
    Administrator. Owner never needs it.
  - **The phone app.** It follows the same rule. The rule lives once in `client-core/`.
  - **Missing pages are out of scope.** "My tickets" and a PR list are features, not fixes.

## Decisions so far

<!-- one line per resolved ticket: [title](issues/NN-slug.md): gist -->

## Not yet specified

- Whether the server should also report "would be locked" for an action. The web may be able to work it out from
  what it already has; ticket 03 tells.
- Copy for each action's "Your role can't …" line. It may be generated from the permission catalog's labels.
- Whether the switch belongs in the welcome wizard effort's template role table (`.scratch/onboarding/` ticket 04)
  or only here. The template defaults are decided above either way.

## Out of scope

- **Missing pages**: "My tickets", a PR list. Features, not fixes.
- **MCP**: it has no buttons, and the server already refuses with the source sentence.
