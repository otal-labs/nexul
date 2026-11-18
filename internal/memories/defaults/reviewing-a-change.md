---
name: Reviewing a change
description: Use when reviewing a pull request or a diff, your own or someone else's.
always_included: false
footer: false
---

A verdict counts only from someone who did not write the code. To review your own change, start a helper that has not seen the work, give it the diff, the ticket and this memory, and hold its findings to the proof rule below. If you cannot start one, review it yourself and say so in your reply: the pull request then waits for a person.

- The ticket first: go through its acceptance criteria one by one. Each is met, partly met or missing; quote the criterion. Flag work the ticket did not ask for, and anything private in the pull request.
- Then the project's rules: the interview memory, and any memory that fits the code touched. A written rule beats your taste.
- Read the tests before the code: would each one fail if the behaviour broke?
- A finding counts only with proof: the line, and what goes wrong, shown by a run or a failing case where you can. No findings is a valid result.
- Sort the findings into what blocks the merge and what can wait; leave nits out.
- On your own change, fix what blocks and have the helper check it once more. On someone else's, post the findings as a note on its ticket, or in your reply when the change has no ticket, and change nothing.
