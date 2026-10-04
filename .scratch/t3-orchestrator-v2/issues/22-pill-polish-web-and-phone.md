# 22 — Hand-off pill polish on web and phone

**What to build:** The small UI gaps the reviews of tickets 14 and 19 found.

Web (`web/src/components/handoff/`, `web/src/components/chat/AgentStreamBubble.tsx`,
`web/src/hooks/useLiveEvents.tsx`):
1. A pill with no model is announced without a stray space before the comma (`HandoffPill.tsx`).
2. The pulsing "…" placeholder does not show under the pills when a frame has hand-offs but no text or
   activity yet.
3. A `left_running` hand-off with no reply says "Still running in T3 Code." instead of "No reply came back."
4. A mid-turn question (and a live reconnect that misses frames) no longer drops the live hand-off pills;
   keep the stream's hand-offs when a question clears the text.
5. Import order in `HandoffStateDot.tsx` per `practices/react-guide.md`.

Phone (`native/src/components/chat/HandoffConversation.tsx`, `HandoffScreen.test.tsx`):
6. Match the web conversation: a finished hand-off with no reply says so ("No reply came back.", or "Still
   running in T3 Code." for left running), a running one with no steps shows "Working…", and the driver mark
   shows where the web shows it, if `native/` has a shared mark; otherwise leave it out.
7. Test names say what the fixture is (drop "(the reply was deleted)" or set `deleted_at`); the api-call
   assertion uses `expect.stringContaining('/conversations/c1/messages')`.

**Blocked by:** None — can start immediately

**Status:** done

Read first: `practices/react-guide.md` (F1–F7 and the self-review checklist), `practices/design-language.md`, `practices/typescript.md`, `practices/native.md`, `practices/testing.md`, the spec.

- [x] Component tests for items 1–4 and 6; web verified at 768, 1024 and 1440 px with seeded hand-offs
- [x] `bun run lint`, `typecheck`, `test`, `build` green in `web/`; `bun run lint`, `typecheck`, `test` green in `native/`

## Comments

- **1, the pill's name.** Chrome names a button from its flex items with a space around each, so the `sr-only`
  ", Done" read "Review the auth module , Done" on every pill, model or not; jsdom joins inline spans and hid it.
  `DialogPill` takes an optional `name` (its `aria-label`), and `HandoffPill` passes "title model, State" (the model
  only when there is one). The `{" "}` and `sr-only` spans are gone; `NotePill` passes no name and is unchanged.
- **2.** The live bubble's "…" placeholder needs no hand-offs as well as no text, step or play block.
- **3.** A finished hand-off with no reply says "Still running in T3 Code." when it is `left_running`, else "No reply
  came back."
- **4, questions.** `useLiveEvents`'s `yieldStream`: an Agent question message keeps the stream when it holds
  hand-offs, with its text and step cleared and its start time and streaming flag kept, so the pills and Stop stay
  under the question card. The reply (any Agent message that is neither a note nor a question) and the pipeline's
  empty clear frame still remove it all, and a question with no hand-offs clears the bubble as before. This replaces
  ticket 14's note that a question drops the live pills.
- **4, reconnects (reading).** Nothing on the web drops hand-offs on a reconnect: frames merge by id and the socket's
  reopen clears no stream. A hand-off whose every frame fell in the gap shows on its next change, and the stored reply
  carries it. Showing it sooner needs a frame that carries the whole set, which spec decision 16 rules out (one
  changed child per frame), so it is left there; ticket 16 can watch a live reconnect.
- **6, phone.** `HandoffConversation` shows `LoadingDisplay` "Working…" for a running hand-off with no steps, and a
  muted line ("No reply came back.", or "Still running in T3 Code." for left running) for any other state without a
  reply; a state the app does not know counts as finished, as on the web. `native/` has no shared provider mark
  (ticket 19 kept brands off the phone), so the conversation shows none.
- **Browser check.** This branch's server on its own port with dev login, Vite through an uncommitted config proxying
  to it, `bootstrap-status` stubbed in headless Chromium. Six seeded hand-offs (every state, two without a model, two
  without a reply) on an Agent reply in #general, and a question message in a second channel with the live store set
  as a question leaves it. At 768, 1024 and 1440 px: six pills, Chrome's names "title model, State" with no stray
  space, the two no-reply lines in the dialog, the pills under the question card with no placeholder, and no console
  errors.
- **Device check.** The `nexul_pass_2` debug build ran this branch's bundle (Metro on the port the build expects,
  through `adb reverse`). It was signed in to ticket 19's stopped server, so it was signed out and connected to this
  branch's server with the `nexul://connect` deep link. The left-running, running-with-no-steps and done-with-no-reply
  screens read right in dark and light. An `agent-device` session from earlier today still held the device, so taps
  went through adb. The app now points at a stopped server on `localhost:18122`; reconnect it before the next check.
- **Jest.** In a fresh worktree the first `jest` run sat idle on watchman; `--no-watchman` ran, and plain
  `bun run test` passed afterwards.
