# One versions store, or one per entity

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

Two shapes of history are now decided: body versions (docs, ticket
descriptions, play instructions, memories, automations) and field-change
lists (tickets, plays). Do they live in one shared versions table and one
shared changes table keyed by entity, with one HTTP route and one web dialog
reading them, or in a table per entity like `doc_versions`,
`memory_versions` and `automation_versions` today? This is a schema change
either way, and it also shapes how the existing three tables migrate.

## Answer

- A table per entity, the pattern `doc_versions` and `memory_versions`
  already follow: new tables for ticket description versions, play
  instruction versions, ticket changes and play changes. Each domain owns
  its history, a delete cascades to it through the foreign key, and nothing
  existing migrates. The cost is a handful of similar tables, each with its
  own pruning query.
- The shared parts are the shape, not the storage: every entity's history
  route returns the same shape, and the web has one history dialog.
- `automation_versions` stays as it is. Its pending and active states drive
  what runs, so the dialog reads it and nothing reshapes it.
- Rejected: one shared table keyed by entity type and id. It loses the
  cascade, so every delete path would have to clean up by hand; it crosses
  the domain seam; and the three existing tables would have to migrate.
