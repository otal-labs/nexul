# 04 — How logs reach the browser and the phone

**Type:** task
**Status:** open
**Blocked by:** 01, 03

## Question

Decide the HTTP route for a snapshot and the live path for a tail: a topic on the existing `/ws/events` hub with a `live_audience` rule (the agent-stream pattern, never stored), or a dedicated socket per viewer. Covers reconnect, back-pressure on a chatty container, and keeping the route off the setup-pass allowlist.
