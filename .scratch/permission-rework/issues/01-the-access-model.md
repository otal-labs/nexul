# 01 — The access model after the rework

**Type:** grilling
**Status:** resolved
**Blocked by:** None

## Question

What replaces "Every project", Restricted members and the Everyone target: who sees which project, how roles
combine, what happens on upgrade, and what a hidden doc or play looks like?

## Answer

Decided with the owner in three rounds on 2026-10-11, the first two on top of that morning's three decisions.

**Targets and roles**
- A rule targets **a role or a person**. There is no Everyone target anywhere.
- A person holds **several roles**, or none. **Zero roles is allowed** and means they see nothing, which also
  makes a missed permission obvious in E2E. Invitations still give a role on joining.
- A role only **allows**: a role's set is what it grants, and deny lives only in rules on things. The owner's
  example: deny role Frontend `channels:read` on #backend.
- Several roles combine as the union of what they allow. **Between role rules on the same thing, Allow wins.**
  A rule naming the person beats every role rule ("the user is superior"). The nearer thing still beats the
  farther one, as in the original spec.
- Who may give a role keeps today's rule: you can't give a role that holds more than you hold. There is no
  Discord-style role ordering.
- The workspace Owner is never denied anything, except a personal computer, unchanged.

**Projects are opt-in**
- Opt-in: being in a workspace shows you no project. You see a project only once it lists one of your roles,
  or you by name. A project you're not in, and everything in it, is **not found**.
- Each project's settings show a plain **"People in this project"** list of roles and people. It is a rule
  underneath, and opening an entry shows the full Allow / Fallback / Deny panel.
- **"Manage this project's access"** is a new permission. A role can hold it workspace-wide, or a rule can give
  it on one project. The creator holds it on their own project, and hold-what-you-give still applies.
- A new project's wizard gets a **"Who's in it"** step, pre-ticked with the creator's own roles. The creator is
  always in.
- Only projects are opt-in, because they are the domain with quantity. The others stay workspace-wide:
  - **Channels:** public channels are seen by every member, and private channels are private. Hiding a public
    channel is the Owner's job, with a channel rule that denies a role or a person. A client sees public
    channels unless a rule says otherwise.
  - **Plays:** each play has its own **See** permission, because some plays shouldn't be seen at all. A play
    you can't see is gone from the plays list, run menus, auto plays and agents' play lists, and its link is
    not found. Past runs stay visible to whoever started them and to people who can see the play.

**Docs and editing**
- A doc you can't read is gone from lists and search, and its direct link is **not found**. Its **mention pill
  still shows the title with a lock icon**, not clickable, with a "You don't have access" tooltip. There is no
  Request access flow for now.
- Permissions are edited by **right-clicking** a doc, doc folder, project, channel or play. The same menu sits
  on its ⋯ button for keyboard and touch.

**Upgrade: a clean start**
- On every install that upgrades (Nexul is in beta), every per-person permission is **wiped**:
  - Team's workspace-wide per-person allow and deny
  - project access levels
  - the Restricted flag
  - doc sharing with a person
  - play exclusions
- **Kept:** roles, who holds which role, and each doc creator's own rights to their doc.
- Right after the upgrade, **no project lists anyone**, so only Owners see projects. An Owner's first visit
  shows a one-time **"Who's in each project"** grid of projects × roles, and the same pass offers hiding public
  channels from chosen roles. The release notes say "permissions were rebuilt; set up project access again".
- Nothing preserves access, so the original spec's matrix gate and its trim migration are dropped.

**Unchanged from the original spec:** three states per permission, the source sentence on every answer and
forbidden error, computers (owner-only, shared by name), instance areas (role or person, written by the instance
Owner, never Everyone), the panel's structure, and no new MCP tool.

**Round 4 (2026-10-11), after the code research**
- **Zero roles hides projects and everything in them.** Things that need only membership still work: People,
  public channels, DMs and the inbox. A person with no role can still message an Owner.
- **"Create projects" is a workspace-level ability** that a role can hold. The creator is in the new project, with
  their pre-ticked roles. Every other project ability counts only inside projects you're in.
- **Team's workspace-wide per-person allow and deny is removed** by the same migration that wipes, along with
  `account_update`'s `allow` and `deny`. To give one person more, give them a small role. Person rules stay only on
  things: projects, folders, docs, stacks, channels, plays, computers and instance areas.
- **Mention pills from a project you're not in show their full content** (a ticket's title, status and developer;
  a doc's title), with a lock icon, and aren't clickable.
- **Kept out:** a person denied See on a project is listed under "Kept out" on that project's list.
- **A doc's panel refuses** someone who isn't in the doc's project, and offers to add them to the project first.
- Decided without the owner, as technical calls:
  - "In this project" is its own permission, separate from `projects:read`, which already gates repositories and
    the delete warning.
  - Hiding public channels ships in the release that wipes, because Restricted also hid them.
  - Open invitations keep their role and lose any per-person parts.
  - An Owner may hold other roles, which changes nothing.
  - A play's See is `plays:read` set per play, and a run's starter always sees their run.
  - Hold-what-you-give is added to doc and play rules.
  - You can't remove a role you couldn't have given.
  - An invitation needs at least one role.
  - The Access page has its own sidebar row.
