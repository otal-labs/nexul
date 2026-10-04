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

**Status:** ready-for-agent

Read first: `practices/react-guide.md` (F1–F7 and the self-review checklist), `practices/design-language.md`, `practices/typescript.md`, `practices/native.md`, `practices/testing.md`, the spec.

- [ ] Component tests for items 1–4 and 6; web verified at 768, 1024 and 1440 px with seeded hand-offs
- [ ] `bun run lint`, `typecheck`, `test`, `build` green in `web/`; `bun run lint`, `typecheck`, `test` green in `native/`
