# Grants are edited as one level per domain

A role, an automation's scopes, and an invitation's allow overrides were
edited as a checkbox per action: 28 domains and up to six columns, around 75
boxes, most of them clicked in the same read-then-write-then-delete order.

Decision: the editor offers one level per domain, None, Read, Write, or
Delete, where each level grants every action below it, plus a toggle for any
verb the domain declares (ADR 0057) and an "Every domain" row that sets them
all. Storage and enforcement are unchanged: a grant is still the string set
ADR 0010 describes, and a role still holds exactly what was saved.

The cost is expressiveness in the editor. A set that skips a rung, write
without read or delete without write, can no longer be built from it. Such a
set still displays at its highest action and is only rewritten when that
domain's level is changed. Deny overrides keep the checkbox grid, because a
deny ladder runs the other way (denying delete alone is the common case) and
a level control would hide it.

Decided 2026-09-28, building on ADRs 0010 and 0057.
