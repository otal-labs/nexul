# 19 — Doc threads

**What to build:** A doc header gains a Thread button that opens the chat dock on the doc's one thread, created on first use like a ticket's. An `@Agent` mention there carries the doc's title and body as markdown, trimmed with a note when too long. The thread is listed on the chat page as "Doc thread" with the doc's title. Only users holding `docs:thread` on that doc (per role, or by per-doc overwrite) can see or post in it; the check lives in the chat use-case so the gateway and the MCP message tools agree.

**Blocked by:** 13, 15

**Status:** done

- [ ] Pressing Thread on a doc opens the dock on that doc's thread; a second doc has a different thread; one doc never has two
- [ ] An Agent reply in the thread references the doc's content
- [ ] A user with `docs:read` but not `docs:thread` sees no Thread button, cannot list or read the messages through the gateway or MCP, and gets a permission error, not an empty list
- [ ] The chat page lists the thread under a "Doc thread" label; unread counts and mentions work as for channel threads
