# Runs T3 Code starts on a thread without Nexul asking

Read from pingdotgg/t3code at 31a9da17. Covers delegate_task, wake runs, PR watch, scheduled tasks, usage-limit resume and restart continuation, and how a client can tell a run its own turn caused.

Paths are relative to apps/server/src in pingdotgg/t3code unless they start with packages/ or docs/. Nexul paths start with internal/. Re-checked at 1945ce82c0.

## 1. delegate_task

**1a. Modes and timeout**
- `mode` is "async" | "wait" and defaults to async. The schema text says: "Defaults to async. Use wait only when this turn needs the child's result before you can continue." (packages/contracts/src/orchestratorMcp.ts:176-180)
- `timeoutMs` defaults to 10 min and is clamped to 1 ms .. 60 min (mcp/OrchestratorMcpService.ts:74-75, 1432-1435).
- **Wait mode keeps Nexul's run open.** The parent agent blocks inside the MCP tool call while waitForTask polls (OrchestratorMcpService.ts:1154-1161, 1436-1438). The run stays `running`, so the result comes back as the tool result inside Nexul's own watched run.
- The T3 doc says wait mode waits "including nested work and completion follow-ups" (docs/orchestration-v2/orchestrator-mcp-server.md:243-246).
- If the wait times out, the child keeps running. `waitTimedOut` comes back true, and the task is upgraded to `completionWake:"always"` through `delegated_task.wake-policy` (OrchestratorMcpService.ts:1440-1455).
- T3 gives Claude a 65-minute MCP timeout so it can outlast the 1-hour cap (orchestration-v2/Adapters/ClaudeAdapterV2.ts:922-927, 959). Codex's `t3-code` entry sets no timeout (CodexAdapterV2.ts:1218-1225).

**1b. Wake policy**
The contract default is `settled_only`, but that only applies to wait mode:
- The MCP tool sets `completionWake: input.mode === "wait" ? "settled_only" : "always"` (OrchestratorMcpService.ts:1404-1407).
- So async, the default mode, is "always". The contract default `settled_only` only applies when the field is omitted (packages/contracts/src/orchestrationV2.ts:646-651, 2826-2828).

How each policy behaves:
- `settled_only` holds back while the spawning run is live, not while any run on the parent thread is (Orchestrator.ts:8699-8705 at eac52f0087). `hasLiveRun` covers preparing, starting and running, but not `waiting` (Orchestrator.ts:440-449).
- `always` with a `running` run on the thread: if the session `supportsActiveSteering`, the wake is steered into that run (Orchestrator.ts:4537-4571). Claude, Codex, OpenCode, OpenCode2 and Pi support this; Cursor and ACP do not.
- Otherwise the wake becomes a new run. ProviderContinuationService dispatches `message.dispatch` with:
  - `dispatchMode:{type:"queue_after_active"}`, `createdBy:"agent"`, `creationSource:"server"`
  - `delegatedCompletion:{parentRunId, generation, taskIds}`
  - text "Delegated task <id> reached a terminal state. Use task_status with taskId <id> to read the result."
  - Source: orchestration-v2/ProviderContinuationService.ts:115-146.
- Wake runs jump ahead of queued user runs (orchestration-v2/QueuedRunOrder.ts:12-31).

**1c. What marks a wake run**
- Run has no `trigger` or `source` field. It does have `userMessageId`, `restartContinuationOfRunId`, `workStartedAt` (set on wake runs) and `delegatedCompletion` (orchestrationV2.ts:511-552).
- The run's user message, sent as `message.updated`, carries `createdBy`, `creationSource`, `notification?`, `scheduledTaskId?`, `senderThreadId?` and `delegatedCompletion{parentRunId}` (orchestrationV2.ts:1029-1053; Orchestrator.ts:4827-4849). In the queued path `run.created` is emitted before `message.updated` (Orchestrator.ts:4861 vs 4909).
- **The exact causal link** is `message.delegatedCompletion.parentRunId == Nexul's run.id`.
- Nexul learns the wake's messageId even earlier: its own run gets a post-terminal `run.updated` whose `delegatedCompletion.delivery.messageId` names the wake (Orchestrator.ts:8595-8620, 9143-9157). This is another post-terminal run.updated the watch must not treat as a second terminal.
- The task itself shows up as `subagent.updated` with `origin:"app_owned"`, `runId` = parent run, `completionWake`, `completionDelivery.state` (Orchestrator.ts:6395-6462; orchestrationV2.ts:636-660).

## 2. The injected instructions

T3 appends one block to every session: Claude system prompt (ClaudeAdapterV2.ts:880-886), Codex developer instructions (provider/CodexDeveloperInstructions.ts:218), OpenCode and OpenCode2 system prompts, Cursor's first-run prompt (CursorAdapterV2.ts:2107), the ACP wrapper and the Pi extension. Key sentences (provider/T3OrchestrationInstructions.ts:9-12):
- "Prefer native subagent tools for same-provider work only when they support the chosen model. Use `delegate_task` with that provider instance and model when native tools cannot, including for same-provider work. Also use `delegate_task` for cross-provider or explicitly T3-owned child tasks."
- "`t3_thread_launch` and `create_threads` ... Use them only when the user explicitly asks for separate/new/top-level threads"
- "`schedule_task` creates persistent recurring work ... By default runs return to the current thread"

The block itself does not tell agents to delegate or schedule. The push toward async comes from the tool descriptions:
- delegate_task: "Prefer mode='async' for long work ... An async child's completion wakes this thread through a notification, steered into active turns where supported or queued otherwise, so end the turn instead of polling or spawning watchers" (mcp/toolkits/orchestrator/tools.ts:60).
- watch_pull_request: "...instead of polling ... then end your turn" (mcp/toolkits/pullRequests/tools.ts:261).

So any agent that delegates is told to end Nexul's run early with a placeholder.

## 3. Other runs that can start without Nexul

| Source | On by default? | How the run is marked |
|---|---|---|
| Native provider background work: Claude background Bash/Task, Monitor (no MCP involved) | Yes, whenever the agent uses it | ClaudeAdapterV2.ts:5224-5230 offers a continuation. The run is queued with `creationSource:"provider"`, `createdBy:"agent"`, `notification{source,outcome,summary}`, a random messageId, and no run link except `workStartedAt` (ProviderContinuationService.ts:148-174; Orchestrator.ts:310-339). The shell or provider-thread `pendingBackgroundTasks` lists the work; T3 holds its own "done" alert for kinds subagent, monitor and background_task (packages/shared/src/orchestrationV2PendingBackgroundWork.ts:62-85, 211-268). |
| PR watch | Opt-in: the agent calls `watch_pull_request` (pullRequests/handlers.ts:232-242) or the user uses the T3 panel (apps/web/src/components/pullRequest/ThreadPullRequestsPanel.tsx:280) | Polled once a minute; settled threads are skipped (PullRequestWatchReactor.ts:60-65, 134). Wake: messageId `message:pr-watch:<uuid>`, `createdBy:"agent"`, `creationSource:"server"`, notification source `monitor`, queue_after_active (Orchestrator.ts:2261-2277; pullRequestWatch.ts:200). No run link. Ends when the PR merges or closes, or after 15 minutes of failed reads. |
| schedule_task | Opt-in by the agent. The tool defaults `bindToCurrentThread=true` and stores `createdBy:"agent"`, `creationSource:"mcp"` (OrchestratorMcpService.ts:1192-1208). | Each fire: messageId `scheduled-task-message:<taskId>:<ms>:<trigger>` with `scheduledTaskId` set (ScheduledTaskService.ts:507-551). Mode `auto` steers into any running turn, which can be a live Nexul run, otherwise start_immediately, which queues behind a blocking run (ThreadManagementService.ts:361-373, 547-552; Orchestrator.ts:4660-4666). Recurring and open-ended. |
| Usage-limit auto-resume | Off: `autoResumeLimitedThreads` and `snoozeLimitedThreads` are false (packages/contracts/src/settings.ts:1275-1276). Can be armed per thread through `limitRecovery`. | messageId `limit-resume:<thread>:<runId>:<resetMs>:<requestId>`, text "Continue where you left off.", `createdBy:"user"`, `creationSource:"server"`, start_immediately (UsageLimitRecoveryWorker.ts:34-71). `usageLimitContinuationOfRunId` is not stored on Run, so the only link is the messageId string. |
| Restart continuation | Off: `continueThreadsAfterServerUpdate` is false, with a per-project override (settings.ts:1205-1208; RestartContinuation.ts:104-108) | messageId `message:restart-continuation:<sourceRunId>`, `createdBy:"agent"`, `creationSource:"server"`, and `run.restartContinuationOfRunId` set (RestartContinuation.ts:98, 120-134). |

Two more: another T3 agent in the same project can post with `t3_thread_send` (the message has `senderThreadId`), and async-answer runs behave as described in protocol-2-wire.md.

All of these arrive on `orchestration.subscribeThread` as ordinary `run.created`/`run.updated`, `message.updated`, `turn-item.updated` and `provider-thread.updated` events.

## 4. Can a client opt out?

There is no switch for the capability or for wakes.
- `ProviderSessionManager` always grants `["orchestration","worktree","pull-requests"]`. Only preview and device are gated by agent-access settings (orchestration-v2/ProviderSessionManager.ts:436-442).
- The comment on `enableAgentBrowserAccess` ("withholds the MCP credential ... never attached", settings.ts:1209-1220) is stale. Turning it off only drops `preview`; the credential and the `t3-code` server are still attached.
- `configureMcp:false` is a layer option that only the test replay harness uses (ProviderSessionManager.ts:227-228, 422; testkit/ProviderReplayHarness.ts:337).
- Runtime mode does not help. Claude pre-approves `mcp__t3-code__*` in every mode except a read-only sandbox (ClaudeAdapterV2.ts:929-949). That sandbox is only set through `RuntimePolicy.layerWithOverride` (RuntimePolicy.ts:141-160), not through any thread.create field. Nexul sends full-access anyway (internal/t3client/harness.go:149, 188).

Cleanup levers exist per instance only, not as opt-outs:
- WS `delegated_task.completion-delivery.dispose{parentThreadId,taskId}` cancels one task's wake (packages/contracts/src/orchestrationV2.ts:2845-2850; Orchestrator.ts:1818-1858).
- `thread.pull-request.watch{watching:false}` stops a PR watch.
- `scheduledTasks.setEnabled` / `scheduledTasks.delete` disable or remove a schedule.

## 5. Nexul limits that interact with this
- Chat turns have a hard ceiling: `maxTurnDuration` is 10 min (internal/agent/pipeline.go:245, 260, 837). That equals the wait-mode default, so a waiting chat turn dies at the same moment the wait returns.
- Plays have a 15-minute silence timer that resets on snapshots and activity (internal/plays/run.go:30-31, 677-682). During a wait the parent stream is probably quiet; this is inferred, not checked (see the open questions).

## Recommendation

Use follow-through, limited to runs Nexul's own run caused.

**Scope.** Inside internal/t3client's V2 watch (thread.go, turnWatch), replace "my run" with a causal run set S, starting with Nexul's run.
- Add a run to S when its user message has `delegatedCompletion.parentRunId` in S, or its userMessageId equals an S run's `delegatedCompletion.delivery.messageId`.
- Also add it when `restartContinuationOfRunId` is in S, or it is a `creationSource:"provider"` notification wake that arrives while S's own work is still pending.
- S's work is still pending if any of these hold:
  - an S run is non-terminal;
  - an app_owned task with runId in S is non-terminal, or its completionDelivery is pending or claimed;
  - `pendingBackgroundTasks` from provider-thread.updated has a kind that holds completion (subagent, monitor or background_task; T3 uses the same rule for its own "done" alert).
- Send Terminal once, when nothing is pending. Feed the wake runs' Snapshots and Activities into the same Update stream, so the final reply is the agent's real answer and not "I delegated X".

**What stays ignored:**
- PR-watch wakes (`message:pr-watch:`)
- scheduled runs (`scheduledTaskId`)
- `senderThreadId` posts from other agents
- the user's own T3-UI turns

They are open-ended or not Nexul's, and none of them links back to S.

**Cost:**
1. t3client watch logic, about 150-250 lines plus table tests. No harness.Client interface change and no DB migration.
2. Chat ceiling: pipeline.go:245 bounds the whole follow-through. Give the follow-through its own cap (60 min, matching T3's MAX_WAIT_TIMEOUT_MS) and emit an Activity note such as "Waiting for delegated work in T3 Code…" when Nexul's run ends but S is still open.
3. Plays: pause or reset the 15-minute silence timer while S is open, the same way OnQuestion already parks it.
4. Interrupt: target the latest non-terminal run in S, use `queued-run.cancel` for a queued wake, and optionally dispose open deliveries.
5. Edge case: an "always" wake can be steered into a later live Nexul turn on the same thread. S must then close with a note that the result arrives in the next reply.
6. Record it as an ADR. It changes what one T3 turn means, which is a real trade-off.

**Why not the other three options:**
- **Ignore:** the T3 tool text tells agents to end the turn after async delegation, so placeholder replies would be the normal case, with results visible only in T3.
- **Prompt-steer to wait mode:** it fights T3's own tool description ("Prefer mode='async' for long work"). Its 10-minute default collides with Nexul's 10-minute chat ceiling. It cannot cover native background subagents or monitors. On Codex, T3 sets no MCP timeout, so long waits are untested. A one-line hint can still go alongside follow-through, but not instead of it.
- **Session-level mirror:** it needs a new harness capability and a long-lived subscription per thread. It also has to post replies with no requesting user and would echo the user's own T3 turns into Nexul. That is a lot more to build than the problem needs.

## Still unknown

- Codex default MCP tool timeout for the t3-code server: T3 sets none (CodexAdapterV2.ts:1218-1225). If Codex keeps its usual 60 s default, a wait-mode delegate_task would fail on the Codex side, and it is unclear whether the server-side wait is cancelled and the wake policy stays settled_only. codex-rs is not in the tree, so this needs a live check on nexul-box.
- Whether the parent thread stream carries any event while a wait-mode child runs (heartbeat for Nexul's play silence timer). Only finalizeAppOwnedSubagent was found propagating child state to the parent (Orchestrator.ts:8640-8700, 9643, 9704); intermediate progress was not traced.
- Not verified live: whether delegated_task.completion-delivery.dispose and delegated_task.wake-policy, which are in the public OrchestrationV2Command union, are accepted from a paired WS client without extra authorization.
- Whether the pendingBackgroundTasks roster on provider-thread.updated is complete enough for follow-through for Codex, OpenCode2 and Pi native background work, or only for Claude SDK tasks (ProviderContinuationService and the Codex adapter offer continuations at CodexAdapterV2.ts:4240; not traced per provider).
- Exact steering behaviour when a delegated 'always' wake targets a Cursor or ACP session (no active steering): presumably it stays queue_after_active, but this was not traced end to end.
- Product decision still for Onik: the follow-through cap (60 min suggested) and whether a timed-out follow-through posts a 'still running in T3 Code' note or a later second reply.