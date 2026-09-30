# 05 — Container logs through MCP

**Type:** task
**Status:** open
**Blocked by:** 01, 02

## Question

The MCP surface is at its ceiling and `logs` is not an allowed verb. Decide how agents read container output: a bounded snapshot option on an existing tool (`stack_get` with `logs: {service, lines, since}`), or a new tool behind an ADR that raises the budget. MCP here is stateless, so no streaming; decide the default and maximum lines and how a long line is cut.
