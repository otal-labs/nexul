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

Superseded in part by ADR 0148: the doc list leaves out a doc the user cannot read, and its direct link
answers not found. The pill rule changes: a mention pill shows its target locked and inert with "You don't have
access", a doc's title or a ticket's title, status and developer, including from a project the user is not in.

The search box (`GET /api/search`, `mention_search` with `kind`) fans out from the mentions service rather than a
search package: tickets and docs through their FTS indexes held to the workspace, and pull requests, stacks,
automations, channels and memories through a source per kind that lists what the caller opens with that domain's own
access checks, matched on the title in Go. These name-only kinds take no FTS table; a pull request's title is read
from its ticket link with a substring scan of the newest links, until a workspace links enough that it needs one.
Ranking is the same for every kind and stable while typing: a title equal to the query, then one holding each word
whole, then one holding each word as a prefix, ties to the most recently updated; a ticket or doc matched only in its
body or notes is a separate body hit, ordered by recency. A word inside another word ("deploy" in "redeploy") is no
match, as in the FTS index.
