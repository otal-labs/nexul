# 13 — Restore a doc version, from the web and MCP

**Status:** ready-for-agent
**Blocked by:** 12

Spec: `../spec.md`, decision 6 and Surfaces (MCP).

`POST /api/docs/{id}/versions/{n}/restore` writes the old title and body as a
new version, saving the open sitting first, then resets the room. `doc_get`
takes `version` and lists recent versions; `doc_update` takes
`revert_to_version`, sent alone.

Done when: tests cover a restore on top, the sitting saved before it, a
locked doc refused, a reader without write refused, the MCP version read and
revert; the route is in the route permission list and the audit test.
