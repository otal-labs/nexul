# 03: How an agent leaves a note over MCP

Type: grilling
Status: resolved
Blocked by: 02

## Question

An agent needs one call that posts the note's message and attaches its
file to the ticket, shown as the Agent rather than as the person, and never
starting an agent turn. What is that call?

- Extend `message_post` with an optional file (name and markdown content)
  that only a ticket's thread accepts, or add a tool inside the tool budget.
- Authorship: only notes show as the Agent ("from" the person whose token
  posted), or every `message_post` call over MCP does.
- Permissions: what posting a note needs (ticket write, thread post, or
  both), checked in the use-case so the gateway and MCP agree.
- The HTTP gateway counterpart, so the browser and integrations can leave a
  note too.

Technical: arrives as a decided answer for a yes or no.

## Answer

- `message_post` gains an optional file (name and markdown) that only a
  ticket's thread accepts; the name is forced to end in `.md`. No new tool:
  the server is at 104 against a budget of 105.
- A post with a file is a note: it posts as the Agent on behalf of the
  caller ("Agent · via <person>") and never starts a turn, because turns
  start only for user-authored messages. A post without a file keeps
  posting as the caller, so `@Agent` over MCP still works.
- Leaving a note needs `tickets:write` on the ticket, checked in the
  use-case so the gateway and MCP agree; Access.tsx gets the line.
- Agents read notes through `message_list` on the ticket's thread, which
  returns a note's markdown with its message. Today a non-image file in a
  turn becomes "[attachment omitted]" and follow-up prompts drop
  Agent-authored messages, so both paths need to carry note text.
- The HTTP gateway takes the same file field on its post-message route.
- Build-time traps: the code already calls system messages "notes"
  (`PostSystemMessage`, `byNoteBody`, `passedNote`), so rename one side;
  `clearStream` in `useLiveEvents.tsx` and `answeredAfter` in
  `MessageList.tsx` treat any Agent message as a turn's end and must skip
  notes.
