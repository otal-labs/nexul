# 05: How a new action is kept from slipping back

Type: grilling
Status: resolved
Blocked by: 01, 03

## Question

A rule nobody checks drifts. How does the repo make sure a new write action is hidden or locked like the rest?
- A shared component every action goes through, which takes the permission and renders hidden, locked or live.
- A lint rule, if one can spot an unguarded write.
- A test that renders each page as a read-only role and lists what's live.
- Or a mix.

Where does it run: `make`, the web job, `AGENTS.md`'s enforcement table?

## Answer

Owner's calls, 2026-10-13.

- **One shared piece draws every action.** That is a button, menu item, inline edit or drag handle that changes
  something.
  - It takes the permission and works out live, locked or hidden from the rule. The rule is the role-wide answer,
    plus the item's own answer where the page has one, plus the Show locked actions switch.
  - The rule lives once in `client-core/`, and the web and the phone each have their components.
  - Nobody writes a check by hand.

  Recorded from the recommendation; the owner made no objection after it was explained.
- **The guard ties the permission to the request.**
  - Every web and phone request that writes declares its permission where it is defined, and the shared piece reads
    it from there.
  - A generated route → permission file comes from the server's route table, through `make route-permissions`, the
    same pattern as `make live-topics`.
  - A test fails when a request declares no permission or a different one than its route checks. That also fixes
    the 8 actions research 01 found gated on the wrong permission.
  - The check goes in `AGENTS.md`'s enforcement table.
- **Locked by a rule on one item**:
  - An item's own page (stack, doc, play, project) reads that item's answers once through `GET
    /api/permissions/explain`, so its actions lock with the rule's sentence.
  - Lists stay role-wide. A refusal on click names the rule.
  - explain grows to stacks, machines and the instance areas as the rework's tickets 19, 23 and 24 add their rules.
