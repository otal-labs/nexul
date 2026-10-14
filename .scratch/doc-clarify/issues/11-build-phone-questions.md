# 11 — Answering on the phone

**What to build:** The phone view picked in 06, on the doc screen in
`native/`, answering through 07's routes, with the clarification topics added
to the live-event invalidation. Radio and checkbox rows come from React
Native Reusables through the shadcn CLI. An app from before this change
keeps showing the doc without questions.

**Blocked by:** 06, 07

**Status:** resolved

- [ ] Verified on the emulator through the device panel at the default and a
      360dp width, default and largest font scale, light and dark

## Answer

Built on the doc screen in `native/`: the switch, the Questions side, answering through the existing clarification routes, and the clarification topics in the phone query's `refreshes`. The logic both apps run (the model, the status line, the row and fold rules, the pick rules) moved to `client-core/clarification.ts`, `client-core/questionChecklist.ts` and `client-core/question.ts`. Radio, checkbox, textarea and toggle group come from React Native Reusables. Editing follows `docs:write` on the doc's project, "no gaps left" follows the server's `can_close`, and the API is unchanged, so an older app keeps showing the doc without questions. Verified on the emulator at the default and a 360dp width, default and largest font scale, light and dark.
