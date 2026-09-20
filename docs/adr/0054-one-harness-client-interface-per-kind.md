# 0054. One harness client interface per kind

The code that talks to a paired computer's agent tool used to be six seams,
each shaped differently and each T3-only: the turn interface in `agent`
(`Backend`), the token exchange in `pairing`, two lister callbacks, the
presence keeper's dial callback, and a version-probe callback. `Backend`
also collided with what "backend" means everywhere else here (the Go
server).

Decision: one interface, `harness.Client`, in its own bottom package
`internal/harness` that imports nothing from the domains, so `pairing`,
`agent` and `presence` all consume it without a cycle. It covers pairing,
version probe, project and provider listing, presence hold, start turn and
interrupt. A `harness.Registry` keyed by `Kind` replaces all six seams, and
every paired computer stores its kind. T3 Code's implementation lives in
`internal/t3client`, so no other package names T3.

Why one interface rather than a turn interface plus a pairing interface:
every planned harness (OpenCode 2 next) maps onto all of it, so a split
would only mean two registries and two fakes. The one T3-shaped method is
`Hold`; a harness with no long-lived connection returns a nil `Conn` and
the keeper shows it as connected.

Rejected names: `Backend` (the Go server), `Harness` alone (T3 calls itself
a control surface, and the interface is the client of one), `Runtime`,
`Driver`, `HarnessController` (reads as an HTTP controller).
`harness.Client` rather than `harness.HarnessClient` because the package
already says it.

Harness-specific concepts stay behind the interface: only harness-neutral
types cross it (`Session`, `Target`, `Update`, `Project`, `Provider`).
