# 02 — Who may read container logs

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Container output often carries secrets: env values an app prints at start-up, tokens in stack traces, connection strings in errors. Is reading logs covered by `stacks:read` (anyone who can see the stack), or is it its own verb (`stacks:logs`, per ADR 0057) that a role grants separately? Also: do agents (MCP tokens) get it by the same rule, and is any masking wanted, for example of the stack's own env values?

## Answer

Decided 2026-09-30 by the owner.

- **A verb of its own: `stacks:logs`** (ADR 0057). Every role that holds
  `stacks:write` gets it by default, and any role can be given it
  separately. Tokens and agents follow the same table; `stacks:read` alone
  does not show logs.
- **Masking.** Before a line leaves the server, any value of the stack's own
  env (the values Nexul holds for it) becomes `••••`. Values shorter than 6
  characters are left alone, so common words are not masked.
