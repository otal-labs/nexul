# 07: Write the ADR for plays that start by themselves

Type: task
Status: open
Blocked by: 01, 02, 03

## Question

ADR 0055 says nothing fires a play but a person, and ADR 0066 makes the
decisions check the one exception. Write the ADR that replaces both rules
with auto plays: an auto play starts a play when a moment matches, on the
person it runs on, through their queue, with the per-ticket daily cap as
the loop guard, and the decisions check becomes an ordinary play with a
seeded auto play. Mark 0055 and 0066 amended. Update `CONTEXT.md`'s Play
and Decisions check entries to match.
