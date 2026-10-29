# Who reaches history, and where

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

Four smaller edges left on the map: who may see an item's history, how
agents read and restore it over MCP within the tool budget, whether an open
history list updates when someone else saves, and whether the phone app
shows history at all.

## Answer

- Anyone who can read an item sees its history. It holds only text and
  fields they could have seen at the time, and reviewers, who often can't
  edit, are the people who want it most. Restoring still needs edit
  permission (ticket 04).
- Agents reach history the way memories already do, with no new tools:
  `doc_get`, `ticket_get` and `play_list` take a `version` to read an old
  one and list the recent versions; `doc_update`, `ticket_update` and
  `play_update` take `revert_to_version`, sent alone. A ticket's and a
  play's field changes come back from the same `get`, newest first. The
  tool budget is untouched.
- An open history list does not update by itself. It loads when the dialog
  opens; reopening shows anything new. No live topic, follower or audience
  rule.
- The phone app shows no history. Comparing versions needs a wide screen and
  is too much on a phone; agents reach history over MCP from anywhere.
