# Per-user permission overwrites are one table for every resource type, and the workspace Owner bypasses all of them

Per-user grants live in a single `permission_overwrites(resource_type,
resource_id, user_id, allow, deny)` table holding the same string sets roles
use (ADR 0010). A workspace-wide override is `resource_type="workspace"`; a
per-item override uses that item's own type and id. One mechanism rather than
a grant table per domain: documents were the first resource type to need it
and would otherwise have set the pattern for every domain after them.

Both allow and deny exist, and precedence is most-specific-wins — role set,
then workspace-wide overwrite, then resource-instance overwrite, with deny
beating allow inside a layer. Additive-only grants were tried first and
rejected: a permission handed out by a role has to be revocable for one person
on one item.

The workspace Owner role bypasses the whole chain, and the bypass is checked
first *inside* `HasPermission` rather than at each call site, so no overwrite
can ever lock an Owner out of their own workspace and no call site can forget
to check it.
