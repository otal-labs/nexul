# 18 — Memory clone

**What to build:** From a memory's page, a member with `memories:clone` copies it to another project in this workspace or to a project in another workspace they belong to. The clone is a new, independent memory with the same title, when-to-use, body, and attachments as new rows; it starts a fresh version history. MCP gains a clone tool.

**Blocked by:** 16

**Status:** done

- [ ] Clone within a workspace and across workspaces works from the page and from MCP
- [ ] Cloning needs `memories:clone` in the source and `memories:write` plus membership in the destination workspace; a failure says which is missing
- [ ] Editing the clone leaves the source untouched and the reverse
- [ ] Embedded images render in the clone
