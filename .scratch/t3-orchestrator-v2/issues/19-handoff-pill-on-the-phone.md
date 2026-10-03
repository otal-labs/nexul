# 19 — Hand-off pills on the phone

**What to build:** `native/` shows the same pills on Agent replies that carry `handoffs` (next to
`NoteFilePill` in `native/src/components/chat/MessageRow.tsx`), opening a sheet with the helper's
prompt, steps and reply from the stored hand-off. The model in `native/src/models/Chat.tsx` gains the
field. Stored hand-offs only; the live bubble on the phone stays as it is.

**Blocked by:** 13

**Status:** ready-for-agent

Read first: `practices/native.md`, `practices/react-guide.md`, `practices/design-language.md`, `practices/testing.md`, the spec.

- [ ] Component tests: a pill per hand-off, the sheet shows prompt, steps and reply
- [ ] `bun run lint`, `bun run typecheck`, `bun run test` green in `native/`; checked on the emulator through the T3 Device panel
