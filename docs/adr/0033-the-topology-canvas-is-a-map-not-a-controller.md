# The topology canvas is a map, not a controller — and only the hand-drawn part is stored

Drawing an edge between two service nodes looks like it should wire a Docker
network, and removing one looks like it should unwire it. It does neither:
the canvas is descriptive, and real wiring is deliberate configuration that
lives in the repository's compose file or in a `run` stack's network setting.
Making edges behavioural would give one workspace two competing sources of
truth for the same network, with the canvas — the one nobody reviews — able
to take a stack down by accident.

Everything the rest of the system already knows is therefore *derived at
render time and never persisted*: the dashed network boxes, a gateway's route
rows, and the hostname pills are computed from the stack, gateway, and
exposure APIs on every load, while the stored canvas JSON holds only nodes,
edges, and positions. Hand-drawn network nodes stay valid in the schema but
the palette no longer offers them, since the derived boxes are always right
and a hand-drawn one goes stale the moment a stack moves.

Concurrent edits are last-write-wins, and the docs' real-time collaboration is
deliberately not extended to the canvas. Docs earn it because two people write
prose in the same document all day; a canvas is edited rarely, by one person,
and the part that changes constantly — status, addresses, network boxes,
route rows — is derived on every render and never in conflict to begin with.
The stored map is small enough that a clobbered layout costs a drag, not work.

Decided: 2026-09-08 (derived layer; the map/controller split predates it)
