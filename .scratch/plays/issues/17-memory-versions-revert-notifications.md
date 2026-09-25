# 17 — Memory versions, revert, and told-not-asked

**What to build:** Every save of a memory appends a version with author and time; the memory page shows the history and any version can be reverted to, which appends a new version. An agent updating a memory through MCP does so with no approval step, and the update is attributed to the Agent via the mentioning user. `memory.updated` reaches the inbox of every member with `memories:read`, naming the author and version. MCP gains list versions and revert.

**Blocked by:** 16

**Status:** done

- [ ] Editing a memory twice shows two versions; reverting to the first produces a third whose body equals the first
- [ ] The notification says who changed which memory and to which version, and links to it
- [ ] A memory written by the Agent in a chat turn shows the Agent-via-user author on its version
- [ ] `memory_get` and `memory_update` tools and routes exist and are gated by `memories:read` and `memories:write`
