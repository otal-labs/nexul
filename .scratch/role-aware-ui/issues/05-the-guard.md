# 05: How a new action is kept from slipping back

Type: grilling
Status: open
Blocked by: 01, 03

## Question

A rule nobody checks drifts. How does the repo make sure a new write action is hidden or locked like the rest?
- A shared component every action goes through, which takes the permission and renders hidden, locked or live.
- A lint rule, if one can spot an unguarded write.
- A test that renders each page as a read-only role and lists what's live.
- Or a mix.

Where does it run: `make`, the web job, `AGENTS.md`'s enforcement table?
