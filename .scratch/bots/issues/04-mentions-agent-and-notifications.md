# 04 — Mentions, the Agent, and notifications

**Type:** grilling
**Status:** open
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
