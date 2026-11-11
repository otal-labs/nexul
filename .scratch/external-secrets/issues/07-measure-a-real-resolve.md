# 07 — Measure a real resolve with a test service account

**Status:** open
**Type:** task
**Blocked by:** 01

## Question

Not a decision. With a throwaway service account scoped to a test vault holding 20 fields, resolve them
from the Go SDK and from `@1password/sdk` under Bun, and record: the reads one `ResolveAll` costs
(`op service-account ratelimit` before and after), latency cold and warm, the error type and text for a
missing item, a missing field and a vault the token cannot reach, and what a call does with the network
cut. Also record which 1Password plan the account is on, since that sets the daily budget. Tickets 04
and 05 decide on caching from these numbers.
