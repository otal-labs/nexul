# 03: What the web can know about one thing's permissions after the switch

Type: research
Status: resolved
Blocked by: None — can start immediately

## Question

Hiding or locking an action needs the answer for this person on this one thing, and for a locked one, its reason.
The original question used the `permission-switch` branch and its spec, preserved in git history at `bf6a135f`.
For current work, read ADR 0148 and master, then
answer:
- what the web gets today and after the switch: `/me` per workspace, `ProjectPermissions`, `useAreaAccess`,
  `client-core/` helpers, source sentences
- whether a per-entity answer exists for a doc, play, stack, machine or channel, and what tickets 19, 23 and 24 add
- whether "your role can't" (the role union lacks it) can be told apart from "a rule on this thing denied it"
  without a new endpoint
- how much the web would have to fetch per list row, so the sweep keeps the fast-by-default rule

## Answer

Details are in `research/03-what-the-web-can-know.md`. The switch merged as #614, so this reads master.

- `/me` answers project, workspace and instance actions. That is enough to hide or lock most buttons with no new
  endpoint.
- For the instance and workspace areas, "absent" means "your role can't".
- `/me` can't tell a role refusal from a rule refusal on project-area actions, and it can't see a rule on one doc,
  play or stack.
- `GET /api/permissions/explain` (and MCP `permission_overwrite_list`) gives the answer for one entity, with
  `source.kind` set to `role` or `rule` and the sentence. It covers docs, plays (`plays:run` only) and projects.
  Stacks, areas, machines and channels were refused at that snapshot.
- The remaining permission slices have since merged, including stack, instance-area, computer and channel rules.
  Check current entity explanations rather than treating the snapshot above as the current API contract.
- Lists are guarded to constant statements, and their rows carry no write or delete answers. One explain call per
  row would be N requests with no guard. A locked action on a single row therefore needs explain widened to stacks
  and areas, or a flag the server computes on each row. That choice goes to ticket 05.
