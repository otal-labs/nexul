# 16 — Memories as their own entity

**What to build:** A Memories page in the workspace lists memories grouped by project; a member with `memories:write` creates and edits one in the rich-text editor with a title, a one-line when-to-use, and an always-included switch; `memories:read` sees the page. The "Agent memory" switch disappears from docs, and memories never appear in the docs list or search. Every Agent chat turn's memories index now reads from memories, and the `nexul-memory` skill text in settings speaks of memory tools. MCP gains list, get, create, update, delete. Creating a project seeds one always-included memory, "Working in this project", with a starter body. Ticked docs are not migrated: the schema restarts before any public deploy.

**Blocked by:** 15

**Status:** done

- [ ] Memories page lists, creates, edits, deletes under the new permissions; gateway routes and MCP tools mirror each other
- [ ] The docs domain no longer has a memory flag or when-to-use; docs list, doc page, and search show no memory
- [ ] The chat prompt's memories index lists the project's memories by title and when-to-use; `doc_get` in the prompt text becomes `memory_get`
- [ ] The skill text in settings and the fallback text in the prompt say the same thing about memory tools
- [ ] A new project has "Working in this project" marked always included
- [ ] `memory.created`, `memory.updated`, `memory.deleted` are catalog rows written through the outbox
