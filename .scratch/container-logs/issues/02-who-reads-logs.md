# 02 — Who may read container logs

**Type:** grilling
**Status:** open
**Blocked by:** None — can start immediately

## Question

Container output often carries secrets: env values an app prints at start-up, tokens in stack traces, connection strings in errors. Is reading logs covered by `stacks:read` (anyone who can see the stack), or is it its own verb (`stacks:logs`, per ADR 0057) that a role grants separately? Also: do agents (MCP tokens) get it by the same rule, and is any masking wanted, for example of the stack's own env values?
