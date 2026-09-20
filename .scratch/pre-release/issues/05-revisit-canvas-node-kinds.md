# 05 — Revisit canvas node kinds once topology ships

**Status:** needs-triage

**Blocked by:** Real day-to-day use of the topology canvas.

**What to build:** Nothing yet — this is a decision to revisit, not work to do.

External nodes on the topology canvas are currently generic: a name plus a label
(domain / proxy / database / API / other). Once the canvas is in use in anger,
decide whether dedicated node kinds are worth adding — e.g. a first-class Domain
node with its own fields.

Owner kept it generic for momentum during the topology domain discussion
(2026-08-11) but expects their mind may change once they use the real canvas.
Triage this once there's actual usage to judge against; if the answer is "generic
is fine", close it and record that as the decision.

- [ ] Decision made, with the reasoning recorded (an ADR if it changes the model)

## Surface when

- Topology/canvas work is complete, or the canvas is being used daily.
- The main domain phase completes or a new roadmap is derived.
- Anyone starts extending canvas nodes or the node detail sheet.
