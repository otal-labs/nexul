# An agent prompt names its context instead of carrying it

Supersedes what remained of ADR 0065 for `@Agent` mentions, and extends ADR 0105 to mentions. Amends ADR 0064 (the
origin is named, not carried) and ADR 0103 (a kind that lives only at the instance).

The prompt a turn sent the harness had grown far past what the agent needed. One play run on a ticket came to over
11,000 characters: the ticket body inlined as the stored editor tree rather than markdown, about 7,000 characters of
it; a memories protocol paragraph of over a thousand characters with a clause about the skill's version; the memories
listed twice, once as the index and once to read first; the "Started" message twice, in the history and as the
request; and a system notice about the harness release. A mention carried the always-included memories in full and
the index of every other memory on top.

Decision: a turn's prompt names each piece of context and the tool that reads it, and carries none of it.

- **The shape.** A full prompt opens with the Intro, says what is running and for whom ("running a play, started
  by", "running @Agent in a chat, mentioned by"), and closes with the Footer. Between them a mention has the ticket
  or doc line, the memories to read, the conversation so far, and the new message; a play has the play and its
  instructions, the ticket or doc line, its link blocks, the memories to read, and the starter's instructions for
  the run.
- **The ticket or doc** is a line: its key or id, its title, and "Read it with `ticket_get`" or `doc_get` "before
  you start". Images its body embeds still travel as attachments when the harness takes their type and they fit the
  size caps, and a line under it says so only when at least one was attached. The others are listed by link for
  `attachment_get`, so the prompt never claims an image the agent did not get. The body is read as markdown for
  that, which is also what makes a ticket's images reach the harness; the stored editor tree has none of the markdown
  image links the scan looks for.
- **Memories.** Both turn kinds list the memories to read first by name and id, in one place in code: a play its
  ordered selection (ADR 0105), a mention only the project's always-included memories, the interview memory first.
  No turn carries a memory's body, its when-to-use line, or an index of the rest; the agent finds those with
  `memory_list`, as the Footer says. The protocol paragraph and the skill-version clause are gone: the Footer keeps
  the rules that matter in every turn, and pairing already tracks which skill version a computer has.
- **A play turn carries no conversation.** The "Started" message is still posted to the thread, but the prompt has
  no history and no request block: the play is the request. The conversation's cursor still moves to its newest
  message, so a later mention on the same session does not send what the run skipped. An answer that resumes a run
  on a session the harness lost sends the play, the ticket or doc line, the trail's recorded memories, and the
  answer; a live session gets the answer alone.
- **System messages** (a harness release notice, "Agent turn failed") are left out of every prompt, full and
  follow-up. They are the server's notices to people, not conversation.
- **The origin** of a bug is a line naming its key and title, telling the agent to read its body, doc, and pull
  requests with `ticket_get`, still one hop.
- **Intro and Footer are instance templates** of the kind `agent_prompt`, keys `intro` and `footer`, with the code
  defaults until an owner edits them. They live only at the instance: no workspace or project layer and nothing to
  clone to, because the Agent is the instance's and its framing should not differ by workspace. An empty template
  leaves its part out of the prompt. A failed read falls back to the code default and logs a warning.

The trade-offs: the agent makes one more tool call for each thing named before it starts, which costs a round trip
per memory and one for the ticket or doc. A harness without Nexul's MCP tools now gets no ticket, doc, or memory text
at all, where a mention used to carry the always-included memories and the ticket body. Ticket and doc images are the
exception, because a harness cannot fetch an attachment URL, so they still ride with the turn. The read-first line no
longer tells the agent what to do when a memory cannot be read; it reports the failed call as it would any other.

Rejected: inlining the ticket as markdown instead of the editor tree, which fixes the worst of the size but still
carries a body that can be long and goes stale within a long session, where `ticket_get` reads it as it stands;
keeping a capped memories index, which repeats in every prompt what `memory_list` answers on demand; and giving the
Intro and Footer a workspace layer, which nobody asked for and would let two workspaces frame one Agent differently.

Decided 2026-10-03.
