# 04 — Expose-a-service UX

**Type:** grilling
**Status:** open
**Blocked by:** 01, 02

## Question

With the network/exit-node model (01) and routing mechanics (02) settled:
where and how does an owner say "host this service at
`app.example.com`"?

- Which surface: the service creation flow, the service/deploy page, the
  topology canvas (drawing an edge to a `domain` node), or several?
- What the flow shows: zone picker, hostname input, exit-node selection
  (or is it implied by the service's network?), propagation/health state.
- What "unexpose" looks like, and what happens to the record/route when
  the service is deleted.
- MCP/API parity: DN6 says tools mirror use-cases — confirm the exposed
  operations list.

If the discussion turns visual, hand the look to design-mode per the map
Notes.

## Comments

Owner delegated execution (2026-08-27). Implementation proceeds on this
direction, ticket stays **open** for the owner's reaction to the built v1:
an "Expose" action on the service page — dialog with hostname input +
gateway (implied by the service's network) + propagation state; unexpose
removes record and route; service deletion cleans up its exposures. MCP
mirrors expose/unexpose/list. Topology-canvas entry point is fog.
