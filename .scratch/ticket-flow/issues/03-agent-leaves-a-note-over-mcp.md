# 03: How an agent leaves a note over MCP

Type: grilling
Status: open
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
