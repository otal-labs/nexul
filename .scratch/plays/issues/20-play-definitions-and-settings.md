# 20 — Play definitions and settings

**What to build:** A Plays page in workspace settings lists the workspace's plays and lets a member with `plays:write` create, edit, enable or disable, and with `plays:delete` delete one: label, type (ticket or doc), description, base instructions, show-when stage for ticket plays, excluded projects. `plays:read` sees the page. A new workspace is seeded with "Fix with AI" (ticket, progress stage) and "To tickets via AI" (doc) using the instruction texts in the spec; existing workspaces get them once. MCP gains list, create, update, delete for definitions. No button appears anywhere yet.

**Blocked by:** 15

**Status:** done

- [ ] Plays page CRUD under the new permissions; routes and MCP tools mirror each other
- [ ] A ticket play must have exactly one show-when stage; a doc play must have none; the form and the API both reject the other way round
- [ ] Both seeded plays exist in a fresh workspace and are ordinary plays: editable, deletable, not restored by an upgrade
- [ ] Plays list sorts by label; disabled plays show as such
