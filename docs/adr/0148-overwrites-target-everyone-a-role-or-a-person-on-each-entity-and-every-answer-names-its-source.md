# Overwrites target everyone, a role or a person on each entity, and every answer names its source

Status: proposed, with the permission overrides effort (`.scratch/permission-overrides/`).

Supersedes ADR 0042 in part (its one table and the Owner bypass stand). Amends ADR 0097, ADR 0088, ADR 0135 and
ADR 0140.

A person's permissions came from their one role plus per-person exceptions stored in four shapes: a doc's
sharing only added or removed an allow, a play's exclusion only a deny, a Restricted member's Project access
held levels with no deny, and Team's workspace-wide override held both as two grids. Nothing could say "every
Contractor on this project", and nothing said why a person could or could not do something, to them or to an
agent.

Decision: the role stays the base, and an entity that is an access boundary carries rules on top of it.

- **Three states.** A rule holds each permission as Allow, Fallback or Deny. Fallback means the rule says
  nothing; it is stored as neither allow nor deny, so the string sets of ADR 0010 are unchanged.
- **Three targets.** A rule names everyone in the workspace, a role, or one person. `permission_overwrites`
  gains a `target` column and a nullable `role_id` beside `user_id`, each cascading on delete.
- **Entities.** Rules apply to everything that is a natural access boundary. Workspace (person rules only, the
  existing Team override), project, doc and play first; then doc folders, stacks and channels; then computers
  and instance areas. A computer takes rules for named people only. An instance area takes a role or a person,
  never everyone, written only by the instance Owner.
- **Order.** Not a member: not found. The Owner: allowed, never deniable by any rule, except on a personal
  computer. Then the role, then the workspace
  person rule, then each entity from the project inward, and on each entity everyone, then the role, then the
  person. The nearer entity wins over the farther one whatever the target, as most-specific-wins did; inside an
  entity the narrower target wins.
- **Restricted members.** A project-area action starts as not held; on project areas, everyone and role rules
  may only deny, and only a rule naming the person allows. Project access is those person rules, now read for
  every member, so one store serves Team and the project's own panel. A project where a Restricted member
  holds nothing stays not found.
- **A denied project is hidden.** When the rules on a project itself end in Deny for `projects:read`, the
  project is not found for that person, like a private channel (ADR 0098), and nothing inside it is reachable.
  A deny elsewhere hides nothing.
- **Computers are private, and rules are how they are shared.** A personal computer is hidden from everyone,
  workspace Owners and the instance Owner included, unless its owner shares it by name: the Owner's bypass and
  every-bit roles do not reach it. Computer rules are written only by the computer's owner, for named people,
  as See this computer, Run agents and Run commands, and take effect on the next check. They are the computer
  sharing ADR 0146 describes. Reading the audit record of what ran on computers stays an instance permission
  that shows no computer.
- **Instance areas carry rules.** The instance itself, sign-in providers, each connector, DNS, templates and
  the other instance areas each take rules for a role or a person, so one area can be handed out without a
  custom role. Only the instance Owner, the Owner role of the default workspace, writes them.
- **Source.** The one evaluator returns whether the action is held and the rule that decided it, as ids.
  Enforcement reads the first; an explain path and a forbidden error turn the second into a sentence ("Denied
  on project Storefront for role Contractor"). Explain and enforcement run the same function, so they cannot
  disagree.
- **Managing rules** keeps each entity's bit from before: `members:write` for the workspace and a project,
  `permissions:write` for a doc, `plays:write` for a play; a computer's owner for a computer and the instance
  Owner for an instance area. Nobody allows, or lifts a deny on, what they do not hold on that entity (ADR
  0088); adding a deny is never limited.

The trade-offs: a rule for a role or everyone changes many people's access in one write, so its live event
reaches the role's members or the workspace. A doc's role rule can open a doc to someone denied on its
project, which the explain line names. Role and everyone rules widen the overwrite read, so they come back in
the same statement as the person's own rules and lists still filter in SQL. A member switched back from None
to From role keeps dormant Project access rows that would start to apply, so the upgrade trims each to what
the person already holds there.

Rejected: a person rule beating every role rule at any depth, which breaks most-specific-wins and today's doc
sharing inside a restricted project; letting role and everyone rules widen a Restricted member's access, which
would show one client's project to every member of a shared role; making the Owner deniable, which can leave a
doc, project or play nobody can manage (ADR 0098 kept the Owner on private channels for the same reason); a
separate explain evaluator, which could drift from enforcement; and one unified manage bit, which would change
who can share docs or exclude people from plays on upgrade.

Rejected for computers: letting the Owner or a role holding every bit see another person's computer, which breaks
the privacy ADR 0146 promises; and Everyone or role rules on a computer, which would share it with people its
owner never named.

Amends ADR 0146: its "checked by ownership rather than permission" now also admits the owner's own rules naming
people, and nothing else. Amends ADR 0097: Project access is the person rules on a project, read for every member. Amends ADR 0088: its
grant rule covers rules for a role or everyone. Amends ADR 0135 and ADR 0140: the memo and `DocsWith` read role
and everyone rules with the person's own. Proposed 2026-10-10; owner's answers on restricted members, hidden projects, the Owner, computers and instance areas recorded the same day.
