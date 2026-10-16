# 18: Build request counts and the error-rate rule

Type: task
Status: resolved
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

## Built so far

- Request counts (migration 0122, `stack_request_minutes`, NULL counts mean missing): Traefik's
  metrics read by the machine runner as a `request_counts` frame; Cloudflare's per-hostname
  analytics once a minute per zone for tunnel hostnames; the advisory `analytics_read` check; the
  Requests card; `stack_get`'s `requests` and `GET /api/stacks/{id}/requests`.
- Routers are matched by `dns.RouterName(exposureID)`, `nexul-<exposure id>`; nothing writes those
  routers yet (ticket 21), so proxy hostnames show no data for now.
- `wantsCounts(stack)` returns true for every stack until the rule narrows it.
- Left: the rule, `error_rate_episodes`, the crossed and recovered events, watchers, the crash
  ticket switch, the `stack.error_rate_crossed` moment, the Error rate card and its greyed reasons.

## Built

- Migration 0123: the rule on the stack (5% / 5 minutes / 20 requests by default, bounded) and
  `error_rate_episodes`, one open per stack. The evaluator runs after each minute's count poll.
- Recovery is every one of the last N minutes at or under the limit; a minute with no requests
  counts as under; a missing minute blocks crossing and recovery alike.
- `wantsCounts` now means the rule is on, for proxy frames and Cloudflare alike; the Requests card
  says "Not counted" while the rule is off.
- Switching the rule off ends an open episode without a recovered event.
- The analytics reason comes from probing a minute of the zone's counts, cached a minute per zone;
  only a refusal counts, so a Cloudflare outage never reads as a missing permission. Not run
  live yet: ticket 20 does, with a real token.
