# Team is one instance section for people and their workspace access

Configuration had two lists of the same people: Members, under This
workspace, showed the selected workspace's roster with its roles and the
invitation links; Registered accounts, under Whole instance, showed every
account with its status. Seeing what one person could reach meant switching
workspace and opening both, and giving an existing account access to another
workspace had no path at all short of a new invitation.

Decision: one section, **Team**. An instance administrator finds it under
Whole instance and sees everyone; anyone else who holds `members:write` in a
workspace finds it under This workspace and sees only the workspaces they
manage and the people in them, with no account actions. For an administrator
it lists every registered account with its status and a
summary of its workspace access; opening a person shows every workspace on the
instance with their role and workspace-wide overrides there, lets the role
be changed, the person be removed from or added to a workspace, and carries
the account's own disable, reactivate, remove, and restore. Invitations move
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

Decided 2026-09-29, amending ADR 0024 and ADR 0061.
