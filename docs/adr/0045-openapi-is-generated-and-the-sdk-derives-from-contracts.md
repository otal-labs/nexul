# The OpenAPI document is generated from the mounted routes, and the SDK derives from contracts

`/openapi.json` is built from the gateway's own route registrations rather
than hand-written or kept in a checked-in YAML file, so a route that exists is
documented and a route that is renamed cannot leave a stale entry behind. A
hand-maintained spec for an API this wide drifts within a release.

The same rule runs downstream: the SDK's event payload types and fixtures are
generated from the published event schemas, not typed out. Contracts first,
client second — no hand-written SDK before the contract it claims to describe
exists. The one surface still hand-written is the small HTTP client the
automations SDK uses; it is a deliberate exception, not the pattern.

Decided: 2026-07-25.
