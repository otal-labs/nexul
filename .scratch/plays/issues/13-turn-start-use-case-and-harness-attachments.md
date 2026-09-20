# 13 — Turn start becomes a use-case, and the harness seam carries attachments

**What to build:** Chat behaves exactly as today, but starting an Agent turn is one callable in the agent domain that the mention handler calls and a play can call too: "start a turn on this conversation, as this user, with this request text and these extra request blocks". The harness turn type gains an attachments list (name, MIME, bytes); the T3 client encodes each as a base64 data URL on the turn message, and the fake harness records what it received. Prefactor: no user-visible change.

**Blocked by:** None — can start immediately

**Status:** done

- [ ] An `@Agent` mention in a channel and in a ticket thread runs unchanged; existing agent and chat tests pass without edits to their expectations
- [ ] The new use-case accepts extra request blocks that land inside the request block of both the full and the incremental prompt, and a test proves a reused session receives them
- [ ] Harness turn prompts carry attachments; the T3 client sends them in T3's attachment shape and a test asserts the wire payload; the fake harness exposes received attachments
- [ ] No T3 type crosses the harness seam (ADR 0054)
