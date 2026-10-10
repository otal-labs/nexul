# Permission overwrites per entity, with a source for every answer

**Status:** ready-for-agent (tickets 01 to 14, built in order: 01 to 07, then 08, then 09 to 14)

ADR 0148 (proposed) records the model. The glossary term stays **Permission overwrite**. The UI calls the three
states Allow, Fallback and Deny.

## Problem

Today a permission comes from a person's role, with per-person exceptions stored in four different shapes, and
nobody can see why a person can or cannot do something:

- The role is the only thing that names a group of people. An exception for "every Contractor on this
  project" has to be written once per person, and is missed for the next Contractor.
- The exceptions disagree on their states. A doc's sharing dialog only adds or removes an allow
  (`internal/access/usecase.go:515-520`). A play's dialog only adds or removes a deny (`:507-513`). A
  Restricted member's Project access is levels with no deny (ADR 0097). Only the workspace-wide override in
  Team holds both, as two separate grids (`web/src/components/team/TeamOverridesForm.tsx`).
- No surface says where an answer came from. A member who gets "forbidden" cannot tell whether their role,
  their Team override, the project or the doc decided it, and neither can an agent.

The owner's direction (2026-10-10): the role stays the base, the **fallback**. Any Nexul entity that is a
natural access boundary may carry rules for a role, for everyone, or for one person, each permission in one
of three states, Allow, Fallback or Deny. Every effective permission has a **source**, the rule that decided
it, shown in the web app and to agents.

## What exists today

| Piece | Where | What it does |
|---|---|---|
| The permission table | `internal/platform/permissions/permissions.go:135-174` | One row per domain: its actions and its area (project, workspace, instance, `:117-125`). `GET /api/permissions/catalog` serves it (`internal/access/handler.go:183-185`). |
| Roles | `roles` and `workspace_members.role_id`, `internal/platform/storage/migrations/0001_schema.sql:307-321` | **One role per membership.** The Owner role (`is_owner_role`) bypasses every check (ADR 0042, 0024). |
| Instance powers | ADR 0088 | Instance-area bits held in any workspace the person is in unrestricted (`internal/access/gate.go:146-157`). |
| Overwrite storage | `permission_overwrites(resource_type, resource_id, user_id, allow, deny)`, `0001_schema.sql:607-617` | One table, keyed per person, `user_id` cascades on account delete. Four resource types in use: `workspace`, `project`, `doc`, `play`. |
| Workspace-wide override | `internal/tenancy/usecase.go:636-683` | Per person, allow and deny, set from Team or `account_update`, takes `members:write` and the hold-what-you-give rule. |
| Project access | `internal/tenancy/project_access.go:69-102`, ADR 0097 | Allow-only levels per Restricted member per project, in `resource_type = "project"` rows. Rows are read only while the member is restricted (`internal/access/usecase.go:145-155`), so a member switched back to From role keeps dormant rows. |
| Doc sharing | `internal/access/usecase.go:376-379`, `:357-366` | Per person; the creator gets `CreatorGrant` (`permissions.go:287`). Managed with `permissions:write` on the doc (`:487-493`). |
| Play exclusion | `internal/access/usecase.go:381-390` | A deny of `plays:run` for one person on one play, managed with `plays:write`. |
| Private channels | ADR 0098 | Membership, not overwrites: participant rows. The Owner sees every one. |
| Memories | ADR 0099 | Project-scoped only; checked through the project. No overwrites. |
| Computers | `.scratch/personal-runners/spec.md`, Access and privacy, rule 3 | Ownership checks that never go through the permission gate. Sharing a computer is not built yet; ticket 12 builds it as rules. |
| The evaluator | `internal/access/usecase.go:95-110` (`decide`), `:162-176` (`has`), `:227-236` (`applyOverwrite`) | Owner bypass, then hidden-project, then role, then workspace-wide override, then the resource's own row; deny beats allow inside a row. Every check path ends here: `Require`, `RequireProject` (`gate.go:27-60`), `HasPermission`, `CanDocs`, `ProjectsWith` (`gate.go:269-290`), `DocsWith` (`usecase.go:270-295`). |
| The memo | `internal/access/memo.go:12-20`, ADR 0135 | Reads memoised per request until the next commit. |
| Lists in SQL | ADR 0140, `practices/go.md` section 17 | `CallerProjects`, `ProjectsAnywhere` and `DocsWith` turn the evaluator's answers into id sets a list filters by in SQL. `server/cmd/list_paging_test.go:221` holds every paged list to the row-by-row answer; `server/cmd/access_memo_test.go:268-336` pins statement counts. |
| Live | `server/cmd/live_audience.go:76`, `:225-235` | `access.grant.changed` reaches the person whose row changed and, for a project, its workspace's `members:write` holders. |
| MCP | `internal/access/mcp.go:41-106`; `account_update` (`internal/mcp/composite/accounts.go:74-82`) | `permission_overwrite_list` and `_update` on docs and plays; `account_update` sets the workspace-wide override and Project access. The surface holds 110 tools against a budget of 111 (`internal/mcp/surface_test.go:21`). |
| Web | `web/src/components/access/PermissionsForm.tsx` (doc and play dialog), `components/settings/ProjectPeopleAccessSection.tsx` (read-only "People with access"), `components/team/TeamOverridesForm.tsx`, `TeamProjectAccess.tsx` | Four editors for four shapes. |

Because a member holds exactly one role, "two roles disagree" cannot happen today.

## The model

### States

Each rule holds, for each permission it covers, one of three states:

- **Allow**: the permission is held, whatever came before.
- **Fallback**: this rule says nothing; the answer is whatever came before it. A permission the rule does not
  mention is Fallback.
- **Deny**: the permission is not held, whatever came before.

Storage is unchanged in shape: Allow is membership in `allow`, Deny in `deny`, Fallback in neither, and a
permission in both is refused on write (`internal/tenancy/usecase.go:685-698` already does this for Team).

### Targets

A rule targets one of:

- **Everyone**: every member of the workspace, including people who join or get a role later.
- **A role**: every member holding it.
- **A person**: one member.

Which targets an entity takes depends on the entity: a computer takes named people only, and an instance area
takes a role or a person but never Everyone (see Entities).

### Entities

A rule is set on an entity. Rules apply to everything in Nexul that is a natural access boundary, including a
person's computers and the instance's own areas (owner, 2026-10-10). The chain a check walks, outermost first:

```
workspace (person rules only: the Team override that exists today)
└─ project
   ├─ doc folder (ticket 09)
   │  └─ doc
   └─ stack (ticket 10)
workspace
├─ play
└─ channel (ticket 11)
outside any workspace
├─ computer (ticket 12)
└─ instance area (tickets 13 and 14)
```

The first set, deliberately small, is the entities that already carry overwrites, so the effort unifies what
exists before adding anything. The build order after it is doc folders, stacks, channels, then computers, then
instance areas.

| Entity | Targets | Permissions a rule may set | Written with |
|---|---|---|---|
| Workspace | Person | every catalog action (today's Team override) | `members:write` |
| Project | Everyone, role, person | the project-area actions | `members:write` in its workspace |
| Doc | Everyone, role, person | `docs:*`, `permissions:write` | `permissions:write` on the doc |
| Play | Everyone, role, person | `plays:run` | `plays:write` on the play |
| Doc folder (09) | Everyone, role, person | `docs:*` for the docs in it | `permissions:write` on the folder |
| Stack (10) | Everyone, role, person | `stacks:*`, `deploys:*` | `stacks:write` on the stack |
| Channel (11) | Everyone, role, person | `channels:read`, a new posting permission | `channels:write` on the channel |
| Computer (12) | Named people only | See this computer, Run agents, Run commands | only the computer's owner |
| Instance area (13, 14) | A role or a person, never Everyone | the area's actions (Instance areas below) | only the instance Owner |

Not entities: tickets, memories and attachments one by one, which follow their project; and Everyone or role
rules on the workspace itself, which would be a second way to edit a role.

### Evaluation order

For person `P` asking for action `A` on entity `E`:

1. **Not a member** of `E`'s workspace: not found. The server's own calls pass. (Unchanged.)
2. **Owner** of the workspace: allowed. No rule on any entity can deny the Owner, so nothing is ever locked away
   from everyone (ADR 0042). The one exception is a personal computer, which the Owner's bypass does not reach
   (Computers below).
3. **Base.** The role's set. For a Restricted member, a project-area action starts as not held, and an
   instance-area action is never held. (Unchanged, ADR 0097.)
4. **Workspace person rule** applies on top, except to a Restricted member's project-area actions.
   (Unchanged.)
5. **Each entity on the chain from the project inward**, and on each entity three rules in this order:
   Everyone, then `P`'s role, then `P`. A rule in Allow or Deny replaces the answer and the source; Fallback
   leaves both.
6. **Restricted members.** On a project-area action, Everyone and role rules may only Deny; only a rule naming
   the person may Allow. A project where a Restricted member holds no project-area action after step 5 is not
   found. (This keeps ADR 0097's promise: a client sees a project only when it is given to them by name.)
7. **A denied project is hidden.** When the rules on the project itself, taken in the order above, end in Deny
   for `projects:read` ("Read projects and board settings"), the project is not found for `P`: in lists, links,
   search and live frames, as a private channel is (ADR 0098). This holds for any member who is not the Owner,
   Restricted or not. It is checked at the project before the chain walks inward, so nothing inside a hidden
   project is reachable, a doc's own rule for `P` included. A deny of `projects:read` anywhere else, such as the
   workspace rule or a role, hides nothing, as today.
8. **The source** is the last rule that set Allow or Deny, or the base when none did.

Decisions inside the order:

- **The nearer entity wins over the farther one**, whatever the target: a doc rule for a role beats a project
  rule for a person. This is today's most-specific-wins (ADR 0042), and Project access already works this way
  for a doc's own rows (ADR 0097).
- **Inside one entity, the narrower target wins**: person over role over Everyone.
- **One rule cannot both allow and deny** a permission; the write is refused.
- **Two roles** cannot conflict while a member holds one role. If several roles per member ever ship, allow
  wins between role rules on the same entity and the person rule still beats both. Not built now.

### Computers

A person's computer is the one entity the Owner's bypass does not reach. Computers are private: hidden from
everyone, workspace Owners and the instance Owner included, unless the computer's owner shares it by name.
For person `P` asking for permission `X` on computer `C`:

1. `P` is signed in and active.
2. `P` owns `C` (the person who installed its runner): allowed, everything. The owner is never denied their own
   computer.
3. Otherwise `C` is found only through a rule that names `P` and allows See this computer, and only while the
   owner's account is active and `P` shares a workspace with the owner. `P` then holds exactly the permissions
   that rule allows.
4. Nothing else counts: no role, no Everyone rule, no workspace person rule, no Owner role, no role holding
   every bit, no instance Owner. No match is not found, never forbidden, so a computer's existence is not
   confirmed.

The permissions on a computer are **See this computer**, **Run agents** and **Run commands**. Allowing either of
the last two also allows See, since a person must find a computer to use it. They live in their own small set,
not in the role grid: no role can hold them, which is what keeps every-bit roles out. Only the computer's owner
writes the rules; a grantee cannot re-share and nobody else (an admin, the instance Owner, an Owner) can write one.
Targets are named people only: no Everyone, no role. A computer rule holds Allow or Fallback; Deny is refused, since
a person with no rule holds nothing and the owner cannot be denied. Revoking is immediate: a computer check
reads the rules in its own query and is never served from the memo. A rule ends when the person shares no
workspace with the owner or either account is disabled or removed, and it is deleted with the computer or the
account.

`computer_activity:read` is a different thing and is unchanged: an instance-area permission that reads the
audit record of what ran on computers (personal runners spec, Access and privacy, rule 5). It lists no
computer, shows no facts and grants no use, so it does not make a computer visible.

### Instance areas

An instance area is a part of the instance that belongs to no workspace (`AreaInstance` in
`internal/platform/permissions`). Today a person holds one by a role bit in any workspace they are in unrestricted
(ADR 0088). Each area now carries rules, so the instance Owner can hand out one area without a custom role:
"bob manages DNS but not connectors".

The areas, from the permission table's instance rows. A rule on an entity sets the listed actions for that part
only; when two entities share an action, the entity is the part of the instance the rule limits.

| Entity | Actions a rule may set |
|---|---|
| The instance itself (URL, upgrades, failed events, public address) | `instance:read`, `instance:write`, `events:read` |
| Sign-in providers | `instance:read`, `instance:write`, for the providers only |
| Each connector (GitHub, Cloudflare, LiveKit, and any added later) | `connectors:read`, `connectors:write` |
| DNS and gateways (zones, records, tunnels, gateways, exposures) | `dns:read`, `dns:write`, `dns:delete` |
| Instance templates | `templates:read`, `templates:write` |
| Accounts (the Team directory) | `accounts:read`, `accounts:write`, `accounts:delete` |
| Runners and machines | `runners:read`, `runners:write`, `runners:delete`, `machines:read`, `machines:write`, `runners:shell` once built |
| Topology registry | `topology:read`, `topology:write`, `topology:delete` |
| Automation hosts | `automations:read`, `automations:write`, `automations:delete` |
| Integrations | `integrations:read`, `integrations:write`, `integrations:delete` |
| Audit log | `audit:read` |
| Computer activity | `computer_activity:read` once built |

A guard test fails when an instance-area action in the table belongs to no entity, so a new area cannot ship
without being coverable. `workspaces:create` is a workspace-area verb and stays a role bit.

Targets are a role or a person, never Everyone, because Everyone would hand an instance power to every member of
every workspace. A role is one workspace's role: the rule reaches the members who hold it there. Only the instance
Owner writes these rules, the holder of the Owner role of the default workspace, the one the owner wizard made
(ADR 0088). For person `P` asking for action `A` on area entity `E`:

1. `P` is signed in and active.
2. `P` is an Owner of any workspace: allowed. No instance-area rule can deny an Owner.
3. Base: `A` held through a role in a workspace where `P` is unrestricted, with that workspace's person rule on
   top (ADR 0088, unchanged). A Restricted membership holds no instance area.
4. On `E`: the rules for the roles `P` holds, then the rules naming `P`. Allow or Deny replaces the answer and
   the source; Fallback leaves them. A person rule beats role rules. If `P` holds roles in several workspaces and
   their rules on `E` disagree, Allow wins, the same as holding an action in any one workspace counts today.
5. A Restricted member gains nothing from a role rule; only a rule naming the person allows. A role rule may
   still deny.
6. The source names the rule, or the role and workspace that hold the bit, or the Owner.

Reads that need only membership today (a signed-in member reads instance templates, for one) stay open whatever
a rule on that area says. `/api/auth/me`'s `instance_permissions` lists an action when `P` holds it on at least one
entity, so Settings shows the entry; the entry's page asks per entity.

### Worked example

Workspace Acme. Roles: Engineer grants `tickets:*`, `docs:read`, `docs:write`, `stacks:read`, `deploys:write`;
Contractor grants `tickets:read`, `docs:read`. People: alice is Owner; bob is an Engineer with a workspace
person rule denying `tickets:delete`; sam is a Contractor; lena is a Contractor and a Restricted member.

Rules: project Storefront: Everyone Deny `deploys:write`; role Engineer Allow `deploys:write`; bob Deny
`docs:write`; lena Allow `tickets:read`, `docs:read`. Doc Pricing (in Storefront): role Contractor Deny
`docs:read`; sam Allow `docs:read`; role Engineer Allow `docs:write`.

| Who | Action | On | Steps that set a state | Answer | Source shown |
|---|---|---|---|---|---|
| alice | `docs:read` | Pricing | Owner | Allow | Owner of Acme |
| bob | `deploys:write` | Storefront | role Allow, Everyone Deny, role Engineer Allow | Allow | Allowed on project Storefront for role Engineer |
| sam | `deploys:write` | Storefront | base not held, Everyone Deny | Deny | Denied on project Storefront for everyone |
| bob | `docs:write` | Storefront's other docs | role Allow, bob Deny | Deny | Denied on project Storefront for bob |
| bob | `docs:write` | Pricing | role Allow, bob Deny on the project, role Engineer Allow on the doc | Allow | Allowed on doc Pricing for role Engineer |
| bob | `tickets:delete` | Storefront | role Allow, workspace person rule Deny | Deny | Denied for bob in workspace Acme |
| sam | `docs:read` | Pricing | role Allow, role Contractor Deny on the doc, sam Allow on the doc | Allow | Allowed on doc Pricing for sam |
| lena | `tickets:read` | Storefront | Restricted base not held, lena Allow on the project | Allow | Allowed on project Storefront for lena |
| lena | `docs:read` | Pricing | lena Allow on the project, role Contractor Deny on the doc | Deny | Denied on doc Pricing for role Contractor |
| lena | `tickets:write` | Storefront | Restricted base not held, nothing allows | Not held | Not given to lena on project Storefront |
| lena | anything | project Backoffice (no rules) | holds nothing there | not found | (no explanation: the project does not exist for her) |
| sam | `tickets:write` | Storefront | base not held, nothing sets it | Not held | Not in role Contractor |

Instance areas and computers, in the same workspace. Role Engineer holds no instance bit. Rules: DNS and gateways:
bob Allow `dns:read`, `dns:write`. Computer: sam owns the laptop and wrote a rule naming bob that allows See and
Run commands.

| Who | Action | On | Steps that set a state | Answer | Source shown |
|---|---|---|---|---|---|
| bob | `dns:write` | DNS and gateways | base not held, bob Allow on the area | Allow | Allowed on DNS and gateways for bob |
| bob | `connectors:write` | connector GitHub | base not held, nothing sets it | Not held | Not in role Engineer |
| lena (Restricted) | `dns:write` | DNS and gateways | Restricted holds no instance area, role rules cannot allow her | Not held | Not given to lena |
| alice | `dns:write` | DNS and gateways, with a rule naming alice that denies it | Owner | Allow | Owner of Acme |
| sam | anything | sam's laptop | owner | Allow | Owner of this computer |
| bob | Run commands | sam's laptop | rule naming bob | Allow | Shared with bob by sam |
| bob | Run agents | sam's laptop | the rule does not allow it | Not held | Not shared with bob |
| alice | See this computer | sam's laptop | no rule names alice; the Owner role does not reach a computer | not found | (no explanation) |

## Source and explain

**One evaluator.** `decide` becomes the one function every check calls, and it returns a decision, not a
bool: whether the action is held and the source, a small value of ids (`kind`: owner, role, workspace rule,
entity rule, restricted base; the entity's type and id; the target's type and id; the state). Enforcement
reads only the bool. The explain path calls the same function and resolves the ids to names. Since both run
the same code on the same layers, they cannot disagree; a parity test still holds it
(`TestExplain_AnswersWhatTheCheckAnswers`, ticket 01), over the `list_paging_test.go` fixture's viewers, every
catalog action, and every entity.

**The explain path.** `GET /api/permissions/explain?resource_type=&resource_id=&user_id=` returns, per
action the entity's rules may set (every catalog action for the workspace), `{action, allowed, source}` with
the source's names and a sentence ("Denied on doc Pricing for role Contractor"). Anyone may explain their own
access on what they can open; explaining someone else's takes the entity's manage permission, the same as
listing its rules: the computer's owner for a computer, the instance Owner for an instance area. An entity the
caller cannot open is not found, and for a computer that means anyone it is not shared with.

**Forbidden errors say why** (ticket 01). A `forbidden` answer carries the source sentence, so an agent and a person
both read "docs:write required; denied on project Storefront for bob". A not-found answer never does.

## Performance plan

- **No new statement on the hot path.** The source is ids already in hand. Role and Everyone rules come back
  in the same statement as the person's own rules (`WHERE resource_type = ? AND resource_id IN (...) AND
  (user_id = ? OR role_id = ? OR target = 'everyone')`), memoised under the same commit fence (ADR 0135).
- **Lists stay in SQL.** `ProjectsWith` reads every project rule that can touch the person in its workspace in
  one statement. `DocsWith` adds the docs whose role or Everyone rules turn the project's answer, from one
  read of doc rules for the person, their role in each workspace, and Everyone, bounded by the number of
  rules, not docs. Nothing becomes a per-row check.
- **Guards.** The `TestStatements_*` guards in `server/cmd/access_memo_test.go` stay green with unchanged
  counts, and each slice that widens a read adds a guard that a list with rules on every row costs the same
  statements as one with none (`practices/testing.md` section 10).
- **Numbers.** Ticket 02 and ticket 04 each report `doc_list`, `ticket_list` and a live frame's statement
  count and median time on the heavy copy, production build, before and after.
- **Computers cost one statement.** A computer check is one query keyed by the owner or the named person, read
  fresh each time so a revoke is immediate. A list of computers is that query, never a per-row check.
- **Instance areas ride the existing reads.** Area rules come back with the role reads `HoldsAnywhere` and
  `PermissionsAnywhere` already do, memoised under the same commit fence; `/api/auth/me` costs the same
  statements with rules on every area as with none.
- **Explain is off the hot path.** It may look names up; nothing else calls it.

## Migration plan

Production has real users, so no one's effective access may change.

1. **Ticket 01, no schema change.** Adds `access.Matrix(ctx, db)`: every member × every catalog action × the
   workspace, every project, every doc and every play, as sorted JSON. A test helper runs it on a database
   copy named by `NEXUL_ACCESS_MATRIX_DB`, skipped when unset.
2. **Ticket 02, the table.** A forward-only migration rebuilds `permission_overwrites` with a `target`
   column (`person`, `role`, `everyone`), a nullable `user_id` (cascading on account delete, as today) and a
   nullable `role_id` (cascading on role delete), a check that exactly the right one is set, and a unique
   index over `(resource_type, resource_id, target, coalesce(user_id, role_id, ''))`. Every existing row is
   copied as `target = 'person'`. Indexes name the query each serves.
3. **Ticket 04, dormant Project access.** For a member who is not restricted today, their `project` rows
   start to apply in the new model. A migration trims each such row's allow to what the person already
   holds in that project from their role and workspace rule, so the row adds nothing now and still opens
   that project if they are switched to None later.
4. **The gate.** Before ticket 02 and ticket 04 merge, the matrix is taken from a copy of the production
   database at the commit before and at the branch, and the diff is empty. Each ticket also carries
   `TestMigration_EffectiveAccessUnchanged`: a fixture with every row kind (Owner, From role with dormant
   rows, Restricted, workspace allow and deny, doc creator grants, play exclusions) at the previous schema,
   upgraded, and compared with a golden matrix generated before the change.

5. **Computers and instance areas add rows, not a rebuild.** Their rules are new `permission_overwrites` rows
   with `resource_type` `computer` and `instance_area`; nothing is shared and no area has a rule yet, so there is
   nothing to backfill and no one's access changes. The matrix gains the computer and instance-area rows, so the
   diff stays empty.

Invitations keep their shape: `allow_json`, `deny_json` and `project_access_json` become person rules when
redeemed, as now.

## UI structure

Structure only; the look goes through design mode when ticket 03 starts.

- **One permissions panel**, reused on each entity: a list of targets on one side (Everyone first, then roles
  with a rule, then people with a rule, and an add control for a role or a person), and the selected target's
  permissions on the other, grouped by domain, each with a three-state control: Allow, Fallback, Deny. A
  permission the giver does not hold cannot be set to Allow, and the control says so. At 768px the target
  list sits above the permissions; at 1024 and 1440 they sit side by side. The panel is built from the
  settings kit (`SettingsCard`, `EmptyRow`) and the existing access components where they fit.
- **Check access**: inside the panel, pick a person and read each permission's answer and its source
  sentence; a source that names another entity links to that entity's panel.
- **Where it lives.** Doc: the doc's existing Permissions dialog, replacing `PermissionsForm`. Project: its
  settings page, where the read-only "People with access" card becomes the panel. Play: its settings page,
  replacing the exclusion dialog. Workspace person rules: Team's person dialog, where the Allow levels and
  Deny grid become the same three-state list.
- **Team keeps its view of a person.** Project access in Team edits the same person rules on each project, so
  there is one store with two ways in.
- The role editor stays the place roles are edited; it is the fallback the panel shows.
- **Computers.** The computer row's sharing dialog (personal runners ticket 18) is the panel limited to named
  people and the three computer permissions, with the warning that a grantee's agents and commands run as the
  owner's own OS user. Design mode decides how much of the panel it reuses.
- **Instance areas.** Each Settings entry for an instance area gains an access section for the instance Owner,
  the panel with a role or person picker (ticket 14); everyone else does not see it.

## MCP

No new tool; the surface is at 110 of 111.

- `permission_overwrite_list`: `resource_type` gains `project`, `workspace`, `computer` and `instance_area`;
  each row says its target
  (`target`, `user_id` or `role_id`). A new optional `user_id` returns that person's `effective` answers,
  each with `allowed` and `source`, the explain path above.
- `permission_overwrite_update`: gains `target` (`person`, `role`, `everyone`), `role_ids`, and `state`
  (`allow`, `fallback`, `deny`). `grant` stays and keeps its meaning when `state` is absent, so existing
  callers are unaffected. It is also how a computer is shared and an instance area is opened to someone, with
  the entity's own writer check (the owner, the instance Owner), so sharing needs no tool of its own.
- `account_update`'s `allow`, `deny` and `project_access` keep working; `project_access` is accepted for any
  member, not only a Restricted one.
- Forbidden errors carry the source sentence (above).

## What personal runner tickets 15 to 19 should assume

- **15 and 16 add instance-area bits** (`computer_activity:read`, `runners:shell`). They wait for ticket 01
  here, so the bits land in the evaluator that names sources. Once ticket 13 ships, both are entities in the
  instance-area table (Computer activity, Runners and machines) and the instance Owner may hand them to a role
  or a person with a rule; until then the role is the only way, and nothing in 15 or 16 changes.
- **Sharing a computer is the computer rules of ticket 12.** The grant storage, its checks, the immediate
  revoke and its privacy tests live there. `computer_grants` is no longer a table: a grant is a computer rule
  naming a person. Personal runners 18 and 19 wait for ticket 12 and keep only the runner-side work (the shell
  job gate, the per-run starter identity for Run agents, the attribution line, the warning dialog, and revoking
  queued and running work).
- **17 stays owner-only.** Container logs are not a computer permission and no rule shares them.
- The privacy rules of the personal runners spec hold in full. The checks stay ownership, extended only by the
  owner's own rules naming people; no role, Owner or every-bit role reaches a computer.

## Out of scope

Several roles per member; rules on single tickets, memories or attachments; token scopes (held at the
gateway, ADR 0087), which explain does not cover; time-limited rules; Deny on a computer; Everyone or role rules
on a computer; Everyone rules on an instance area; the phone app's editing (it refreshes on the new event fields
and shows nothing new).

## Risks

- **A quiet change in someone's access** on upgrade. The matrix diff on a production copy is the gate, not a
  review.
- **Nearer entity wins can surprise**: a doc's role rule opens a doc to someone denied on its project. The
  explain line names the doc rule, and the panel shows the inherited answer under each Fallback.
- **Fan-out.** A role or Everyone rule change affects many people at once. `access.grant.changed` gains
  `target` and `role_id` beside its fields, and its audience becomes the role's members or the workspace;
  each socket's frames are re-checked by the memo as today.
- **Width.** A project's rules cover about thirty actions; the panel lists one target's permissions at a time
  so it fits 768px.
- **An instance-area rule can take away someone's instance power in one write.** The Owner is never deniable, so
  the instance Owner cannot lock themselves out, and the explain line names the rule.
- **A computer is the only place the Owner is not trusted.** An Owner who is named by a computer rule holds
  exactly what is named; an Owner who is not named finds nothing. The tests assert both.
- **`ListUsers`** gates the user picker on holding `permissions:write` on any doc through a person row
  (`internal/access/usecase.go:456`); ticket 02 counts role and Everyone rows too.

## Owner's decisions (2026-10-10)

1. **Role and Everyone rules cannot give a Restricted member more.** They can only deny; only a rule naming the
   person grants, so a rule for "Contractor" never shows one client's project to another client. (Evaluation
   order, step 6.)
2. **A deny hides a project.** When the rules on the project itself end in Deny for "Read projects and board
   settings", the project reads as not found, like a private channel. Only a deny on that project hides it; a
   workspace-level or role deny elsewhere hides nothing. Ticket 08 builds it.
3. **The workspace Owner can never be denied anything**, so nothing is locked away from everyone. The one
   exception is a personal computer (decision 5).
4. **Rules apply to everything,** computers and instance areas included, not only the workspace, project, doc and
   play. The build order after the first set is doc folders, stacks, channels, then computers and instance
   areas (tickets 09 to 14).
5. **Personal computers are hidden from everyone,** workspace Owners and the instance Owner included, unless the
   computer's owner shares it with them by name. An Owner or an every-bit role gets no bypass on a computer.
   Reading the record of what ran on computers (`computer_activity:read`) stays as the personal runners spec
   decided: it reads the record only and shows no computer.
6. **Computer rules are the computer sharing feature.** Only the person who installed the runner writes them;
   targets are named people only; the permissions are See this computer, Run agents and Run commands. Run agents
   runs with the starter's own identity and shows "Run by bob using alice's laptop", and the dialog warns that a
   grantee's agents and commands run as the owner's OS user. Revoking is immediate, and only the owner grants or
   revokes.
7. **Each instance area carries rules** for a role or a person, never Everyone, written only by the instance
   Owner: the instance itself, sign-in providers, each connector, DNS and gateways, templates, and the other
   instance areas in the table. The purpose: "bob manages DNS but not connectors" without a custom role.

## Tickets

`issues/01` to `issues/14`, in build order: 01 to 07 the model, the doc, project and play panels and Team; 08 a
denied project is hidden; 09 to 11 doc folders, stacks and channels; 12 computers (sharing); 13 and 14 instance
areas.
