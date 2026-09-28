# 29 — Docs reader on the phone

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 25
**Decided in:** tickets 11, 12

## What to build

Under More: Docs lists a project's docs (project picker as on Board), and a
doc screen renders its markdown read-only (headings, lists, code blocks in
mono, images, links; mention chips as plain mono text). Same endpoints as
`web/src/hooks/DocHooks.tsx`.

## Acceptance criteria

- [ ] A doc reads cleanly at phone width, including long code blocks
- [ ] Links to other docs open the doc screen

## Surfaces

- UI (native)

## Read first

`AGENTS.md`, `practices/native.md` (written by ticket 24), `practices/react-guide.md` (F1–F7 and the self-review checklist, applied to React Native), `practices/testing.md`, `practices/borrowed-practices.md`, ticket 11.

## Verification

In `native/`: `bun run lint`, `bun run typecheck`, `bun run test`. Screenshots from an Android emulator or a web preview of the screens at a phone width, kept out of git.

## Files likely touched

- `native/src/app/(tabs)/more/docs/`
- `native/src/api/docs.ts`

**Size:** M
