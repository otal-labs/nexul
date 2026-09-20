# MCP tools may start executions, marked by their provenance

The MCP adapter originally refused to expose anything that kicked off real
work: tools read and mutated domain records, and starting a deploy stayed a
UI-only button. That made the agent path strictly weaker than the browser
path, which contradicts the rule the adapter exists for — an agent gets
nothing the UI doesn't get, and nothing less either. A product whose thesis is
that an agent can drive the whole loop cannot stop at the deploy step.

Retired 2026-09-10: `stack_deploy` and `stack_rollback` enqueue exactly what
their UI buttons enqueue. Accountability is handled by provenance rather than
by refusal — the execution records the token's user with a `:mcp` suffix, so
the trigger's origin is visible wherever the deploy is.
