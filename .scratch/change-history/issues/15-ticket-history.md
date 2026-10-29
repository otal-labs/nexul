# 15 — Ticket history: description versions and field changes

**Status:** resolved
**Blocked by:** 12

Spec: `../spec.md`, decisions 2, 5, 10, Storage, The route shape.

`ticket_description_versions` and `ticket_changes`; every ticket save writes
one change per field it changed, with the request's actor; the description
follows the sitting rule through `CommitSitting`; caps of 10 and 100; the
migration seeds one "Created" change per existing ticket. History and
restore routes; `ticket_get` takes `version`, `ticket_update` takes
`revert_to_version`. Adds Change to `CONTEXT.md`.

Done when: tests cover each recorded field, position moves writing nothing,
an automation's and a play's actor, both prunes, the seed, restore on top,
and the MCP read and revert.
