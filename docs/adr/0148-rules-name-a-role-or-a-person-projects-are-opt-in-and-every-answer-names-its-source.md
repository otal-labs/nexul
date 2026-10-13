# Rules name a role or a person, projects are opt-in, and every answer names its source

Status: accepted.

Supersedes ADR 0097 (Every project and the Restricted member) and ADR 0042 in part (its per-person rows and the
workspace-wide override; the one table and the Owner bypass stand). Supersedes ADR 0028 in part (the doc list's inert
title, and the pill rule). Amends ADR 0061, 0085, 0087, 0088, 0098, 0102, 0135, 0140 and 0146.

A person held one role, plus per-person exceptions stored in four shapes: a doc's sharing only added or removed an
allow, a play's exclusion only a deny, a Restricted member's Project access held levels with no deny, and Team's
workspace-wide override held both as two grids. Every member saw every project unless switched to Restricted, which
took them all away at once. Nothing could say "every Client on this project", and nothing said why a person could or
could not do something, to them or to an agent.

Decision: roles are the only grouping, projects are opt-in, and things carry rules for a role or a person.

- **Roles.** A member holds a set of roles, or none. A role only allows; several combine as the union. Zero roles
  shows no project and no play; what needs only membership (People, public channels, DMs, the inbox) still works, and
  a DM with the workspace's Owners takes membership alone. Giving a role keeps ADR 0088's rule, and a role you
  couldn't give you can't take away. An invitation gives at least one role and nothing per person.
- **Three states, two targets.** A rule holds each permission as Allow, Fallback or Deny, stored as ADR 0010's string
  sets with Fallback in neither. It names a role or a person: `permission_overwrites` gains `target` and a nullable
  `role_id` beside `user_id`, each cascading. There is no Everyone target and no rule on the workspace itself.
- **Projects are opt-in.** A non-Owner is in a project only when its rules allow In this project (`projects:open`) for
  a role they hold or for them by name; it is held only through project rules. A project someone is not in, and
  everything in it, is not found, checked before the chain walks inward. A person denied In this project is kept out
  whatever their roles. A role's project-area actions count only inside projects the holder is in; Create projects
  (`projects:create`) is a workspace action, and Manage this project's access (`projects:access`) writes a project's
  rules. Creating a project lists its creator and the creator's roles in the same write.
- **Channels and plays stay workspace-wide.** A public channel is read by every member unless a rule denies a role or
  a person; each play carries its own See (`plays:read` per play), and a run's starter always reads their run.
- **Order.** Not a member: not found. The Owner: allowed, never deniable, except on a personal computer. Then the union
  of the roles; then, for anything in a project, being in it; then each entity from the project inward, and on each
  the rules for the person's roles, Allow winning between them, then the rule naming the person, which beats them. The
  nearer entity beats the farther one whatever the target.
- **Unreadable is not found.** Something whose read a person does not hold is left out of links, lists, search and
  frames, and a direct read answers not found, never forbidden. A mention pill still shows what it points at, locked
  and inert with "You don't have access", across the project line too: a doc's title, a ticket's title, status and
  developer.
- **Source.** The one evaluator returns whether the action is held and the rule that decided it, as ids. Enforcement
  reads the first; the explain path and a forbidden error turn the second into a sentence ("Denied on project
  Storefront for role Client"). `roles.requireAction` and the invitation SQL read the same role set.
- **Managing rules.** Manage this project's access for a project, `permissions:write` for a doc or folder,
  `plays:write` for a play, `channels:write` for a channel, `stacks:write` for a stack; a computer's owner for a
  computer and the instance Owner for an instance area. Nobody allows, or lifts a Deny on, what they don't hold there
  (ADR 0088), on doc and play rules too; adding a Deny is never limited.
- **Computers are private, and rules are how they are shared.** The Owner's bypass and every-bit roles don't reach a
  personal computer. Only its owner writes rules on it, for named people, as See this computer, Run agents and Run
  commands, in force on the next check. They are the computer sharing ADR 0146 describes.
- **Instance areas carry rules** for a role or a person, written only by the instance Owner, so one area can be handed
  out without a custom role. A role bit in any workspace still holds an area.
- **The upgrade wipes.** Every per-person permission goes: Team's allow and deny, Project access, the Restricted flag,
  play exclusions and doc sharing, except each doc creator's own row. Roles and who holds them stay. No project lists
  anyone afterwards, so an Owner fills a one-time "Who's in each project" grid, which becomes the workspace's Access
  page. Everything that changes access lands as one release.

The trade-offs: every install loses its per-person access at once and depends on an Owner to rebuild it, so the
release is tested by upgrading a production copy and proving the rollback first. A former Restricted member reads
public channels until an Owner hides them, and gains any instance bit their roles carry. A role rule changes many
people's access in one write, so its live event reaches the role's holders. Every member's project check now reads
the project rules a Restricted member's did, batched once per workspace per request. A doc's role rule can open a doc
to someone denied on its project, which the source line names.

Rejected: an Everyone target, which hands every new member whatever it allows; keeping access across the upgrade by
translating Restricted members and Every project into rules, which carried the old model's shapes into the new one;
`projects:read` as what lets a person in, since it gates board settings a role may lack while its holders work in the
project; a table of project members, which would repeat the overwrite store for one bit; a role ordering, which the
hold-what-you-give rule already covers; a feature flag around the switch, which can't gate a migration that deletes
rows; making the Owner deniable, which can leave something nobody can manage; and a separate explain evaluator, which
could drift from enforcement.

Rejected for computers: letting the Owner or a role holding every bit see another person's computer, which breaks the
privacy ADR 0146 promises; and role rules on a computer, which would share it with people its owner never named.

Proposed 2026-10-10 with an Everyone target and Restricted members kept. Rewritten 2026-10-11 for the owner's
decisions on the permission rework map: roles only, several roles or none, opt-in projects, See on each play, and a
wipe with a one-time setup. Accepted 2026-10-13, when the switch was tested by upgrading a copy.
