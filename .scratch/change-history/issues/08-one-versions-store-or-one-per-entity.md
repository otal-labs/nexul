# One versions store, or one per entity

Type: grilling
Status: open
Blocked by: None — can start immediately

## Question

Two shapes of history are now decided: body versions (docs, ticket
descriptions, play instructions, memories, automations) and field-change
lists (tickets, plays). Do they live in one shared versions table and one
shared changes table keyed by entity, with one HTTP route and one web dialog
reading them, or in a table per entity like `doc_versions`,
`memory_versions` and `automation_versions` today? This is a schema change
either way, and it also shapes how the existing three tables migrate.
