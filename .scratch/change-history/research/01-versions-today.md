# How versions are written today, and how big they get

Answers ticket `issues/01-how-versions-are-written-today.md`. Read-only research,
measured 2026-10-28 on a copy of the demo instance database (made on a worktree
at `20bc0300`). Paths are repo-relative.

## 1. What writes a version

| Entity | Table | Written by | Not written by |
|---|---|---|---|
| Doc | `doc_versions` | Create (v1); every `Service.Update` (HTTP `PUT /api/docs/{id}`, MCP `doc_update`, plays); explicit named snapshot `POST /api/docs/{id}/versions` | Every collaborative-editing flush (`CommitCollab`) |
| Memory | `memory_versions` | Create (also decisions, interview, clone, the per-project seed memory); every `Service.Update` (browser save, MCP `memory_update`); `Revert` (goes through `Update`) | Nothing skips it |
| Automation | `automation_versions` | Push (pending), seed (first active, later pending), merge/rollback only flip `status` | A merge or rollback writes no row |

### Docs

- Insert sites: `internal/platform/storage/docs_repo.go:46` (Create) and `:128`
  (`DocsRepo.Update`), both through `insertDocVersion` (`:333`) ->
  `InsertDocVersion` (`internal/platform/storage/queries/docs.sql:43`). Named
  snapshot: `docs_repo.go:199` -> `InsertNamedDocVersion` (`docs.sql:46`).
- `Service.Update` (`internal/docs/usecase.go:359`) always reaches `DocsRepo.Update`,
  and `DocsRepo.Update` always inserts a row. There is no "body changed" check on
  the insert: a title-only save, or a save that sends the same body, still writes a
  row. `bodyChanged` (`usecase.go:384`) only decides whether the collab room is reset.
- Collab flush: the browser sends a `commit` every 5 s while dirty
  (`web/src/lib/collab/provider.ts:11`, `:263`) -> `session.handleCommit`
  (`internal/collab/session.go:140`) -> `writeBody` (`:188`) ->
  `Service.CommitCollab` (`internal/docs/usecase.go:598`) -> `DocsRepo.CommitBody`
  (`docs_repo.go:172`). `CommitBody` updates `docs.body` and `docs.version` and
  writes no version row; its comment says so. The session also skips a commit
  whose body equals the last one (`session.go:168`).
- `live.Reset` (`usecase.go:402`, `internal/collab/hub.go:204`) is not a version
  writer. It runs the `repo.Update` write (which is the one that inserts the row)
  fenced against the room, then drops the room's Y.js state (ADR 0109). It happens
  only when a server-side write changes the body.
- Who calls `Service.Update`: the web UI only on a brand-new doc
  (`CreateDocForm.tsx`, `useUpdateDoc`); every later edit in the editor goes through
  collab. So for a doc written only in the editor, `doc_versions` holds v1 plus
  whatever someone named by hand. Agents (`doc_update`) and plays are the main
  version writers.
- Consequences for a history feature:
  - `docs.version` increments on every collab flush, but rows are only written on
    the paths above, so `doc_versions.version` has gaps and "version N" is not the
    N-th save.
  - Unnamed rows carry no author (`InsertDocVersion` has no `author_id`; ADR 0101
    notes it). Named rows have `name` and `author_id`.
  - A named snapshot copies the current body, so it duplicates the row before it
    (demo: v2 and v3 are both 808 bytes). It also bumps `docs.version` to
    `version+1` (`docs_repo.go:206`).
  - A row stores the body after the save, never before it.

### Memories

- Insert sites: `internal/platform/storage/memories_repo.go:47` (Create) and `:152`
  (`Update`), through `insertMemoryVersion` (`:192`) -> `InsertMemoryVersion`
  (`queries/memories.sql:27`). Also the seed memory a new project gets:
  `internal/platform/storage/workspace_repo.go:121`.
- Every save writes a row, changed or not (`internal/memories/usecase.go:222-271`, no
  equality check). The editor has no autosave (`web/src/pages/MemoryPage.tsx` calls
  `useUpdateMemory` on Save), so rows = explicit saves + agent calls. `Revert`
  (`usecase.go:331`) appends a new row, never deletes history.
- Rows carry `author_id` and `author_via` (`"mcp"` or empty). Numbering is done
  inside the write (`cur.Version + 1`, `memories_repo.go:138`), so versions are
  contiguous.

### Automations

- Insert sites: `internal/platform/storage/automation_versions_repo.go:31`
  (`InsertActive`, only the first seeded version) and `:52` (`ReplacePending`),
  through `queries/automation_versions.sql:2`. Callers: `VersionsService.Push`
  (`internal/automations/versions.go:70`, HTTP/SDK push) and `SeedVersion` (`:155`).
- `ReplacePending` deletes the previous pending row before inserting
  (`DeletePendingAutomationVersion`), so an unmerged push that is superseded is
  gone. Only versions that were active (or are the one pending) survive. Sequence
  numbers are never reused (`nextVersionSequence`, `:135`).
- Merge and rollback are one operation, `Activate` (`versions.go:94`): it flips
  `status` and writes no row. `SeedVersion` is a no-op when the active code already
  matches, so a restart does not add rows.
- `code` is the built bundle (BLOB); `source` (migration 0131) is the file it was
  built from, and the pending-vs-active diff uses `source` when both sides have it
  (`web/src/models/AutomationVersion.tsx:24`).

## 2. Real numbers (demo instance copy)

Copied to `.verify/dev.db` (no `-wal`/`-shm` existed beside the original);
`PRAGMA wal_checkpoint` returned `0|0|0`. The original was never opened. The demo
database predates migration 0131, so `automation_versions` has no `source` column.

| Table | Rows | Entities | Versions per entity (max / median) | Body bytes (max / median / total) |
|---|---|---|---|---|
| `doc_versions` | 39 (3 named) | 31 docs | 6 / 1 | 7,113 / 493 / 32,311 |
| `memory_versions` | 23 | 21 memories | 2 / 1 | 2,813 / 243 / 8,435 |
| `automation_versions` (`code`) | 13 (9 active, 2 inactive, 2 pending) | 10 automations | 3 / 1 | 1,781 / 1,105 / 17,110 |

Other facts:

- Docs: 29 of 31 docs have one version; one has 4, one has 6. Current `docs.body`:
  max 3,693 bytes, median 405. The 7,113-byte row is a version 1 whose doc was later
  cut to 456 bytes.
- Memories: 19 of 21 have one version, 2 have two. 16 of 23 bodies are Tiptap JSON;
  7 are plain text (the legacy seed memory).
- Automations: 8 of 10 have one row. 6 of 13 rows were landed by the seeding hook
  (`nexul-seed`).

Seeding or real use: seeding. Reasons: the README says the instance is built by
`seed.mjs`/`seed-content.mjs`; 29 of 31 docs are single-version; the multi-version
docs have timestamps exactly 8 h 13 m 02 s apart (v1 to v4 of one doc) and
named versions sit on the same step; the 6-version doc has a 7 KB v1 followed by a
456-byte v2, which is not how a doc grows. The sizes are a floor and the counts are
not a forecast. What the code analysis says about real use: docs edited in the editor
pile up almost no rows (collab writes none); memories and agent-edited docs pile up
one row per save, without dedup, with full bodies each time.

## 3. What a doc version's body holds, and whether a text diff is stable

- `doc_versions.body` is the Tiptap/ProseMirror document as compact JSON
  (`{"type":"doc","content":[...]}`), the canonical form (`internal/docs/richtext/richtext.go:1-40`,
  ADR 0026). `Normalize` (`richtext.go:61`) passes structured JSON through and converts
  legacy markdown to JSON on write. Old rows can still hold plain markdown; all 39 demo
  rows are JSON, 7 of 23 memory bodies are plain text.
- `richtext.ToMarkdown` (`richtext.go:73`) returns plain bodies unchanged and renders JSON
  via `JSONToMarkdown` (`internal/docs/richtext/to_markdown.go:11`). The same call already
  feeds FTS (`SearchText`) and the MCP reads, so it is the shared text view.
- Deterministic: yes for the same tree. No map iteration reaches the output; the one place
  that formats a map (`sharedMarks`, `to_markdown.go:322`) uses `fmt.Sprint`, which sorts
  keys. Blocks join with a blank line, one paragraph is one line, lists/tables/code are
  line-based. Runs with equal marks merge (`**a**` + `**b**` renders `**ab**`), so editor
  run-splitting does not create noise.
- Gaps to know about, not blockers:
  - No test asserts "same tree twice gives the same text" or "different JSON, same
    content gives the same text". Tests (`richtext_test.go:151,264,384`; `mention_test.go:87`)
    check content with `Contains`/`Equal` on small cases. The determinism claim above comes
    from reading the code.
  - Mark order is kept as stored (`bold` then `italic` differs from `italic` then `bold`).
  - Lossy: marks and attrs with no markdown form render nothing (`markDelimiter` returns
    `""` for unknown marks such as underline or highlight; image sizes, text alignment, table
    column widths are dropped). A formatting-only change in those can produce an empty diff.
    Unknown blocks flatten to their text.
- Verdict: a line diff of `ToMarkdown(a)` vs `ToMarkdown(b)` shows only content changes,
  and every real text change shows. Compute it per version at read time, since the stored
  JSON is the source of truth. Memory `when_to_use` and titles are plain strings, and
  automation bodies are code, so they diff as they are.

## 4. Does the line diff keep up, and does prose need a word diff

`web/src/utils/DiffUtility.tsx:9-49` builds a full `(n+1) x (m+1)` table of numbers, then
walks it. Its own comment (`:8`) says to swap in a library if inputs grow. Its only caller
is `AutomationVersionDiff.tsx` (code files).

- Largest real body from part 2: the 7,113-byte JSON doc renders to 2,316 bytes of
  markdown, 72 lines (longest line 590 chars). `73 x 73` = 5,329 cells, about 0.005 M.
  The next largest doc is 30 lines. Nowhere near a limit.
- Measured the function itself (bun, synthetic 10 % changed lines, equal sizes; browsers
  will differ some):

  | Lines per side | Cells | Time | Heap growth |
  |---|---|---|---|
  | 72 | 5 k | 1 ms | none |
  | 1,000 | 1.0 M | 24 ms | none |
  | 2,000 | 4.0 M | 69 ms | about 23 MB |
  | 3,000 | 9.0 M | 164 ms | about 23 MB |
  | 5,000 | 25.0 M | 459 ms | about 150 MB |

  It starts to matter around 3,000 to 5,000 lines (a 100 to 200 times larger doc than
  anything in the demo): half a second of blocked main thread and a table of about 150 MB.
  Because a paragraph is one markdown line, 5,000 lines is 5,000 blocks, far more than
  a normal doc. A common prefix/suffix trim before the table removes most of the cost for
  small edits to big docs.
- Diff library: none. Not in `web/package.json`, `native/package.json`, `sdk/package.json`,
  `automations/package.json` or `website/package.json`, and `client-core/` has no
  `package.json` (it is shared source) or any diff helper. `web/bun.lock` has no diff
  package. `native/bun.lock` has `jest-diff` and `diff-sequences`, but only as test
  tooling under jest, not importable app dependencies. Adding `diff` (jsdiff) would be a
  new dependency, which needs Onik's sign-off (global rule).
- Word-level: yes, for prose. A tiptap paragraph is one line, so a one-word fix in a
  590-character paragraph shows as the whole paragraph removed and re-added, and a reader
  must hunt for the change. Code lines (automations) are short and read fine as line diffs.
  Prose needs a second pass inside each changed line, paired remove/add, highlighting
  changed words (jsdiff `diffWords`, or a small tokenised LCS over the two lines, which
  is cheap since lines are short). Memories and docs share this; ticket and play bodies
  would too.

## Open points for later tickets

- Docs: collab flushes write no row, so a "history" of editor work needs a decision
  (snapshot on a timer, on session close, or squash by time window). Counts in the demo
  cannot tell us the real rate.
- No-op saves write a row for docs (`Update`) and memories; a "skip when the body and title
  are unchanged" rule is a one-line check each.
- Unnamed doc rows have no author; memories do.
- Automation history keeps only the last pending row, so superseded pushes are not history.
