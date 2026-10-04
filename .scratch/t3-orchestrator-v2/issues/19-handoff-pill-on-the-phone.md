# 19 — Hand-off pills on the phone

**What to build:** `native/` shows the same pills on Agent replies that carry `handoffs` (next to
`NoteFilePill` in `native/src/components/chat/MessageRow.tsx`), opening a sheet with the helper's
prompt, steps and reply from the stored hand-off. The model in `native/src/models/Chat.tsx` gains the
field. Stored hand-offs only; the live bubble on the phone stays as it is.

**Blocked by:** 13

**Status:** done

Read first: `practices/native.md`, `practices/react-guide.md`, `practices/design-language.md`, `practices/testing.md`, the spec.

- [x] Component tests: a pill per hand-off, the sheet shows prompt, steps and reply
- [x] `bun run lint`, `bun run typecheck`, `bun run test` green in `native/`; checked on the emulator through the T3 Device panel

## Comments

- **Where it lives.** `models/Chat.tsx`: `Message.handoffs`, `Handoff`, `HandoffStep`, `handoffStateLabel` and
  `handoffStateDot`. `HandoffPill` sits under the body in `MessageRow`, after the note pill, one per hand-off in a
  wrapping row. A pill pushes `/chat/handoff/[id]` with `conversationId` and `messageId`; `HandoffScreen` finds the
  hand-off in the thread's own messages query (`useFetchMessages`), so it needs no fetch of its own and follows the
  thread's refetches. `HandoffConversation` renders the state line, the prompt as the opening bubble, one
  `HandoffStepRow` per step and the reply as Markdown.
- **Judgment calls.**
  - A pushed screen, not a form sheet. `practices/native.md` keeps sheets for pickers and confirms and pushes detail
    screens; the note pill beside it pushes `NoteScreen` the same way; and a form sheet fitted to its content is
    sized for a short list, not 200 steps and a long reply. The screen gets the header title and the back gesture for free.
  - No brand marks on the phone: the pill leads with lucide's `Bot`, the web's own fallback glyph, then the model in
    mono when there is one, the title, and the state dot. Porting `HarnessProviderMark`'s SVGs means
    `react-native-svg` paths for each brand; add them if the owner wants the phone to match exactly.
  - Dots: running `warning`, done `success`, failed `destructive`, interrupted `muted-foreground`, left running
    `info`. A state the app does not know (the server can be newer) reads as its raw value with the in-flight dot.
  - Steps are flat rows (tool in mono, then the summary on two lines at most, notes muted and italic). The web's turn
    folding and `stepLabel` rewriting stay web-only; T3 strips output, so `detail` is not shown.
  - A reply whose hand-offs are gone (deleted while the screen is open) shows "This hand-off is no longer on its
    reply."; a 404 on the thread drops the stale messages, as `ChatThreadScreen` does.
- **Device check (2026-10-04).** Debug build of this branch on the `nexul_pass_2` AVD through the Device panel,
  against this branch's server on a copy of the device-pass database with one seeded Agent reply carrying three
  hand-offs (done with six steps and a Markdown reply, running with no steps, left running with no model). Pills,
  truncation, the screen's state line, prompt bubble, steps and reply read in dark and light; back returns to the
  thread. The AVD now has the debug build instead of the 0.1.3 release APK.
- **Ticket 16.** Live data: a real hand-off reply opened on the phone after the turn ends, and a long one (close to
  200 steps) scrolled to the end.

