# Stay a modular monolith; the EventBus interface is the seam, not a split

Domains talk to each other only through the `EventBus` interface in
`internal/platform/eventbus`, whose day-1 implementation is in-process
channels. The interface exists so a domain *could* become its own process
later — but splitting now would break the two things the product is built on:
the single SQLite writer (ADR 0011) and the transactional outbox, which only
works because the domain change and its event commit in one transaction.
A broker across processes is the only thing NATS buys, and in-process the
outbox already gives durability, retry, dead-lettering and replay. No scale
problem exists to justify the cost.

Decided: 2026-08-04 (the seam itself 2026-07-24)

## Consequences

Domains stay the unit of ownership, not deployment. Watermill over NATS is the
planned day-N implementation, so the swap is a composition-root edit rather
than a rewrite. Anything that would only work in one process — a shared
in-memory map, a direct function call across domains — is a seam violation
even though it compiles.
