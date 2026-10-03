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

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, and `research/protocol-2-wire.md`.

- [ ] Table tests per item type over the fixtures: `nexul.ticket_get` → `nexul · ticket_get`; a failed command is marked failed; one item updated twice keeps its CallID; empty option descriptions decode; an item with `runId: null` maps
- [ ] The watch ignores items of other runs; Shell, Edit, Grep and MCP rows parse through the existing `web/src/models/Trail.tsx` (no web change)
