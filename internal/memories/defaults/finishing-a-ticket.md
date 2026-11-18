---
name: Finishing a ticket
description: Use at the end of a run that built a ticket, to merge it and set its board column.
always_included: true
footer: true
---

This is for a run that built a ticket; a run that tested it, and any other run, skips it. Such a run is finished only when the ticket sits in the right column: a merged pull request with the ticket still in progress is not finished.

1. If an acceptance criterion is unmet, or the project's tests and linters, where it has any, have not passed, merge nothing, post a note on the ticket saying which and why, leave the ticket where it is, and stop.
2. Merge the way the rules say only if this project's rules let an agent merge its own pull request and a review by someone who did not write the code found nothing that blocks; otherwise leave the pull request open. After a merge, keep the branch if step 4 sends the ticket to a testing column, where its preview is tested; otherwise delete it.
3. Post a note on the ticket (`message_post` with `file`): one line per acceptance criterion saying what changed and where to see it, with the proof, and each blocking finding still open.
4. Move the ticket with `ticket_update` to a column of the right stage; `project_get` lists each column's `stage`. Merged, and `ticket_get` shows a tester or a `test_target` url: the first testing column, or the first done column when the board has none. Merged, with neither: the first done column. Left open for a person: the first review column, or where it is when the board has none.
5. After the move, do nothing but reply.
