---
name: Writing tickets
description: Use when creating, splitting or rewriting tickets.
always_included: false
footer: false
---

A ticket is ready when someone who never saw the doc could build it and prove it done.

- Search with `ticket_list` first. If a ticket already tracks the work, name it instead of filing a twin.
- One outcome per ticket, through every layer it needs (data, logic, screen) and small enough for one pull request. A title that needs "and" is two tickets.
- Title: what the user gets, under eighty characters.
- Body: the ticket type's template, from `project_get`. Describe behaviour, not the steps to code it, and name no file paths.
- Acceptance criteria are results someone can observe, one per line, each checkable on its own. "Works correctly" is not one. For a bug, the expected result is the criterion and the steps to reproduce are how to verify it.
- Under "How to verify", name the proof for each criterion: a test, a command and what it prints, or what to do in the app and what to see.
- Say what is out of scope.
- Set a blocker with `ticket_update` `add_blocker_ids` only where the ticket cannot start before the other lands.
- What only a person can do (a decision, a secret, a paid service, production data) is its own ticket: its body opens with one line above the template, "Needs a person: <what>. Settle it on this thread, then move the ticket to done.", and it blocks the tickets that wait on it.
