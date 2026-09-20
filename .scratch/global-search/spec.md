# Global search: one box, every searchable entity

**Status:** needs-triage

Carried over from the retired search requirements document, where it sat as a
future direction. Search exists and works, but only from inside the docs and
tickets surfaces and from the MCP tools. Worth re-confirming that a single
global box is still the shape wanted before building it.

## Problem Statement

A person who remembers something exists but not where it lives has nowhere to
type it. Search is reachable only from the surface that owns the content: the
docs list searches docs, the ticket views search tickets, the `@` picker
searches both but only inside an editor. Finding a service definition, a pull
request, or an automation by name is not possible at all — they were never
registered as searchable. Meanwhile an agent driving the product through MCP
has a better search surface than the person sitting in front of the browser.

## Solution

One search box in the app header, reachable from any page and from the
keyboard, returning results across every registered entity in one list
grouped by kind, each result linking straight to the thing. The same index
and the same permission rules that search already uses — this adds a surface
and more registered entities, not a second search engine.

Making a new kind of entity searchable stays the small, repeatable move it
already is: the owning domain declares its indexable text and gets an index
kept in step by the database, the same way a domain registers its MCP tools.
The candidates named when this was first specified were service definitions,
pull requests and reviews, and automations.

## User Stories

1. As a member, I want a search box in the header on every page, so that I can look something up without first navigating to the surface that owns it.
2. As a member, I want to open search from the keyboard, so that I do not have to reach for the mouse mid-thought.
3. As a member, I want results grouped by what kind of thing they are, so that I can tell a doc from a ticket at a glance.
4. As a member, I want each result to say enough to identify it — title, and the project or status that disambiguates it, so that two similarly named things are distinguishable.
5. As a member, I want to open a result with the keyboard, so that the whole interaction is one gesture.
6. As a member, I want to find a service by name, so that I can jump to it without walking the topology.
7. As a member, I want to find a pull request by title, so that I can reach the ticket it belongs to.
8. As a member, I want to find an automation by name, so that I can check what it does without browsing the list.
9. As a member, I want results I cannot open to be absent rather than shown and inert, so that search never tells me a document exists that I was not shown.
10. As a member, I want archived and deleted things left out, so that results reflect what is live.
11. As a member, I want results to appear as I type without the page stalling, so that searching feels like filtering rather than submitting.
12. As a member, I want search scoped to the workspace I am in, so that another workspace's content never appears.
13. As an agent acting as a user, I want the MCP search tools to cover the same entities as the browser's box, so that the two surfaces do not drift apart.

## Implementation Decisions

- No new search engine and no new index: the same full-text tables, kept in
  step by database triggers, that search already uses (ADR 0028). Registering
  a new entity means adding its index and its domain's own search method.
- One gateway endpoint fans out across the registered entities and returns a
  single grouped result set, so the browser makes one request per query
  rather than one per kind.
- Permission filtering stays at query time in each domain, unchanged: results
  the requester cannot open are excluded entirely, never disclosed as an
  inert placeholder (ADR 0028). The global endpoint must not become a place
  where a domain's filter is bypassed for convenience.
- Results are scoped to the active workspace.
- Per-kind result limits so one noisy entity cannot crowd out the rest, with
  a way to see more of one kind.
- Each newly registered entity needs its indexable text decided explicitly —
  what a person would plausibly type to find it — rather than indexing every
  column.
- The existing in-surface searches stay; the global box is an addition, not a
  replacement.
- Whether the box is a header input or a command palette overlay is a design
  decision, not settled here.

## Testing Decisions

Tests cover what a caller sees: given seeded content and an identity, which
results come back and in what shape.

- An integration test against real SQLite is the main seam, because the index
  is maintained by database triggers and a test with a fake repo would prove
  nothing about it. Prior art: the existing storage integration tests that
  assert archived content disappears from search.
- Per-entity: content is findable right after it is created, disappears when
  archived or deleted, and never appears for an identity that cannot open it.
- The fan-out endpoint: one query returns results from several kinds, per-kind
  limits hold, and a query matching nothing returns empty rather than an
  error.
- The header surface is covered at component level against a stubbed
  endpoint — keyboard open, result navigation, empty and loading states — not
  by driving the whole app.

## Out of Scope

- Semantic or vector search. Deferred; the event consumer already sitting on
  the re-index topics is where it would arrive later, without touching the
  full-text path.
- Saved searches, search history, and search-based filters on the board.
- Searching chat messages, unless chat is one of the entities confirmed for
  registration.
- Cross-workspace search.
- Replacing the in-surface searches.

## Further Notes

The original requirement made the global box conditional on there being more
than docs and tickets indexed — worth deciding whether that condition still
holds, or whether a box over just docs and tickets is already worth having on
its own.
