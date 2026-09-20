# Borrowed practices

Rules adopted from other codebases that shipped at scale, adapted to this
repository. They apply to every language. If a rule here fights a hard rule in
`AGENTS.md`, the hard rule wins.

## Non-negotiables

1. A one-way door is a bug. Every way in needs a way out and a way to see it.
   Archive needs restore, close needs reopen, and a soft delete names the rows
   it affects in its warning. Users do not read the code to learn what a
   button undid.
2. The smallest model that makes correct behavior unsurprising. Complexity is
   never preserved because it already exists, and machinery is never added
   because it looks impressive. Understand the real constraint first, then
   build the least that satisfies it.
3. Complexity belongs at the adapter boundary. Domains stay pure, the UI stays
   dumb, and adapters (HTTP, MCP, the git provider, the runner protocol) do
   every translation. This is ADR 0017 and ADR 0019 stated as a habit.
4. If a rule fights the task in front of you, say so out loud and get the
   owner's sign-off before breaking it. A silently broken rule is
   indistinguishable from a rule nobody knew about.

## SQLite discipline

- Every index carries a comment naming the query it serves. The reason lives
  in the migration, where the next person changing the index will read it.
- An ordering index includes its tiebreaker column, for example
  `(ticket_id, created_at, id)`, so a pagination cursor stays index-only.
- List queries paginate by keyset, never by `OFFSET`. Offset pagination
  rescans every skipped row and drifts when rows are inserted mid-scroll.
- A list index on a soft-deleted table leads with the archive or delete
  column, so live queries scan only live rows.
- Migrations are idempotent: `CREATE ... IF NOT EXISTS` everywhere, and a
  column add checks `PRAGMA table_info` first, because SQLite has no
  `ADD COLUMN IF NOT EXISTS`.
- An expensive computation that is stored is stored under a `UNIQUE` key, so
  recomputing it is an upsert and not a duplicate.

## Async and retries

- One component owns the retry policy for a path: the bus middleware for
  events, the runner client for its connection. Everything below it tries
  once; everything above it renders state. Scattered retry loops multiply
  each other and hide which layer is failing.
- A transport failure and a domain failure are handled differently. Transport
  down means reconnect or replace the session. A domain error keeps the
  healthy connection and surfaces the error.
- Async code is tested by draining, never by sleeping. Wait on the real signal
  (a channel, a receipt, a wait group). A test that needs a timeout to pass is
  wrong, because the timeout is the bug wearing a disguise.
- Idempotency is a stored receipt, not a hope. The processed-events store is
  the receipt (see `practices/architecture.md`, sections 3 and 6), so
  at-least-once delivery is safe everywhere.

## Frontend discipline

- A performance audit checks three things first: too much data over the wire,
  animations that repaint continuously, and lists that are hard to render.
  A continuously repainting animation pegs the GPU on a high-refresh display.
- New design tokens are never invented per component. Theme through the
  existing CSS variables and the existing type scale
  (`practices/design-language.md`).
- State is split by lifetime: server cache in TanStack Query, cross-surface
  client state in Zustand, ephemeral UI state local to the component.
- Loading and connection state derive from real state, never from whether an
  object or cache entry happens to exist.

The checklist for calling a feature done on every surface lives in
`AGENTS.md` under "Hit every surface".

## Git and pull requests

- One topic per pull request. A description that says "also" is two pull
  requests.
- The branch is rebased onto the latest `master` before the pull request
  opens, so the diff is only the change.
- Every pull request is squash-merged. Its title is one sentence describing
  the change from the code's point of view, because that sentence becomes the
  commit on `master` and the line in the release notes.
- The body states the problem in a sentence or two, then how the change fixes
  it. A UI change carries before and after screenshots at 320, 375, 414, and
  768px; a change that depends on motion carries a short video.
- Nothing shipped names the tool that produced it. Commit messages, pull
  request text, comments, and docs describe the change, not its author.

## Ways to hurt yourself

- Never write to a live or shared database from a test. Tests use a temporary
  SQLite file; data is copied in, never symlinked, and never written back out.
- Never commit a secret. A dev-only shared value is tracked as an open item in
  `.scratch/pre-release/` until it is rotated.
- Never kill a process by pattern (`pkill -f`, `pgrep | kill`). Your own
  session matches the pattern. Kill only a PID you captured at spawn.
