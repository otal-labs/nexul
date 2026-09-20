# The event catalog is a published, versioned, additive-only contract

Every event topic carries a versioned JSON Schema, served from
`GET /api/events/catalog` and seeded from the catalog at startup. This makes
the field names in the domains' event structs a public contract rather than an
internal detail: fields may be added, never renamed or removed, and a breaking
change ships as a new schema version alongside the old one.

The event surface is the thing a third-party developer builds against, so
freezing it is what makes integrations in any language possible — and what
lets the SDK's typed payloads and test fixtures be generated from the catalog
instead of hand-written. The cost is that a payload mistake is permanent for
the life of a version.

Decided: 2026-07-25.
