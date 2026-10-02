# 07: Live editing a note

Type: grilling
Status: open
Blocked by: None — can start immediately

## Question

A note's markdown file is edited live, the way a doc body is: people in the
same note see each other's changes as they type, and a save replaces the
file with no history. How does the doc collaboration machinery serve a
note's file?

- What the live session is keyed on and who may join it (anyone who may
  write the ticket).
- What is stored: the markdown file as the source of truth with the live
  state rebuilt from it, or a live state persisted beside it.
- When the file is written back, and what the thread and other viewers see
  when it changes.
- How an agent's edit over MCP lands while people are in the session.
- Whether the dialog reuses the doc editor as is.

Technical: arrives as a decided answer for a yes or no.
