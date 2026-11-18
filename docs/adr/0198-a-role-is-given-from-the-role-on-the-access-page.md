# A role is given from the role, on the Access page

Amends ADR 0085 and ADR 0148.

What a role allows was on the Access page's Roles tab; who held it was on Team, one person at a time in a dialog that
held every change until Confirm. Making a role and giving it to three people took two pages and three dialogs, and
nothing showed a role's holders beside what it allows. A workspace manager without `accounts:read` had a third place,
Team under Configuration, scoped to the workspaces they manage.

Decision: the Access page's Roles tab becomes **Roles and people**, a list pane of the workspace's roles beside the open
one. A role's page holds who holds it, the ways to give it (Add people, for anyone the viewer's Team shows, adding them
to the workspace if they are not in it yet; Invite, a link that starts out giving the role) and what it allows. Each
holder's menu takes the role, views Nexul as them, or takes them out of the workspace. Two more entries sit under the
roles: No roles, the members holding none, and Compare roles, the matrix of every role side by side. Every change saves
at once. The tab opens to whoever holds `roles:write` or `members:write`; the Access page adds `members:write` to the
permissions that open it.

Team keeps the account: it is an instance section only, for an `accounts:read` holder, and a person's dialog shows
their roles per workspace read-only with a link to that workspace's Access page, beside the account actions, sign-in
accounts, setup progress and View as. Configuration no longer has a Team.

Nothing on the server changes: the page writes through the member use-cases ADR 0085 named (`members:write`, the Owner
role never given or taken, nobody giving a role holding more than they hold), and `account_list` and `account_update`
over MCP are unchanged.

The trade-off: someone who looks after one person across several workspaces now opens each workspace's Access page
instead of one dialog, and adding an existing account to a workspace starts from a role rather than from the person.
Roles are per workspace, so a role-first page is the one place every change in a workspace can be seen together.

Rejected, after building each with seeded data: a People tab on the Access page that opened a person with their roles
(the same one-person-at-a-time path, one page closer), and a people by roles tick grid saved from a footer (dense and
quick to scan, but it showed no role's permissions beside its holders and had no place for invitations).

Decided 2026-11-18.
