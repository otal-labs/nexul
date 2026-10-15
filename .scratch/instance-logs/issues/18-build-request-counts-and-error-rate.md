# 18: Build request counts and the error-rate rule

Type: task
Status: open
Blocked by: 14, 16, 17

## What to build

Tickets 08 and 09, as decided: Traefik's metrics on proxy gateways read by the machine runner;
Cloudflare's per-hostname analytics for tunnel hostnames with Analytics Read added to the
connector's checks; `stack_request_minutes` kept a day with gaps stored as missing; the per-stack
rule; `error_rate_episodes`; `stack.error_rate_crossed` and `stack.error_rate_recovered`; watchers
notified, the crash ticket switch and stack plays reused.

## Surfaces

- Events: as ticket 14.
- MCP: `stack_get` returns the rule, the open episode and the last hour's counts; `stack_update`
  sets the rule.
- UI: the rule, the error-rate line and the small chart, placed where ticket 10 puts them; the
  greyed-out reasons.
- Docs: the stacks guide, the Cloudflare connector's token instructions.
