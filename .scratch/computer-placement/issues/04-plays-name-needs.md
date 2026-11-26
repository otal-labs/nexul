# 04 — Plays name needs

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Spec: `../spec.md`, Placement (Ready, Needs); Surfaces.

`plays.needs`, versioned with the play. The play form gets **Needs**, a label picker fed by every label on the editor's
own computers plus free entry. `place` takes the play's needs in `RunRequest` and passes a computer over as
`missing_need` ("laptop doesn't have gpu") after the model check and before the checkout. Pressed, queued and Run again
all pass the play's needs. Saving a play whose needs changed wakes the queue. `play_create`, `play_update` and
`play_list` carry `needs`, paid for within the MCP text budget.

Done when: placement tests cover a need met, a need missed by the first computer and met by the second, and no computer
meeting it (the waiting line names each); `make event-schemas` and the contract test pass. `plays.md` documents Needs;
`CONTEXT.md` defines need.
