# Protocol 2 wire shapes Nexul needs

Read from pingdotgg/t3code at 31a9da17 (main the day after the orchestration V2 merge). Paths are relative to that repo unless they start with `internal/`. T3 promises no compatibility for this protocol: re-check against the current source before relying on a line.

PROTOCOL-2 WIRE SPEC FOR internal/t3client
Checked line by line against V2 @31a9da17. Path roots: C = packages/contracts/src, S = apps/server/src, O = S/orchestration-v2, effect-smol = .repos/effect-smol, N = internal/t3client (Nexul). Re-checked at 1945ce82c0: no protocol change since 31a9da17.

=== 0. Envelope and errors ===
The Effect RPC framing is unchanged (Request/Chunk/Exit/Ack/Interrupt/Ping/Pong/Defect), so N/client.go works as it is. Its call() still has to send `{}` for no-input calls.

A failed unary call looks like this:
`{"_tag":"Exit","requestId":"7","exit":{"_tag":"Failure","cause":[{"_tag":"Fail","error":{"_tag":"OrchestrationV2DispatchCommandError","commandId":"c1","commandType":"message.dispatch","message":"Runtime request r1 is resolved.","detail":"..."}}]}}`
- Error class: C/orchestrationV2.ts:3174-3183. Cause encoding: .repos/effect-smol/packages/effect/src/Schema.ts:10535-10540.
- `message` is the user-facing text, or "Failed to dispatch orchestration V2 command" (S/ws.ts:1767-1777).
- If the payload fails schema decode, the cause is `[{"_tag":"Die","defect":...}]`. Only that request fails; the socket stays open.
- A missing scope fails with `{"_tag":"EnvironmentAuthorizationError","message","requiredScope"}` (C/auth.ts:297-303).
- Scopes needed (S/auth/RpcAuthorization.ts:24-37, 147):
  - orchestration:operate: dispatchCommand, projects.mutate, assets.persistChatAttachments.
  - orchestration:read: subscribeThread, subscribeShell, getThreadProjection, server.getConfig.
  - Nexul's existing scopes already cover all of these.

=== 1. Dial ===
Detecting the protocol:
- GET /.well-known/t3/environment returns `orchestrationProtocolVersion`, an optional int. Absent means 1 (C/environment.ts:210-218). server.getConfig().environment carries the same descriptor.
- Probe on every connect, because V1 and V2 can share port 3773.

URL: `<server>/ws?orchestrationProtocol=2&wsTicket=<ticket>`
- The param value must be exactly "2" (C/environment.ts:13-16; S/ws.ts:507-512).
- The ticket mint (POST /api/auth/websocket-ticket) is unchanged.

Optional identity params. They are lenient: an invalid value is silently dropped (S/ws.ts:519-584; mirrors client-runtime/src/authorization/remote.ts:41-81).
- `clientSurface` ∈ web | desktop | mobile | cli (C/baseSchemas.ts:164)
- `clientAppVersion`: trimmed, 1–64 chars
- `clientDeviceType` ∈ desktop | phone | tablet | unknown
- `clientOs` ∈ macOS | Windows | Linux | iOS | Android | ChromeOS | other | unknown
- `clientWebDeployment` ∈ hosted | server, and `clientBrowser` (≤64): only read when surface=web
- `clientOsMajorVersion` (int > 0) and `clientDeviceModel` (≤80): only read when surface=mobile
- `connectionMethod` ∈ direct | ssh | relay | unknown

Only surface and appVersion are stored on the auth session (S/ws.ts:3749-3750); the rest is analytics. None of the surface values describes a server integrator. Leave clientSurface out (or send "cli") and send `clientAppVersion=nexul/<ver>`.

426 response (S/ws.ts:3727-3741):
`HTTP 426 {"code":"orchestration_protocol_incompatible","message":"Update this client to one that supports orchestration protocol 2.","orchestrationProtocolVersion":2}`
- The check runs before auth, so the ticket is not consumed.
- coder/websocket v1.8.15 returns `resp` with StatusCode and the first 1024 body bytes (~/go/pkg/mod/github.com/coder/websocket@v1.8.15/dial.go:147-161).
- Today client.go:155-158 drops `resp` and marks the error Retryable. It must check `resp != nil && resp.StatusCode == 426` and return a non-retryable error.

```go
type protocolMismatch struct {
	Code                         string `json:"code"`
	Message                      string `json:"message"`
	OrchestrationProtocolVersion int    `json:"orchestrationProtocolVersion"`
}
```

=== 2. Commands (orchestration.dispatchCommand) ===
- The payload is the bare command object, discriminated by `type` (no `_tag`). Success is `{"sequence":<int>}` (C/orchestrationV2.ts:3014-3017).
- There is no createdAt on any command.

Ids:
- CommandId, ThreadId, MessageId, ProjectId, RunId and RuntimeRequestId are all "any trimmed non-empty string" (C/baseSchemas.ts:134-147, 202).
- The client mints commandId, threadId and messageId; ids.New() is fine.
- The server mints:
  - runId: `run:thread:<threadId>:ordinal:<n>` (O/IdAllocator.ts:194-195, 404). Treat it as opaque.
  - runtime request ids and turn-item ids.
- commandId is the idempotency key, via command receipts. Reuse the same commandId and messageId when retrying one logical send.

Provenance:
- `createdBy` ∈ user | agent | system and `creationSource` ∈ web | mobile | mcp | provider | server are both REQUIRED on thread.create and message.dispatch (C/orchestrationV2.ts:74-89).
- ws.ts overwrites createdBy to "user" and keeps the client's creationSource (S/ws.ts:1757-1762; O/ThreadManagementService.ts:44-58).
- Nothing in the server branches on creationSource for user-created threads or messages. The server/provider checks at O/Orchestrator.ts:4484-4506 only apply to notification and delegatedCompletion. Send "web", which is T3's own client default (client-runtime operations/commands.ts:395).

```go
type modelSelection struct {
	InstanceID string          `json:"instanceId"`        // slug ^[a-zA-Z][a-zA-Z0-9_-]*$, ≤64 (C/providerInstance.ts:40-56)
	Model      string          `json:"model"`             // trimmed non-empty (C/modelSelection.ts:17-21)
	Options    []optionSetting `json:"options,omitempty"` // C/model.ts:46-52
}
type optionSetting struct {
	ID    string `json:"id"`
	Value any    `json:"value"` // non-empty string or bool only
}
```

thread.create (C/orchestrationV2.ts:2439-2464; handler O/Orchestrator.ts:2097-2178):
```go
type threadCreate struct {
	Type            string         `json:"type"`           // "thread.create"
	CreatedBy       string         `json:"createdBy"`      // "user"
	CreationSource  string         `json:"creationSource"` // "web"
	CommandID       string         `json:"commandId"`
	ThreadID        string         `json:"threadId"`
	ProjectID       string         `json:"projectId"`
	Title           string         `json:"title"`           // trimmed non-empty
	ModelSelection  modelSelection `json:"modelSelection"`
	RuntimeMode     string         `json:"runtimeMode"`     // approval-required|auto-accept-edits|auto|full-access (C/providerPolicy.ts:25-31)
	InteractionMode string         `json:"interactionMode"` // default|plan (C/providerPolicy.ts:34)
	Branch          *string        `json:"branch"`          // key required, null allowed
	WorktreePath    *string        `json:"worktreePath"`    // key required, null allowed
}
```
`{"type":"thread.create","createdBy":"user","creationSource":"web","commandId":"cmd_01","threadId":"thr_01","projectId":"prj_01","title":"Fix login","modelSelection":{"instanceId":"claudeAgent","model":"claude-opus-4-5","options":[{"id":"effort","value":"high"}]},"runtimeMode":"full-access","interactionMode":"default","branch":null,"worktreePath":null}`

message.dispatch (C/orchestrationV2.ts:2664-2712):
```go
type messageDispatch struct {
	Type           string           `json:"type"` // "message.dispatch"
	CreatedBy      string           `json:"createdBy"`
	CreationSource string           `json:"creationSource"`
	CommandID      string           `json:"commandId"`
	ThreadID       string           `json:"threadId"`
	MessageID      string           `json:"messageId"`
	Text           string           `json:"text"`        // Schema.String
	Attachments    []chatAttachment `json:"attachments"` // required; send [] not null
	ModelSelection *modelSelection  `json:"modelSelection,omitempty"`
	DispatchMode   dispatchMode     `json:"dispatchMode"` // REQUIRED
}
type dispatchMode struct {
	Type        string `json:"type"`                  // defer_start|steer_active|restart_active|queue_after_active|start_immediately
	TargetRunID string `json:"targetRunId,omitempty"` // only steer_active/restart_active
}
type chatAttachment struct {
	Type      string `json:"type"` // "image"
	ID        string `json:"id"`
	Name      string `json:"name"`
	MIMEType  string `json:"mimeType"`
	SizeBytes int    `json:"sizeBytes"`
}
```
`{"type":"message.dispatch","createdBy":"user","creationSource":"web","commandId":"cmd_02","threadId":"thr_01","messageId":"msg_01","text":"...","attachments":[],"dispatchMode":{"type":"queue_after_active"}}`

How the server treats a message.dispatch:
- Without deliveryIntent, dispatchMode is used verbatim (O/CommandPolicy.ts:120-124).
- While any run is preparing, starting, running or waiting (isBlockingRun, O/Orchestrator.ts:431-438), start_immediately, queue_after_active and defer_start all create a QUEUED run (O/Orchestrator.ts:4659-4666).
- When the thread is idle, start_immediately and queue_after_active create a "starting" run. defer_start creates "preparing", which waits for prepared-run.release (O/Orchestrator.ts:5124). Never use defer_start.
- steer_active and restart_active merge the message into another run. No run carrying this messageId is created. Never use them.
- In every queue/start case, Run.userMessageId == messageId (O/Orchestrator.ts:4746, 5120).

There is no runtimeMode on message.dispatch; runtime mode is a thread property. Setting modelSelection overrides the thread's selection, and a different instanceId triggers a provider switch through a handoff (O/Orchestrator.ts:4458, 4985-4992). Omit it unless the selection really changes.

run.interrupt (C/orchestrationV2.ts:2728-2735):
`{"type":"run.interrupt","commandId":"cmd_03","threadId":"thr_01","runId":"run:thread:thr_01:ordinal:4","reason":"Stopped from Nexul"}`
- Works on preparing, starting and running runs.
- A waiting run with no running provider turn and no background work fails with "Run X is not interruptible." (O/Orchestrator.ts:8103-8109).
- `holdQueue:true` sets queueHeld on every queued run, which then needs queue.resume (O/Orchestrator.ts:7946-7961). T3's own Stop sends it (client-runtime operations/commands.ts:798-806). Nexul must omit it.

queued-run.cancel (C/orchestrationV2.ts:2755-2760):
`{"type":"queued-run.cancel","commandId":"cmd_04","threadId":"thr_01","runId":"..."}`
- Only works on a run with status "queued"; otherwise it fails with "Run X is not queued." (O/Orchestrator.ts:7286-7292).
- On success it emits run.updated with status "cancelled".

runtime-request.respond (C/orchestrationV2.ts:2772-2780):
```go
type runtimeRequestRespond struct {
	Type      string         `json:"type"` // "runtime-request.respond"
	CommandID string         `json:"commandId"`
	ThreadID  string         `json:"threadId"`
	RequestID string         `json:"requestId"`          // the item's requestId (RuntimeRequestId)
	Decision  string         `json:"decision,omitempty"` // accept|acceptForSession|acceptAlways|decline|cancel (C/providerPolicy.ts:50-56)
	Answers   map[string]any `json:"answers,omitempty"`  // Record<string,unknown> keyed by question.id (C/providerPolicy.ts:66)
}
```
- Approval: `{"type":"runtime-request.respond","commandId":"c","threadId":"t","requestId":"rq","decision":"decline"}`
- Question: `{"type":"runtime-request.respond","commandId":"c","threadId":"t","requestId":"rq","answers":{"Which DB?":"sqlite","Pick features":["auth","search"]}}`

Answer value shapes:
- T3's web client sends a string for free text or a single selection, and a string[] for multiSelect. The option key is `option.value ?? option.label` (apps/web/src/pendingUserInput.ts:41-66, 132-147).
- Live Claude joins arrays with ", " (O/Adapters/ClaudeAdapterV2.ts:2832-2845).
- Live Codex accepts a string or string[] (O/Adapters/CodexAdapterV2.ts:592-600).
- Message-mode requests:
  - Every question not marked required:false must get a non-empty STRING; an array is rejected with "Answer each question before sending." (O/Orchestrator.ts:6914-6930). Join multi-select values into one string.
  - The server then dispatches its own run with messageId `async-answer:<requestId>`, using queue_after_active or steer_active (O/Orchestrator.ts:6931-6982).

Server validation:
- The request must exist and be "pending", otherwise the error is "Runtime request X is resolved|expired|cancelled".
- not_resumable fails with its `reason` (O/Orchestrator.ts:6781-6806).

questionTextById is NOT a command field. The server writes it into the resolved user_input_request item's `questionAnswer {requestId, questionTextById, answers, attachmentsByQuestionId}` (O/Orchestrator.ts:6880-6897; C/providerPolicy.ts:76-81).

Question ids:
- Claude: the id is the question text itself (ClaudeAdapterV2.ts:2820-2827).
- Codex async: "0", "1", ... (CodexAdapterV2.ts:4390-4395).

thread.runtime-mode.set (C/orchestrationV2.ts:2639-2644):
`{"type":"thread.runtime-mode.set","commandId":"c","threadId":"t","runtimeMode":"full-access"}`

=== 3. assets.persistChatAttachments ===
Defined at C/rpc.ts:360, 1208-1212 and C/chatAttachment.ts:192-216, 236-249; handler at S/ws.ts:268-330 and 3193-3197.

Request:
`{"threadId":"thr_01","messageId":"msg_01","attachments":[{"type":"image","name":"shot.png","mimeType":"image/png","sizeBytes":48213,"dataUrl":"data:image/png;base64,iVBOR..."}]}`

Limits:
- name ≤255 characters.
- mimeType ≤100 characters and must match ^image/.
- sizeBytes ≤10 MiB.
- dataUrl ≤14,000,000 characters, and must be `data:<mime>;base64,<b64>`.
- The data-URL mime must equal the lowercased mimeType.
- The decoded byte count must equal sizeBytes.
- Images only. Files go through attachments.createUploadUrl, which Nexul does not need.

How the server handles it:
- Any `id` the client sends is ignored.
- The id is computed as createDeterministicAttachmentId(threadId, "<messageId>:<index>") = "<thread-segment>-<uuid derived from sha256>" (S/attachmentStore.ts:91-103). A retry with the same thread, message and order gives the same id.
- The server does not check that the thread exists.

Response:
`{"attachments":[{"type":"image","id":"thr_01-3f2ab9c1-...","name":"shot.png","mimeType":"image/png","sizeBytes":48213}]}`
On error: PersistChatAttachmentsError {message}.

Wiring it into the send:
1. Mint messageId first.
2. Persist with the same threadId and messageId.
3. Put the returned objects verbatim into message.dispatch.attachments.

Message.dispatch then also checks: ≤100 attachments, ≤80 MiB of images in total, no duplicate ids. Ids that are not "pending" pass through (O/AttachmentClaims.ts:62-78; C/chatAttachment.ts:218-234). Providers accept only gif, jpeg, png and webp (C/chatAttachment.ts:17-22), so filter before uploading. Inline dataUrl on message.dispatch is rejected because ChatImageAttachment requires `id` (C/chatAttachment.ts:149-156).

=== 4. orchestration.subscribeThread ===
Input (C/orchestrationV2.ts:3040-3054):
```go
type subscribeThreadInputV2 struct {
	ThreadID                string `json:"threadId"`
	AfterSequence           *int64 `json:"afterSequence,omitempty"` // omit on the first subscribe
	RequestCompletionMarker bool   `json:"requestCompletionMarker,omitempty"`
	AcceptBoundedSnapshot   bool   `json:"acceptBoundedSnapshot,omitempty"` // send true
}
```

Server flow (S/ws.ts:648-838):
- It hydrates the legacy V1 transcript first (S/ws.ts:661).
- Without afterSequence: [snapshot] → [synchronized, if a marker was requested] → live events after snapshotSequence.
- With afterSequence:
  - It replays when the gap is ≤128 events and ≤1 MiB, contains no create event, and afterSequence ≤ the high-water mark. Otherwise it sends a snapshot instead (O/ThreadStream.ts:10-20, 79-91).
  - A replay sends no snapshot.

With acceptBoundedSnapshot:true:
- The snapshot holds the latest ≤10 user turns / ≤75 items (O/threadHistoryPaging.ts:14-23).
- `runs` still contains EVERY queued, preparing, starting, running and waiting run, plus the runs in the window (O/ProjectionStore.ts:2796-2803).
- Without the flag you get the full projection, which can be very large.

Failures:
- Missing thread: Exit Failure OrchestrationV2GetThreadProjectionError "Failed to load orchestration V2 thread X" (S/ws.ts:726-742).
- Slow consumer: the server fails the stream with LiveStreamBufferError (limits 1000 items or 8 MiB, O/LiveStreamBudget.ts:17-18, 72-73). That error is not in subscribeThread's error union (C/rpc.ts:1574-1579), so it reaches the client as `{"_tag":"Exit","exit":{"_tag":"Failure","cause":[{"_tag":"Die",...}]}}`, not a tagged failure. A store error mid-stream arrives as OrchestrationV2GetThreadProjectionError, the same tag as a missing thread. Rule: once the first snapshot has arrived, any Exit and any stream end resubscribe with afterSequence; only an OrchestrationV2GetThreadProjectionError on the initial subscribe, before a snapshot, means the thread is missing.

Items (C/orchestrationV2.ts:3147-3170):
```go
type threadStreamItem struct {
	Kind             string          `json:"kind"`             // "synchronized" | "snapshot" | "event"
	Sequence         int64           `json:"sequence"`         // event: the cursor sits HERE, not in event
	Event            json.RawMessage `json:"event"`
	SnapshotSequence int64           `json:"snapshotSequence"` // snapshot: also the cursor
	Projection       json.RawMessage `json:"projection"`
	HasMoreHistory   bool            `json:"hasMoreHistory"`
}
type wireEventV2 struct { // C/orchestrationV2.ts:1489-1498
	ID                 string          `json:"id"`
	Type               string          `json:"type"`
	ThreadID           string          `json:"threadId"`
	RunID              string          `json:"runId"`
	NodeID             string          `json:"nodeId"`
	ProviderInstanceID string          `json:"providerInstanceId"`
	OccurredAt         string          `json:"occurredAt"` // ISO string
	Payload            json.RawMessage `json:"payload"`
}
```
- Snapshot: `{"kind":"snapshot","snapshotSequence":4120,"projection":{...},"historyCursor":null,"hasMoreHistory":false,"latestLocalTurnOrdinal":12}`
- Event: `{"kind":"event","sequence":4133,"event":{"id":"evt_9","type":"run.updated","threadId":"thr_01","runId":"run:thread:thr_01:ordinal:4","nodeId":"...","providerInstanceId":"claudeAgent","occurredAt":"2026-10-03T10:00:01.234Z","payload":{...Run}}}`
- Sequences are global and not contiguous. Running tool updates are coalesced per item id in 50 ms windows; terminal updates are never dropped (O/ThreadLiveEventCoalescer.ts:18-38).
- Skip unknown event types but still advance the cursor (C/orchestrationV2.ts:3108-3145).
- Events are strictly filtered to this threadId, so subagent child threads never appear.

Projection (C/orchestrationV2.ts:1640-1659). Only the fields Nexul needs:
```go
type projectionV2 struct {
	Thread           appThreadV2         `json:"thread"`
	Runs             []runV2             `json:"runs"`
	TurnItems        []turnItemV2        `json:"turnItems"`
	RuntimeRequests  []runtimeRequestV2  `json:"runtimeRequests"`
	ProviderSessions []providerSessionV2 `json:"providerSessions"`
}
type appThreadV2 struct { // C/orchestrationV2.ts:357-433
	ID                     string         `json:"id"`
	ProjectID              string         `json:"projectId"`
	ProviderInstanceID     string         `json:"providerInstanceId"`
	ModelSelection         modelSelection `json:"modelSelection"`
	RuntimeMode            string         `json:"runtimeMode"`
	HistoryOrigin          string         `json:"historyOrigin"` // absent | "native" | "v1_import"
	ActiveProviderThreadID *string        `json:"activeProviderThreadId"`
	ArchivedAt             *string        `json:"archivedAt"`
	DeletedAt              *string        `json:"deletedAt"`
	LimitRecovery          *struct {
		RunID      string `json:"runId"`
		ResetAt    string `json:"resetAt"`
		AutoResume bool   `json:"autoResume"`
	} `json:"limitRecovery"`
}
type runV2 struct { // C/orchestrationV2.ts:511-554
	ID                 string         `json:"id"`
	Ordinal            int            `json:"ordinal"`
	ProviderInstanceID string         `json:"providerInstanceId"`
	ModelSelection     modelSelection `json:"modelSelection"`
	UserMessageID      string         `json:"userMessageId"`
	RootNodeID         *string        `json:"rootNodeId"`
	Status             string         `json:"status"` // preparing|queued|starting|running|waiting|completed|interrupted|failed|cancelled|rolled_back (446-457)
	QueuePosition      *int           `json:"queuePosition"`
	QueueHeld          bool           `json:"queueHeld"`
	StartedAt          *string        `json:"startedAt"`
	CompletedAt        *string        `json:"completedAt"` // null while waiting
	CheckpointID       *string        `json:"checkpointId"`
}
type runtimeRequestV2 struct { // C/orchestrationV2.ts:941-959
	ID                 string `json:"id"`
	NodeID             string `json:"nodeId"`
	Kind               string `json:"kind"`   // command|file-read|file-change|mcp-elicitation|permission|dynamic_tool_call|user_input|auth_refresh
	Status             string `json:"status"` // pending|resolved|expired|cancelled
	ResponseCapability struct {
		Type              string `json:"type"` // live|message|not_resumable
		ProviderSessionID string `json:"providerSessionId"`
		Reason            string `json:"reason"`
	} `json:"responseCapability"`
}
type providerSessionV2 struct { // C/orchestrationV2.ts:693-704
	ID                 string  `json:"id"`
	ProviderInstanceID string  `json:"providerInstanceId"`
	Status             string  `json:"status"`
	LastError          *string `json:"lastError"`
}
type turnItemV2 struct { // base fields C/orchestrationV2.ts:1222-1240; variants 1265-1463
	ID           string  `json:"id"` // stable across updates → CallID
	ThreadID     string  `json:"threadId"`
	RunID        *string `json:"runId"`
	NodeID       *string `json:"nodeId"`
	ParentItemID *string `json:"parentItemId"`
	Ordinal      int     `json:"ordinal"`
	Status       string  `json:"status"` // idle|pending|running|waiting|completed|failed|cancelled|interrupted (1158-1167)
	Title        *string `json:"title"`
	StartedAt    *string `json:"startedAt"`
	CompletedAt  *string `json:"completedAt"`
	UpdatedAt    string  `json:"updatedAt"`
	Type         string  `json:"type"`
	ToolSource   *struct {
		Key  string `json:"key"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	} `json:"toolSource"`
	// assistant_message / user_message
	MessageID string `json:"messageId"`
	Text      string `json:"text"`
	Streaming bool   `json:"streaming"`
	// command_execution: input is a string; dynamic_tool: input is any JSON
	Input                  json.RawMessage `json:"input"`
	OutputIndicatesFailure bool            `json:"outputIndicatesFailure"`
	ExitCode               *int            `json:"exitCode"`
	// file_change
	FileName  string `json:"fileName"`
	Additions *int   `json:"additions"`
	Deletions *int   `json:"deletions"`
	Changes   []struct {
		Operation string `json:"operation"`
		Path      string `json:"path"`
		OldPath   string `json:"oldPath"`
	} `json:"changes"`
	// file_search / web_search
	Pattern  string          `json:"pattern"`
	Patterns []string        `json:"patterns"`
	Results  json.RawMessage `json:"results"`
	// dynamic_tool
	ToolName        *string         `json:"toolName"`
	ViewedImagePath string          `json:"viewedImagePath"`
	Output          json.RawMessage `json:"output"`
	// subagent (prompt is shared with approval_request)
	SubagentID    string  `json:"subagentId"`
	ChildThreadID *string `json:"childThreadId"`
	Prompt        string  `json:"prompt"`
	Progress      string  `json:"progress"`
	Result        *string `json:"result"`
	// user_input_request / approval_request
	RequestID    string              `json:"requestId"`
	Questions    []userInputQuestion `json:"questions"`
	ResponseMode string              `json:"responseMode"` // "message" = async question
	RequestKind  string              `json:"requestKind"`
	Options      []struct {
		Decision string `json:"decision"`
		Label    string `json:"label"`
		Warning  string `json:"warning"`
	} `json:"options"` // approval_request only
	// error
	Failure *struct {
		Class     string  `json:"class"`   // usage_limit|provider_error|transport_error|permission_error|validation_error|unknown
		Message   string  `json:"message"` // ≤4096
		Code      *string `json:"code"`
		Retryable *bool   `json:"retryable"`
		ResetAt   *string `json:"resetAt"`
	} `json:"failure"` // 1195-1208
	Retry *struct {
		Attempt     int  `json:"attempt"`
		MaxAttempts *int `json:"maxAttempts"`
		RetryDelayMs *int `json:"retryDelayMs"`
	} `json:"retry"`
	// system_notice / run_interrupt_request / run_interrupt_result
	Message string `json:"message"`
}
type userInputQuestion struct { // C/orchestrationV2.ts:1067-1081
	ID       string `json:"id"`
	Header   string `json:"header"`
	Question string `json:"question"`
	Options  []struct {
		Label       string `json:"label"`
		Description string `json:"description"`
		Value       string `json:"value"` // optional → fall back to Label
	} `json:"options"`
	MultiSelect       bool  `json:"multiSelect"`
	AllowCustomAnswer *bool `json:"allowCustomAnswer"`
	Required          *bool `json:"required"`
}
```

Events to decode (C/orchestrationV2.ts:1500-1638; JSON variants 2291-2425):
- run.created and run.updated: payload is the full Run. The first one carrying userMessageId == your messageId identifies your run. A queued send arrives as status "queued".
- turn-item.updated: payload is the full TurnItem. Upsert by id and filter on payload.runId == your run.
- runtime-request.updated: carries responseCapability. For Claude it arrives before the matching turn item (ClaudeAdapterV2.ts:6566-6583).
- provider-session.attached and provider-session.updated: payload.lastError.
- thread.deleted, thread.archived, thread.runtime-mode-updated, thread.model-selection-updated, thread.provider-switched: payload is the AppThread.
- subagent.updated: the Subagent (id, origin, runId, status, title, model, providerInstanceId, childThreadId, completionWake, completionDelivery, result). Needed for hand-offs; `projection.subagents` holds the same rows in a snapshot.
- message.updated with role user: carries delegatedCompletion.parentRunId, notification, scheduledTaskId and senderThreadId, which link wake runs to the run that caused them (C/orchestrationV2.ts:1029-1053). For a queued wake, run.created arrives before its message.updated.
- Skip everything else: run-attempt.*, node.updated, provider-thread.updated, provider-turn.updated (token usage), plan.updated, checkpoint.*, context-*.
- message.updated with role assistant duplicates the assistant text that turn-item.updated already carries (ClaudeAdapterV2.ts:2517-2548). Decode only one of the two.

Item types:
- assistant_message: {messageId, text, streaming}. `text` is the FULL cumulative text, not a delta, and a run can have several. Claude sends each block once with streaming:false. Codex streams, throttled.
- Tool-like items: command_execution, file_change, file_search, web_search, dynamic_tool, subagent. There is NO mcp_tool_call or image_view type.
  - MCP calls arrive as dynamic_tool:
    - Claude toolName is the raw tool name, e.g. "mcp__nexul__ticket_get", "Read", "Grep", "TodoWrite" (ClaudeAdapterV2.ts:1565-1606, 3780-3815).
    - Codex toolName is "server.tool", or "namespace.tool" for dynamic tools; MCP errors arrive in output.error (CodexAdapterV2.ts:457-500).
  - Image views: dynamic_tool.viewedImagePath, from Claude Read on an image.
  - command_execution, file_change and web_search carry NO tool name. Claude's Bash, Edit/Write/MultiEdit/NotebookEdit, and WebFetch/WebSearch all collapse into these types (ClaudeAdapterV2.ts:1573-1590). Synthesize Tool = "Shell", "Edit", "WebSearch", "Search", or "Agent" for subagent. Summary comes from:
    - command_execution: `input` (the command string)
    - file_change: fileName, and changes[].path
    - file_search: pattern
    - web_search: patterns[0]
    - dynamic_tool: title, else viewedImagePath, else a preview of input
  - Status mapping: pending, running or waiting → ActivityToolCall; anything else → ActivityToolResult.
  - Treat the step as failed when status=="failed", outputIndicatesFailure, exitCode≠0, or the dynamic_tool output has isError:true.
- user_input_request: {requestId, questions, responseMode?}. Its status is "waiting" while pending (ClaudeAdapterV2.ts:4632-4660). A live question leaves the run "running". Claude may also emit a dynamic_tool named "AskUserQuestion"; keep V1's mapping of that to ActivityQuestion.
- approval_request: {requestId, requestKind, prompt?, appName?, options?}.
- error: Use it only as the turn's root failure when status=="failed" and runId==run.id and nodeId==run.rootNodeId (packages/shared/src/orchestrationV2ThreadError.ts:9-33). An error item with status "running" is a provider retry still in flight (O/ProviderFailure.ts:205-250).
  - Usage limit: failure.class=="usage_limit" with failure.resetAt (ISO), plus thread.limitRecovery.
  - If there is no root error item, fall back to provider-session.lastError.

Stripped from the wire (O/WireProjection.ts:66-150). This applies to snapshots, events and HTTP alike:
- command_execution.output is removed. outputIndicatesFailure:true is added when the output looked failed or exitCode≠0.
- file_change diffStr, oldStr and newStr are removed.
- dynamic_tool.input becomes {summary, truncated:true} when its JSON exceeds 16 KiB.
- dynamic_tool.output is reduced to {isError?, threadId?, messageId?, taskId?, scheduledTaskId?, status?, thread?, threads?}, or dropped entirely (packages/shared/src/toolOutput.ts:97-160). Tool RESULTS are therefore not available.
- subagent prompt, progress and result are truncated at 32 KiB.
- handoff.summary is removed.
- No RPC returns the full output. Only getTurnDiff returns diffs.

Typical successful run: run.created(starting) → user_message item → run.updated(running) → tool items → assistant_message (streaming:false) → run.updated(waiting, completedAt:null) → checkpoint.captured + checkpoint item → run.updated(completed, completedAt, checkpointId).

=== 5. Project list and create ===
Over WS: orchestration.subscribeShell
- Input: {afterSequence?, requestCompletionMarker?} (C/orchestrationV2.ts:3025-3038).
- The first item is `{"kind":"snapshot","snapshot":{"schemaVersion":1,"snapshotSequence":N,"projects":[...],"threads":[...],"archivedThreads":[...]}}` (C/orchestrationV2.ts:1781-1797).
- Each project is OrchestrationProjectShell {id, title, workspaceRoot, repositoryIdentity?, defaultModelSelection, defaultThreadEnvMode?, autoPull?, faviconPath?, projectIcon?, scripts, createdAt, updatedAt} (C/orchestrationProject.ts:9-28). It has NO deletedAt; removed projects simply drop out.
- A metadata-only snapshot with threads:[] can follow the first one (S/ws.ts:985-1015). Take the first.
- Map to harness.Project{ID: id, Title: title, Path: workspaceRoot}. N/projects.go keeps working.

Over HTTP (V2 only): GET /api/projects
- Header: `Authorization: Bearer <token>` only, with no protocol header (C/environmentHttp.ts:61-64, 562-575). Scope: orchestration:read (S/project/http.ts:44-50).
- Returns ProjectSnapshot {projects:[Project], updatedAt}, where Project = {id, title, workspaceRoot, repositoryIdentity?, faviconPath?, projectIcon?, defaultModelSelection, defaultThreadEnvMode?, autoPull?, scripts, createdAt, updatedAt, deletedAt} (C/project.ts:137-159). Filter out rows with deletedAt != null.

Create: POST /api/projects/mutate
- Same Bearer header; scope orchestration:operate (S/project/http.ts:51-58). The same payload also works over WS as `projects.mutate` (C/rpc.ts:1154-1158).
- Body (C/project.ts:171-198):
  `{"type":"project.create","commandId":"cmd_p1","projectId":"prj_new","title":"nexul","workspaceRoot":"/path/to/project","createWorkspaceRootIfMissing":false}`
  scripts and defaultModelSelection are optional.
- 200 returns the Project.
- An active project already owns the root: ProjectConflictError → HTTP 400 `{"_tag":"EnvironmentRequestInvalidError","code":"invalid_request","reason":"invalid_command","traceId":"..."}` (S/project/http.ts:19-30; O/../project/ProjectService.ts:280-287). Re-GET /api/projects and match on workspaceRoot. The server normalizes the root (ProjectService.ts:197-216), so compare normalized paths.
- A missing root with createWorkspaceRootIfMissing:false fails normalization → 500 `{"code":"internal_error","reason":"project_mutation_failed"}`.
- Reusing a commandId replays its receipt.

=== 6. orchestration.getThreadProjection ===
- Input: {threadId}. Output: the bare OrchestrationV2ThreadProjection, not wrapped (C/orchestrationV2.ts:3019-3022; C/rpc.ts:1533-1540).
- The server reads a window (rowLimit 77) and applies the wire stripping (S/ws.ts:1853-1880).
- runs[] ALWAYS includes every queued, preparing, starting, running and waiting run (O/ProjectionStore.ts:2796-2803). It is a reliable fallback for a non-terminal runId. Terminal runs outside the window may be missing.
- Lookup order: runs.findLast(r.userMessageId == msgID), else runs.findLast(non-terminal).

=== 7. When to report Terminal ===
Evidence:
- A successful provider turn is persisted as "waiting" with completedAt:null (O/RunExecutionService.ts:635-650). Only success maps to waiting, and a pending question never sets a run to waiting.
- The server itself calls waiting "post-terminal drain, so its agent turn is over" (O/Orchestrator.ts:440-449), and "provider-finished with checkpoint capture still pending" (O/ThreadForkService.ts:37).
- Checkpoint capture then flips waiting to completed (O/CheckpointCaptureService.ts:201-216).
- Capture failures inside the checkpoint store are swallowed into a "missing" checkpoint, so the run still completes (O/CheckpointService.ts:405-430).
- The run stays waiting only if the capture target is incomplete (CheckpointCaptureService.ts:88-102), until restart recovery handles it (O/ProviderRuntimeRecoveryService.ts:185-210).
- isTerminalRunStatus excludes waiting (O/ThreadManagementService.ts:335-345), and so does T3's MCP thread wait.

What the T3 UI shows while a run is waiting:
- The sidebar shows "working" (apps/web/src/components/Sidebar.logic.ts:956-959).
- The phase is "running" (apps/web/src/session-logic.ts:1013), and the run is not settled (session-logic.ts:194-207).
- Stop is not offered (client-runtime state/threadExecution.ts:35).

While a run is waiting, the next Nexul dispatch becomes a QUEUED run. It starts on its own when the run completes (startNextQueuedRun, O/Orchestrator.ts:1180-1215).

DECISION: report Done at the first run.updated(status "waiting") for the watched run. If the waiting update was missed (resume, snapshot), report Done at "completed". Emit Terminal exactly once per run and ignore the second run.updated, which carries the checkpointId (CheckpointCaptureService.ts:201-216). By "waiting", the agent's reply is final; checkpointing is T3's own rollback bookkeeping. Waiting for "completed" adds git-snapshot latency and a rare risk of hanging.

Status mapping:
- waiting, completed → done
- interrupted → interrupted
- cancelled (queued run cancelled, or restart recovery) → interrupted, with a note
- rolled_back → interrupted
- failed → error. LastError = the root error item's failure.message. For class usage_limit, name the resetAt.

While the watched run sits in "queued", surface an ActivityNote. If queueHeld is true, it needs queue.resume.

## Pitfalls

Implementation pitfalls for the V2 path in internal/t3client:

1. **Dial.** Add `orchestrationProtocol=2` on every /ws dial, including the Hold presence socket. Check `resp.StatusCode==426` from websocket.Dial and return a NON-retryable "T3 Code speaks protocol N" error; today it is Retryable and the presence keeper loops forever. Probe the descriptor on every connect, because V1 and V2 can share port 3773.

2. **Required command fields.**
   - Always send createdBy ("user"), creationSource ("web") and dispatchMode. The server forces createdBy to "user".
   - Send explicit `branch:null` and `worktreePath:null`.
   - Never send createdAt, `role`, or runtimeMode on message.dispatch.
   - A non-empty title is required.
   - Option values must be a non-empty string or a bool.

3. **dispatchMode.**
   - Use `queue_after_active` (or `start_immediately`; they behave the same) with no deliveryIntent.
   - Never `defer_start`: it creates a "preparing" run that waits for prepared-run.release and never starts.
   - Never steer or restart: no run carries your messageId, so the watch hangs.

4. **Attachments.** Mint messageId first, call assets.persistChatAttachments with the same threadId and messageId, then dispatch the returned refs. Inline dataUrl is rejected. Filter to gif/jpeg/png/webp up front.

5. **Wait for the first snapshot before dispatching.** Subscribe with acceptBoundedSnapshot:true, wait for the snapshot item, then decide:
   - Send the Full prompt when historyOrigin=="v1_import" and runs is empty, or when snapshot thread.providerInstanceId != target.Provider (that dispatch is a provider switch).
   - Send thread.runtime-mode.set if thread.runtimeMode != full-access.
   - Only after that, call message.dispatch.
   The current subscribeAndStart fires StartTurn without waiting.

6. **Run identity.** Your run is the run.created/run.updated whose userMessageId == your messageId. Filter turn-item events on payload.runId. Store threadID→runID in memory for Interrupt; fall back to getThreadProjection, whose runs always include non-terminal runs. Ignore runs Nexul didn't start: wake runs, `async-answer:<requestId>` runs, delegated completions.

7. **Terminal.**
   - Done at "waiting", exactly once.
   - Map cancelled and rolled_back to interrupted.
   - On failed, read the error item with status=="failed" and nodeId==run.rootNodeId. Error items with status "running" are retries, not failures.
   - Show a note while your run is "queued" (and queueHeld, if set).

8. **Interrupt.**
   - Send run.interrupt for preparing, starting or running runs, WITHOUT holdQueue. holdQueue freezes later Nexul turns until queue.resume.
   - Send queued-run.cancel for queued runs.
   - A waiting run is interruptible only when it is the thread's latest run and background work (for example a running subagent) is pending (O/Orchestrator.ts:7907-7943); otherwise the server replies "not interruptible". Interrupting the parent disposes its delegated-completion cohort but does not interrupt delegated child threads (O/Orchestrator.ts:1984-2065); T3's own cancel_task interrupts the child run itself (S/mcp/OrchestratorMcpService.ts:1520-1545).

9. **Answers.** Key them by question.id; for Claude that id is the full question text. Use `option.value`, falling back to `option.label`. For message-mode requests (responseMode "message" or responseCapability.type=="message"), send one non-empty STRING per question and join multi-selects, or the server rejects it. Do not start your own follow-up turn: the server dispatches `async-answer:<requestId>`. not_resumable/expired requests cannot be answered.

10. **Tool rows.**
    - command_execution, file_change and web_search have no tool name; synthesize one (Shell, Edit, WebSearch).
    - MCP tools arrive as dynamic_tool: Claude names them `mcp__server__tool`, Codex `server.tool`.
    - Use the turn item `id` as CallID.
    - Command output, diffs and tool results are stripped server-side. Detail can only carry the input (truncated past 16 KiB) and a compact {isError,...}, so drop the V1 result-detail expectations.

11. **Stream hygiene.**
    - The cursor is item.sequence, not event.sequence.
    - Skip unknown event and item types but still advance the cursor.
    - Decode one of message.updated / turn-item.updated for assistant text, not both (they duplicate).
    - assistant text is cumulative, so do not sum it.
    - After the first snapshot, any Exit (LiveStreamBufferError arrives as a Die) means resubscribe with afterSequence, not TurnError. Only a failure of the initial subscribe means the thread is missing. A deleted thread is a soft delete: the subscribe succeeds and the snapshot has thread.deletedAt set.

12. **Projects.** Shell project rows have no deletedAt. For create-from-folder, use POST /api/projects/mutate (or WS projects.mutate) with project.create. On a 400 invalid_command (root already registered), re-list and match the normalized workspaceRoot.

## Still unknown

- Exact per-adapter ordering of the root `error` turn item relative to run.updated(failed). If failed can arrive first, LastError must be resolved late (from a later item, provider-session lastError, or a getThreadProjection read). I did not trace this for every adapter.
- Whether a failed run always has either a root error item or a provider-session lastError. Some setup failures may surface only in shell lastError.
- Claude AskUserQuestion options with an empty description violate TrimmedNonEmptyString on the wire schema (ClaudeAdapterV2.ts:2811-2816 vs orchestrationV2.ts:1074). Whether the server's encode fails, stripping or killing that event, is untested.
- What Nexul's watch sees when a T3 user promotes Nexul's queued run to steer or cancels it from T3's UI. The run disappears or is cancelled, and no completion carries Nexul's messageId.
- What happens on dispatch to an antigravity provider instance: it is listed in builtInDrivers but has no V2 adapter.
- Whether Codex streaming assistant_message items always reach streaming:false before run.updated(waiting). This is inferred from provider turn completion, not traced line by line.
- Everything here comes from reading source. No shape was checked against a running 0.0.46 nightly.