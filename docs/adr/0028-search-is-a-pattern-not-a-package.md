# Search is a pattern each domain opts into, not a domain package of its own

There is no `internal/search`. Searchable content is FTS5 virtual tables kept
in step with their source tables by SQLite triggers, and each domain exposes
its own `Search` over its own index — the same way each domain registers its
own MCP tools. The event consumer subscribed to the re-index topics only
validates and acknowledges; it does not write the index, because the triggers
already did, inside the same transaction as the change.

A search domain owning one index would have had to import every other
domain's content or have every domain push into it, which is the shape the
EventBus seam exists to avoid. The consumer stays subscribed anyway despite
having nothing to do: it is the seam where embeddings arrive later as a second
consumer on the same events, without touching the FTS path (semantic search is
deferred per D14).

Search results are **filtered, not disclosed**: a document the requesting user
cannot open is excluded entirely, not even its title. This is deliberately
different from the `@`-mention chip and the doc list, which do show an inert
title-and-status placeholder for an inaccessible target — there the user
already knows the thing exists because somebody linked it, whereas search
would otherwise turn into an oracle for enumerating titles the user was never
shown. Archived and soft-deleted items never appear either. The MCP tools
enforce the same rule against the user the agent acts as.
