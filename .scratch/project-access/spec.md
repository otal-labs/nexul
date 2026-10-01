# Spec: Project access and private channels

**Status:** ready-for-agent
**Map:** [map.md](map.md), decisions in `issues/01` to `07`
**ADRs:** 0097 (Restricted members), 0098 (private channels), 0099 (memories
are project-scoped only)

## Problem

A workspace member sees every project in it. A client giving feedback on one
OTAL project has to be added to OTAL to work there, and then sees all of
OTAL's projects, names included. A role cannot narrow that: ADR 0087 made the
project list something every member reads. Channels have the same shape:
every member reads every channel, so there is no place to talk that the
client should not see.

## The model

- **Restricted member.** A membership is either unrestricted (today's
  behaviour) or restricted. A Restricted member sees only the projects they
  hold Project access to; every other project is invisible, name included,
  and a direct link reads as not found. Projects created later stay hidden
  from them. The Owner is never restricted.
- **Every project row.** The UI never says "restricted". A person's access in
  a workspace shows an **Every project** row set to *From role* (the role's
  project areas apply on every project, new ones included) or *None* (a
  Restricted member). Every existing member upgrades as From role.
- **Project access.** Under None, each project has its own level per project
  area (None, Read, Write, Delete, plus the area's verbs, the same editor as
  roles, ADR 0078). For a Restricted member it is the whole answer inside
  that project; the role adds nothing there. A doc's own sharing still sits
  on top of the project's level.
- **Areas.** Every permission domain has one area:

  | Area | Domains | Restricted member |
  |---|---|---|
  | project | tickets, docs, memories, stacks, deploys, pull requests (`repos`), code reviews, repositories, attachments, permissions (doc sharing), projects and board settings | Project access per project |
  | workspace | chat, channels, voice, notifications, mentions, plays, members and invites, roles, workspaces | role and workspace-wide overwrite, as today |
  | instance | runners, machines, topology, DNS, connectors, integrations, automations, events, audit log, accounts, instance settings | nothing, whatever the role carries |

  Projects keep one row: `projects:write` both creates and edits. Creating
  has no project, so it answers from the Every project levels, and a
  Restricted member never creates a project. Stacks belonging to no project
  stay instance-level.
- **Private channel.** A text or voice channel only its members see and
  read. Switchable both ways with `channels:write`, the bit that renames a
  channel. `#general` is always public. The Owner sees every private channel, messages included.

## Behaviour per surface

### Server checks

The resolver, after the Owner bypass, for a restricted membership:

- a workspace-area action answers from the role and workspace-wide
  overwrite;
- an instance-area action is refused, in the workspace and anywhere;
- a project-area action answers from that project's access row alone, and is
  refused when asked with no project;
- "may open this project" (the project list, a project's own read, its board
  settings, project-scoped live frames) means holding a row with anything in
  it;
- a doc check applies the project layer, then the doc's own overwrite.

Unrestricted members and server calls answer exactly as today.
`RequireProject` (`internal/access/gate.go`) is the seam; tickets, docs,
deploys, stacks, repositories, reviews, mentions, and the live audience
already route through it. The project list and a project's own read move
from workspace membership to it. Tokens and paired agents act as their
creator and inherit the limit (ADR 0087).

Instance-level checks (`HoldsAnywhere`, `PermissionsAnywhere`, so
`instance_permissions` on `/api/auth/me`) skip restricted memberships. A
person restricted in A and unrestricted in B gets instance areas from B only.

"Nobody grants what they don't hold" (ADR 0088) extends to Project access:
a giver may set on a project only levels they hold on it, and setting Every
project to From role, or adding a project area to a role, needs those levels
without a project, which only an unrestricted giver has. Redeeming an
invitation re-checks every level.

### Storage

- `workspace_members.restricted`, default 0.
- Project access is a `permission_overwrites` row with `resource_type =
  "project"`, `resource_id` the project, `allow` the levels expanded to
  actions, `deny` empty (ADR 0042's one table). Rows are read only while the
  member is restricted. Switching to None shows what is stored (nothing the
  first time, so every project starts at None); switching back to From role
  keeps the rows, so a round trip loses nothing.
- Deleting a project deletes its rows. Removing someone from a workspace
  deletes their rows for that workspace's projects. Deleting a user cascades
  as today.

### Lists, search, pickers

- The project list, the project switcher, search, the @ picker, the board,
  labels, PR counts, and every cross-project list leave out what the caller
  cannot open. They already drop unreadable rows per row through
  `RequireProject`; batching per person is the lever if a restricted
  member's lists get slow.
- People (ADR 0086) is unchanged: a Restricted member still sees every
  member's name and picture, and everyone sees theirs.
- A ticket's developer and tester pickers leave out Restricted members who
  cannot open its project, from a server list of the people who can open a
  project. Setting one of them as developer or tester is refused as invalid.
- An @mention of such a person stays in the text and creates no notice
  (a notice only goes to someone who may read its subject, ADR 0087).
- A link to a hidden ticket or doc shows the generic section chip and opens
  to not found. Already true; no change.
- A play's excluded-projects list keeps ids of projects the editor cannot
  see when saved, and never shows their names.

### Notices, inbox, trails

Existing notices, inbox entries, and agent trails that point at a project
now hidden from their reader are filtered at read time, unread counts
included; the rows are kept, so access given back brings them back. A
notice's stored subject title never reaches someone who cannot open the
subject.

### Plays

Running a play needs all three: `plays:run` from the role (and no per-play
exclusion of the person), the play not excluded from the target's project,
and the target readable through Project access. Seeing and managing plays
stays a workspace area.

### Attachments

An attachment is read, uploaded, and deleted with its owner: a ticket's
through the ticket, a doc's through the doc, a memory's through the memory,
a conversation's through reading the conversation. Ticket and conversation
attachments today need only a signed-in user, which would let a hidden
project's or a private channel's file be fetched by id.

### Live socket and events

- The socket re-checks every frame against the same read, so frames for a
  project stop the moment access goes.
- A Project access change publishes `access.grant.changed` with
  `resource_type: "project"` (catalog enum widened to `doc`, `play`,
  `project`; SDK events regenerated; automation scope resolves the
  project's workspace). Its frame reaches the person and holders of
  `members:write` in that workspace.
- Switching Every project publishes `workspace.member.updated`, which
  exists.
- On either, the web invalidates the workspace-me, projects, board,
  tickets, docs, memories, and conversations queries, and the Team and
  People-with-access reads.
- A page whose project goes away while open shows the revoked state (Web).

### Instance-level areas

A restricted membership opens no instance area: runners, machines, the
topology, DNS, connectors, integrations, automations (including the
decisions check), events, the audit log, accounts, instance settings, and
creating workspaces. Their sidebar entries stay hidden because the server
refuses them.

### Memories

Memories are project-scoped only (ADR 0099). A memory always names a
project; the use-case refuses one without. Every memory read, write, clone,
version, attachment, and live frame checks the memory's project through
`RequireProject`. The Clone dialog loses its Workspace destination. A
project's list no longer prepends workspace memories, and an Agent turn's
index is the project's memories only; a plain chat with no ticket or doc
carries none.

### Private channels

- **Members.** A private channel's members are its participant rows (the
  table DMs use). Anyone in it adds people from the workspace's members;
  removing someone else takes `channels:write`; anyone may leave, except the
  last member.
- **Switching.** To private: the switcher is kept, and names who else stays;
  everyone else loses it at once. To public: member rows are dropped and the
  history becomes readable by the whole workspace; the confirmation says so.
  `#general` refuses both. Voice channels follow the same rules.
- **Reads.** A private channel the caller is not in reads as not found in
  lists, unread counts, links, messages, attachments, voice join and
  occupancy, and live frames. The Owner bypasses.
- **Restricted member's chat.** DMs (both ways, with anyone), the private
  channels they are in, and the ticket, doc, and interview threads they can
  read through Project access. No public channel.
- **Events.** Membership changes and switches publish
  `chat.conversation.members_changed` (new catalog topic: conversation,
  workspace, `private`, added and removed user ids, actor), reaching the
  channel's readers and everyone it removed, ids only. A switch to private
  names everyone who lost it as removed. A private channel's
  `chat.conversation.deleted` carries its member ids and reaches only them
  and the Owner.

### Invitations

An invitation's per-workspace grant carries Every project (From role or
None) and, under None, the Project access per project. The client lands
restricted with no window of seeing everything. Redeeming re-checks that the
giver still holds every level given, skips projects deleted since, and, as
today, never changes an existing membership (ADR 0061).

### Project delete

Deleting a project removes everyone's access to it (there is no archive).
The delete impact gains the Restricted members who lose access, by name, and
the confirmation names them: a signal, not a gate.

### Team, roles, project settings (web)

The visual treatment of every level control follows ticket 07 (being
redone); this section fixes what appears where.

- **Role editor:** levels split under two microheaders, Workspace (with the
  instance areas) and Every project ("Applies to members whose Every project
  is From role.", with its own Every area row). Rendered from the catalog's
  `area` field.
- **Team dialog**, per workspace, under the role select: the Every project
  row (From role / None, with a one-line consequence), and under None one
  row per project the viewer can open, with a search field and a count,
  each showing a summary of what is granted and opening (one at a time) into
  the project areas' levels with an Every area row on top.
- **Invitation dialog:** the same block under each workspace's role.
- **Project settings, People with access:** the Restricted members who can
  open the project with a summary of their levels, "Change access from
  Team." and an empty row; shown to holders of `members:write`.
- **Delete confirmation:** a line naming the Restricted members who lose
  access.
- **Revoked page:** the body becomes the empty state "You no longer have
  access to this project", "Someone changed your access. Projects you can
  still open are in the sidebar.", and "Go to Home".
- **Private channel:** in the channel's settings, a Private channel row with
  a switch; turning it on opens "Who stays in #channel?" with the switcher
  checked and fixed; a private channel lists its members with a count, Add
  people, removal per row, and Leave channel.
- **Restricted member's sidebar:** only their projects, only their private
  channels (with a lock), and DMs.
- Every change confirms with a toast naming the person and the project, or
  the channel.

### What the client is told

`GET /api/workspaces/{id}/me` adds `restricted` and, when on, `projects`:
`[{project_id, actions}]`. The web hides what the server refuses.

### MCP

No tool is added; the ceiling does not move.

- `account_list`: each membership gains `every_project` (`role` or `none`)
  and, under `none`, `projects`: `[{project_id, project_name, allow}]`.
- `account_update`: a `workspaces` entry gains `every_project` and
  `project_access`: `[{project_id, allow}]`, replacing each named project's
  levels; an empty `allow` takes the project away; omitted fields keep
  their value.
- `invitation_create`: a `grants` entry gains the same two fields.
- `project_get`: gains `access`, the Restricted members with access and
  their actions, filled only for a holder of `members:write` in the
  workspace; `delete_impact` names the Restricted members who lose access.
- `conversation_list`: each channel gains `private` and, when private,
  `member_ids`; private channels the caller is not in are absent.
- `conversation_update`: retitled "Update channel"; gains `private` (with
  `member_ids` naming who stays when going private), `add_member_ids`, and
  `remove_member_ids` (yourself to leave).
- `role_update`: description says a role's project areas reach only members
  whose Every project is From role.
- `memory_list` and `memory_create`: the workspace option goes; a project is
  required.
- Server instructions gain one line: a person may see only some of a
  workspace's projects, so read not found as access, not a fault.
- The `nexul-memory` skill drops workspace scope; its content-derived
  version changes, so every computer refreshes through `skill_get`.

## Migrations

Numbered, forward-only, each with an upgrade test from the previous schema
(`migrateBefore`). Numbers are pre-assigned so parallel branches do not
collide; take the next free number before merging if master has gained one.

- **0049** (ticket 08): `workspace_members.restricted INTEGER NOT NULL
  DEFAULT 0`; `invitation_grants.restricted` and
  `invitation_grants.project_access_json`. Every existing member and
  invitation upgrades as From role with nothing taken away.
- **0050** (ticket 09): deletes every memory with no project, with its
  versions and attachments deleted explicitly, not left to cascades. No
  table rebuild for NOT NULL; the use-case enforces it.
- **0051** (ticket 10): `conversations.private INTEGER NOT NULL DEFAULT 0`.
  Members reuse `conversation_participants`. Every existing channel stays
  public.

## Open for the owner

Each has the reading the tickets build to until the owner says otherwise.

- **A doc shared with a Restricted member in a hidden project.** Ticket 03
  puts the doc's overwrite on top of the project layer, so an allow on the
  doc opens that one doc even when the project is hidden. Built as written;
  the project's name may then show on the doc page.
- **A Restricted member with `channels:write`.** Creating a channel or making
  one public leaves them with a channel they cannot see. Built as allowed;
  the result simply drops from their list.
- **Creating a channel private.** Ticket 04 settled switching, not creation;
  the create form stays public-only and the switch follows.
- **The phone app.** It reads the same server, so every filter reaches it
  with no change; showing a private channel's lock or the revoked state
  there is not assigned to any ticket.

## Out of scope

- Adjusting one project for a From role person (say, denying Sam deletes on
  one project). Switching them to None covers it; per-project tweaks for
  everyone else are a separate effort.
- Filtering runners, machines, or the topology canvas per project. A
  restricted membership opens no instance-level area at all.
- Hiding people's names. Every member still reads the workspace's people
  (ADR 0086); project names and contents are the leak, names are not.
- A tool to create a channel.
- Editing Project access from project settings; it is read-only there.
