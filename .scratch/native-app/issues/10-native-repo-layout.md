# 10 — native/ layout, shared code, practices and CI

**Type:** grilling
**Status:** open
**Blocked by:** None — can start immediately

## Question

How is `native/` set up in this repo? Its package and bun setup, which code it shares from `web/src/models` and `sdk/` and how (Metro `watchFolders`, no barrels, no root package.json), whether it gets its own practices file and how `AGENTS.md` routes to it, which lint, typecheck and test gates run, and its CI job and `dorny/paths-filter` entry.
