# 04 — Mentions, the Agent, and notifications

**Type:** grilling
**Status:** resolved
**Blocked by:** 02

## Question

A bot message can carry `@user` and `@Agent` tokens, and Discord's
`allowed_mentions` says which are live. Settle what each does in Nexul:

1. `@Agent` in a bot message: an Agent turn runs on the mentioning user's
   own harness (ADR 0029) and a bot has none. Options: never fire; fire on
   the bot creator's harness with the creator's permissions; or fire only if
   the bot was created with an explicit "may summon the Agent" switch.
2. `@user` mentions: notify like a human mention, and honour
   `allowed_mentions` so a CI bot that pastes a commit message does not ping
   everyone it names.
3. Unread state: a bot post bumps unread and the sidebar like any message,
   or is quieter (a per-bot switch, the way Discord channels can be muted).
4. Ticket and doc threads: a bot post counts as thread activity for the
   ticket page and the doc header, or not.

Recommendation going in: `@Agent` never fires from a bot in this spec (it
goes to "not yet specified" as a future bot kind); `@user` notifies with
`allowed_mentions` honoured; unread bumps normally.

## Answer

Decided 2026-10-05; the owner took all four recommendations.

- **`@Agent` never fires from a bot.** The agent pipeline already runs only
  on messages with author kind `user`, so a `bot` message is skipped with no
  new code. A bot that may summon the Agent is a future bot kind and stays in
  "not yet specified".
- **`@user` behaves exactly like a person's mention**, filtered by Discord's
  `allowed_mentions`: a handle the sender did not allow is stored as plain
  text, not a mention. Today a chat mention highlights the name and sends no
  inbox notification; if chat mentions later notify, bot mentions follow
  with `allowed_mentions` already applied.
- **Unread bumps normally.** The unread count counts every message not
  written by the reader, so a bot post counts with no change. No per-bot
  mute.
- **Ticket and doc threads count a bot post as activity**, the same as a
  person's post, since both read the thread's messages.
