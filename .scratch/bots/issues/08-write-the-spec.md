# 08 — Write the spec

**Type:** task
**Status:** resolved
**Blocked by:** 01, 02, 03, 04, 05, 06, 07

## Question

Fold every resolved ticket into `.scratch/bots/spec.md` in the repo's spec
shape (problem, solution, user stories, implementation decisions, testing
decisions, out of scope), update `CONTEXT.md` with **Bot** and the
`botwebhook` permission domain, record the author-identity change as an ADR
if the model ticket made one, and mark the map's destination reached. Then
this effort is ready for `/to-tickets`.

## Answer

Written 2026-10-06: `.scratch/bots/spec.md` folds tickets 01 to 07, with
ticket 06's two corrections (the forward-only author migration, and bots
cascading with a deleted channel). `CONTEXT.md` gains **Bot** and the
`botwebhook` domain; ADR 0129 records that a message's author is no longer
always a user. The MCP ceiling ADR ships with the build, as ticket 06
decided. Ready for `/to-tickets`.
