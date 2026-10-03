# 13 — Hand-off conversations travel with the Agent's reply

**What to build:** Spec decisions 16–17, server side. The owner approved the column on 2026-10-03.
- `harness`: `Handoff{ID, Driver, Model, Title, Prompt, State, Reply string; Steps []Activity}` with
  State one of running, done, failed, interrupted, left_running; `Update.Handoff`. Protocol 1 never
  emits it.
- `t3clientv2`: children from `subagent.updated` and `projection.subagents` whose `runId` is followed,
  keyed by subagent id. Subscribe to `childThreadId` once non-null and map every child turn item with
  ticket 07's mapper (no run filter; provider-native children carry `runId: null`). State from
  `Subagent.status`: pending/running/waiting → running, completed or idle → done, failed → failed,
  cancelled/interrupted → interrupted; still running at the cap → left_running. Driver from the provider
  instance's driver; Model from `Subagent.model`, else the child thread's selection; Title from
  `Subagent.title`, else the prompt's first line; Reply from `Subagent.result`, else the child's last
  assistant text. A child with no child thread: prompt and result, no steps. Caps: newest 200 steps,
  child Detail 2 KiB, 20 children, prompt 2 KiB. One level only. A child waiting on a question adds the
  parent note "a handed-off agent is waiting for an answer in T3 Code".
- `chat` owns the stored and wire type `Handoff` with snake_case tags (spec decision 16) and a
  `HandoffStep` mirroring the trail step shape; `Message.Handoffs`.
- `agent`: `StreamFrame.Handoff *chat.Handoff` (singular) set only on the frame for a Handoff update;
  collect per id; `Conversations.PostAgentReply` gains `handoffs []harness.Handoff`, threaded through
  `redactedConversations` (every string through `redact.Tokens`), the `server/cmd` adapter and
  `chat.PostAgentMessage`; the stored set is trimmed to 256 KiB, oldest steps first.
- Storage: migration `0060` (re-check the number) adds nullable `messages.handoffs`; upgrade test from
  the previous schema; sqlc queries; `DeleteMessage` sets it to NULL.
- Surfaces: HTTP message output gains `handoffs`; `message_list` returns summaries {id, driver, model,
  title, state, reply, step_count} without steps, and its description gains "An Agent reply that handed
  off work carries handoffs: each helper's provider, model, title, state and final reply."; the
  `chat.message.created` catalog description mentions `handoffs`; regenerate the SDK.

**Blocked by:** 07, 11, 20

**Status:** done

Note: ticket 20 makes a `subagent.updated` without `completionDelivery` keep the known state; build the child state on that.

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md` and `research/runs-nexul-did-not-start.md`. Also `practices/mcp.md`.

- [x] Migration upgrade test; old messages read with `handoffs` null; deleting a reply clears them
- [x] Child mapping table: steps mapped with `runId: null`, caps enforced, state mapping, no child thread → prompt and result only, nested hand-offs appear as steps
- [x] A child step carrying a `dep_` token is stored and published redacted
- [x] Pipeline: one hand-off per stream frame, the final set on the reply, none → null
- [x] `message_list` for a reply with hand-offs has no steps and stays under 10,000 tokens; `make sqlc-check`, `make lint`, `make coverage` green

## Comments

- **Where it lives.** `harness.Handoff` and the five `Handoff*` states. `t3clientv2/handoff.go`: `pills` (one `pill` per
  subagent of a followed run, up to 20, ordered by `startedAt` then id) keeps each row's last known state, so a
  snapshot that drops a finished row does not drop its pill, and folds the child thread's items through `mapItem`.
  `pump.go`: `follow` subscribes to each `childThreadId` on the turn's connection (`turn.openThread`), one reader
  goroutine per child feeds `inbox`, whose `put` wakes the pump's wait on its own thread; `hear` emits the changed
  pills each loop, `finish` settles them before the Terminal. `chat`: `Handoff`, `HandoffStep` (field for field
  `harness.Activity`, as `plays.ActivityEntry` is), `NewHandoff`, `Message.Handoffs`, the 256 KiB trim in
  `PostAgentMessage`. `agent`: `StreamFrame.Handoff`, `keepHandoff`, the redaction in `redactedConversations`.
  Storage: migration `0064_message_handoffs.sql` (0060 to 0063 were taken).
- **Wire, for tickets 14 and 19.** The agent stream frame carries `handoff` (one whole hand-off, only on the frame
  for the one that changed; merge by `id`). A message carries `handoffs` only when it has some (absent, not `null`,
  otherwise). Each is `{id, driver, model, title, prompt, state, reply, steps}`; `steps` is always an array, each
  `{kind, call_id?, tool?, summary, detail?, at}`; `driver` is T3's driver (`claudeAgent`, `codex`, ...), the key
  `HarnessProviderMark` lowercases. The stored set is the latest frame of each hand-off, in the order they started.
  `message_list` returns `{id, driver, model, title, state, reply, step_count}` per hand-off.
- **Ticket 16.** Check live: a `delegate_task` child's thread (its items carry its own run ids, which nothing
  filters) and when `childThreadId` turns non-null; a provider's own subagent on Codex, OpenCode and Grok, whose rows
  may name no `model` (the child thread's selection is the fallback); a child that asks a question or an approval in
  T3; the pill reaching `done` with the reply in the stored message after a reconnect mid-turn.
- **Judgment calls.**
  - `chat` imports `harness`, as `plays` and `pairing` do, so `chat.NewHandoff` is the one conversion: the live
    frame and the stored reply use it, and `PostAgentReply` and `PostAgentMessage` take `[]harness.Handoff` as the
    ticket says.
  - The 256 KiB cut is in `chat.PostAgentMessage`, the owner of the stored shape, so the row and
    `chat.message.created` carry the same set. The oldest steps by time go first across all hand-offs. Once no step
    is left, the longest replies are cut to even shares of what the rest leaves, measured as stored JSON; a reply
    within its share stays whole. T3 cuts only the parent's `subagent` turn item at 32 KiB, never the subagent row
    (`WireProjection.ts`), so a row's `result` is the child's whole final text and nothing else bounds it. The live
    frame still carries the whole reply; only the stored set is cut.
  - A child's approval is handled like its question: a step plus the turn note, never auto-declined, since T3 made
    that thread under its own runtime mode and a person can answer it in T3. The note reads "A handed-off agent is
    waiting for an answer in T3 Code", once per request.
  - A pill still running when the turn ends becomes `left_running` at any end but Stop (the cap, a steered result, a
    run that failed or was interrupted in T3); Stop makes it `interrupted`. Only a done or interrupted turn with text
    stores a reply, so a turn that fails keeps no pills, as before.
  - The child's assistant messages are not steps (the mapper skips them); the newest by ordinal is the reply when the
    row has no `result`. The title, the row's or else the prompt's first line, goes through `harness.Preview`
    (160 runes), so a long row title cannot crowd the stored set.
  - A child subscription that fails is logged at Warn and left closed: the pill keeps its row's state and reply,
    and its steps stop. All child subscriptions reopen from their cursors after the turn's own stream resubscribes.
  - A hand-off update counts as a play's heartbeat (`Observer.OnSnapshot`).
  - The `message_list` summary keeps the first 1 KiB of each reply, and its description says so: 20 whole replies
    could pass 10,000 tokens on their own. The check is over 20 hand-offs with 4 KiB replies.
  - `bun run generate:events` ran and changed nothing: the catalog types `message` as an open object, and only its
    description gained `handoffs`. ADR 0116 gained "What the reply carries"; the `CONTEXT.md` term stays with
    ticket 17.
