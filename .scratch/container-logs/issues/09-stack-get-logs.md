# 09 — `stack_get` returns a service's logs

**Type:** task
**Status:** ready-for-agent
**Blocked by:** 08

## What to build

- `stack_get` accepts an optional `logs: {service, lines}` argument:
  `lines` defaults to 200 and is capped at 1000, and a line over 2000
  characters is cut with `…`.
- It returns the masked snapshot through the use-case from ticket 08.
- A logs request without `stacks:logs` fails with a permission error. The
  rest of `stack_get` is unchanged.
- An offline runner is a readable error naming the machine.
- Update the tool description and `practices/mcp.md` if the surface test's
  description rules need it. No new tool.

## Acceptance criteria

- [ ] An MCP test calls `stack_get` with `logs` and gets lines, gets a
      permission error without the verb, and gets the offline error.
- [ ] The surface test still passes at the current ceiling.
