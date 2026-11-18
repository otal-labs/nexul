---
name: Finishing a ticket
description: Use when a run built a ticket, found its pull request merged, or was asked to merge one, to merge it and set its board column.
always_included: true
footer: true
---

This is for a run that built a ticket, found its pull request already merged, or was asked to merge it; a run that tested it, and any other run, skips it. A build run is finished only when the ticket sits in the right column: a merged pull request with the ticket still in progress is not finished.

1. Wait for the pull request's checks, where it has any, to finish; if your change failed one, fix it and wait again. Then, if an acceptance criterion is unmet, or a test, linter or check still fails, merge nothing, post a note on the ticket saying which and why, leave the ticket where it is, and stop.
2. If the pull request is still open, merge it the way the rules say only if someone who did not write the code reviewed it and found nothing that blocks (if no one has and you did not write it, review it now as the memory "Reviewing a change" says), and either a person's message asked for this merge or this project's rules let an agent merge its own pull request; otherwise leave it open. Keep the branch if step 4 sends the ticket to a testing column, where its preview is tested; otherwise, once merged, delete it.
3. Post a note on the ticket (`message_post` with `file`): one line per acceptance criterion saying what changed and where to see it, with the proof, and each finding still open, blocking or not.
4. Move the ticket with `ticket_update` to a column of the right stage; `project_get` lists each column's `stage`. Merged, and `ticket_get` shows a tester or a `test_target` url: the first testing column, or the first done column when the board has none. Merged, with neither: the first done column. A pull request left unmerged: the first review column, or where it is when the board has none.
5. After the move, do nothing but reply.
