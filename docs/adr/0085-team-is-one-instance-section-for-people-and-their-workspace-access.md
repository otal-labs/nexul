# Team is one instance section for people and their workspace access

Configuration had two lists of the same people: Members, under This
workspace, showed the selected workspace's roster with its roles and the
invitation links; Registered accounts, under the instance sections, showed every
account with its status. Seeing what one person could reach meant switching
workspace and opening both, and giving an existing account access to another
workspace had no path at all short of a new invitation.

Decision: one section, **Team**. An instance administrator finds it under
Instance settings and sees everyone; anyone else who holds `members:write` in a
workspace finds it in Configuration and sees only the workspaces they
manage and the people in them, with no account actions. For an administrator
it lists every registered account with its status and its presence; opening a
person shows, in a dialog, the workspaces they are in with their role and
workspace-wide overrides there, lets the role be changed and the person be
removed from a workspace or added to one the viewer manages, and carries the
account's own disable, reactivate, remove, and restore. Invitations move
with it. Roles stay under This workspace, because a role is defined by one
workspace; Team only assigns them. The page reads one endpoint,
`GET /api/team`, whose accounts come from auth through a gate and whose
memberships, roles, and overrides come from one query, so a person is never
fetched per workspace. Over MCP the same read is `account_list` and the
changes are fields on `account_update`, with no new tool.

The trade-off is against ADR 0024's separation of instance administration
from workspace membership: one page now shows both. Authority stays separate.
Reading the Team and changing an account's status need an instance
administrator; every change to a person's access in a workspace goes through
the workspace member use-cases, which check `members:write` in that
workspace, and the Owner role can still never be given, changed, removed, or
overridden there. An instance administrator who is not a manager of a
workspace sees that workspace's rows read-only, with the reason, and the
server refuses the change if it is sent anyway. The scoping is the server's:
`GET /api/team` and `account_list` answer a workspace manager with their own
workspaces only, so a manager never learns who holds access elsewhere, and a
caller who manages nothing is refused. The cost is one page whose contents
differ by who is looking, and a nav entry that moves between the two groups.

Presence, added the same day: a row shows Online while the person holds an
open live-events socket (the one the presence keeper already counts per user,
with its linger so a refresh does not flap), otherwise when they were last
seen, the latest `last_active_at` across their sessions. That column is
written at most once an hour (ADR 0083), so last seen is accurate to the hour,
and signing out of every device deletes it, so the row then says Signed out.
Opening the first socket and the end of the last one's linger push a live-only
`account.presence_changed` frame that names nobody: an open Team page
refetches and the server's scoping decides who it may see. It is not a catalog
event, since presence is not a domain change and must not reach integrations.
Presence is on the Team and `account_list` only, never the people directory.

Decided 2026-09-29, amending ADR 0024 and ADR 0061. Amended by ADR 0088: the
instance administrator is now whoever holds `accounts:read` in any workspace
(seeing everyone) and `accounts:write` or `accounts:delete` (changing or
removing an account).

The instance sections moved from Configuration to the Settings page on
2026-09-30, where the group is called Instance settings and each entry shows
only to a holder of its permission; Team's scoped variant stayed in
Configuration.
