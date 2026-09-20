# 01 — Memories: keep them as designated docs, or make them their own thing

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Today a memory is not a separate object. It is an ordinary doc with a
"this doc is a memory" switch turned on and a one-line "when to use" phrase.
Turning the switch on is what "flagging" meant in the charting session. The
docs editor edits it, `docs:write` gates it, and the Agent's memories index
lists it by name and when-to-use. The workspace-level set lives in a
placeholder project called `project-general`.

The owner said docs and memories are different things that may happen to be
the same markdown files, and asked for a `memories:*` permission split from
docs. Settle the model:

1. Keep a memory as a designated doc (built, reuses the editor and search),
   and add the `memories:read/write` permission on top: editing a designated
   doc's content requires `memories:write`, and `docs:write` alone can no
   longer touch it. Or make Memory its own entity with its own table, page,
   and editor, and stop reusing docs.
2. Whatever the answer, what does `memories:read` gate: the picker in the run
   dialog only, or also seeing the memory in the docs list?
3. Where do memories live in the UI once they have their own permission: a
   "Memories" section in workspace settings, a filter on the docs list, or
   both?
4. The prompt inlines a selected memory "in full": full means the doc's
   markdown export (ADR 0026: rich text is canonical, markdown is a
   conversion surface). Confirm, and set a size ceiling per memory and per
   run so a huge memory cannot blow the harness turn limit
   (`MaxPromptChars` in `internal/agent/prompt.go`).
5. Workspace-level memories today sit in `project-general`; with real
   multi-workspace, confirm they become workspace-scoped rows rather than
   stay in a placeholder project.

Recommendation going in: option 1 (designated doc plus the new permission),
`memories:read` gates the picker and the index, and a settings section that
lists memories by name and when-to-use with a link to the doc.

## Answer

Resolved 2026-09-16 with the owner (two grilling rounds).

- **Memory is its own entity**, not a doc with a tick. Own table, own
  "Memories" page in the workspace, own MCP tools. The "Agent memory" switch
  on docs goes away. Docs are for clients and requirements; memories are
  process and context for agents, and the two never share a list, a page,
  or a search result.
- **Fields**: title, one-line when-to-use, body, project.
- **Scope is the project.** Every memory belongs to exactly one project;
  there is no workspace-wide set. The reason: agents write memories too, and
  what they learn is mostly per project. A memory that should apply
  elsewhere is **copied** to another project or workspace through an
  explicit copy action in the Memories page. A copy is independent and may
  drift; the owner accepts that over an optional project scope.
- **Body is rich text** in the same editor docs use (images and links
  allowed), exported to markdown at run time (ADR 0026). What the export
  does with an embedded image is a prompt-composition question, ticket 05.
- **Permissions**: `memories:read` sees the Memories page and the picker in
  the run dialog (the picker shows titles only); `memories:write` creates,
  edits, copies, and deletes. Reading docs grants nothing about memories and
  the reverse.
- **Agents write memories** through `memory_create` and `memory_update`,
  gated by the mentioning user's `memories:write`, so "@Agent remember X"
  keeps working and an agent can keep its own documentation current.
- **Inlining**: a selected memory is inlined in full as markdown, with a
  per-memory and a per-run character ceiling next to the existing prompt
  limit; a selection over the run ceiling is refused in the dialog with the
  totals shown. Exact numbers are tuned in the build.

Build-time details that follow, not decisions: existing ticked docs migrate
into memories of their own project, and the `project-general` placeholder's
memories are copied into every project of the workspace; the memories index
in the chat prompt and the `nexul-memory` skill text change from doc tools
to memory tools.

### Addendum, same day

- **Every memory is versioned**, the way docs are (`doc_versions`: one row
  per update with author and time; see `DocVersion` in
  `internal/docs/model.go`). The memory page shows the history and any
  version can be **reverted to**, which appends a new version rather than
  rewriting one.
- **No approval step for agent writes.** An agent updates a memory
  directly; the write is still attributed to, and bounded by, the mentioning
  user's `memories:write` (ADR 0029), but nobody is asked to confirm it.
  The owner's reasoning: the agent should have as much good context as it
  can keep current, and a gate would make it stop maintaining it.
- **People are told, not asked.** A memory update publishes `memory.updated`
  with the author (a user, or the Agent via a user) and the version number,
  and the workspace turns it into an inbox notification for members who can
  read memories. The version history is the safety net if a write was wrong.

### Addendum, from ticket 05

A memory gains an **always included** flag. See ticket 05's answer for what
it does in a run; the Memories page shows it as a badge and the flag needs
`memories:write` to change.
