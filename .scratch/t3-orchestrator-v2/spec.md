# T3 Code orchestrator V2

**Status:** ready-for-agent

## Objective

T3 Code merged a new orchestrator (pingdotgg/t3code#2829, 2026-10-02). T3 calls it orchestrator V2,
and its wire protocol is "protocol 2". Nexul speaks protocol 1 today.

- Protocol 2 ships in nightlies from `v0.0.46-nightly.20261003.2610`.
- Stable `v0.0.45` still speaks protocol 1, and T3 has given no date for stable to switch
  (pingdotgg/t3code#14871).

A computer whose T3 Code runs a nightly is unreachable from Nexul today:

- **`/ws` refuses the connection.** It answers HTTP 426 `orchestration_protocol_incompatible`
  unless the URL carries `orchestrationProtocol=2`.
  - `internal/t3client` treats the dial error as retryable, so presence redials forever.
  - Every turn fails setup with "is T3 Code running there?".
  - Pairing still looks fine, because the token exchange did not change.
- **Four of the five commands Nexul sends are gone:** `thread.turn.start`, `thread.turn.interrupt`,
  `thread.approval.respond` and `thread.user-input.respond`.
- **All three events it reads are gone:** `thread.message-sent`, `thread.session-set` and
  `thread.activity-appended`.

This effort does three things:
- makes Nexul work on both protocols;
- moves each computer forward once and never back;
- adopts the one orchestrator V2 capability that changes what a person sees: agents handing work to
  other agents. The handed-off work becomes a pill on the Agent's reply, and the pill opens that
  agent's conversation.

Success means:
- a person on T3 stable notices nothing;
- a person who updates T3 to the nightly keeps working without re-pairing;
- an @Agent mention or play whose agent delegates work replies with the finished result and shows
  what the helper did.

## What did not change

Read from source (`research/protocol-2-wire.md`):

- **Pairing:**
  - `POST /oauth/token` with the `environment-bootstrap` subject token, and the scope string;
  - `POST /api/auth/websocket-ticket`;
  - `t3 pair`.
- **The connection:** the Effect RPC frame envelope, the `server.getConfig` provider and model
  shape, and the `orchestration.subscribeShell` project snapshot.
- **Stored ids:** T3's first V2 start copies `state.sqlite` to `statev2.sqlite` once, keeping thread
  ids, project ids, provider instance ids and bearer sessions. Every T3 id Nexul stores stays valid,
  and nobody re-pairs.
- **The protocol number:** both protocols put `orchestrationProtocolVersion` in the
  `/.well-known/t3/environment` descriptor and in `server.getConfig`'s `environment` field.
  `v0.0.45` sends `1`; absent means 1.

## Decisions

Owner's calls (2026-10-03):

1. **A separate client for orchestrator V2, under its own harness kind.**
   - `internal/t3clientv2` implements `harness.Client` for `harness.KindT3CodeV2` (`"t3code-v2"`).
   - `internal/t3client` keeps `harness.KindT3Code` (`"t3code"`) and protocol 1. It stays frozen
     except for the three changes ticket 03 lists.
2. **A computer moves forward once and never back.**
   - When the protocol-1 client finds protocol 2, the computer's stored kind becomes `t3code-v2` and
     the call is retried on the new client.
   - A `t3code-v2` computer whose T3 answers as protocol 1 is refused with a plain message,
     re-pairing included.
   - No schema migration: `pairing_computers.kind` is free text.
3. **Keep track of handed-off work.**
   - A turn whose agent hands work to another agent waits for that work and replies with the
     finished result.
   - Each handed-off agent is a pill on the reply, and clicking it shows that agent's conversation.
   - This covers T3's `delegate_task` children and providers' own subagents alike. T3 V2 projects
     both as child threads and tracks both in `subagent` rows.
4. **Protocol 1 removal belongs to the owner,** once T3 cuts a stable release on protocol 2. No
   ticket here.
5. **No separate OpenCode 2 harness.** T3 V2 runs OpenCode 2 (2.0.18 and later) as an ordinary
   provider, and Nexul's setup entry loads on 2.0.22. People run T3 Code.

Technical decisions:

6. **Shared transport.** The protocol-neutral parts of `internal/t3client` move to `internal/t3rpc`,
   behind an exported API that ticket 01 spells out:
   - dial with extra query values and a typed 426 error;
   - unary calls and streams;
   - the pairing exchange and the descriptor probe;
   - getConfig providers, the shell project list, and the default-model pick.

   The fake T3 server moves to an importable `internal/t3rpc/t3rpctest`. Removing protocol 1 then
   means deleting `internal/t3client` and its registry entry.
7. **Protocol refusals are one harness sentinel, `harness.ErrProtocol`.**
   - A protocol mismatch Nexul cannot follow wraps it, and the message names the computer or host:
     - protocol 1 on a `t3code-v2` computer: "T3 Code on <computer> went back to its old
       orchestrator; Nexul only moves forward. Update T3 Code there."
     - a 426 or descriptor naming a protocol above 2: "T3 Code on <computer> needs a newer Nexul."
   - Everything downstream treats `ErrProtocol` as the message to show:
     - `requireSetup` passes it through instead of "is T3 Code running there?";
     - re-pairing files it under the server URL, without "run t3 pair";
     - presence keeps its capped retry, so updating T3 recovers on its own, but logs the refusal
       once at Warn and then at Debug.
8. **One place does the switch.**
   - `harness.Forward(from, to Client, moved func(ctx, Session, Kind) error) Client` is registered
     under `KindT3Code`.
   - A Session method that returns `*harness.MovedError` gets `moved` called, then the same call
     retried on `to`.
   - `Pair` and `Version` have no session, so they retry on `to` without calling `moved`.
     `PairResult.Kind` says where a pairing landed.
   - Both clients' `Pair` probe the descriptor BEFORE spending the one-time `t3 pair` token.
   - `moved` is the pairing use-case. It runs one SQL statement:
     `UPDATE … SET kind='t3code-v2' … WHERE id=? AND kind='t3code' RETURNING user_id`.
     - The outbox write for `computer.harness_switched` is in the same transaction.
     - The event and the live frame fire only when a row changed.
   - Re-pairing refuses to lower a stored kind and raises it when the result is higher.
9. **A protocol-2 turn is the T3 run whose `userMessageId` is the message id Nexul minted.**
   - **The stream.**
     - A snapshot item always replaces state and the cursor. Event items below the cursor are
       dropped, and unknown event types are skipped but advance the cursor.
     - Once the first snapshot has arrived, any stream Exit or end resubscribes with
       `afterSequence`, up to three times with backoff. `LiveStreamBufferError` arrives as a `Die`.
   - **When it is done.** At that run's first `waiting`, or `completed` if `waiting` was missed.
     Terminal is emitted exactly once, because a stopped run reports its end twice and delivery
     bookkeeping re-sends `run.updated`.
   - **Interrupted:** `interrupted`, `cancelled` and `rolled_back`.
   - **Error:** `failed`. The message comes from the root error item, a usage limit names its reset
     time, and error items with status `running` are retries.
   - **Queued or held:** a note re-emitted every five minutes under one call id, so the turn waits as
     long as T3 holds it.
10. **Dispatch** is always `message.dispatch` with `dispatchMode: {type: "queue_after_active"}`.
    - There is never a `deliveryIntent`, because a steered message gets no run of its own.
    - `thread.create` sends every required key, including `title`, `interactionMode: "default"`,
      `branch: null` and `worktreePath: null`, plus `createdBy: "user"`, `creationSource: "web"`
      and `runtimeMode: "full-access"`.
    - An empty target model resolves to the provider's default first.
    - `thread.runtime-mode.set` corrects a reused thread only when no run there is queued or live,
      because it detaches provider sessions. Otherwise the turn notes "this T3 thread is not in full
      access".
11. **Prompts.**
    - Send `Full` plus attachments when Nexul just created the thread, or when the thread is an
      import (`historyOrigin == "v1_import"`) with no `completed` run yet. T3 gives that thread only
      an excerpt of its old history.
    - A reused thread that is gone (initial subscribe fails, the snapshot has `deletedAt`, or the
      dispatch fails) is recreated once with `Full`. Otherwise send `Incremental` (ADR 0106).
    - Images go through `assets.persistChatAttachments` first. Only gif, jpeg, png and webp; anything
      else is skipped with a note.
12. **Runs Nexul did not start are not followed,** except runs caused by its own run's handed-off
    work (decision 15).
    - Excluded: PR-watch wakes, scheduled tasks, other threads' sends, auto-resume, and providers'
      background commands and monitors.
    - Nexul never sends `queue.resume`.
13. **Stop**, in this order:
    1. dispose each followed `app_owned` delegated task's undelivered completion;
    2. interrupt each followed child thread's live run;
    3. cancel a followed queued wake run, or interrupt a live one;
    4. stop Nexul's own run:
       - queued → `queued-run.cancel`;
       - live, or waiting as the thread's latest run with background work → `run.interrupt` without
         `holdQueue`, treating "not interruptible" as done;
       - nothing at all → `ErrConflict`.

    Interrupting the parent alone does not stop T3's delegated children, which is why steps 1 and 2
    come first. A turn that gives up while its run is still queued cancels that run.
14. **An answer when Nexul's watch is gone** goes back as `runtime-request.respond`, not as a new
    message. A live question keeps its run `running`, and a new message would queue behind it
    forever.
    - `agent.TurnRequest` and `harness.TurnPrompts` gain `Answer *PendingAnswer`.
    - The protocol-2 client reads the request's state from the snapshot's `runtimeRequests` before
      choosing respond, done, or plain text.
15. **Waiting for handed-off work.**
    - The watch follows causal links only:
      - a user message whose `delegatedCompletion.parentRunId` is in the followed set;
      - a run named by a followed run's `delegatedCompletion.delivery.messageId`;
      - a user message whose `notification` names a followed subagent's child thread;
      - `restartContinuationOfRunId` in the set.
    - Runs whose message is not known yet wait in a pending map. An absent `delegatedCompletion` on
      `run.updated` keeps the previous value.
    - The turn stays open while:
      - a followed run is live, or
      - a `subagent` of a followed run (either origin) is pending, running or waiting, or
      - an `app_owned` one's completion is `pending` or `claimed`.
    - The step "Waiting for work handed off in T3 Code" is re-emitted every five minutes under one
      call id while anything is pending, whatever the parent run's status. It keeps chat's and
      plays' silence windows alive.
    - If a linked result lands in a run outside the set (T3 steered it into a later turn), the wait
      ends done with the note "The handed-off result went to a later reply in T3 Code".
    - Cap: 60 minutes after Nexul's own run reached `waiting`. The watch then ends done, and the
      pipeline appends this exact line to the stored reply: "Part of this work is still running in
      T3 Code." A play still ends `done`.
    - The pipeline's in-flight turn map is keyed per turn, so a second mention in the same
      conversation cannot orphan the first one's Stop or Answer.
    - Chat's fixed ten-minute ceiling becomes a 15-minute silence window for chat callers only.
      Plays keep their own timer, and a pending question pauses the window.
16. **Hand-off pills.**
    - **Seam.** `harness.Update.Handoff` carries one cumulative snapshot of one child, re-emitted on
      change and keyed by subagent id. Chat owns the stored and wire shape (snake_case):

      ```json
      {"id": "", "driver": "", "model": "", "title": "", "prompt": "", "state": "",
       "reply": "", "steps": [{"kind": "", "call_id": "", "tool": "", "summary": "", "detail": "", "at": ""}]}
      ```

      - `state` is one of `running`, `done`, `failed`, `interrupted` or `left_running` (still
        running at the cap).
      - `title` falls back to the prompt's first line, and `prompt` is capped at 2 KiB.
    - **What the client reads.** The protocol-2 client takes children from `subagent.updated` and
      `projection.subagents` whose `runId` is followed. It subscribes to each `childThreadId` once
      that is non-null, and maps every child turn item with the shared mapper.
      - There is no run filter. Provider-native children have no runs and their items carry
        `runId: null`.
      - State comes from `Subagent.status`. Model comes from `Subagent.model`, else the child
        thread's selection.
      - A child with no child thread gets a pill built from its prompt and result, without steps.
    - **Caps.** 200 newest steps per child, 2 KiB of detail per child step, 20 children per reply,
      and 256 KiB of stored hand-offs per reply (oldest steps go first).
    - **One level only.** A child's own hand-offs appear as steps inside it. A child's question shows
      as a step plus the parent note "a handed-off agent is waiting for an answer in T3 Code".
    - **Live and stored.**
      - The stream frame carries only the child that changed, and the web merges by id.
      - The final set is redacted like the reply body and stored with the Agent's reply.
      - Plays reply through the same pipeline, so pills show in chat, ticket and doc threads, and a
        play's thread alike.
17. **Storage: one new nullable column, `messages.handoffs`** (JSON, default `NULL`).
    - It goes in the next numbered forward-only migration, with an upgrade test from the previous
      schema.
    - It is the effort's only schema change, and it **needs the owner's OK before ticket 13 starts**.
    - Deleting the message sets it back to `NULL`.
    - The alternatives were worse. A note file is limited to ticket threads by ADR 0108. A live fetch
      from T3 on click fails when the computer is offline or the T3 thread is gone.
18. **Tool step names** are normalized to what `web/src/models/Trail.tsx` already parses:
    - `Shell` (command_execution), `Edit` (file_change), `Grep` (file_search), `WebSearch`,
      `Agent` (subagent);
    - Claude `mcp__` names stay as they are, and Codex `server.tool` becomes `server · tool`.
    - T3 strips command output, diffs and tool results from the wire for every client, so a
      protocol-2 step shows what ran but not its output.

Not adopted, on purpose, until a ticket asks for one:
- T3's scheduler and PR watch, which would be a second scheduler Nexul cannot see;
- steering mentions into a running turn, which breaks one reply per mention;
- `queue.resume`, fork and merge-back, `launchThread` worktrees and usage-limit auto-resume;
- tokens and cost per run;
- reasoning, plan and todo rows.

## Surfaces

- **UI:**
  - The pill and its dialog show on the Agent's reply (`web/src/components/chat/MessageRow.tsx`, next
    to the note pill) and in the live working bubble (`MessageList.tsx`). Ticket 14 names the
    components it reuses.
  - Notes from the protocol-2 client arrive as `ActivityNote` steps.
  - The settings row labels `t3code-v2` as "T3 Code".
  - Check at 768, 1024 and 1440 px.
- **Phone app:** a pill on Agent replies too (ticket 19). Until then, `native/` ignores the additive
  field and shows the reply as today.
- **HTTP gateway:** messages gain an additive `handoffs` field. No new route.
- **MCP:**
  - `message_list` returns hand-off summaries without steps, so results stay under the 10,000-token
    guidance. Its description gains one sentence about them.
  - `computer_*` results show the new kind value.
- **Events:**
  - `chat.message.created` carries the whole message; its catalog description gains `handoffs`.
  - `computer.harness_switched`:
    - a topic in `internal/pairing/events.go`;
    - a schema in `internal/integrations/catalog.go`, with the SDK regenerated;
    - instance scope in `server/cmd/automation_scope.go`;
    - an `ownFrame` live rule;
    - web invalidation of the computers and readiness queries.
- **Live push:**
  - The agent stream frame carries the changed hand-off.
  - The computers list refreshes from the `computer.harness_switched` live frame. `OnComputersChanged`
    only refreshes presence.
- **Search:** hand-off conversations are not indexed.
- **Permissions:** unchanged. Pills inherit the conversation's read check; Stop and Answer keep
  theirs.
- **Reverse states:**
  - Stop cancels a queued run and stops handed-off work.
  - A held queue can be resumed in T3 or stopped from Nexul.
  - Deleting a reply removes its hand-offs.
  - Going back to protocol 1 is refused by design and said plainly, and updating T3 again recovers
    on its own.
- **Docs:**
  - ADR 0113 (ticket 03): two kinds for T3 Code, the one-way switch, the protocol-2 turn. It amends
    `docs/adr/0029-the-agent-turn-runs-on-the-mentioning-users-own-environment.md` (Nexul refuses a T3
    that went back) and `docs/adr/0054-one-harness-client-interface-per-kind.md` (two T3 kinds, no
    OpenCode 2 harness).
  - ADR 0114 (ticket 11): a turn waits for the work it handed off, and the reply carries it.
  - Re-check both numbers against `origin/master` before writing.
  - Note ADR 0106 (imported threads get the Full prompt).
  - `CONTEXT.md`: Harness, Trail, hand-off.
  - `website/` paired-computers and computer-setup guides.
  - `ROADMAP.md` and `website/src/pages/roadmap.astro`.
  - The `t3client` package comment.

## Commands

- **Go,** from the repo root: `make lint`, `make vet`, `make test`, `make coverage` (80% floor),
  `make vuln`. Run `make sqlc` and `make sqlc-check` when queries change.
- **Web,** in `web/`: `bun run lint`, `bun run typecheck`, `bun run test`, `bun run build`.
- **Native,** in `native/`: `bun run lint`, `bun run typecheck`, `bun run test`.
- **SDK,** when the event catalog changes: `bun run generate:events`, `bun run typecheck` and
  `bun run test` in `sdk/`.

## Structure

```
internal/harness/            Kind, Session.ComputerID, PairResult.Kind, MovedError, ErrProtocol, Forward,
                             Update.Handoff, Handoff, TurnPrompts.Answer, PendingAnswer
internal/t3rpc/              protocol-neutral T3 transport (ticket 01 API), descriptor, providers, shell projects
internal/t3rpc/t3rpctest/    fake T3 server shared by both clients' tests
internal/t3client/           protocol 1; MovedError, Pair probes first, reconnect stops on MovedError
internal/t3clientv2/         protocol 2: client.go, turn.go (dispatch + watch reducer), items.go (turn items
                             to Activity), handoff.go, stop.go, testdata/*.ndjson
internal/pairing/            the one-way kind switch, re-pair rules, ErrProtocol in requireSetup, event
internal/presence/           quiet retry on ErrProtocol
internal/agent/              per-turn active map, chat silence window, Answer passthrough, hand-off collection
internal/plays/              answer fallback passes the pending answer
internal/chat/               Handoff type, messages.handoffs, message_list summaries
internal/platform/storage/   migration 0060 (handoffs) + sqlc queries for the switch and handoffs
server/cmd/                  registry {t3code: Forward(t3client, t3clientv2, switch), t3code-v2: t3clientv2},
                             automation scope, live rule
web/src/                     hand-off pill and dialog, HARNESS_LABELS, live invalidation
native/src/                  hand-off pill on Agent replies
```

## Code style

The repo's hard rules apply (AGENTS.md): early return and no `else`, `slog`, no barrels, one-line
comments only for a why. Protocol-2 parsing is a pure reducer, table-tested over recorded frames:

```go
// apply folds one stream item into the watch and returns what the harness should emit for it.
func (w *watch) apply(item streamItem) ([]harness.Update, *harness.TurnResult) {
	if item.Kind == "snapshot" {
		w.cursor = item.SnapshotSequence
		return w.reset(item.Projection)
	}
	if item.Kind != "event" || item.Sequence <= w.cursor {
		return nil, nil
	}
	w.cursor = item.Sequence
	return w.event(item.Event)
}
```

## Testing

- **Fixtures.** Protocol-2 behavior is tested over frames in `internal/t3clientv2/testdata`.
  - Files are named after the T3 build that produced them, with ids, paths, tokens and emails
    scrubbed.
  - Frames read from source come first, then captures from nightly `0.0.46-nightly.20261003.2632`
    (ticket 04), then provider captures (ticket 16).
- **Fake server.** `internal/t3rpc/t3rpctest` serves both protocols, including the
  `orchestrationProtocol` check, a 426 mode and single-use pairing tokens.
- **Order.** Error paths first: refusals, dispatch failures, missing and deleted threads, double
  terminals, queued and held runs, going back, and a protocol change on reconnect.
- **Timers.** Tests for the hand-off cap, the silence windows and the five-minute steps run under
  `testing/synctest`, fed through channel-backed fakes, never the WebSocket fake: network I/O blocks
  fake time.
- **Concurrency.** Two concurrent switches yield one event (real SQLite).
- **Protocol 1.** Existing scenarios and assertions pass. Only receivers and imports change, apart
  from ticket 03's three listed changes.
- **End to end.** On an Incus image with T3 `0.0.45` and nightly `2632` side by side, plus a flip of
  one computer from stable to nightly and back (ticket 16).

## Boundaries

- **Always:**
  - Read the practices files the ticket names before editing.
  - Refresh the local T3 source before reading it; T3 makes no compatibility promise.
  - Keep protocol-1 behavior unchanged apart from ticket 03's three changes.
  - Run `rg --hidden` for design-source and credit names before a PR.
- **Ask first:**
  - The `messages.handoffs` column (ticket 13).
  - The owner's provider login and host firewall rules (ticket 16).
- **Never:**
  - Steer a Nexul message into a running T3 turn, or send `queue.resume`.
  - Store the protocol anywhere but the kind.
  - Edit an old migration.
  - Use the production `nexul` MCP tools, the owner's live T3 home, or the host's port 3773 for
    tests. Ports inside an Incus clone are fine.

## Success criteria

- A computer on T3 stable pairs, shows connected, runs @Agent turns and plays, answers questions,
  and stops exactly as before.
- A computer that updates to the nightly switches to `t3code-v2` on its next call, with no re-pair
  and no "re-pair" warning. After that:
  - its turns reply;
  - images reach the agent;
  - questions can be answered live and after a Nexul restart;
  - Stop ends a running or queued run;
  - an imported old thread's first turn gets the Full prompt.
- Pairing a fresh computer on the nightly lands on `t3code-v2` with one `t3 pair` token.
- The same computer moved back to stable gets the "went back" message in the turn and on the
  settings row. Updating T3 again recovers it.
- An agent that delegates work replies with the finished result. Its reply carries a pill per
  handed-off agent, Claude's own subagents included, and the pill opens that agent's conversation,
  live while it runs and kept afterwards.
- Every gate in AGENTS.md's table is green, and the end-to-end checklist passes on both protocols.

## Answered from source (2026-10-03)

- **Does interrupting a parent stop its delegated children?** No. It disposes the parent's
  delegated-completion cohort and cancels a queued delivery run, but child threads keep running
  (`apps/server/src/orchestration-v2/Orchestrator.ts:1984-2065`). T3's own `cancel_task` interrupts
  the child run itself (`apps/server/src/mcp/OrchestratorMcpService.ts:1520-1545`).
- **May a paired client dispose a delivery?** Yes. `delegated_task.completion-delivery.dispose` is in
  the public command union and needs only `orchestration:operate`.
- **Can a paired client read a child thread?** Yes. Subscribing to any child thread needs only
  `orchestration:read`; there is no lineage check.
- **Can setup gate on Pi's version?** Yes. `server.getConfig` providers carry `version`, which T3
  fills from `pi --version`.
