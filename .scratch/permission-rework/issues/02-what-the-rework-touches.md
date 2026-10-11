# 02 — What the rework touches in the code

**Type:** research
**Status:** open
**Blocked by:** None

## Question

What does each decision in ticket 01 change in today's code?
- every check path and list filter
- every writer of `permission_overwrites`
- every use of Restricted, `every_project` and project access
- the live audience rules
- the MCP fields
- invitations
- the web editors
- the phone app

What must the wipe migration delete and keep, row by row? Which statement-count guards will move, and why?
Write the findings to `research/what-the-rework-touches.md`, with file:line references, grouped so the rewrite
can slice tickets from them.
