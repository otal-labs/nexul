# 09: What an error-rate rule is and what happens when it trips

Type: grilling
Status: resolved
Blocked by: 08

## Question

A stack's app can be up and still failing every request. With request counts per stack (ticket
08), what rule does a team set, what counts as crossing it and coming back, and what happens
then? Does the same apply to Nexul's own server?

## Answer

- **One rule per stack, off until someone turns it on** (`stacks:write`): "5xx errors above 5%
  for 5 minutes, with at least 20 requests". The three numbers can be changed. The rule counts
  all the stack's hostnames together. The request floor keeps one failed request out of two at
  3am from waking anyone.
- **Crossing and coming back.** The server checks every stack with a rule each minute, over the
  last 5 minutes. It crosses when the share of 5xx is above the limit and the requests are at or
  over the floor. It recovers once the share stays under the limit for 5 minutes in a row. A
  window with missing minutes neither crosses nor recovers (ticket 08).
- **An episode, stored like an outage.** A new forward-only migration adds `error_rate_episodes`
  (stack, started_at, peak share, requests, errors, the rule's numbers at the time, ended_at).
  Only one is open per stack. Events `stack.error_rate_crossed` and `stack.error_rate_recovered`
  sit beside the container events, with the same audience (`stacks:read`), outbox and schemas.
- **It does what a crash does** (ticket 07):
  - The stack page shows "errors at 12% since 10:42", and `stack_get` returns the open episode.
  - Watchers get one notification when it crosses and one when it recovers.
  - The ticket switch opens one ticket per episode while none is open, and a repeat adds a
    comment.
  - Stack plays get the moment `stack.error_rate_crossed`.
- **No hostname, no rule.** A stack with no exposure has no requests to count. The rule shows
  greyed out with that reason, and also when a tunnel stack's Cloudflare token can't read
  analytics.
- **Nexul's own error rate is a follow-up, not this effort.** Its request lines are already in
  OpenObserve, but plays run on tickets, docs and stacks, and Nexul itself is none of those. It
  will be an instance rule that notifies the instance Owner, with no play.
