# 05 — Container logs through MCP

**Type:** task
**Status:** resolved
**Blocked by:** 01, 02

## Question

The MCP surface is at its ceiling and `logs` is not an allowed verb. Decide how agents read container output: a bounded snapshot option on an existing tool (`stack_get` with `logs: {service, lines, since}`), or a new tool behind an ADR that raises the budget. MCP here is stateless, so no streaming; decide the default and maximum lines and how a long line is cut.

## Answer

Decided 2026-09-30.

- **`stack_get` takes an optional `logs: {service, lines}`** and returns
  that service's last lines, masked, in the snapshot shape. There is no
  new tool: the surface is at its ceiling, and reading a stack's state
  already lives on `stack_get`.
- `lines` defaults to 200 and is capped at 1000. A line longer than 2000
  characters is cut and ends in `…`.
- Asking for logs without `stacks:logs` fails that call with a permission
  error. The rest of `stack_get` stays under `stacks:read`.
- An offline runner is a readable error naming the machine.
