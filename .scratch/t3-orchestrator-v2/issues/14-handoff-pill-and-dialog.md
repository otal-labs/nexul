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

**Status:** done

Read first: `practices/react-guide.md` (F1–F7 and the self-review checklist), `practices/design-language.md`, `practices/typescript.md`, `practices/testing.md`, the spec.

- [x] Component tests: a pill per hand-off, a dot per state, the dialog shows prompt, steps and reply, a live frame updates an open dialog, `NotePill` unchanged in behavior
- [x] On the local debug stack, seed `messages.handoffs` with fixture JSON and check stored pills at 768, 1024 and 1440 px in chat, a ticket thread and a play's thread (live data is checked in ticket 16)
- [x] `bun run lint`, `bun run typecheck`, `bun run test`, `bun run build` green in `web/`

## Comments

- **Where it lives.** `models/Handoff.tsx` (the wire type, `HANDOFF_STATE_LABELS`, `mergeHandoff`), `Message.handoffs`;
  `components/handoff/`: `HandoffPills` (the row), `HandoffPill` (pill plus its dialog), `HandoffConversation` (the
  dialog body), `HandoffStateDot`. The pill shell is `components/DialogPill.tsx`, and `NotePill` renders through it with
  the same markup as before. `SetupTranscript`'s segment renderer moved to `components/play/TurnSegment.tsx` so the
  dialog renders `segmentSetupTurn` the same way.
- **Live.** `useLiveEvents` passes a stream frame's `handoff` to `setStream`, and `agentStreamStore` keeps
  `AgentStreamFrame.handoffs` across frames, replacing by `id` and appending new ones in arrival order. The live bubble
  shows the pills under its latest step. An open dialog reads the pill's current hand-off, so a frame updates it in
  place.
- **Judgment calls.**
  - Dot colors: running `warning` with the `status-pulse` keyframe, done `success`, failed `destructive`, left running
    `info`, interrupted `muted-foreground` (Stop is the person's own act, not an error). The dot is decorative; the
    pill's accessible name ends with the state ("…, Left running") and the dialog says it in words beside the dot.
  - The dialog takes `NoteMessage`'s sizing and scrolls as one block, with no follow-the-end scroller; live rows append
    at the bottom of the open turn group.
  - A finished hand-off with no reply shows "No reply came back." A running one with no steps yet shows "Working…".
  - A dialog opened from the live bubble closes when the reply lands, because the bubble yields to the stored message;
    the same pill on the reply opens it again. A question message mid-turn clears the stream (existing behavior), so
    the live bubble drops earlier hand-offs until each one changes again; the stored reply carries them all.
  - An empty `model` shows the title and dot alone.
- **Browser check.** Run from the worktree, not the docker stack: the Go server on its own port with dev login, Vite
  through an uncommitted config proxying to it, `bootstrap-status` stubbed in Playwright, and `messages.handoffs`
  seeded with five fixture hand-offs (one per state) in #general and a ticket thread, plus two on a play's reply with
  its trail row. Pills and dialogs checked at 768, 1024 and 1440 px, no console errors.
- **Ticket 16.** Watch the live bubble with a real `delegate_task` child: pills arriving per frame, an open dialog
  growing, and the swap to the stored pills when the reply lands.
- **Ticket 19.** The phone mirrors the five labels in `HANDOFF_STATE_LABELS` and the dot colors above.
