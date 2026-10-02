# 10 — The body-or-note rule in what agents read

**What to build:** The rule from [What agents are told](06-what-agents-are-told.md):
the `ticket_update` body field's description, the `message_post` file
field's description, and one line after the ticket body in `ComposePrompt`
(`internal/agent/prompt.go`). The server instructions, the versioned Nexul
skill, the interview memory, and the built-in play texts stay unchanged.
A play's trail result stays its final word; the line lets its agent leave
a note only when it judges there is lasting context.

**Blocked by:** 09

**Status:** ready-for-agent

- [ ] The three texts carry the rule in the decided words, and the MCP surface test's description checks pass
- [ ] Every agent turn on a ticket, mention or play, includes the line; a plain chat turn does not
