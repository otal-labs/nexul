# Event schemas are generated from the payload types, and a contract file holds them to additive-only

The published event schemas (ADR 0044) were hand-written JSON beside, not with, the Go structs that produce the
payloads. A test checked that every topic had a schema, never that the schema matched what was sent, and the two had
drifted: `service.created` documented a `service` field the wire has never carried (it sends `stack`),
`ticket.status_changed` documented `execution_id` where the wire sends `run_id`, `deploy.requested` listed fields
removed long ago, and the ticket status enum named four values while a ticket's status is its column's id.

Decision: each domain's `Topics()` declares every topic with its payload type, and the schemas are generated from
those types with `jsonschema-go`, the library the MCP adapter already uses. Field descriptions are `jsonschema` tags.
What reflection cannot see is a tag too: `enum`, `minimum`, `deprecated`, and `type` for a raw JSON field. A type with
its own `MarshalJSON` names the type it encodes as through a `WireShape` method. A topic published with two payload
types is declared once per type and merged: either may arrive, so only what both always carry is required.

`make event-schemas` writes the result to `internal/eventcatalog/schemas.json`, the contract file reviewers diff and
the SDK generates its types from, and to `schemas_gen.go`, the same texts compact, which the server publishes at boot.
A test regenerates and compares, so the published schemas can never drift from the structs again. It refuses a change
that would break a consumer of the file: a topic or field removed or renamed, a type changed, an enum value dropped, a
field no longer always sent. An addition passes that check and asks for `make event-schemas`, the way `sqlc diff` asks
for `make sqlc`. A deliberate break has to be made by hand in the contract file, where review sees it.

Schemas follow what `encoding/json` writes. A field without `omitempty` is required. A slice without it may be `null`,
because the outbox encodes payloads with plain `json.Marshal`, and the schema says so rather than promise an array the
code does not guarantee.

The boot reads the generated texts and generates nothing: generating takes about 20ms and parsing the contract file
about 4ms, against well under 1ms to publish an unchanged catalog. Because every text changed once in the move,
an instance's first boot on this version publishes each topic's generated schema as its next version, beside the
hand-written one; later boots publish nothing until a payload type changes.

Decided 2026-10-09, amending ADR 0044.
