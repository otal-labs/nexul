# 14 — The hand-off pill and its conversation

**What to build:** One pill per hand-off on an Agent message with `handoffs` (next to `NoteMessage` in
`web/src/components/chat/MessageRow.tsx`) and in the live working bubble (`MessageList.tsx`, from
`web/src/stores/agentStreamStore.tsx`, which merges incoming frames by hand-off id). The pill: provider
mark (`web/src/components/model/HarnessProviderMark.tsx`), model, title, a status dot (running, done,
failed, interrupted, left running). Clicking opens a dialog with that agent's conversation: its prompt
as the opening bubble, its steps and final reply, live while running and stored afterwards.
- Extract the pill shell from `web/src/components/note/NotePill.tsx` into a shared component that takes
  an icon and trailing content, and reuse it for `NotePill`.
- Use the `Dialog`/`DialogContent` sizing from `web/src/components/note/NoteMessage.tsx`.
- Render the conversation the way `web/src/components/pairing/SetupTranscript.tsx` does
  (`segmentSetupTurn` + `TrailTurnGroup` + `TrailReplyProse`), which needs no Trail.
- Mono Console tokens, color only for the status dot. F1–F7.

**Blocked by:** 13

**Status:** ready-for-agent

Read first: `practices/react-guide.md` (F1–F7 and the self-review checklist), `practices/design-language.md`, `practices/typescript.md`, `practices/testing.md`, the spec.

- [ ] Component tests: a pill per hand-off, a dot per state, the dialog shows prompt, steps and reply, a live frame updates an open dialog, `NotePill` unchanged in behavior
- [ ] On the local debug stack, seed `messages.handoffs` with fixture JSON and check stored pills at 768, 1024 and 1440 px in chat, a ticket thread and a play's thread (live data is checked in ticket 16)
- [ ] `bun run lint`, `bun run typecheck`, `bun run test`, `bun run build` green in `web/`
