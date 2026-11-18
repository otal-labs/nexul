# Imports run through one framework and one tool

Linear, Asana, Trello, Notion and GitHub issues all need the same machinery to move a team's records into Nexul: a
scan before anything is written, a job that survives a restart, progress, a credential that is deleted afterwards,
people and statuses mapped onto the workspace, files copied, and a second run that does not duplicate the first.

Decision: an import is one framework in `internal/imports`, and each tool is a `Source` that only reads: `Scan` counts
and names the people and statuses it found, `Import` pages through and hands each record to a sink. The framework owns
everything else, so every source behaves alike:

- An import moves `scanning` → `ready` → `importing` → `completed`, `failed` or `cancelled`, run by one loop woken by
  the commit broadcast. A ready import waits 24 hours to be confirmed. A workspace has one open import at a time.
- The credential is sealed with the instance key while the import runs; a final state clears it, and a CHECK
  constraint refuses a final state that keeps it.
- Every record gets an id derived from the workspace, the source and its source id, so a write whose record was lost (a
  restart, then a cancel) conflicts on the next attempt or the next import instead of writing twice. `import_items` says how far each record got; `import_source_ids` makes a later
  import of the same source skip what came in before, rather than update it, so a ticket edited since is never
  overwritten.
- Statuses land in a column of the same name, else the first of their kind, else a new column. People match a member
  who can open the project by GitHub login, then email; anyone unmatched is named in the body. Files come down over
  plain HTTPS from the source's own hosts, redirects included, never carrying the credential, up to 10 MB each; the
  rest stay links. A ticket body is markdown, so copied files are swapped in by URL.
- An imported ticket's reporter is `import` with its source, and an imported comment is an `external` message under
  its author's name there. Notifications and auto plays pass both by; every event is still published.
- The workspace's Owner alone runs an import.

One MCP tool, `import_run`, carries every source: `source` and `project_id` start a scan, `id` reads progress, and
`confirm` or `cancel` act on it. A new source adds a word to `source` and a clause to `from`, not a tool, so the count
goes to 114 under the budget of 115 and stays there. The tool's text raises `advertisedBudget` from 193,000 to 195,000
characters; no existing tool could carry it, since an import is an execution with its own confirmation (practices/mcp.md
section 4), and the room left above it is what the next sources' words will use.

The GitHub source imports one attached repository's issues as the GitHub App, so it takes no credential, and creates no
sync links. It refuses a repository whose issue sync is on (ADR 0200), whose issues are already tickets, and maps an
issue's title, body, state, labels and assignees with sync's own `FieldsFromIssue`, so an issue reads the same either way.

Rejected: a tool per source (five tools for one task); updating previously imported records (it would overwrite edits
made in Nexul since); posting imported comments as the importing person (their name on words they never wrote, and their
@handles would notify).

The Notion source (amended the same day) is the first that writes docs, so the sink gained a doc path beside the ticket
one: each root page becomes a folder named after it, found by name first so a re-run reuses it, and every page and
database row under it a doc. Links between imported pages are written in one pass: the plan comes first, from a search
of everything shared with the integration, and a doc's id is derived, so a link names a doc before it exists. Blocks are
read and converted page by page, so Notion's signed file links, which expire within the hour, are downloaded minutes
after they are handed out. A doc written but never filled is archived when its import fails or is cancelled, and
forgotten, so a re-run writes it again. The import form takes the integration secret without a ticker: nothing keeps the
secret, and the scan is the check, failing with what to fix (a refused secret, nothing shared, a page not shared).

Decided 2026-11-18.
