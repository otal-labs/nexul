# 04: runPlay in the SDK, with the play name checked as a literal union

Type: research
Status: resolved
Blocked by: None — can start immediately

## Question

Automations get `runPlay(play, ticket)`, where `play` must be one of the
workspace's play names as a string literal union, so a typo or a renamed
play fails the type check.

- How the SDK reaches the server today (`sdk/src/protocol.ts` frames,
  `context.ts` `buildCtx`), and how a new call is added end to end through
  the automations host.
- Play names are workspace data, not code: where the union comes from.
  How the automation editor gets its types now (`events.generated.ts` is
  generated from the Go catalog at build time), and whether the editor can
  load a per-workspace declaration at edit time. What happens on the host
  at run time when the name no longer exists.
- Does `runPlay` go through the same queue as an auto play (developer by
  default, a `runOn` option), and whose permission it checks: the
  automation token's, plus `plays:run` for the person it lands on.
- `createMockContext` in `sdk/src/testing.ts` and the docs page for the
  SDK.

Findings go in `research/04-run-play-in-the-sdk.md`.

## Answer

Recommendation in `research/04-run-play-in-the-sdk.md`, with one change from
the owner (2026-10-08): **play names are unique per workspace.** A forward
migration renames existing duplicates by appending " (2)", " (3)" in
creation order, a unique index on `(workspace_id, label)` holds it after,
and create and rename refuse a taken name with a plain error in the play
dialog and on `play_create`/`play_update`. So a name is an identity, and
`runPlay` and `nexul types` no longer need the skip-duplicates rule.
