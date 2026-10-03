# 07 — Show protocol-2 steps in chat and trails

**What to build:** A turn-item mapper in its own file (`items.go`), with no run filter (the watch
decides which items it passes; ticket 13 passes a child thread's items, whose `runId` is null). Spec
decision 18: CallID is the item id; pending/running/waiting → `ActivityToolCall`, otherwise
`ActivityToolResult`; names `Shell`, `Edit`, `Grep`, `WebSearch`, `Agent`; Claude `mcp__` names
kept, Codex `server.tool` → `server · tool`; Summary from the command, file path, pattern, viewed image
path, then an input preview, then the title; " · failed" on status failed, a failure output flag, a
nonzero exit code or an error result; Detail is the input through `harness.CapDetail`.
`user_input_request` → `Question` (once, keyed by the app-owned request id; Codex sends empty option
descriptions). `approval_request` → `Approval`, auto-declined through `runtime-request.respond` as
protocol 1 does. The watch from ticket 05 passes its run's items through it.

**Blocked by:** 05

**Status:** done

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, and `research/protocol-2-wire.md`.

- [x] Table tests per item type over the fixtures: `nexul.ticket_get` → `nexul · ticket_get`; a failed command is marked failed; one item updated twice keeps its CallID; empty option descriptions decode; an item with `runId: null` maps
- [x] The watch ignores items of other runs; Shell, Edit, Grep and MCP rows parse through the existing `web/src/models/Trail.tsx` (no web change)

## Comments

- **Where it lives.** `items.go` holds `turnItem` (moved out of `watch.go`) and `mapItem(it) (harness.Update, bool)`,
  a pure function of one item with no run filter. `watch.item` sends every item of the turn's run that is not an
  assistant message or a failed error through `watch.step`, which emits a step only when its `Activity` changed (so a
  resubscribe snapshot re-sending items emits nothing) and a Question or Approval once per request id.
- **Auto-decline.** `watch.step` queues an approval's request id in `w.declines`; `pump.fold` calls
  `pump.declineApprovals` right after `apply`, before forwarding, which sends `runtime-request.respond` with
  `decision: "decline"` through `turn.decline` on the turn's own connection. A refused decline is logged at Warn and
  the turn carries on, as protocol 1 ignores it.
- **Ticket 08.** `runtimeRequestRespond` in `turn.go` has `decision` only, always sent; Answer needs an `answers`
  field and `decision` with `omitempty`. `refused` already wraps T3's message.
- **Ticket 13.** Call `mapItem` on each child item. It returns a Question for an open `user_input_request` and an
  Approval for an open `approval_request`; for a child, 13 turns the question into a step plus the parent note and
  decides whether to decline. A `subagent` item on the parent maps to an `Agent` step keyed by the item id, beside
  the pill. `testdata/subagent-child-thread.source-65731f986b.ndjson` holds a Claude subagent's own thread, whose items
  carry `runId: null`.
- **Judgment calls.**
  - The `server.tool` → `server · tool` rule covers ACP providers too (Grok runs on T3's ACP adapter), because T3
    names their MCP calls the same way. ACP's other tools carry the call's own title as `toolName`, so a name with
    whitespace stays as it is. Claude's names never hold a dot, so no `mcp__` check is needed.
  - No title fallback for a null `toolName`: ACP sends null only when the title is null too.
  - Summary order: `fileName`, then `pattern` (file_search) or `patterns[0]` (web_search), `viewedImagePath`, the
    input (a command's string as is, other input as compact JSON, `{}` and null skipped), the title, and last the
    subagent's prompt, since `Subagent.title` is nullable. Then " · failed".
  - Detail is `{"input": …}`, the shape the web's step detail labels "Arguments", through `harness.CapDetail`; empty
    when the item has no input. `At` is the item's `updatedAt`.
  - A question or approval is raised only while its item is `pending`, `running` or `waiting`: one first seen
    answered or declined (a resubscribe snapshot) raises nothing and is not declined.
  - Claude's `AskUserQuestion` dynamic tool stays an `ActivityQuestion` step, as on protocol 1; the Question itself
    comes from `user_input_request`. The approval's summary is its `prompt`, else its title.
  - A `defer_start` run's "Preparing workspace" item maps to a Shell step. Nexul never uses `defer_start`;
    `TestWatch_AnotherRunCancelledMidTurn_IsNotTheTurnsEnd` now asserts only on terminals.
- **Web.** Checked by running the web's `stepLabel` on the mapper's outputs: Shell reads as the command, Edit as
  `Edit: <path>`, Grep as `Grep: <pattern>`, both MCP shapes as `Nexul · <tool>`, and " · failed" marks the row
  failed. Not kept as a test: `web/src/models/Trail.test.tsx` already covers those tool shapes.
- **Fixtures.** `tool-steps.source-65731f986b.ndjson` is built by hand from T3 at `65731f986b`: Nexul's run on
  Claude (Bash, Edit, Grep, an MCP call, an image Read, WebSearch, AskUserQuestion with its question, an approval, a
  subagent), queued behind a run typed in T3, then runs typed in T3 on Codex (a nonzero exit, an MCP error, an async
  question with empty option descriptions) and Grok over ACP (a file search, an MCP call, a titled tool). Ticket 16
  should check them against captures from logged-in providers. On today's wire a nonzero exit always comes with
  `outputIndicatesFailure`, and an MCP error with status `failed`; the mapper checks each signal on its own anyway.

