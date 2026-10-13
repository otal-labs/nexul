# Permissions on roles: opt-in projects, rules on each entity, and a source for every answer

**Status:** ready-for-agent (slices `issues/01` to `issues/24`; slices 09 to 17 are built on the `permission-switch`
branch and land as one PR)

ADR 0148 (proposed) records the model. The glossary term stays **Permission overwrite**, the rule on one entity for a
role or a person. The UI calls the three states Allow, Fallback and Deny. The decisions come from the permission
rework map (`.scratch/permission-rework/map.md`, tickets 01 to 06 and 08), charted with the owner on 2026-10-11, which
replaced the first version of this spec (2026-10-10). Keep `.scratch/permission-rework/` until slices 06, 07, 08, 12,
13, 14 and 15 merge: its tickets 03, 04, 05 and 08 hold the locked looks those slices build.

## Problem

Today a permission comes from a person's one role, with per-person exceptions stored in four shapes, and nobody can
see why a person can or cannot do something:

- A person holds exactly one role, so "a developer who also does frontend" needs a third role or a per-person
  override in Team.
- Every member sees every project unless they are switched to Restricted, which takes all of them away at once.
  Keeping a project to part of the team means restricting everyone outside it and giving them every other project
  back by name.
- The exceptions disagree on their states. A doc's sharing only adds or removes an allow
  (`internal/access/usecase.go:515-520`), a play's exclusion only a deny (`:507-513`), a Restricted member's Project
  access holds levels with no deny (ADR 0097), and Team's workspace-wide override holds both as two grids
  (`web/src/components/team/TeamOverridesForm.tsx`).
- Nothing names a group of people on a thing: an exception for "every Client on this project" is written once per
  person, and missed for the next Client.
- No surface says where an answer came from. A member who gets "forbidden" cannot tell whether their role, their
  Team override, the project or the doc decided it, and neither can an agent.
- A doc a person cannot open still shows its title in the doc list (ADR 0028), and a direct link answers forbidden,
  which confirms it exists.

The owner's direction (2026-10-11): **roles only**. A person holds several roles, or none. Projects are opt-in: being
in a workspace shows you no project until the project lists one of your roles, or you. Rules on things name a role
or a person, never "everyone", each permission in one of three states, and every answer names its **source**. The
upgrade wipes every per-person permission except a doc creator's own, and an Owner sets who's in each project once.

## What exists today

| Piece | Where | What it does |
|---|---|---|
| The permission table | `internal/platform/permissions/permissions.go:135-174` | One row per domain: its actions and its area (project, workspace, instance, `:117-125`). `GET /api/permissions/catalog` serves it. |
| Roles | `roles`, `workspace_members.role_id`, `migrations/0001_schema.sql:307-323` | **One role per membership**, `NOT NULL`. The Owner role (`is_owner_role`) bypasses every check (ADR 0042). Assumed in 13 sqlc queries, the tenancy use-cases, invitations, `/me`, client-core and five MCP tools (research section 2). |
| Restricted member | `workspace_members.restricted` (migration 0051), ADR 0097 | Sees only the projects with Project access, where the levels are the whole answer; holds no instance area (`internal/access/gate.go:146-157`); reads no public channel (`internal/chat/usecase.go:145-154`). |
| Project access | `internal/tenancy/project_access.go`, ADR 0097 | Allow-only levels per Restricted member per project, as `resource_type = 'project'` rows, read only while restricted. |
| Workspace-wide override | `internal/tenancy/usecase.go:636-683` | Per person, allow and deny, from Team or `account_update`. |
| Doc sharing | `internal/access/usecase.go:357-379` | Per person. The creator gets `CreatorGrant` (`permissions.go:286-287`). Managed with `permissions:write` on the doc. No hold-what-you-give check. |
| Play exclusion | `internal/access/usecase.go:381-390` | A deny of `plays:run` for one person on one play, managed with `plays:write`. No hold-what-you-give check. |
| Overwrite storage | `permission_overwrites(resource_type, resource_id, user_id, allow, deny)` | One table, keyed per person. Four resource types: `workspace`, `project`, `doc`, `play`. |
| Channels | ADR 0087, 0094, 0098 | A public channel is read with membership alone, except by a Restricted member; a private channel by its members and the Owner. |
| Unreadable docs | `internal/docs/usecase.go:303-337`, ADR 0028 | The doc list shows an unreadable doc's title with `can_open: false`; a direct read answers forbidden; search leaves it out. A mention chip shows the title with a lock (`MentionChip.tsx:52`), with no tooltip. |
| The evaluator | `internal/access/usecase.go:95-110` (`decide`), `:162-176` (`has`), `:228-236` (`applyOverwrite`) | Owner, hidden project, role, workspace override, the resource's own row; deny beats allow inside a row. Every check path ends here. Two more evaluators assume one role: `roles.requireAction` and the invitation SQL. |
| The memo and lists | ADR 0135, ADR 0140 | Reads memoised per request until the next commit; `CallerProjects`, `ProjectsWith`, `DocsWith` filter lists in SQL, held by `server/cmd/list_paging_test.go` and the `TestStatements_*` guards in `server/cmd/access_memo_test.go`. |
| MCP | `internal/access/mcp.go`, `internal/mcp/composite/accounts.go`, `internal/tenancy/invitations_mcp.go` | 110 tools against an effective ceiling of 110 (`internal/mcp/surface_test.go:21`): no headroom. |

File:line facts for every area are in `.scratch/permission-rework/research/what-the-rework-touches.md` (the
research). Production on 2026-10-11: 7 workspaces, 8 projects, 20 memberships, one Restricted; 32 doc rows, each a
creator's own `CreatorGrant`, and 2 project rows.

## The model

### Roles

- A member holds **a set of roles**: several, one, or **none**. Zero roles is allowed and means the person sees no
  project and no play; what needs only membership still works (People, public channels, DMs and the inbox).
- **A role only allows.** A role's set is what it grants; Deny lives only in rules on things.
- Several roles combine as the **union** of what they allow.
- **Giving a role** keeps today's rule: you can't give a role that holds more than you hold, judged against the union
  of your roles (ADR 0088). **You can't remove a role you couldn't have given.** There is no role ordering.
- The **Owner role** is one role in a member's set. It is never given or taken in Team. An Owner may hold other roles,
  which changes nothing: the Owner is never denied anything, except on a personal computer.
- An **invitation** gives at least one role per workspace, and nothing per person.
- Deleting a role that people hold is allowed; each holder loses it, and someone left with none holds zero roles.

### States

Each rule holds, for each permission it covers, one of three states:

- **Allow**: held here, whatever the role or the parent says.
- **Fallback**: this rule says nothing; the answer comes from further out. A permission the rule does not mention is
  Fallback.
- **Deny**: not held here, whatever the role or the parent says.

Storage keeps its shape: Allow is membership in `allow`, Deny in `deny`, Fallback in neither, and a permission in both
is refused on write.

### Targets

A rule targets **a role** (every member holding it) or **a person** (one member). There is no Everyone target
anywhere. A computer takes named people only.

### New permissions

| Permission | Key | Area | What it is |
|---|---|---|---|
| **In this project** | `projects:open` | project, rule only | Being in a project. Held only through rules on the project: no role, token scope or integration install holds it, and the role editor does not show it. Its row on a project's panel reads "See this project". It is separate from `projects:read`, which keeps gating repositories, the delete impact and board settings. |
| **Create projects** | `projects:create` | workspace | A verb (ADR 0057). The one action whose area differs from its domain's; the permission table carries an area per action for it. |
| **Manage this project's access** | `projects:access` | project | A verb. Writes the rules on a project. A role may hold it, meaning in every project the holder is in, or a rule may give it on one project. |
| **See this play** | `plays:read` set per play | workspace | Not a new key: today's `plays:read`, which a play rule may now set. |
| **Post in this channel** | `channels:post` | workspace | Slice 20. Granted by migration to every role, integration install and automation holding `chat:write`. |
| **Edit the workspace Interview template** | `memories:template` | workspace | A verb on memories, with the per-action area `projects:create` has. Saving or resetting the workspace's Interview template takes it, and its holders read it on Settings; reading it through a project needs only `memories:read` there (Q32). |

A migration grants `projects:create` to every role holding `projects:write`, and `projects:access` to every role
holding `members:write`, the way migration 0046 did, so nobody gains or loses either on upgrade. Another grants
`memories:template` to every role holding `memories:write`.

### Projects are opt-in

- A person is **in** a project when the rules on that project, for the roles they hold (Allow wins between them) and
  then for them by name, end in Allow of In this project. Nothing else lets anyone in. The Owner is in every project.
- A project a person is not in, and everything in it, is **not found** for them: lists, links, search, live frames,
  MCP. It is checked at the project before the chain walks inward, so a doc's or folder's rule naming the person never
  reaches into a project they are not in.
- Inside a project a person is in, the project-area actions of their roles are the base, and the project's rules and
  every nearer entity's rules apply on top. **A project-area action asked with no project named is not held.** The
  paths that ask one today move to `projects:create`: creating a project, the wizard's repository list and scan, and
  detaching a GitHub account from the workspace (ADR 0147). Installation frames reach `projects:create` holders.
- **Kept out.** A rule naming a person with In this project denied keeps them out of a project their role is in. They
  are listed under "Kept out" on the project's "People in this project".
- **Creating a project** takes `projects:create`. The same write lists the creator by name, allowed In this project
  and Manage this project's access, plus each of the creator's roles allowed In this project. The wizard's
  **"Who's in it"** step shows those roles pre-ticked; the creator is always in. `project_create` over MCP does the
  same, with no new field.
- **Listing a role or a person** on a project writes Allow of In this project and needs Manage this project's access
  on that project. It gives no other bit, so hold-what-you-give asks nothing more: the role's own bits were checked
  when the role was given.
- **The Access page** (`/:workspace/access`, ticket 05 of the rework) shows projects × roles and public channels ×
  roles as tick grids. Its sidebar row shows to anyone holding Manage this project's access. A non-Owner sees only
  the Projects grid, with the projects they are in and may manage; the Owner sees every project and the Public
  channels grid.

### Channels and plays stay workspace-wide

- **Channels.** A public channel's `channels:read` starts held for every member, whatever their roles, because reading
  one takes membership alone (ADR 0087); a private channel's starts held for its members (ADR 0098). Only a Deny in the
  channel's rules takes it away, and then the channel is not found, as a private one is. The Owner sees every channel.
  Hiding a public channel is the Owner's job: from the Access page's Public channels grid in the release that wipes
  (slice 13), and from the channel's own panel later (slice 20), which also adds `channels:post`.
- **Plays.** Each play has its own **See** (`plays:read` set per play). A play someone can't see is gone from the
  plays list, run menus, auto plays and agents' play lists, and its link is not found. Its past runs stay visible to
  whoever started them and to people who can see the play. Play rules set See, Run, Edit and Delete (`plays:read`,
  `plays:run`, `plays:write`, `plays:delete`) and are written with `plays:write` on the play. See ships after the
  switch (slice 22); until then a play rule sets Run only, as today's exclusion does.
- **DMs.** Starting a DM takes `chat:write`, except a DM whose other participants are all Owners of the workspace,
  which takes membership alone, so a person with no role can still message an Owner.

### Entities

The chain a check walks, outermost first:

```
workspace (roles only: no rules on the workspace itself)
├─ project
│  ├─ doc folder (slice 18)
│  │  └─ doc
│  └─ stack (slice 19)
├─ play
└─ channel
outside any workspace
├─ computer (slice 21)
└─ instance area (slices 23 and 24)
```

| Entity | Targets | Permissions a rule may set | Written with |
|---|---|---|---|
| Project | role, person | In this project, and every project-area action | Manage this project's access on the project |
| Doc | role, person | `docs:*`, `permissions:write` ("Manage this doc's access") | `permissions:write` on the doc |
| Play | role, person | `plays:run`; with slice 22, `plays:read`, `plays:write`, `plays:delete` | `plays:write` on the play |
| Channel | role, person | `channels:read`; with slice 20, `channels:post` | the Owner, through the Access page's Public channels grid; with slice 20, `channels:write` on the channel |
| Doc folder (18) | role, person | `docs:*`, `permissions:write`, for the docs in it | `permissions:write` on the folder |
| Stack (19) | role, person | `stacks:*`, `deploys:*` | `stacks:write` on the stack |
| Computer (21) | named people only | See this computer, Run agents, Run commands | only the computer's owner |
| Instance area (23, 24) | role, person | the area's actions (Instance areas below) | only the instance Owner |

Not entities: the workspace itself (a rule there would be a second way to edit a role, and Team's per-person override
is removed); tickets, memories and attachments one by one, which follow their project.

**Hold what you give** applies to every rule: nobody allows, or lifts a Deny on, a permission they don't hold on that
entity (ADR 0088). Adding a Deny is never limited. This is new on doc and play rules.

**A doc's or folder's panel refuses** a role or person who isn't in its project: the add popover lists them disabled
with "Add <name> to <project> first", linking to the project's list.

### Evaluation order

For person `P` asking for action `A` on entity `E` in a workspace:

1. **Not a member** of the workspace: not found. The server's own calls pass. (Unchanged.)
2. **Owner**: `P` holds the workspace's Owner role among their roles. Allowed; no rule on any entity can deny an
   Owner, so nothing is ever locked away from everyone (ADR 0042). The one exception is a personal computer.
3. **Base**: the union of `P`'s roles' sets. Zero roles is the empty set. A public channel's read is held for every
   member, a private channel's for its members.
4. **In the project**: when `E` is a project or inside one, `P` must be in it (Projects are opt-in). Otherwise the
   project and everything in it is not found, and the walk stops.
5. **Project-area actions** count only inside a project `P` is in. With no project named they are not held; Create
   projects is a workspace action.
6. **Each entity on the chain**, from the project inward (project, folder, doc; project, stack), or the play or
   channel itself. On each entity:
   - the rules for the roles `P` holds, merged: **Allow if any of them allows**, otherwise Deny if any denies,
     otherwise Fallback;
   - then the rule naming `P`, which beats the role rules.
   A rule in Allow or Deny replaces the answer and the source; Fallback leaves both.
7. **Unreadable is not found.** When the answer for the entity's read is not held, the entity is not found for `P`,
   never forbidden: links, API and MCP reads, lists, search and live frames leave it out. The read is In this project
   for a project, `docs:read` for a doc, `stacks:read` for a stack, `plays:read` for a play and `channels:read` for a
   channel. A folder shows while it holds a doc `P` can read. Whether a rule denied the read or nothing gave it, the
   answer is the same, so it never tells `P` a rule exists.
8. **Forbidden carries the source**: an action refused on something `P` can read answers forbidden with the source
   sentence. Not found carries none.

Decisions inside the order:

- **The nearer entity wins over the farther one**, whatever the target: a doc rule for a role beats a project rule
  naming the person. Being in a project is the one answer no nearer entity changes (step 4).
- **Inside one entity, the person beats the roles** ("the user is superior"), and **between role rules, Allow wins**.
- **One rule cannot both allow and deny** a permission; the write is refused.

### The source sentence

Every answer has a source: the last rule that set Allow or Deny, or else the base. As ids on the hot path; as a
sentence on the explain path and in a forbidden error. Named entities and roles are links in the panel.

| Source | Sentence |
|---|---|
| The Owner | "Owner of Acme" |
| A rule for a role | "Allowed on doc **Pricing** for role **Frontend**", "Denied on project **Storefront** for role **Client**" |
| A rule naming the person | "Allowed on project **Storefront** for lena", "Denied on project **Storefront** for bob" |
| The roles, no rule | "Allowed by role **Developer**" (the first of `P`'s roles that grants it, in the order they were given) |
| Not held, no rule | "Not allowed: role **Client** doesn't grant it", "Not allowed: no role of bob's grants it", "Not allowed: sam holds no role" |
| Not in the project (to someone who may check another person) | "Not listed on project **Storefront**", "Not found: sam isn't in project **Storefront**" |
| A public channel, no rule | "Read by every member" |
| Several role rules agreeing | the first in the entity's target order |

A not-found answer to the person themselves explains nothing.

### Pills

- A mention pill whose target `P` can't read shows its title (a doc) or its title, status and developer (a ticket),
  with a lock icon, not clickable, and the tooltip "You don't have access". This holds across the project line: a pill
  from a project `P` is not in shows its full content, locked. There is no Request access for now.
- Search stays filtered (ADR 0028). Lists leave unreadable things out; the inert title in the doc list goes.

### The upgrade: a wipe and a one-time setup

Every install that upgrades (Nexul is in beta) wipes every per-person permission. Nothing preserves access, so the
first version's matrix gate and trim migration are dropped.

- **Deleted** from `permission_overwrites`: every `workspace` row (Team's allow and deny), every `project` row (Project
  access, active and dormant), every `play` row (exclusions), every `doc` row other than the doc creator's own, and
  every doc row whose doc is gone.
- **Kept:** roles, who holds which role (as a role set), and each doc creator's own row exactly as stored, now
  `target = 'person'`.
- **Dropped:** `workspace_members.restricted`, Every project, Team's overrides, and the per-person parts of open
  invitations, which keep their role and lose the rest. `workspace_invites` keeps its one role, a set of one on joining.
- **No channel carry-over.** A former Restricted member reads public channels until an Owner hides them on the Access
  page; the upgrade writes no per-person channel rule.
- **Right after the upgrade no project lists anyone**, so only Owners see projects. Each workspace with projects gets
  a setup marker that is unset; a workspace with none starts as set up.
- **The one-time setup** (rework ticket 05): an Owner's first visit redirects once to `/:workspace/access` in its
  setup state, "Who's in each project", with the projects × roles grid and the public channels × roles grid on one
  page and one **Finish setup**, which saves both in one request and sets the marker. Until then the sidebar shows
  **Set up access** to Owners, and members see "Your projects are on their way" with chat working. Afterwards the page
  is the workspace's **Access** page, with the two grids as tabs.
- Things acting as a non-Owner (tokens, automations, auto plays, T3 links, notifications, reattached runs) lose their
  projects until the grid is filled. They answer not found or "may not run here", never a 500.
- The release note is the switch PR's title: "Rebuild permissions on roles: projects are opt-in, people hold several
  roles, and an Owner sets who's in each project again after upgrading".

On production the wipe deletes 2 project rows and one Restricted flag, and keeps the 32 creator rows.

### Computers

A person's computer is the one entity the Owner's bypass does not reach (ADR 0146). Unchanged from the first version
of this spec, without its Everyone and Restricted lines. For person `P` asking for permission `X` on computer `C`:

1. `P` is signed in and active.
2. `P` owns `C` (the person who installed its runner): allowed, everything.
3. Otherwise `C` is found only through a rule that names `P` and allows See this computer, while the owner's account
   is active and `P` shares a workspace with the owner. `P` holds exactly what that rule allows.
4. Nothing else counts: no role, no Owner role, no role holding every bit, no instance Owner. No match is not found,
   never forbidden.

The permissions are See this computer, Run agents and Run commands, outside the role grid; allowing either of the last
two also allows See. Only the computer's owner writes rules, for named people, in Allow or Fallback; Deny is refused.
Revoking is immediate: a computer check reads its rules in its own query and is never memoised. A rule ends when the
two share no workspace or either account is disabled or removed. `computer_activity:read` stays an instance-area
permission that reads the audit record and shows no computer.

### Instance areas

An instance area belongs to no workspace. Today a person holds one by a role bit in any workspace they are in
unrestricted (ADR 0088); after the switch, in any workspace they are in. Each area carries rules so the instance Owner
can hand out one area without a custom role: "bob manages DNS but not connectors".

| Entity | Actions a rule may set |
|---|---|
| The instance itself (URL, upgrades, failed events, public address) | `instance:read`, `instance:write`, `events:read` |
| Sign-in providers | `instance:read`, `instance:write`, for the providers only |
| Each connector (GitHub, Cloudflare, LiveKit, any added later) | `connectors:read`, `connectors:write` |
| DNS and gateways | `dns:read`, `dns:write`, `dns:delete` |
| Instance templates | `templates:read`, `templates:write` |
| Accounts | `accounts:read`, `accounts:write`, `accounts:delete` |
| Runners and machines | `runners:read`, `runners:write`, `runners:delete`, `machines:read`, `machines:write`, `runners:shell` once built |
| Topology registry | `topology:read`, `topology:write`, `topology:delete` |
| Automation hosts | `automations:read`, `automations:write`, `automations:delete` |
| Integrations | `integrations:read`, `integrations:write`, `integrations:delete` |
| Audit log | `audit:read` |
| Computer activity | `computer_activity:read` once built |

A guard test fails when an instance-area action belongs to no entity. Targets are a role (one workspace's role,
reaching its holders there) or a person. Only the instance Owner writes these rules: the holder of the Owner role of
the default workspace (ADR 0088). For `P` asking for `A` on area `E`:

1. `P` is signed in and active.
2. `P` is an Owner of any workspace: allowed, never deniable.
3. Base: `A` held through a role in any workspace `P` is in.
4. On `E`: the rules for the roles `P` holds, in any workspace, merged with Allow winning; then the rule naming `P`,
   which beats them. Allow or Deny replaces the answer and the source.
5. The source names the rule, the role and workspace that hold the bit, or the Owner.

Reads that need only membership today stay open. `/api/auth/me`'s `instance_permissions` lists an action when `P`
holds it on at least one entity.

### Worked example

Workspace Acme. Roles: **Developer** grants `tickets:*`, `docs:read`, `docs:write`, `stacks:read`, `deploys:write`,
`plays:read`, `plays:run`; **Frontend** grants `tickets:read`, `docs:read`, `docs:write`, `plays:read`; **Client**
grants `tickets:read`, `docs:read`. People: alice is an Owner; bob holds Developer and Frontend; lena and sam hold
Client.

Rules:

- Project **Storefront**: role Developer Allow In this project; role Frontend Allow In this project; role Client Allow
  In this project; lena Allow `tickets:write`; bob Deny `docs:write`.
- Project **Backoffice**: role Developer Allow In this project; bob Deny In this project (kept out).
- Doc **Pricing** in Storefront: role Developer Deny `docs:write`; role Frontend Allow `docs:write`; role Client Deny
  `docs:read`; sam Allow `docs:read`.
- Play **Release notes**: role Developer Deny `plays:read`; role Frontend Allow `plays:read`.
- Channel **#backend**: role Client Deny `channels:read`.

| Who | Action | On | Steps that set a state | Answer | Source shown |
|---|---|---|---|---|---|
| alice | `docs:write` | Pricing | Owner | Allow | Owner of Acme |
| bob | In this project | Storefront | role Developer Allow, role Frontend Allow | in | Allowed on project Storefront for role Developer |
| bob | anything | Backoffice | role Developer Allow, bob Deny In this project | not found | (to bob, nothing; to a manager, "Denied on project Backoffice for bob") |
| bob | `deploys:write` | Storefront | roles grant it, no rule | Allow | Allowed by role Developer |
| bob | `docs:write` | Storefront's other docs | roles grant it, bob Deny on the project | Deny | Denied on project Storefront for bob |
| bob | `docs:write` | Pricing | bob Deny on the project; on the doc Developer Deny and Frontend Allow, so Allow | Allow | Allowed on doc Pricing for role Frontend |
| lena | `tickets:write` | Storefront | Client doesn't grant it, lena Allow on the project | Allow | Allowed on project Storefront for lena |
| sam | `tickets:write` | Storefront | Client doesn't grant it, no rule | Not held | Not allowed: role Client doesn't grant it |
| lena | `docs:read` | Pricing | Client grants it, role Client Deny on the doc | not found | (to lena, nothing; her pill shows "Pricing" locked; to a manager, "Denied on doc Pricing for role Client") |
| sam | `docs:read` | Pricing | role Client Deny, sam Allow on the doc | Allow | Allowed on doc Pricing for sam |
| lena | anything | Backoffice | no rule lists Client or lena | not found | (a Backoffice ticket's pill shows its title, status and developer, locked) |
| bob | `plays:read` | Release notes | on the play Developer Deny and Frontend Allow, so Allow | Allow | Allowed on play Release notes for role Frontend |
| lena | anything | Release notes | Client grants no `plays:read` | not found | (gone from her plays list and run menus) |
| lena | `channels:read` | #backend | public, so held; role Client Deny | not found | (to a manager, "Denied on channel #backend for role Client") |
| bob | `channels:read` | #backend | public, so held, no rule for his roles | Allow | Read by every member |

If an Owner unticks Client from sam, sam holds no role: Storefront and everything in it is not found for him; he reads
People, #general and #backend (no rule names his roles any more), his DMs and his inbox; he can start a DM with alice;
his home shows "No roles in Acme yet" with the Owners listed.

Instance areas and computers, in the same workspace. No role holds an instance bit. Rules: DNS and gateways: bob Allow
`dns:read`, `dns:write`; connector GitHub: role Frontend Allow `connectors:read`. Computer: sam owns a laptop and
wrote a rule naming bob that allows See and Run commands.

| Who | Action | On | Steps that set a state | Answer | Source shown |
|---|---|---|---|---|---|
| bob | `dns:write` | DNS and gateways | no role holds it, bob Allow on the area | Allow | Allowed on DNS and gateways for bob |
| bob | `connectors:read` | connector GitHub | no role holds it, role Frontend Allow | Allow | Allowed on connector GitHub for role Frontend |
| bob | `connectors:write` | connector GitHub | nothing sets it | Not held | Not allowed: no role of bob's grants it |
| lena | `dns:write` | DNS and gateways | nothing sets it | Not held | Not allowed: role Client doesn't grant it |
| alice | `dns:write` | DNS and gateways, with a rule naming alice that denies it | Owner | Allow | Owner of Acme |
| sam | anything | sam's laptop | owner | Allow | Owner of this computer |
| bob | Run commands | sam's laptop | rule naming bob | Allow | Shared with bob by sam |
| bob | Run agents | sam's laptop | the rule doesn't allow it | Not held | Not shared with bob |
| alice | See this computer | sam's laptop | no rule names alice; the Owner role doesn't reach a computer | not found | (no explanation) |

## Source and explain

**One evaluator.** `decide` returns whether the action is held and the source as ids (kind: owner, role, rule; the
entity's type and id; the target's type and id; the state). Enforcement reads only the first. `roles.requireAction`
and the invitation SQL read the role set through `access.WorkspacePermissions`, so no evaluator assumes one role.

**The explain path.** `GET /api/permissions/explain?resource_type=&resource_id=&user_id=` returns, per action the
entity's rules may set, `{action, allowed, source}` with names and the sentence. It also returns, per action, the
answer the entity would give the chosen target with its own rule at Fallback, which feeds the panel's line under each
Fallback. Anyone explains their own access on what they can open; explaining another person's takes the entity's
manage permission (Manage this project's access, `permissions:write` on a doc, `plays:write` on a play, the computer's
owner, the instance Owner). An entity the caller can't open is not found.

**A parity test** (`TestExplain_AnswersWhatTheCheckAnswers`) runs explain and the check over the
`list_paging_test.go` fixture's viewers and every catalog action, and keeps running over every later fixture.

**`access.Matrix`**: every member × every catalog action × the workspace, every project, doc, play and channel of a
database, as sorted JSON, run on a copy named by `NEXUL_ACCESS_MATRIX_DB`. Before the switch, every PR that touches the
evaluator or a permission table shows an empty diff on a production copy. At the switch it is the box test's before
and after picture. It lives at the composition root, since storage imports `access` and a channel's read is chat's:
`NEXUL_ACCESS_MATRIX_DB=<copy> go test ./server/cmd -run TestAccessMatrix_OfACopy` writes `<copy>.matrix.json`
(`NEXUL_ACCESS_MATRIX_OUT` names another file), and `TestAccessMatrix_OfTheListParityFixture` pins the fixture's.

## Performance plan

Lists page and filter by access in SQL (AGENTS.md rule 13). Every new read is batched and memoised under the commit
fence (ADR 0135).

- **Roles.** A member's role set is read in 2 statements or fewer, joined, like today's membership and role.
- **Projects.** `ProjectsWith` reads the project rules for the person and their roles across every project of a
  workspace in **one statement per workspace per request**; warm frames read nothing new. `ListProjectPeople` becomes
  one SQL query. The workspace project list uses the `ProjectsWith` batch.
- **Docs.** Role rules come back in the same statement as the person's own (`WHERE resource_type = ? AND resource_id
  IN (...) AND (user_id = ? OR role_id IN (...))`). `DocsWith` stays bounded by the number of rules, not docs, and the
  doc list's no-query path filters by it.
- **Channels.** One batched rule read per workspace per request across the conversation list, unread counts, search,
  voice join and live frames.
- **Plays.** One batched rule read per workspace per request for the plays list, run menus and queued runs.
- **The Access grid.** Its read and its Finish write cost the same statements at 3 projects as at 30.
- **Computers** cost one fresh statement per check. **Instance-area rules** ride `HoldsAnywhere` and
  `PermissionsAnywhere`; `/api/auth/me` costs the same with rules on every area as with none.
- **Explain** is off the hot path.

**Guards** (`practices/testing.md` section 10), named in the slices:

- Every `TestStatements_*` guard in `server/cmd/access_memo_test.go` keeps its count.
- New, each comparing rules on every row with none: `TestStatements_RoleRulesCostTheSameAtAnyLength` (2),
  `…ProjectRulesCost…` (4), `…AccessGridCostsTheSameAtAnyLength` (13), `…FolderRules…` (18), `…StackRules…` (19),
  `…PlayRules…` (22).
- **The fixtures list every non-Owner reader in their projects** from slice 4 on (`access_memo_test.go`,
  `listParityFixture`, `seedPermFixture`), so after the flip no guard passes because a path answered not found.
- `TestStatements_LiveFramesCostOneReadPerPersonOnceWarm` keeps its count with the role set and channel rules.

**Numbers.** Slices 2, 4 and 9 report `doc_list`, `ticket_list` and one live frame, statements and median time, on the
heavy copy with a production build, before and after, in the PR. The target: the statement counts above, and a median
no more than 10% slower than before. A larger regression goes back to the owner before merging.

## Build and release

Slicing is rework ticket 06's, which is authoritative.

- **Slices 01 to 08, on master.** Each changes nobody's access: it shows an empty `access.Matrix` diff on a production
  copy, its migrations only add and copy, and nothing it ships can reach the second half (no route or MCP field writes
  a role rule, several roles, a non-Restricted project rule or a channel rule; no screen of the switch is mounted).
  Every beta cut after one is a working Nexul.
- **Slices 09 to 17, on `permission-switch`.** Cut from master once slices 01 to 05 merge; it lives for days. Each
  slice is a green commit that passes the local CI mirror. Each session starts by rebasing onto master. While it is
  open no other effort works in `internal/access`, `internal/tenancy`, the channel reads in `internal/chat`, or
  `web/src/components/{access,team,member,settings}`. Its migrations take their numbers at the final rebase. It lands
  as **one squash-merged PR** after the box test (slice 17), so no partial wipe reaches a beta.
- **Slices 18 to 24, on master after the switch.** Each adds an ability and changes nobody's access on upgrade.

Rejected: one long-lived branch for everything (the neutral half would go untested by betas and land as one huge
diff), and a feature flag (it can't gate a forward-only migration that deletes rows, and would run two evaluators on
production).

Order: 01 → 02 → 03 → (04, 05); 06 after 01; 07 after 06; 08 any time (its look is locked). Then 09 → 10 → 11 → (12, 13,
14, 15, 16) → 17. Then 18 to 24 in any order; 20 needs 13, 22 needs 14.

## MCP

No new tool: the surface stays at 110 (`internal/mcp/surface_test.go`). The HTTP API only grows (ADR 0082); MCP
parameters change under `practices/mcp.md` section 11.

- `permission_overwrite_list`: `resource_type` gains `project` and `channel` at the switch, then `folder`, `stack`,
  `computer` and `instance_area` with their slices. Each row says its `target` and its `user_id` or `role_id`. An
  optional `user_id` returns that person's `effective` answers with `allowed` and `source` (slice 01).
- `permission_overwrite_update`: gains `state` (`allow`, `fallback`, `deny`) in slice 06, and `role_ids` beside
  `user_ids` at the switch; which list an id is in says its target, so there is no separate `target` input. `grant` stays and keeps its meaning when `state` is absent, on a play
  included. It is how agents list a role or a person on a project, keep someone out, hide a channel, share a computer
  and open an instance area.
- `account_update`: gains `add_roles` and `remove_roles` (child-collection fields, `practices/mcp.md` section 4) and
  loses `role_id`, `allow`, `deny`, `every_project` and `project_access`. Each removed field fails with a schema error.
- `account_list` and `workspace_list` carry the roles held, beside the single fields until the switch, which drops
  them; `workspace_list` names the caller's `your_roles`, since its `roles` lists the workspace's roles.
- `invitation_create` takes `roles` (at least one per workspace) and loses its per-person fields; `invitation_list`
  carries `roles`.
- `project_create` lists the creator and the creator's roles, with no new field.
- `play_list`, `play_run`, `trail_list`, `trail_update`: a play the caller can't see is filtered out or not found
  (slice 22).
- Descriptions are rewritten. `rg -w` finds every removed field and old meaning in the built-in play instructions
  (with a migration rewriting stored copies if one names them), the `nexul-memory` skill (its version moves), and the
  guide's `api-and-tokens.md`.
- Forbidden errors carry the source sentence.

**HTTP keeps every old field answering** (slice 11): `/me` keeps `restricted`, now true for every non-Owner, and sends
`projects` to everyone, so a phone build from before the switch reads per-project permissions correctly; `role_name`
is the roles joined, beside `roles`; member routes take `role_ids` beside `role_id`; `allow`, `deny`, `every_project`
and `project_access` read as empty, and writing a non-empty value is refused with a message naming what replaced it.

## UI

The looks are locked in the rework map's tickets. A slice builds against them, not against nearby code.

- **The permissions panel and the row menu**: `.scratch/permission-rework/issues/03-the-permissions-panel.md`, with
  the owner's changes at its end: the three-state control is **three icon buttons** (check, slash, cross) as a radio
  group, not a select; "Kept out" on a project's list; a doc's panel refuses people outside its project. A dialog with
  Rules | Check access, targets in a column (a scrolling tab row at 768px), the source line under each Fallback, and
  "Permissions…" in every row's right-click and ⋯ menu above Delete. Screenshots: `~/Code/Images/permission-rework/03-panel/`.
- **Several roles on screen**: `.scratch/permission-rework/issues/04-several-roles-on-screen.md`. Role chips in Team
  rows, a roles checklist with "Can't give" in the person dialog and the invitation dialog, Your roles, and the
  zero-roles empty state. Screenshots: `~/Code/Images/permission-rework/04-roles/`.
- **The one-time setup and the Access page**: `.scratch/permission-rework/issues/05-the-one-time-project-setup.md`.
  Screenshots: `~/Code/Images/permission-rework/05-setup/`.
- **The role editor as a dialog**: `.scratch/permission-rework/issues/08-the-role-editor-dialog.md`. Edit and New role
  open a 40rem dialog whose body scrolls under a footer with Save always in view, and leaving with changes asks
  "Discard changes?". A new role can start from a copy of another. Three tabs, **Instance**, **Workspace** and **In
  projects** (replacing "Every project"), each leading with its own bulk row and showing its allowed count; Create
  projects and Manage this project's access are switches; plays' first rung reads **See**; extra verbs sit as
  checkboxes under their area. Screenshots: `~/Code/Images/permission-rework/08-role-dialog/` and `build/08/`.
- **Computers**: the computer row's sharing dialog (personal runners ticket 18) is the panel limited to named people
  and the three computer permissions, with the warning that a grantee's agents and commands run as the owner's own OS
  user.
- **Instance areas**: each Settings entry for an area gains an access section for the instance Owner only (slice 24).

**Copy for a member with no role.** Ticket 04 of the rework locked "no projects, channels or plays" in the zero-roles
empty state and the person dialog's warning, and a sidebar with no conversation sections. The later decision that a
person with no role still reads public channels and DMs wins: the empty state's line reads "Your roles decide what you
see here, so there are no projects or plays for you yet. You can read public channels and message an Owner below, who
can give you a role." (decision 22), the
dialog's warning reads "With no roles, Sam sees no projects or plays in Acme.", the sidebar keeps Channels and Direct
messages when they have rows, and each Owner row in the empty state opens a DM with that Owner. Everything else in
ticket 04 stands.

Every web slice is verified at 768, 1024 and 1440px, passes the react-guide self-review, and adds no motion beyond the
existing locks.

## Out of scope

Token scopes (held at the gateway, ADR 0087), which explain does not cover; time-limited rules; rules on single
tickets, memories or attachments; Deny on a computer; role rules on a computer; Request access on a locked pill; how
the phone shows a project or play that disappeared beyond empty lists; editing permissions on the phone; preserving
anyone's access across the upgrade.

## Risks

- **The wipe is forward-only.** A fix-up after a beta is too late for an install that already upgraded. The box test
  (slice 17) runs on a production copy and a synthetic workspace with every row shape, and proves the rollback once:
  restore the backup and reinstall the previous release.
- **Everything acting as a non-Owner goes dark until the grid is filled**: tokens, automations, auto plays, T3 links,
  notifications. The box test asserts they answer not found or "may not run here", never a 500, and that a running
  play reattaches or ends cleanly.
- **Two widenings on upgrade.** A former Restricted member reads public channels until an Owner hides them, and gains
  any instance bit their roles carry, since every membership now counts. On production the one Restricted client
  holds no instance bit and her workspace's channels are public. The box test's "nothing widens" check allows exactly
  these two.
- **A creator left out of the grid** loses their own docs until listed, because the project is not found first.
- **Nearer entity wins can surprise**: a doc's role rule opens a doc to someone denied on its project. The source line
  names the doc rule, and the panel shows the inherited answer under each Fallback.
- **Fan-out.** A role rule changes many people's access in one write. `access.grant.changed` gains `target` and
  `role_id` beside `user_id`; its audience is the role's holders, and the web and phone refetch `/me` on a frame that
  names no person, the way `role.updated` does.
- **Guards passing for the wrong reason.** Under opt-in an unlisted fixture answers not found cheaply. The fixtures
  list every reader from slice 4 on.
- **The switch branch drifts.** It lives for days, rebases every session, and no other effort works in its areas.
- **No MCP headroom.** Every capability rides an existing tool; the surface test stays at 110.
- **An instance-area rule can take away someone's instance power in one write.** The Owner is never deniable, so the
  instance Owner can't lock themselves out.

## Owner's decisions

**2026-10-10** (first version, still standing)

1. The workspace Owner can never be denied anything, so nothing is locked away from everyone. The one exception is a
   personal computer.
2. Rules apply to everything that is an access boundary, computers and instance areas included.
3. Personal computers are hidden from everyone, workspace Owners and the instance Owner included, unless the owner
   shares one by name. Computer rules are the sharing feature: only the person who installed the runner writes them,
   for named people, as See this computer, Run agents and Run commands; revoking is immediate.
4. Each instance area carries rules for a role or a person, written only by the instance Owner: "bob manages DNS but
   not connectors" without a custom role.

**2026-10-11** (the rework map, ticket 01)

5. Roles only: a rule targets a role or a person. There is no Everyone target.
6. A person holds several roles, or none. Zero roles shows no project and no play; People, public channels, DMs and
   the inbox still work, and a person with no role can still message an Owner. An invitation gives at least one role.
7. A role only allows. Several roles combine as the union; between role rules on the same thing Allow wins; a rule
   naming the person beats every role rule; the nearer thing beats the farther one.
8. Who may give a role keeps today's rule; there is no role ordering.
9. Projects are opt-in, by role or person, with "People in this project" on each project, "Manage this project's
   access" as a permission, and a "Who's in it" step in the wizard pre-ticked with the creator's roles. The creator is
   always in.
10. "Create projects" is a workspace-level ability a role can hold; every other project ability counts only inside
    projects you're in.
11. Channels and plays stay workspace-wide. Public channels are seen by every member; hiding one is the Owner's job,
    with a rule denying a role or a person. Each play has its own See.
12. A doc you can't read is gone from lists and search, and its link is not found. Its pill shows the title with a
    lock and a "You don't have access" tooltip. Pills from a project you're not in show their full content, locked.
    No Request access for now.
13. Permissions are edited by right-clicking a doc, folder, project, channel or play, and from the same menu on ⋯.
14. Every per-person permission is wiped on upgrade, Team's workspace-wide allow and deny included, along with
    `account_update`'s `allow` and `deny`; roles, who holds which role and each doc creator's own rights are kept.
    Owners fill a one-time "Who's in each project" grid that also offers hiding public channels.
15. A person denied See on a project is listed under "Kept out", and a doc's panel refuses someone who isn't in the
    doc's project, offering to add them to the project first.
16. The permissions control is three icon buttons side by side, not a dropdown (rework ticket 03).
17. The role editor opens as a dialog with a Save button; the roles list stays as it is (rework ticket 08).
18. The role editor has three sections, Instance, Workspace and In projects, in that order, each with its own bulk row
    and count; Instance holds the instance-area domains, from the catalog's area. They are tabs inside the dialog, so
    it doesn't grow too tall (2026-10-11).

**2026-10-11** (questions from the switch)

19. Reordering projects in the sidebar takes Create projects (`projects:create`), and orders only the projects the
    person is in; a project hidden from them keeps its place (Q31).
20. Editing the workspace Interview template is its own workspace permission, **Edit the workspace Interview template**
    (`memories:template`), shown in the role editor's Workspace tab. Reading it stays as it is: members read it through
    any project they're in, and a holder of it also opens it on Settings, where no project is named. A migration grants it to every role holding `memories:write`, who could edit it before
    (Q32).

**2026-10-13** (from the upgrade test)

21. The upgrade lifts denies as well as allows: a play exclusion, a doc deny or a Team deny is gone after it, so a
    non-Owner can briefly hold an action they were denied before. This is the third accepted widening, beside a former
    Restricted member reading public channels and an instance bit their roles carry. It follows from starting over;
    the release notes say permissions were rebuilt, and Owners add back any Deny they need.

**2026-10-13** (from channel posting, slice 20)

22. A member with no role is read-only in public channels: they read them but don't post, since posting takes
    `channels:post` from a role. Their DMs still work, so they can message an Owner.
23. A bot posts only while its creator may post in its channel. When they can't, because they left, a rule denies
    them, or they hold no role, the bot's row and the sender's 403 say "Stopped: <creator> can't post in #<channel>".

**Technical calls the owner delegated (2026-10-11)**

- In this project is its own permission, separate from `projects:read`.
- Hiding public channels ships in the release that wipes, because Restricted also hid them. The one Restricted client
  is wiped like everything else and reads the public channels until an Owner hides them.
- Open invitations keep their role and lose any per-person parts.
- An Owner may hold other roles, which changes nothing.
- A play's See is `plays:read` set per play, and a run's starter always sees their run.
- Hold-what-you-give is added to doc and play rules, and you can't remove a role you couldn't have given.
- The Access page has its own sidebar row.
- The build order and the switch branch (rework ticket 06).
- `memories:template` is a verb on the memories domain with a workspace area, as `projects:create` is on projects: the
  template is memories', and a `workspace_templates` domain would have carried a `read` nobody needs and named every
  workspace template while gating one.
- A pending login invite (`workspace_invites`) whose role is deleted is deleted with it: an invite gives its one role and
  nothing else, as an open link keeps its role, so with the role gone there is nothing left to give.
- Made while rewriting this spec: the keys `projects:open`, `projects:create` and `projects:access`; the paths that
  asked a project-area action with no project move to `projects:create`; a DM with Owners only takes membership alone;
  the zero-roles copy above; the creator's doc row is kept as stored; no channel carry-over.

## Tickets

`issues/01` to `issues/24`, one per slice of rework ticket 06, each one working session (03 and 09 may take two on
one branch):

- Before the switch, on master: 01 sources, 02 role targets, 03 role sets, 04 project rules read for every member, 05
  channel rules read, 06 the panel on docs and plays, 07 the row menu and locked pills, 08 the role editor dialog.
- The switch, on `permission-switch`: 09 the flip to opt-in, 10 the wipe, 11 the HTTP and MCP contract, 12 several
  roles on screen, 13 the Access page and setup, 14 People in this project and "Who's in it", 15 the role editor's
  new sections, 16 client-core and the phone, 17 the upgrade test and the PR.
- After the switch, on master: 18 folders, 19 stacks, 20 channel rules and posting, 21 computer sharing, 22 See on
  each play, 23 instance-area rules, 24 their panel.
