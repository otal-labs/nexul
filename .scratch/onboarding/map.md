# Wayfinder map: a welcome wizard for every role

Charted 2026-10-11 with the owner. Joining Nexul is hard. The owner wizard
cannot finish without a paired computer. An invited member gets a profile step,
then a home page about runners with no next step. Only the Owner role exists on a
fresh workspace, so the roles an admin would invite people into have to be
built by hand. Five walks through the code, as a client, a junior developer, a
senior developer, a devops engineer and an administrator, found about 60 gaps
(`research/persona-gaps.md`). This map covers the ones a welcome wizard fixes.

## Destination

A spec in `.scratch/onboarding/spec.md`, ready to slice into build tickets. It
covers the Welcome wizard and its steps, the steps each Role ticks, Template
roles, the account menu entry and its dot, the owner wizard handing over to the
Welcome wizard, and an admin seeing each person's progress on Team.

## Notes

- Skills every session consults: `grilling` and `domain-modeling` for decisions,
  and `CONTEXT.md` (Welcome wizard, Template role, Instance template). The
  owner chose auto design mode for the screen tickets: the agent scores, picks
  and reports with the runner-up, and asks no design questions.
- Builds after the permission rework (`.scratch/permission-rework/`). Roles,
  invitations and ADR 0148 are changing under it, and "Your computer" rests on
  the personal runner (`.scratch/personal-runners/`), which is paused behind the
  same rework.
- Decided while charting:
  - The destination is the Welcome wizard only. The other gaps the walks found
    go to their own homes (Out of scope).
  - Every step can be skipped. A step counts as done when what it sets up
    exists, not when someone clicked through it.
  - The name is **Welcome wizard**. "Onboarding" is an _Avoid_ term twice over.
  - One wizard per person. It shows the steps ticked on any role they hold, in
    any workspace, and a step done once counts everywhere. Alice, a Developer in
    one workspace and a Client in another, sees both sets of steps.
  - The wizard opens on its own only at first sign-in. After that it is the
    top item of the account menu (the name and avatar in the sidebar footer),
    and the menu trigger carries a red dot while any step is left, the way the
    settings cog carries the yellow update dot (`UpdateDot`). There is no
    checklist on home. Existing members, and members of a role that gains a
    step, get only the dot.
  - The owner wizard keeps the instance steps: name, workspace, tools. Its
    "Set up T3 Code" step becomes the owner's Welcome wizard steps, read from the
    Owner role and skippable. The hard block on a paired computer goes.
  - Built-in steps only for now:
    - Profile
    - Connect GitHub
    - Your computer: one step for the personal runner, T3 Code, pairing and the
      setup checks, because one install command does all four
    - Your projects: what you are in and what you can do there
    - Add a deploy server: the runner install command, then a wait for the
      server to connect
    - Get the phone app
  - Five Template roles ship: Client, Junior Developer, Senior Developer, DevOps
    and Administrator. Each is an Instance template, edited in Settings →
    Templates. A new workspace copies them in beside Owner, and the copy is never
    rewritten. Existing workspaces get nothing, so prod's hand-built roles stay.
  - Team shows each person's progress through their steps ("2 of 3, computer not
    added"), so an admin can see why something isn't working for them.
  - Signing in with Google is the owner's to turn on. The invite page already
    offers whichever providers are on. The map adds nothing for it.

## Decisions so far

<!-- one line per resolved ticket: [title](issues/NN-slug.md): gist -->

- [How the permission rework lands on roles, invitations and computers](issues/01-how-the-permission-rework-lands-on-roles.md): roles keep their table and combine as a union; steps need a new field; Template roles are a new template kind copied in at workspace creation; projects are opt-in under ADR 0148; build after the `permission-switch` branch merges
- [What counts as done for each step, and what skipping does](issues/02-what-counts-as-done-and-what-skip-does.md): done is read from existing data, except a per-person record of skips and of the phone's first connect, and a record of who enrolled each deploy runner; a skip lasts until the step is given again; the phone stays done once connected; `first_login_done` now means the first run happened

## Not yet specified

- The phone app step. Can the wizard show the QR connect code itself, or does it
  link to Devices? Should the phone app show the same dot?
- Live updates. The dot should clear when a computer pairs or a runner connects
  without a refresh, which needs a topic, its audience rule and a follower.
- Agents. Should the progress Team shows also be readable over MCP (`account_list`
  or `account_get`), so an agent answering "why can't Bob run plays" sees it?
- The guide page under `website/src/content/docs/docs/guide/`, and the roadmap
  entry, once the spec exists.
- Accepted invitations on Team: who accepted and when. It sits beside the
  progress view and may join that ticket.

## Out of scope

- **Hiding what a role can't do.** Buttons show and are then refused for
  project settings, deploy, rollback, delete stack, gateways, exposures,
  "Link branch/PR", topology "Add a node", and Runners without `machines:read`.
  This is one permission sweep, as its own effort after the rework.
- **One-way doors and missing pages:** delete a machine, delete a tunnel from the
  web, edit a stack's env and secrets after launch, a PR list, "My tickets",
  retry a failed deploy. These go to separate tickets.
- **Dead-end text that should be a button**, as one quick PR: "Add a computer in
  Settings", "No machines yet — connect a runner first", "Connect a DNS
  provider", "Change access from Team", and the readiness lines.
- **Home page copy** that speaks only to developers. This is a copy fix, and the
  Welcome wizard is the guidance.
- **Admin invitation gaps.** An invitation fails silently when its creator's
  access changed, or when the invitee is already in the workspace. One role per
  invitation. "Restricted" in the copy. The permission rework is reshaping
  invitations.
- **Running plays without reading memories breaks the run dialog.** This is a
  bug, filed in `.scratch/latent-bugs/spec.md`.
- **Custom welcome steps**, such as "read the handbook" or "join #support". This
  is a possible future feature, once built-in steps have been lived with.
