# Any member reads a workspace's people

Chat, the board, trails, and memory history named people through the
workspace members list, which needs `members:write`. A member without it saw
raw user ids where names belong, could not start a DM, @mention anyone, or
pick a developer. Granting roles a read bit would have made seeing who you
work with a permission an owner can take away, and widening the members list
would have shown every member each other's roles and overrides.

Decision: a second, smaller read beside the members list.
`GET /api/workspaces/{id}/people` answers any member of that workspace, and
only a member, with each person's user id, login, display name, and picture
URL, nothing else: no role, override, email, or account status. The display
name is the one the person set, else their sign-in account's name; clients
fall back to the login. Over MCP the same list rides on `workspace_list`
with an id, so no tool is added. An uploaded picture is served from
`GET /api/people/{userID}/avatar` to the person, anyone sharing a workspace
with them, and a holder of `accounts:read` (an instance administrator before
ADR 0088); its URL carries a hash of the
picture, so a new upload is a new URL and no cache keeps the old one. Saving
a profile publishes `account.profile_updated` with the account id only, and
open clients refetch the list.

The trade-off: two reads of the same roster exist, and the people list is
deliberately not gated by the permission table, so nothing an owner
configures can hide a member's name from the rest of the workspace. The
members list keeps `members:write` for everything about managing people.

Decided 2026-09-29.
