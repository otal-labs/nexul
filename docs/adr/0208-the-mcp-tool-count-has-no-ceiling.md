# The MCP tool count has no ceiling

Amends ADR 0068 and drops the ceilings ADRs 0081, 0131, 0168 and 0185 raised.

ADR 0068 kept the server under 100 tools because one client caps an agent at 100 across all its servers, and a test
held the count one above today's so each new tool needed an ADR. The surface has been past 100 since ADR 0081, so that
client already drops tools, and every new capability since has spent its review arguing for one more slot rather than
about its shape. Shell jobs on a computer (`command_run`) and the sharing that follows are new, destructive executions
that no existing tool can carry, which ADR 0068 itself keeps as their own tools.

Decision: the tool count has no ceiling and no test pins it. What each session actually pays for, the characters
`tools/list` sends, stays capped by the advertised size budget (ADR 0171), raised only by the ADR of the feature that
needs it. ADR 0068's shaping rules still hold and are checked in review: one tool per task, a capability extends an
existing tool before it gets a new one, and reading, changing, deleting and starting an execution stay separate tools.

Rejected: raising the ceiling by one again, which keeps the ADR ritual without guarding anything the size budget
does not.

Decided: 2026-11-25.
