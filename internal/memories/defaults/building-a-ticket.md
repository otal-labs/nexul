---
name: Building a ticket
description: Use when implementing a ticket or changing code for one.
always_included: false
footer: false
---

- Read the whole ticket first, with its attachments and thread, and the thread of any blocker whose body opens with "Needs a person". A run that is building the ticket stops and says so if this ticket's pull request merged and no failed test report came after, if this ticket's own body opens with "Needs a person", or if a blocker is neither merged nor in a done column.
- A build is one branch and one pull request. Build what the acceptance criteria ask; note anything else you notice on the ticket instead of fixing it.
- For a bug, reproduce it before you touch code, find the cause, and fix the cause.
- Write the test that fails without your change first, at the level this project tests at, and none where its rules say none. A test that restates the code is worse than none.
- Prove it in the real thing: run the project's tests and linters, where it has any, then watch each acceptance criterion hold in the app or by a command. A build that compiles and tests that pass do not prove behaviour.
- Review the diff as the memory "Reviewing a change" says, and fix what blocks before you open the pull request.
- Pull request: the title starts with the ticket key and says what changed; the body says why, what changed and how you proved it. Link it and the branch to the ticket with `ticket_update` `link_branch` and `link_pr`.
- Keep commits and the pull request free of anything private: real names, live addresses, hostnames, home paths, secrets. Screenshots and logs go in a note on the ticket.
