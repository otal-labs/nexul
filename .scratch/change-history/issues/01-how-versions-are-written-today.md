# How versions are written today, and how big they get

Type: research
Status: resolved
Blocked by: None — can start immediately

## Question

For docs, memories and automations: what writes a version (every save, each
collaborative-editing flush, only a named milestone), how many versions an
entity piles up, and how large the bodies are, measured on a copy of the demo
instance's database. Also: is a doc version's stored body something a stable
text diff can be taken of (for example through `richtext.ToMarkdown`), and
does the O(n·m) LCS in `web/src/utils/DiffUtility.tsx` keep up with the
largest real bodies, or does the dialog need a word-level diff or a library?

## Answer

Full findings, with file:line pointers: [research/01-versions-today.md](../research/01-versions-today.md).

- Docs write a version on create, on every `Service.Update` (`PUT
  /api/docs/{id}`, `doc_update`) and on a named snapshot. A collaborative
  editing flush writes no version, so work done in the editor leaves no
  history, and `doc_versions.version` has gaps because the flush still bumps
  `docs.version`. Unnamed rows record no author, and a save that changes
  nothing still writes a row.
- Memories write a version on every create, update and revert, even when
  nothing changed. Automations write one per push; merge and rollback only
  flip a status.
- The demo database is seeded, so its counts (39 doc, 23 memory and 13
  automation versions, largest doc body 7 KB, 72 markdown lines) say nothing
  about how often real use writes versions.
- A doc version is Tiptap JSON (older rows may be markdown).
  `richtext.ToMarkdown` is deterministic, which makes a line diff of the
  markdown sound, but it drops marks with no markdown form (underline,
  highlight), so a formatting-only change shows no diff.
- The current LCS diff takes 24 ms at 1,000 lines and 459 ms at 5,000 lines,
  about 100 times the largest demo doc, so it holds. Prose needs a
  word-level diff inside a changed line, because a paragraph is one line.
  No diff library is installed, and adding one needs the owner's sign-off.
