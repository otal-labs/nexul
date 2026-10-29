# Restore from history

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

Does each entity's history let you restore an old version from the diff
dialog, and what does a restore do: add a new version on top, the way
memory revert does today, or something else? Does a ticket restore every
field or only its description? Is a locked doc's history read-only?

## Answer

- The dialog offers "Restore this version" for doc bodies, ticket
  descriptions, play instructions and memories. A restore saves the old text
  as a new version on top, the way memory revert does today, so nothing is
  lost and a restore can itself be undone. Automations keep their existing
  rollback.
- Field changes (a ticket's status, a play's enabled switch) are shown but
  never undone from history. Each is one click on the entity itself, and an
  undo from history would set off automations, auto plays and notifications
  from an unexpected place.
- A restore while someone has the doc or ticket open in the live editor
  replaces the text and their editor follows, the same as an API or MCP
  save today (ADR 0124). The open sitting is saved as a version first, so
  what they typed stays in the history.
- A restore needs the same permission as an edit. A locked doc refuses a
  restore until it is unlocked; its history can still be read and diffed.
