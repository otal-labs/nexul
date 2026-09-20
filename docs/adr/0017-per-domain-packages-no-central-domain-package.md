# Entities live per domain; there is no central `Domain` package

Each bounded context owns its own model, repository interface, use-cases and
events under `internal/<domain>/`, and domains never import each other. What
looks like the obvious alternative — one shared package holding every
entity so nothing is duplicated — is what turns into a god-package that every
domain depends on and nobody can change. Shared *contracts* instead live in
`internal/platform/*` (event bus, errors, config, permissions, identity) and
in the published schemas; a domain that needs a slice of another declares a
small consumer-side interface and the composition root wires it.

Decided: 2026-07-25

## Consequences

Some types are deliberately mirrored across domains (a `User` in `access` is
not `auth`'s `User`). That duplication is the price of the seam, not an
oversight to be refactored away. `internal/platform/*` may be imported by
anyone; `server/cmd` and `runner/cmd` import domains and are imported by
nothing.
