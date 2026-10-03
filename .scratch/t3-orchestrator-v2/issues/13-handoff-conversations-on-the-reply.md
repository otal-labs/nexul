# 13 — Hand-off conversations travel with the Agent's reply

**What to build:** Spec decisions 16–17, server side. **Needs the owner's OK on the column before it
starts.**
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

**Blocked by:** 07, 11, and the owner's OK on the column

**Status:** needs-info

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md` and `research/runs-nexul-did-not-start.md`. Also `practices/mcp.md`.

- [ ] Migration upgrade test; old messages read with `handoffs` null; deleting a reply clears them
- [ ] Child mapping table: steps mapped with `runId: null`, caps enforced, state mapping, no child thread → prompt and result only, nested hand-offs appear as steps
- [ ] A child step carrying a `dep_` token is stored and published redacted
- [ ] Pipeline: one hand-off per stream frame, the final set on the reply, none → null
- [ ] `message_list` for a reply with hand-offs has no steps and stays under 10,000 tokens; `make sqlc-check`, `make lint`, `make coverage` green
