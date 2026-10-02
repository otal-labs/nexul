# 13 — Ticket search finds note text

**What to build:** A note's markdown is indexed so searching for its text
finds its ticket, as the body's appended context used to be. A new
numbered forward-only migration adds the index and backfills existing
notes; posting, replacing, and deleting a note keep it current.

**Blocked by:** 09

**Status:** ready-for-agent

- [ ] Searching a phrase that appears only in a note returns its ticket, in the web and over MCP
- [ ] Replacing or deleting the note updates the result
- [ ] The migration upgrades a database from the previous schema with notes already present
