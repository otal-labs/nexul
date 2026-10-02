# 06: What agents are told about the body and notes

Type: grilling
Status: resolved
Blocked by: 03

## Question

The rule is decided: the body is the ticket's spec and changes only when a
person asks; everything an agent adds afterwards is a note; an agent may
suggest the person edit the body when the spec itself looks wrong. Where
does that rule live so every agent follows it, and in what words?

- The `ticket_update` and `message_post` tool descriptions.
- The versioned Nexul skill agents install, and the interview memory.
- The play instructions, so a play run follows it too.

Technical: arrives as a decided answer for a yes or no.

## Answer

- `ticket_update`'s `body` field: "The ticket's spec. Change it only when
  a person asks; add anything you learn afterwards as a note with
  message_post's file, and if the spec looks wrong, say so and suggest the
  person edit it."
- `message_post`'s file field says the same in one line.
- One line after the ticket body in `ComposePrompt`
  (`internal/agent/prompt.go`), so every `@Agent` mention and every play
  run on a ticket carries the rule, including plays already copied into
  existing workspaces.
- Unchanged: the server instructions (at 2,042 of their 2,048-character
  cap), the versioned Nexul skill, the interview memory, the built-in play
  texts.
