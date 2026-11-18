# 05 — Run as a person, and the auto play computer list goes

**Status:** ready-for-agent
**Blocked by:** 04

Spec: `../spec.md`, decisions 3 and 4; Moving over. ADR 0197.

`RunOnComputer` becomes `RunOnPerson` (`person`): the run takes `run_as`'s identity and order. `ComputerIDs` and
`ComputerID` go from `AutoPlay`, `QueueItem` and `HarnessChoice`'s queued path; `HandleAutoPlayComputersChanged`
becomes a move of queued runs onto a new `run_as`. `runRequest` in `server/cmd/wire_plays.go` no longer builds a list
from the item. The migration moves every auto play and queued item as the spec says, switching off the ones that named
only shared computers with the history line. The Run on section offers Run as \<person\>; the seeded Investigate a crash
still waits for its person to be picked (ADR 0156's off-until-picked rule, now off until a person is picked).

Done when: the migration test seeds an auto play on own computers, one on a shared computer only, and a queued item, and
asserts each result; the ADR 0156 right checks still run at pick, match and the front of the queue, now for the person.
`plays.md`'s Run on a chosen computer becomes Run as a person; ADRs 0156 and 0165 carry their amendment lines.
