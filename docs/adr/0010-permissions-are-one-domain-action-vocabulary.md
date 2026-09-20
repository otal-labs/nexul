# Permissions are one domain × action vocabulary stored as string sets

Roles held a fixed list of thirteen bespoke actions (`read`, `edit`,
`archive`, `manage_roles`, `automation.update`, ...) packed into a `uint64`
mask, while integration and automation tokens were checked against a separate
`<domain>:<read|write|delete>` grid built from the API's own domains. Two
vocabularies meant two catalogs to keep in step, a web form per vocabulary,
and no way to say "this token may do exactly what this role may do". The
token grid alone is 60 actions across 26 domains, so a 64-bit mask had no
headroom left for roles to adopt it.

Decided 2026-09-11: the domain × action grid is the only permission
vocabulary. It lives in one table in `internal/platform/permissions`, is
served by `GET /api/permissions/catalog` (and unchanged by
`GET /api/integrations/scopes`), and every actor is checked against the same
values: a user through their role and per-resource overwrites, an
integration or automation through its token scopes, and the agent through
whichever of those it acts as. Grants are stored as a sorted JSON string
array (`roles.permissions`, `permission_overwrites.allow`/`deny`), the same
shape `automations.scopes` already used, so adding a domain is one table row
and no schema change. The old document actions fold into the grid
(`edit`/`archive` → `docs:write`, `manage_permissions` →
`permissions:write`) and enforcement maps one to one; no domain gained a
check it did not have. The one asymmetry kept on purpose: a token granted
`X:write` or `X:delete` also receives `X:read` at mint, while a role holds
exactly what was ticked. This supersedes the fixed-width bitmask that
ticket 02 of the workspace redesign chose (the retired `.scratch/workspace-sidebar-redesign/` effort, in git history)
and the `manage_mention_layout` naming in ADR 0007, which is now
`workspaces:write`.

Amended by ADR 0057: a domain may declare a verb beside read, write, and
delete.
