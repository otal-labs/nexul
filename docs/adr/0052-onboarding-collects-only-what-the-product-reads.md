# The onboarding wizard collects only fields the product actually reads

The owner wizard's "Introduce yourself" step asks for a name and an avatar and
nothing else. The reference designs it was built from also collect a role and a
team size, and both were deliberately left out: nothing in the product reads
either one, so storing them would be data kept for its own sake — a column to
migrate, a field to render, and a question to answer on the way in, all with no
consumer on the other side.

The same rule governed the wizard's other steps. Step 2 sets the name and prefix
on the already-seeded default project rather than creating a second project row,
because the step exists to fix those fields, not to establish a second identity.

Decided: 2026-09-03

## Consequences

Adding a field to onboarding means naming the feature that reads it first. A
future redesign that reintroduces role or team size because a reference design
shows them is re-making this decision, not filling an oversight — say so, and
point at the consumer.
