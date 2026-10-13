# 03: What the web can know about one thing's permissions after the switch

Type: research
Status: open
Blocked by: None — can start immediately

## Question

Hiding or locking an action needs the answer for this person on this one thing, and for a locked one, its reason.
Read the `permission-switch` branch and the rework's spec and tickets (`.scratch/permission-overrides/`), then
answer:
- what the web gets today and after the switch: `/me` per workspace, `ProjectPermissions`, `useAreaAccess`,
  `client-core/` helpers, source sentences
- whether a per-entity answer exists for a doc, play, stack, machine or channel, and what tickets 19, 23 and 24 add
- whether "your role can't" (the role union lacks it) can be told apart from "a rule on this thing denied it"
  without a new endpoint
- how much the web would have to fetch per list row, so the sweep keeps the fast-by-default rule
