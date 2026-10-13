# Welcome wizard: every person set up for what they do

**Status:** ready-for-agent. The permission prerequisite has merged.

## Problem

Joining Nexul is hard, and the parts that are hard depend on who is joining.

- An invited member gets a profile step, then a home page about runners and deploys with no next step.
- The owner wizard cannot finish without a paired computer, even for an owner who will never run an agent.
- A fresh workspace has only the Owner role, so the roles an admin invites people into have to be built by
  hand.
- An admin cannot see whether someone finished setting up, so "it doesn't work" has no first place to look.

Five walks through the code, as a client, a junior developer, a senior developer, a devops engineer and an
administrator, found the gaps (`research/persona-gaps.md`). The decisions are on the map (`map.md`) and in its
tickets 01 to 06; this spec puts them together for the build.

## The flow

```
Invited person signs in for the first time
└─ Welcome wizard opens by itself (only this once)
   ├─ Profile                      always first
   └─ the steps ticked on any role they hold, in any workspace
      ├─ Connect GitHub
      ├─ Your computer             install command → connected, T3 Code found, paired → setup
      ├─ Your projects             what they're in and what they can do there
      ├─ Add a deploy server       install command → server connected
      └─ Get the phone app         QR code → phone connected
   each step: do it, or Skip for now
   end: "N things left for later", or "You're set up"
Afterwards
└─ account menu: Welcome on top, a red dot while a step is neither done nor skipped
Admin
└─ Team: a line under each person while a step is left or skipped; accepted invitations listed
Owner of a fresh install
└─ owner wizard (name, workspace, tools) → "Workspace ready" → the Welcome wizard
```

## Decisions

From the map's charting notes and tickets 01 to 06. Each links to where it was decided.

1. **One Welcome wizard per person** (charting). It shows Profile, then every step ticked on any role the person
   holds, in any workspace. A step done once counts everywhere.
2. **Opens by itself only at first sign-in** (charting, ticket 02). `first_login_done` now means "the Welcome
   wizard has had its first run". It is set when the person finishes or closes the wizard the first time.
   Afterwards the wizard is reached from the top of the account menu. Existing members, and members given a
   step later, only get the dot.
3. **No checklist on home** (charting). The account menu is the one reminder.
4. **Done is read from data** (ticket 02), except where noted:

   | Step | Key | Done while | Lights the dot |
   |---|---|---|---|
   | Profile | — | the first run happened | never |
   | Connect GitHub | `github` | a working row in `github_user_links` | yes |
   | Your computer | `computer` | one of the person's computers is paired and its setup confirmed | yes |
   | Your projects | `projects` | the person is in at least one project | only when the step was given to them |
   | Add a deploy server | `deploy_server` | a deploy runner this person enrolled is connected | yes |
   | Get the phone app | `phone` | a phone has signed in once (recorded, never cleared) | yes |

5. **Skip** (ticket 02). Stored per person per step. A skipped step stops lighting the dot and shows as Skipped,
   and can be done any time. The skip lasts until the step is **given** to the person again: a role they hold
   gains the tick, or they are given a role that ticks it. Then the dot comes back until it is done or skipped
   again.
6. **Ticks live on the role** (ticket 03). A JSON list of step keys beside `permissions`. Profile is never a tick.
   Edited in a **Welcome** tab of the role editor by whoever may edit the role (`roles:write`); a step gives no
   power, so the "grant only what you hold" check does not apply. Clones copy the ticks. The HTTP role request
   and `role_update` take `welcome_steps`; `workspace_list` returns them.
7. **The Owner role** (ticket 03). It stays uneditable. Its steps are fixed in code: `github`, `computer`.
8. **Owner wizard hand-over** (charting, ticket 05). The owner wizard keeps name, workspace and tools. Its "Set up
   T3 Code" step becomes "Workspace ready", which opens the Welcome wizard. The hard block on a paired computer
   goes.
9. **Template roles** (ticket 04). Client, Junior Developer, Senior Developer, DevOps and Administrator ship as
   Instance templates, each with a permission grid and welcome ticks (the table is in ticket 04). Edited in
   Settings → Templates → Roles with the same role editor. The shipped five reset but don't delete. The instance
   can add and delete its own template roles. Every new workspace, a fresh install's first included, copies every
   template role beside Owner and never rewrites the copy. Existing workspaces get nothing.
10. **Looks** (tickets 05 and 06), decided in auto design mode. Prototypes on the never-merged branches
    `proto/welcome-wizard` (Rail frame) and `proto/team-progress` (Gap line); the values are in the tickets.
11. **Progress on Team** (ticket 06). A gap line under each person while a step is left or skipped, details in
    the person dialog, invitations accepted in the last 30 days on the Invitations tab. "Not in a project yet"
    shows on Team whenever the person is in no project, because the admin is the one who fixes it. Visible to
    whoever can see Team's member list.
12. **The phone app shows no dot and no wizard** (decided in this spec). It has no account menu, and every step
    but "Get the phone app" is done on the web. A phone shows the web's progress nowhere for now.
13. **Live** (decided in this spec). The dot and the Team line update without a refresh, through one derived
    topic (Architecture).
14. **Agents** (decided in this spec). `account_get` and `account_list` return each person's welcome progress, to
    whoever may read the account, so an agent asked "why can't Bob run plays" sees "computer not added".

## Architecture

### A `welcome` domain

`internal/welcome/` owns the per-person record and works out progress. It reads other domains through narrow
read seams wired in the composition root, the way the pairing domain reads runner rows, and never writes their
tables.

| Seam | Answers | Owner |
|---|---|---|
| ticked steps | the union of `welcome_steps` over the person's roles in every workspace, plus Owner's fixed pair | roles, tenancy |
| GitHub linked | a working `github_user_links` row | repository / auth |
| computer ready | a paired computer of the person's with `setup_confirmed_at` set | pairing |
| in a project | at least one project lists the person or a role they hold | access |
| deploy server | a connected deploy runner with `enrolled_by` = the person | runner |
| first run | `first_login_done` | auth |

`Progress(ctx, userID)` returns each ticked step with a state, `done`, `skipped` or `left`, and the time of the
skip or the phone's first connect. "Lights the dot" is `left` for every step but `projects`, which lights only
while its skip-record says it was given (below).

### Records

- `welcome_steps` on `roles`: `TEXT NOT NULL DEFAULT '[]'`, a JSON list of keys.
- `welcome_step_states (user_id, step, skipped_at, given_at, done_at, PRIMARY KEY (user_id, step))`.
  - `skipped_at` is set by Skip and cleared when the step is given.
  - `given_at` is set when the step is given after the person's first run. Only `projects` reads it, to decide
    whether it lights the dot.
  - `done_at` is set only for `phone`, on the first phone sign-in.
- `enrolled_by` on `runner_enrollment_codes` and `runners`: the person who made the install command. Copied
  from the code to the runner at enrollment. Empty for rows that predate it, and for the `instance` runner.

All three are one forward-only migration, numbered after the permission switch's.

### Giving a step

- `role.updated` gains `welcome_steps_added`, the keys the save added. A consumer clears `skipped_at` and sets
  `given_at` for those steps for every holder of the role.
- Giving someone a role (the member-roles event the switch publishes) does the same for the role's ticks.
- First sign-in gives nothing: every step starts unskipped.

### Live updates

One derived topic, `welcome.progress_changed {user_id}`, written by a consumer of:

- `computer.paired`, `computer.setup_confirmed`, `computer.setup_unconfirmed`, and removing a computer
- a GitHub link made or revoked
- `runner.connected` and `runner.disconnected` for a runner with `enrolled_by`
- a phone session created
- project lists changing (the switch's access events)
- `role.updated` with ticks changed, and a role given or taken
- the person's own Skip, Unskip and first run

Its audience rule: the person, plus everyone who may see Team's member list. It is bridged in `livePushTopics`.
The web follows it in the welcome hooks (the account menu's dot and the wizard) and the Team hooks. It names
only `user_id`, so a frame refreshes one person's progress.

### HTTP and MCP

| Route | Does |
|---|---|
| `GET /api/me/welcome` | the caller's progress |
| `POST /api/me/welcome/{step}/skip` | skip one of the caller's steps |
| `DELETE /api/me/welcome/{step}/skip` | unskip it |
| `POST /api/me/welcome/first-run` | sets `first_login_done` (replaces the first-login finish) |
| role create and update | take `welcome_steps` |
| Team people list | each `TeamPerson` gains `welcome` (the progress) and `joined {invited_by, accepted_at}` from the invitation's `redeemed_by`, `redeemed_at` |
| invitations list | also returns invitations accepted in the last 30 days |
| templates | the `role` template kind: list, create, update, reset (shipped) or delete (added) |

MCP: `role_update` takes `welcome_steps`, and `workspace_list` returns them. `account_get` and `account_list`
return `welcome`. Template roles are edited through `template_get` and `template_update` with the `role` kind.
Skip is a person's own UI action, with no tool.

### Template roles

A new template kind `role`, with keys `client`, `junior_developer`, `senior_developer`, `devops` and
`administrator` for the shipped five. Each one's code default is ticket 04's grid and ticks. An added template
role gets a generated key and exists only as its row. A body is `{name, permissions, welcome_steps}`.

Workspace creation, next to `SeedWorkspaceDefaults`, creates one role per template role from its resolved body.
Owner comes first, as today.

## Hit every surface

- **UI**:
  - the Welcome wizard at `/wizard/welcome`
  - the account menu row and dot
  - the role editor's Welcome tab
  - Settings → Templates → Roles
  - Team's gap line, person dialog block and accepted invitations
  - the owner wizard's hand-over

  Each is verified at 768, 1024 and 1440.
- **HTTP gateway**: the routes above.
- **MCP**: `role_update`, `workspace_list`, `account_get`, `account_list`, `template_get`, `template_update`.
- **Events**:
  - `welcome.progress_changed` in `Topics()` with its payload type
  - `role.updated` gains `welcome_steps_added` beside its fields
  - `make event-schemas`
  - outbox writes
- **Live push**: the topic above, its audience rule, `make live-topics`, followers in the web's welcome and Team
  hooks. No phone query shows welcome progress, so no `refreshes` line.
- **Both apps**: the step keys and the "lights the dot" rule live once in `client-core/`, since the web uses them
  today and the phone may later.
- **Search**: nothing indexed.
- **Permissions**:
  - A person reads and skips only their own steps.
  - Team progress follows Team's visibility.
  - Ticks follow `roles:write`.
  - Template roles follow `templates:write`.
- **Reverse states**:
  - Skip has Unskip.
  - A removed computer or a revoked GitHub link makes a step left again.
  - A deleted added template role leaves the workspace copies alone.
  - Reset brings a shipped one back to its default.
- **Docs**:
  - `CONTEXT.md` already has Welcome wizard and Template role.
  - An ADR records "template roles are copied once and never rewritten, and a skip lasts until the step is given
    again", since both surprise a reader.
  - `practices/` needs nothing new.
- **User guide**:
  - A new `website/src/content/docs/docs/guide/welcome-wizard.md`.
  - `people-and-access.md` gains template roles, Welcome ticks and Team progress.
  - `runners.md` gains "added by".
  - `setup-wizard.md` gains the hand-over.

## Out of scope

As on the map. Hiding what a role can't do, the one-way doors, the dead-end text, home page copy, admin
invitation gaps and custom welcome steps each have their own home. A super admin role above Administrator is the
owner's for later.

## Risks

- **Permission prerequisite met.** The permission switch and remaining permission slices have merged.
  Build on the shipped role editor, invitations and Team (ADR 0148).
- **The progress read on Team.** Working out six steps per person on a long list must stay one statement per
  seam per page, not one per person. It joins the `TestStatements_*` guards in `server/cmd/access_memo_test.go`.
- **Tunnel-paired computers.** They count once setup is confirmed. Removing tunnel pairing (personal-runners
  ticket 14) doesn't change that.

## Build order

The tickets are in `issues/`, 08 to 15. 08 first. Then 09. Then 10 to 14, which can run in parallel. 15 last.
